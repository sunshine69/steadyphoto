package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"steadyphoto/internal/database"
	"steadyphoto/internal/domain"
	"steadyphoto/internal/processor"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

const (
	defaultAPIPort = "8081"
	storageEnv     = "STORAGE_DIR"
	defaultStorage = "./storage"

	dbURLKey = "DATABASE_URL"
)

const usage = `ExifUpdater CLI Tool

Usage:
  exifupdater [flags]

Flags:
  -verbose              Verbose output (JSON per-item + detailed logs)
  -dry-run              Show what would be updated without writing to DB
  -media-id <uuid>      Process a specific media item by UUID
  -user <email>         Process all media owned by user (by email)
  -help                 Show this help message

Modes:
  No flags              Scan all media in the database
  -media-id <uuid>      Process a single media item
  -user <email>         Process all media owned by a user

Examples:
  # Show all available options
  exifupdater -help

  # Dry-run scan of all media
  exifupdater -dry-run

  # Process a specific media item
  exifupdater -media-id a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11

  # Process all media for a user
  exifupdater -user admin@steadyphoto.com

  # Verbose output with JSON per item
  exifupdater -verbose
`

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Failed to load .env file: %v", err)
	}

	// CLI flags
	verbose := flag.Bool("verbose", false, "Verbose output (JSON per-item + detailed logs)")
	dryRun := flag.Bool("dry-run", false, "Show what would be updated without writing to DB")
	mediaID := flag.String("media-id", "", "Process a specific media item by UUID")
	userEmail := flag.String("user", "", "Process all media owned by user (by email)")
	showHelp := flag.Bool("help", false, "Show usage examples")
	flag.Parse()

	// Show help if requested
	if *showHelp {
		fmt.Print(usage)
		return
	}

	// Validate flags
	if *mediaID != "" && *userEmail != "" {
		log.Fatal("Cannot specify both -media-id and -user (mutually exclusive)")
	}

	// Get database connection
	dbURL := os.Getenv(dbURLKey)
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	// Get storage root
	storageRoot := os.Getenv(storageEnv)
	if storageRoot == "" {
		storageRoot = defaultStorage
	}

	log.Printf("Connecting to database...")
	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Printf("Database connected successfully")

	// Create repositories
	mediaRepo := database.NewPostgresMediaRepository(db)
	userRepo := database.NewPostgresUserRepository(db)

	// Create EXIF reader
	exifReader := processor.NewExifReader()

	// Determine which media items to process
	ctx := context.Background()
	var mediaList []*domain.Media
	mode := "all media"

	switch {
	case *mediaID != "":
		// Single media item mode
		uid, parseErr := uuid.Parse(*mediaID)
		if parseErr != nil {
			log.Fatalf("Invalid media ID format '%s': %v", *mediaID, parseErr)
		}

		media, getErr := mediaRepo.GetByID(ctx, uid, nil)
		if getErr != nil {
			log.Fatalf("Failed to get media by ID %s: %v", *mediaID, getErr)
		}

		mediaList = []*domain.Media{media}
		mode = fmt.Sprintf("single media: %s", media.Path)
		log.Printf("Processing single media item: %s", media.Path)

	case *userEmail != "":
		// User mode - find user and get all their media
		user, userErr := userRepo.GetByEmail(ctx, *userEmail)
		if userErr != nil {
			log.Fatalf("Failed to find user with email '%s': %v", *userEmail, userErr)
		}

		userMediaList, total, userListErr := mediaRepo.List(ctx, 1000000, 0, &user.ID)
		if userListErr != nil {
			log.Fatalf("Failed to list media for user: %v", userListErr)
		}

		mediaList = userMediaList
		mode = fmt.Sprintf("user: %s (%d items)", *userEmail, total)
		log.Printf("Processing %d media items for user: %s", total, *userEmail)

	default:
		// No flags - scan all media in the database
		allMedia, total, listErr := mediaRepo.ListAll(ctx, 1000000, 0)
		if listErr != nil {
			log.Fatalf("Failed to list all media: %v", listErr)
		}

		mediaList = allMedia
		mode = fmt.Sprintf("all media in database (%d items)", total)
		log.Printf("Processing all %d media items in database", total)
	}

	// Process each media item
	processed := 0
	updated := 0
	skipped := 0
	errors := 0
	var results []ExifUpdateResult

	for i, media := range mediaList {
		if *verbose {
			fmt.Printf("Processing %d/%d: %s\n", i+1, len(mediaList), media.Path)
		}

		// Build full file path
		fullPath := media.Path
		if !strings.HasPrefix(fullPath, "/") {
			fullPath = storageRoot + "/" + fullPath
		}

		// Check if file exists
		if _, statErr := os.Stat(fullPath); os.IsNotExist(statErr) {
			if *verbose {
				log.Printf("  File not found, skipping: %s", fullPath)
			}
			skipped++
			continue
		}

		// Read EXIF data from file
		file, openErr := os.Open(fullPath)
		if openErr != nil {
			if *verbose {
				log.Printf("  Failed to open file: %v", openErr)
			}
			errors++
			continue
		}

		exifInfo, readErr := exifReader.ReadExif(file)
		file.Close()

		if readErr != nil {
			if *verbose {
				log.Printf("  Failed to read EXIF: %v", readErr)
			}
			errors++
			continue
		}

		if exifInfo == nil {
			if *verbose {
				log.Printf("  No EXIF data found")
			}
			processed++
			continue
		}

		// Build metadata from EXIF info
		metadata := buildMetadataFromExif(exifInfo)

		if *verbose {
			// Verbose: print per-item JSON
			resultJSON, _ := json.MarshalIndent(metadata, "  ", "  ")
			fmt.Printf("  %s: metadata extracted (%d tags)\n", media.Path, len(metadata))
			fmt.Printf("  metadata: %s\n", string(resultJSON))
		}

		processed++
		updated++

		// Store for final summary
		result := ExifUpdateResult{
			ID:       media.ID.String(),
			Path:     media.Path,
			Metadata: metadata,
		}
		results = append(results, result)

		if *dryRun {
			continue
		}

		// Update the media record
		media.Metadata = metadata
		media.UpdatedAt = time.Now()

		updateErr := mediaRepo.Update(ctx, media)
		if updateErr != nil {
			if *verbose {
				log.Printf("  Failed to update database: %v", updateErr)
			}
			errors++
			continue
		}
	}

	// Only output JSON if verbose mode
	if *verbose {
		// Output clean JSON array - exactly what will be saved to DB
		output := map[string]interface{}{
			"mode":       mode,
			"totalItems": len(mediaList),
			"processed":  processed,
			"updated":    updated,
			"skipped":    skipped,
			"errors":     errors,
			"mediaItems": results,
		}
		jsonBytes, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(jsonBytes))
	}

	// Print final stats (always shown)
	log.Printf("\n=== EXIF Update Complete ===")
	log.Printf("Mode: %s", mode)
	log.Printf("Total items: %d", len(mediaList))
	log.Printf("Processed: %d", processed)
	log.Printf("Updated: %d", updated)
	log.Printf("Skipped: %d", skipped)
	log.Printf("Errors: %d", errors)
}

// ExifUpdateResult represents the result of an EXIF update
type ExifUpdateResult struct {
	ID       string          `json:"id"`
	Path     string          `json:"path"`
	Metadata domain.Metadata `json:"metadata"`
}

// buildMetadataFromExif converts ExifInfo to Metadata map
func buildMetadataFromExif(info *processor.ExifInfo) domain.Metadata {
	metadata := domain.Metadata{}

	// Add orientation if not normal
	if info.Orientation != processor.OrientationNormal {
		metadata["orientation"] = fmt.Sprintf("%d", int(info.Orientation))
	}

	// Extract DateTimeOriginal for display purposes
	var dateTimeOriginal string

	// Add other EXIF tags
	for _, tag := range info.Tags {
		// Skip orientation as it's already handled
		if strings.EqualFold(tag.Tag, "Orientation") {
			continue
		}

		// Capture DateTimeOriginal if present
		if strings.EqualFold(tag.Tag, "DateTimeOriginal") && dateTimeOriginal == "" {
			dateTimeOriginal = tag.Value
		}

		// Add tag to metadata (use tag name as key)
		if _, exists := metadata[tag.Tag]; !exists {
			metadata[tag.Tag] = tag.Value
		}
	}

	// Store normalized GPS fields so location search works.
	// NOTE: info.GPSLatitude/GPSLongitude are already signed by parseGPSCoordinate
	// (which applies S/W ref), so we just use them directly without re-applying signs.
	if info.GPSLatitude != 0 && info.GPSLongitude != 0 {
		metadata["gps_latitude"] = fmt.Sprintf("%.6f", info.GPSLatitude)
		metadata["gps_longitude"] = fmt.Sprintf("%.6f", info.GPSLongitude)
		metadata["gps_altitude"] = fmt.Sprintf("%.1f", info.GPSAltitude)
	}

	// Set DateTimeOriginal: prefer the actual EXIF tag, fallback to ModifyDate
	if dateTimeOriginal != "" {
		metadata["DateTimeOriginal"] = dateTimeOriginal
	} else if md, ok := metadata["ModifyDate"]; ok {
		metadata["DateTimeOriginal"] = md
	}

	return metadata
}

