package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"steadyphoto/internal/database"
	"steadyphoto/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	dbURL := flag.String("db", "", "PostgreSQL connection URL")
	validate := flag.Bool("validate", false, "Validate thumbnail files on disk and update job statuses accordingly")
	flag.Parse()

	if *dbURL == "" {
		*dbURL = os.Getenv("DATABASE_URL")
	}

	if *dbURL == "" {
		log.Fatal("Database URL must be provided via -db flag or DATABASE_URL environment variable")
	}

	// 1. Connect to DB
	db, err := sqlx.Connect("postgres", *dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 2. Initialize Repositories
	photoRepo := database.NewPostgresMediaRepository(db)
	jobRepo := database.NewPostgresJobRepository(db)

	ctx := context.Background()

	// 3. Get all photos
	allMedia, _, err := photoRepo.List(ctx, 10000, 0, nil)
	if err != nil {
		log.Fatalf("Failed to list media: %v", err)
	}

	// Filter for photos only (MediaTypePhoto or empty MediaType treated as photo)
	var photos []*domain.Media
	for _, m := range allMedia {
		if m.MediaType == domain.MediaTypePhoto || m.MediaType == "" {
			photos = append(photos, m)
		}
	}

	fmt.Printf("Found %d photos in database.\n", len(photos))

	createdCount := 0
	updatedCount := 0
	skippedCount := 0
	validatedCount := 0

	for _, photo := range photos {
		// Check if ANY job already exists for this photo (regardless of status)
		var jobExists bool
		var existingJobID uuid.UUID
		query := `SELECT EXISTS(SELECT 1 FROM jobs WHERE media_id = $1 AND job_type = $2)`
		err := db.GetContext(ctx, &jobExists, query, photo.ID, string(domain.JobTypeThumbnail))
		if err != nil {
			log.Printf("Error checking job status for photo %s: %v", photo.ID, err)
			continue
		}

		// Get the existing job ID if it exists
		if jobExists {
			err = db.GetContext(ctx, &existingJobID, `SELECT id FROM jobs WHERE media_id = $1 AND job_type = $2 LIMIT 1`, photo.ID, string(domain.JobTypeThumbnail))
			if err != nil {
				log.Printf("Error getting existing job ID for photo %s: %v", photo.ID, err)
				continue
			}
		}

		if *validate {
			// Validate mode: Check if thumbnail file exists on disk
			fullThumbPath := getThumbnailPath(photo)
			thumbExists, thumbSize := checkThumbnailFile(fullThumbPath)

			if !thumbExists || thumbSize == 0 {
				// Thumbnail missing or empty - need to create/update job to pending
				if !jobExists {
					// Create new pending job
					job := createPendingJob(photo.ID)
					if err := jobRepo.Create(ctx, job); err != nil {
						log.Printf("Failed to create job for photo %s: %v", photo.ID, err)
						continue
					}
					createdCount++
					fmt.Printf("Created pending job for photo %s (thumb missing)\n", photo.ID)
				} else {
					// Update existing job to pending
					if err := jobRepo.UpdateStatus(ctx, existingJobID, domain.JobStatusPending, ""); err != nil {
						log.Printf("Failed to update job to pending for photo %s: %v", photo.ID, err)
						continue
					}
					updatedCount++
					fmt.Printf("Updated job to pending for photo %s (thumb missing)\n", photo.ID)
				}
			} else {
				// Thumbnail exists and is valid - ensure job is completed
				if !jobExists {
					// Create new completed job
					job := createCompletedJob(photo.ID)
					if err := jobRepo.Create(ctx, job); err != nil {
						log.Printf("Failed to create job for photo %s: %v", photo.ID, err)
						continue
					}
					createdCount++
					fmt.Printf("Created completed job for photo %s (thumb exists)\n", photo.ID)
				} else {
					// Update existing job to completed
					if err := jobRepo.UpdateStatus(ctx, existingJobID, domain.JobStatusCompleted, ""); err != nil {
						log.Printf("Failed to update job to completed for photo %s: %v", photo.ID, err)
						continue
					}
					updatedCount++
					fmt.Printf("Updated job to completed for photo %s (thumb exists)\n", photo.ID)
				}
			}
			validatedCount++
		} else {
			// Original repair mode: Only create jobs if they don't exist
			if !jobExists {
				job := createPendingJob(photo.ID)
				if err := jobRepo.Create(ctx, job); err != nil {
					log.Printf("Failed to create job for photo %s: %v", photo.ID, err)
					continue
				}
				createdCount++
			} else {
				skippedCount++
			}
		}
	}

	fmt.Printf("\n✅ Repair Complete!\n")
	if *validate {
		fmt.Printf("Validated: %d photos\n", validatedCount)
		fmt.Printf("New jobs created: %d\n", createdCount)
		fmt.Printf("Jobs updated: %d\n", updatedCount)
	} else {
		fmt.Printf("New jobs created: %d\n", createdCount)
		fmt.Printf("Jobs already existing: %d\n", skippedCount)
	}
}

// getThumbnailPath constructs the thumbnail file path for a photo
// Pattern: storage/.thumbnails/{user_id}/{YYYY}/{MM}/{DD}/{media_id}_thumb.webp
func getThumbnailPath(photo *domain.Media) string {
	storageDir := os.Getenv("STORAGE_DIR")
	if storageDir == "" {
		storageDir = "storage"
	}

	thumbRoot := filepath.Join(storageDir, ".thumbnails")
	if photo.UserID != uuid.Nil {
		thumbRoot = filepath.Join(thumbRoot, photo.UserID.String())
	}

	// Use captured_at for date-based directory structure
	// If captured_at is zero time, fall back to created_at
	capturedAt := photo.CapturedAt
	if capturedAt.IsZero() {
		capturedAt = photo.CreatedAt
	}

	thumbFilename := fmt.Sprintf("%s_thumb.webp", photo.ID.String())
	thumbPath := filepath.Join(thumbRoot, capturedAt.Format("2006"), capturedAt.Format("01"), capturedAt.Format("02"), thumbFilename)

	return thumbPath
}

// checkThumbnailFile checks if a thumbnail file exists and returns its size
func checkThumbnailFile(path string) (exists bool, size int64) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, 0
		}
		log.Printf("Error checking file %s: %v", path, err)
		return false, 0
	}
	return true, info.Size()
}

// createPendingJob creates a new job with pending status
func createPendingJob(mediaID uuid.UUID) *domain.Job {
	return &domain.Job{
		ID:        uuid.New(),
		Type:      domain.JobTypeThumbnail,
		Status:    domain.JobStatusPending,
		MediaID:   mediaID,
		CreatedAt: time.Now(),
	}
}

// createCompletedJob creates a new job with completed status
func createCompletedJob(mediaID uuid.UUID) *domain.Job {
	return &domain.Job{
		ID:        uuid.New(),
		Type:      domain.JobTypeThumbnail,
		Status:    domain.JobStatusCompleted,
		MediaID:   mediaID,
		CreatedAt: time.Now(),
	}
}