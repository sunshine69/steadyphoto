package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"steadyphoto/internal/ai"
	"steadyphoto/internal/database"
	"steadyphoto/internal/domain"
	"steadyphoto/internal/processor"
)

func main() {
	// CLI flags
	userEmailFlag := flag.String("user", "", "User email to process all media for")
	mediaIDFlag := flag.String("media-id", "", "Specific media ID to generate thumbnail for (single mode)")
	flag.Parse()

	if *userEmailFlag != "" && *mediaIDFlag != "" {
		log.Fatal("cannot use both -user and -media-id flags together")
	}

	// Load .env from the project root
	absPath, err := filepath.Abs(".")
	if err != nil {
		log.Fatalf("failed to get absolute path: %v", err)
	}
	log.Printf("Loading .env from: %s", absPath)

	envFile := filepath.Join(absPath, ".env")
	if err := godotenv.Load(envFile); err != nil {
		log.Printf("[WARN] Failed to load %s: %v (using env vars)", envFile, err)
	}

	// Read config from env (now populated from .env)
	dbURL := getEnv("DATABASE_URL", "")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set in .env or environment")
	}

	storageRoot := getEnv("STORAGE_ROOT", "storage")
	thumbRoot := getEnv("THUMBNAIL_ROOT", filepath.Join(storageRoot, ".thumbnails"))

	log.Printf("DB URL: %s", maskDBPassword(dbURL))
	log.Printf("Storage root: %s", storageRoot)
	log.Printf("Thumbnail root: %s", thumbRoot)

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	jobRepo := database.NewPostgresJobRepository(db)
	mediaRepo := database.NewPostgresMediaRepository(db)
	faceRepo := database.NewPostgresFaceRepository(db)

	detector := &ai.NoopFaceDetector{}
	engine := processor.NewStandardImageEngine(85)
	thumbProcessor := processor.NewThumbnailProcessor(engine, storageRoot, thumbRoot)
	faceProc := processor.NewFaceDetectionProcessor(detector, faceRepo, mediaRepo, storageRoot)

	ctx := context.Background()

	// User email mode: find user by email, process all their media
	if *userEmailFlag != "" {
		log.Printf("[USER MODE] Processing all media for user: %s", *userEmailFlag)
		err = processUserMedia(ctx, jobRepo, mediaRepo, thumbProcessor, faceProc, *userEmailFlag)
		if err != nil {
			log.Fatalf("error processing user media: %v", err)
		}
		log.Println("User media processing completed.")
		return
	}

	// Single media mode
	if *mediaIDFlag != "" {
		log.Printf("[SINGLE MODE] Processing media ID: %s", *mediaIDFlag)
		err = processSingleMedia(ctx, jobRepo, mediaRepo, thumbProcessor, faceProc, *mediaIDFlag)
		if err != nil {
			log.Fatalf("error processing single media: %v", err)
		}
		log.Println("Single media processing completed.")
		return
	}

	// One-shot: process all pending jobs until none remain
	processed, err := processAllJobs(ctx, jobRepo, mediaRepo, thumbProcessor, faceProc)
	if err != nil {
		log.Fatalf("error processing jobs: %v", err)
	}

	if processed == 0 {
		log.Println("No pending jobs found. Exiting.")
	} else {
		log.Printf("Processed %d job(s). Exiting.", processed)
	}
}

// processUserMedia finds a user by email and processes all their media items
func processUserMedia(ctx context.Context, jobRepo *database.PostgresJobRepository, mediaRepo *database.PostgresMediaRepository, thumbProc *processor.ThumbnailProcessor, faceProc *processor.FaceDetectionProcessor, userEmail string) error {
	// Find user by email
	userID, err := findUserIDByEmail(ctx, mediaRepo, userEmail)
	if err != nil {
		return fmt.Errorf("failed to find user %q: %w", userEmail, err)
	}

	log.Printf("Found user ID: %s for email: %s", userID, userEmail)

	// Get all media items for this user
	materials, _, err := mediaRepo.List(ctx, 10000, 0, &userID)
	if err != nil {
		return fmt.Errorf("failed to list media for user %s: %w", userID, err)
	}

	log.Printf("Found %d media items for user", len(materials))

	processed := 0
	for _, media := range materials {
		err = processSingleMedia(ctx, jobRepo, mediaRepo, thumbProc, faceProc, media.ID.String())
		if err != nil {
			log.Printf("[ERROR] Failed to process media %s: %v", media.ID, err)
			continue
		}
		processed++
	}

	log.Printf("Processed %d out of %d media items for user %s", processed, len(materials), userEmail)
	return nil
}

// findUserIDByEmail queries the users table to get the ID for a given email
func findUserIDByEmail(ctx context.Context, mediaRepo *database.PostgresMediaRepository, userEmail string) (uuid.UUID, error) {
	// Use the user repository if available, otherwise query directly
	// For now, we'll need to add this method to the media repo or use raw SQL
	// Let's add a direct query approach

	var userID uuid.UUID
	query := `SELECT id FROM users WHERE email = $1 LIMIT 1`
	err := mediaRepo.GetDB().GetContext(ctx, &userID, query, userEmail)
	if err != nil {
		return uuid.Nil, fmt.Errorf("user not found: %w", err)
	}

	return userID, nil
}

// processSingleMedia handles a single media ID by checking for existing jobs,
// printing their status, generating the thumbnail (overriding if exists), and updating DB.
func processSingleMedia(ctx context.Context, jobRepo *database.PostgresJobRepository, mediaRepo *database.PostgresMediaRepository, thumbProc *processor.ThumbnailProcessor, faceProc *processor.FaceDetectionProcessor, mediaIDStr string) error {
	mediaID, err := uuid.Parse(mediaIDStr)
	if err != nil {
		return fmt.Errorf("invalid media ID %q: %w", mediaIDStr, err)
	}

	// 1. Get the media
	media, err := mediaRepo.GetByID(ctx, mediaID, nil)
	if err != nil {
		return fmt.Errorf("failed to get media %s: %w", mediaID, err)
	}
	if media == nil {
		return fmt.Errorf("media not found: %s", mediaID)
	}

	log.Printf("Found media: ID=%s Path=%s Type=%s UserID=%s",
		media.ID, media.Path, media.MediaType, media.UserID)

	// 2. Check for existing jobs for this media
	existingJobs, err := jobRepo.GetJobsByMediaID(ctx, mediaID)
	if err != nil {
		return fmt.Errorf("failed to get existing jobs for media %s: %w", mediaID, err)
	}

	var job *domain.Job
	if len(existingJobs) > 0 {
		log.Printf("Found %d existing job(s) for this media:", len(existingJobs))
		for _, j := range existingJobs {
			log.Printf("  Job ID=%s Status=%s Type=%s CreatedAt=%s",
				j.ID.String(), j.Status, j.Type, j.CreatedAt.Format(time.RFC3339))
		}

		// Check if thumbnail exists on disk and is valid (>0 bytes)
		thumbAbsPath, err := thumbProc.GetThumbnailAbsPath(media.Path)
		if err != nil {
			log.Printf("[WARN] Could not determine thumbnail path: %v", err)
		} else {
			if _, statErr := os.Stat(thumbAbsPath); statErr == nil {
				info, _ := os.Stat(thumbAbsPath)
				if info.Size() > 0 {
					log.Printf("[EXISTING] Thumbnail already exists on disk: %s (%d bytes)", thumbAbsPath, info.Size())
					// Reuse the most recent job and mark it completed
					job = existingJobs[0]
					log.Printf("Reusing existing job: %s (marking as completed)", job.ID.String())

					// Update job status to completed if not already
					if job.Status != domain.JobStatusCompleted {
						err := jobRepo.UpdateStatus(ctx, job.ID, domain.JobStatusCompleted, "")
						if err != nil {
							return fmt.Errorf("failed to update job status: %w", err)
						}
					}
					return nil // Done - no need to generate thumbnail
				} else {
					log.Printf("[EMPTY] Thumbnail exists but is empty (0 bytes). Will regenerate.")
				}
			} else {
				log.Printf("[NEW] No existing thumbnail found. Will generate new one.")
			}
		}
	} else {
		log.Printf("No existing jobs found for this media.")
	}

	// 3. Generate the thumbnail (new or regenerate)
	if job == nil {
		// Create a new job if none exists
		job = &domain.Job{
			ID:        uuid.New(),
			UserID:    media.UserID,
			Type:      domain.JobTypeThumbnail,
			Status:    domain.JobStatusPending,
			MediaID:   media.ID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := jobRepo.Create(ctx, job); err != nil {
			return fmt.Errorf("failed to create job for media %s: %w", mediaID, err)
		}
		log.Printf("Created new job: %s", job.ID.String())
	}

	// Process the job (generate thumbnail)
	log.Printf("Processing thumbnail for media: %s", media.Path)
	processErr := thumbProc.ProcessJob(ctx, job, media)
	if processErr != nil {
		// Update job status to failed
		log.Printf("Thumbnail generation failed: %v", processErr)
		return jobRepo.UpdateStatus(ctx, job.ID, domain.JobStatusFailed, processErr.Error())
	}

	// 4. Update job status to completed (this is critical - was missing before!)
	log.Printf("Thumbnail generated successfully. Updating job status to completed.")
	err = jobRepo.UpdateStatus(ctx, job.ID, domain.JobStatusCompleted, "")
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	return nil
}

// processAllJobs processes all pending jobs one at a time until the queue is empty.
func processAllJobs(ctx context.Context, jobRepo *database.PostgresJobRepository, mediaRepo *database.PostgresMediaRepository, thumbProc *processor.ThumbnailProcessor, faceProc *processor.FaceDetectionProcessor) (int, error) {
	processed := 0
	skippedErrors := 0

	for {
		jobs, err := jobRepo.GetPending(ctx, 1)
		if err != nil {
			return processed, err
		}

		if len(jobs) == 0 {
			break
		}

		job := jobs[0]
		log.Printf("Processing job ID=%s type=%s mediaID=%s", job.ID.String(), job.Type, job.MediaID)

		processErr := func() error {
			media, err := mediaRepo.GetByID(ctx, job.MediaID, nil)
			if err != nil {
				return logError(jobRepo, job.ID, "failed to get media: "+err.Error())
			}
			if media == nil {
				return logError(jobRepo, job.ID, "media not found")
			}
			log.Printf("  [DEBUG] Media: ID=%s Path=%s Type=%s UserID=%s CapturedAt=%s",
				media.ID, media.Path, media.MediaType, media.UserID, media.CapturedAt)

			switch job.Type {
			case domain.JobTypeThumbnail:
				log.Printf("  [DEBUG] Generating thumbnail for: %s", media.Path)
				err = thumbProc.ProcessJob(ctx, job, media)
				if err != nil {
					return logError(jobRepo, job.ID, "thumbnail generation failed: "+err.Error())
				}
				log.Printf("  [DEBUG] Thumbnail job %s completed successfully", job.ID)
			case domain.JobTypeFaceDetection:
				log.Printf("  [DEBUG] Running face detection for: %s", media.Path)
				err = faceProc.ProcessJob(ctx, job, media)
				if err != nil {
					return logError(jobRepo, job.ID, "face detection failed: "+err.Error())
				}
				log.Printf("  [DEBUG] Face detection job %s completed successfully", job.ID)
			default:
				return logError(jobRepo, job.ID, "unknown job type: "+string(job.Type))
			}

			log.Printf("job %s completed successfully", job.ID)
			return jobRepo.UpdateStatus(ctx, job.ID, domain.JobStatusCompleted, "")
		}()

		if processErr != nil {
			log.Printf("error processing job %s: %v", job.ID, processErr)
			skippedErrors++
		}

		processed++
	}

	if skippedErrors > 0 {
		log.Printf("[WARNING] %d jobs had errors during processing", skippedErrors)
	}

	return processed, nil
}

// logError updates a job's status to failed and logs the error.
func logError(jobRepo *database.PostgresJobRepository, jobID uuid.UUID, errMsg string) error {
	log.Printf("job %s failed: %s", jobID, errMsg)
	return jobRepo.UpdateStatus(context.Background(), jobID, domain.JobStatusFailed, errMsg)
}

// maskDBPassword replaces password in DSN for safe logging.
func maskDBPassword(dsn string) string {
	masked := dsn
	if idx := len("postgres://"); idx < len(dsn) {
		if pwEnd := strings.Index(dsn[idx:], "@"); pwEnd != -1 {
			masked = dsn[:idx] + "****" + dsn[idx+pwEnd:]
		}
	}
	return masked
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
