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
  -v                    Verbose output
  -json                 Output as JSON
  -dry-run              Show what would be updated without writing to DB
  -force                Force update even if metadata already exists
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

  # Force update all media with verbose output
  exifupdater -force -v

  # JSON output for programmatic use
  exifupdater -json
`

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Failed to load .env file: %v", err)
	}

	// CLI flags
	verbose := flag.Bool("v", false, "Verbose output")
	jsonOutput := flag.Bool("json", false, "Output as JSON")
	dryRun := flag.Bool("dry-run", false, "Show what would be updated without writing to DB")
	forceUpdate := flag.Bool("force", false, "Force update even if metadata already exists")
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
			log.Printf("[%d/%d] Processing: %s", i+1, len(mediaList), media.Path)
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

		// Check if metadata already exists and force is not set
		if !*forceUpdate && hasExistingMetadata(media.Metadata) {
			if *verbose {
				log.Printf("  Metadata already exists, skipping (use -force to override)")
			}
			skipped++
			continue
		}

		// Read EXIF data from file
		file, openErr := os.Open(fullPath)
		if openErr != nil {
			log.Printf("  Failed to open file: %v", openErr)
			errors++
			continue
		}

		exifInfo, readErr := exifReader.ReadExif(file)
		file.Close()

		if readErr != nil {
			log.Printf("  Failed to read EXIF: %v", readErr)
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

		processed++
		updated++

		result := ExifUpdateResult{
			ID:          media.ID.String(),
			Path:        media.Path,
			Orientation: int(exifInfo.Orientation),
			Tags:        exifInfo.Tags,
			Metadata:    metadata,
		}
		results = append(results, result)

		if *dryRun {
			if *verbose {
				log.Printf("  [DRY-RUN] Would update: %s", media.Path)
			}
			continue
		}

		// Update the media record
		media.Metadata = metadata
		media.UpdatedAt = time.Now()

		updateErr := mediaRepo.Update(ctx, media)
		if updateErr != nil {
			log.Printf("  Failed to update database: %v", updateErr)
			errors++
			continue
		}

		if *verbose {
			log.Printf("  Updated metadata for: %s", media.Path)
		}
	}

	// Print results
	if *jsonOutput {
		data, jsonErr := json.MarshalIndent(results, "", "  ")
		if jsonErr != nil {
			log.Fatalf("Failed to marshal results to JSON: %v", jsonErr)
		}
		fmt.Println(string(data))
	} else {
		printTextSummary(processed, updated, skipped, errors, mode)
	}

	log.Printf("\n=== EXIF Update Complete ===")
	log.Printf("Mode: %s", mode)
	log.Printf("Processed: %d", processed)
	log.Printf("Updated: %d", updated)
	log.Printf("Skipped: %d", skipped)
	log.Printf("Errors: %d", errors)
}

// hasExistingMetadata checks if media has existing EXIF metadata
func hasExistingMetadata(metadata domain.Metadata) bool {
	if metadata == nil {
		return false
	}
	return len(metadata) > 0
}

// ExifUpdateResult represents the result of an EXIF update
type ExifUpdateResult struct {
	ID          string                `json:"id"`
	Path        string                `json:"path"`
	Orientation int                   `json:"orientation"`
	Tags        []processor.ExifTag   `json:"tags,omitempty"`
	Metadata    domain.Metadata       `json:"metadata"`
}

// buildMetadataFromExif converts ExifInfo to Metadata map
func buildMetadataFromExif(info *processor.ExifInfo) domain.Metadata {
	metadata := domain.Metadata{}

	// Add orientation if not normal
	if info.Orientation != processor.OrientationNormal {
		metadata["orientation"] = fmt.Sprintf("%d", int(info.Orientation))
	}

	// Add other EXIF tags
	for _, tag := range info.Tags {
		// Skip orientation as it's already handled
		if strings.EqualFold(tag.Tag, "Orientation") {
			continue
		}

		// Add tag to metadata (use tag name as key)
		if _, exists := metadata[tag.Tag]; !exists {
			metadata[tag.Tag] = tag.Value
		}
	}

	return metadata
}

// printTextSummary prints a human-readable summary
func printTextSummary(processed, updated, skipped, errors int, mode string) {
	fmt.Println("\n=== EXIF Update Summary ===")
	fmt.Printf("Mode: %s\n", mode)
	fmt.Printf("Processed: %d\n", processed)
	fmt.Printf("Updated: %d\n", updated)
	fmt.Printf("Skipped: %d\n", skipped)
	fmt.Printf("Errors: %d\n", errors)
}
