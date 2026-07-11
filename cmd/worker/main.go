package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/jbrodriguez/mlog"
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

// VideoMetadataProcessor handles video metadata extraction jobs.
type VideoMetadataProcessor struct {
	mediaRepo   *database.PostgresMediaRepository
	storageRoot string
}

// ProcessJob extracts video metadata for the media item associated with the job.
func (p *VideoMetadataProcessor) ProcessJob(ctx context.Context, job *domain.Job, media *domain.Media) error {
	absPath := filepath.Join(p.storageRoot, media.Path)
	mlog.Info("Extracting video metadata from: %s", absPath)

	vm, err := processor.ExtractVideoMetadata(ctx, absPath)
	if err != nil {
		return fmt.Errorf("video metadata extraction failed: %w", err)
	}

	if vm == nil {
		return fmt.Errorf("no video metadata extracted from %s", absPath)
	}

	media.VideoMetadata = *vm

	// Update the media record with video metadata
	return p.mediaRepo.Update(ctx, media)
}

func main() {
	// CLI flags
	userEmailFlag := flag.String("user", "", "User email to process all media for")
	mediaIDFlag := flag.String("media-id", "", "Specific media ID to generate thumbnail for (single mode)")
	forceFlag := flag.Bool("force", false, "Force thumbnail generation, ignoring existing thumbnails")
	videoMetaFlag := flag.Bool("video-meta", false, "Extract video metadata for all media items (or just those with missing data)")
	flag.Parse()

	if *userEmailFlag != "" && *mediaIDFlag != "" {
		mlog.Fatal("cannot use both -user and -media-id flags together")
	}

	// Load .env from the project root
	absPath, err := filepath.Abs(".")
	if err != nil {
		mlog.Fatalf("failed to get absolute path: %v", err)
	}
	mlog.Info("Loading .env from: %s", absPath)

	envFile := filepath.Join(absPath, ".env")
	if err := godotenv.Load(envFile); err != nil {
		mlog.Info("[WARN] Failed to load %s: %v (using env vars)", envFile, err)
	}

	// Read config from env (now populated from .env)
	dbURL := getEnv("DATABASE_URL", "")
	if dbURL == "" {
		mlog.Fatal("DATABASE_URL not set in .env or environment")
	}

	storageRoot := getEnv("STORAGE_ROOT", "storage")
	thumbRoot := getEnv("THUMBNAIL_ROOT", filepath.Join(storageRoot, ".thumbnails"))

	mlog.Info("DB URL: %s", maskDBPassword(dbURL))
	mlog.Info("Storage root: %s", storageRoot)
	mlog.Info("Thumbnail root: %s", thumbRoot)

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		mlog.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		mlog.Fatalf("failed to ping database: %v", err)
	}

	jobRepo := database.NewPostgresJobRepository(db)
	mediaRepo := database.NewPostgresMediaRepository(db)
	faceRepo := database.NewPostgresFaceRepository(db)

	detector := &ai.NoopFaceDetector{}
	engine := processor.NewStandardImageEngine(85)
	thumbProcessor := processor.NewThumbnailProcessor(engine, storageRoot, thumbRoot)
	faceProc := processor.NewFaceDetectionProcessor(detector, faceRepo, mediaRepo, storageRoot)
	videoMetaProcessor := &VideoMetadataProcessor{mediaRepo: mediaRepo, storageRoot: storageRoot}

	ctx := context.Background()

	// Video metadata backfill mode: extract video metadata for all media items
	if *videoMetaFlag {
		mlog.Info("[VIDEO META BACKFILL] Extracting video metadata for all media...")
		err = processVideoMetaBackfill(ctx, jobRepo, mediaRepo, videoMetaProcessor)
		if err != nil {
			mlog.Fatalf("error processing video meta backfill: %v", err)
		}
		mlog.Info("Video meta backfill completed.")
		return
	}

	// User email mode: find user by email, process all their media
	if *userEmailFlag != "" {
		forceMode := *forceFlag
		if forceMode {
			mlog.Info("[FORCE MODE] Will regenerate all thumbnails for user: %s", *userEmailFlag)
		}
		mlog.Info("[USER MODE] Processing all media for user: %s", *userEmailFlag)
		err = processUserMedia(ctx, jobRepo, mediaRepo, thumbProcessor, faceProc, videoMetaProcessor, *userEmailFlag, forceMode)
		if err != nil {
			mlog.Fatalf("error processing user media: %v", err)
		}
		mlog.Info("User media processing completed.")
		return
	}

	// Single media mode
	if *mediaIDFlag != "" {
		forceMode := *forceFlag
		if forceMode {
			mlog.Info("[FORCE MODE] Will regenerate thumbnail for media: %s", *mediaIDFlag)
		}
		mlog.Info("[SINGLE MODE] Processing media ID: %s", *mediaIDFlag)
		err = processSingleMedia(ctx, jobRepo, mediaRepo, thumbProcessor, faceProc, videoMetaProcessor, *mediaIDFlag, forceMode)
		if err != nil {
			mlog.Fatalf("error processing single media: %v", err)
		}
		mlog.Info("Single media processing completed.")
		return
	}

	// Global force mode: regenerate ALL thumbnails for ALL media across ALL users
	if *forceFlag {
		mlog.Info("[GLOBAL FORCE MODE] Regenerating all thumbnails for ALL users...")
		err = processGlobalForce(ctx, jobRepo, mediaRepo, thumbProcessor, faceProc, videoMetaProcessor)
		if err != nil {
			mlog.Fatalf("error processing global force: %v", err)
		}
		mlog.Info("Global force processing completed.")
		return
	}

	// One-shot: process all pending jobs until none remain
	processed, err := processAllJobs(ctx, jobRepo, mediaRepo, thumbProcessor, faceProc, videoMetaProcessor)
	if err != nil {
		mlog.Fatalf("error processing jobs: %v", err)
	}

	if processed == 0 {
		mlog.Info("No pending jobs found. Exiting.")
	} else {
		mlog.Info("Processed %d job(s). Exiting.", processed)
	}
}

// processVideoMetaBackfill finds all media items and extracts video metadata for videos.
// For non-videos, it also creates video_metadata extraction jobs if missing.
func processVideoMetaBackfill(ctx context.Context, jobRepo *database.PostgresJobRepository, mediaRepo *database.PostgresMediaRepository, proc *VideoMetadataProcessor) error {
	mediaList, total, err := mediaRepo.ListAll(ctx, 1000000, 0)
	if err != nil {
		return fmt.Errorf("failed to list all media: %w", err)
	}

	mlog.Info("[VIDEO META BACKFILL] Found %d total media items", total)

	processed := 0
	for _, media := range mediaList {
		if media.MediaType != domain.MediaTypeVideo {
			continue
		}

		// Check if a video metadata job already exists for this media
		existingJobs, err := jobRepo.GetJobsByMediaID(ctx, media.ID)
		if err != nil {
			mlog.Info("[WARN] Failed to check existing jobs for media %s: %v", media.ID, err)
			continue
		}

		hasVideoMetaJob := false
		hasVideoMetadata := media.VideoMetadata.Duration > 0
		for _, j := range existingJobs {
			if j.Type == domain.JobTypeVideoMetadata {
				hasVideoMetaJob = true
				break
			}
		}

		if hasVideoMetaJob {
			mlog.Info("[SKIP] Media %s already has a video metadata job", media.ID)
			continue
		}

		if hasVideoMetadata {
			mlog.Info("[SKIP] Media %s already has video metadata in DB", media.ID)
			continue
		}

		// Create a new video metadata job
		job := &domain.Job{
			ID:        uuid.New(),
			UserID:    media.UserID,
			Type:      domain.JobTypeVideoMetadata,
			Status:    domain.JobStatusPending,
			MediaID:   media.ID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := jobRepo.Create(ctx, job); err != nil {
			mlog.Info("[ERROR] Failed to create video metadata job for %s: %v", media.ID, err)
			continue
		}
		mlog.Info("[CREATE JOB] Created video metadata job %s for media %s (%s)", job.ID, media.ID, media.Filename)
		processed++
	}

	mlog.Info("[VIDEO META BACKFILL] Created %d new video metadata jobs", processed)
	return nil
}

// processUserMedia finds a user by email and processes all their media items
func processUserMedia(ctx context.Context, jobRepo *database.PostgresJobRepository, mediaRepo *database.PostgresMediaRepository, thumbProc *processor.ThumbnailProcessor, faceProc *processor.FaceDetectionProcessor, videoMetaProc *VideoMetadataProcessor, userEmail string, force bool) error {
	// Find user by email
	userID, err := findUserIDByEmail(ctx, mediaRepo, userEmail)
	if err != nil {
		return fmt.Errorf("failed to find user %q: %w", userEmail, err)
	}

	mlog.Info("Found user ID: %s for email: %s", userID, userEmail)

	// Get all media items for this user
	materials, _, err := mediaRepo.List(ctx, 10000, 0, &userID)
	if err != nil {
		return fmt.Errorf("failed to list media for user %s: %w", userID, err)
	}

	mlog.Info("Found %d media items for user", len(materials))

	// If force mode, reset all existing job statuses to pending so they get regenerated
	if force {
		mlog.Info("[FORCE MODE] Resetting all existing jobs for user %s to pending...", userID)
		resetCount, err := jobRepo.ResetJobsByUserID(ctx, userID)
		if err != nil {
			mlog.Info("[WARN] Failed to reset jobs: %v", err)
		} else {
			mlog.Info("[FORCE MODE] Reset %d existing job(s) to pending", resetCount)
		}

		// Also delete existing thumbnail files so they get regenerated
		thumbCount := 0
		for _, media := range materials {
			thumbAbsPath, err := thumbProc.GetThumbnailAbsPath(media.Path)
			if err != nil {
				continue
			}
			if _, statErr := os.Stat(thumbAbsPath); statErr == nil {
				if err := os.Remove(thumbAbsPath); err == nil {
					thumbCount++
					mlog.Info("  Deleted existing thumbnail: %s", thumbAbsPath)
				}
			}
		}
		mlog.Info("[FORCE MODE] Deleted %d existing thumbnail file(s)", thumbCount)
	}

	processed := 0
	for _, media := range materials {
		err = processSingleMedia(ctx, jobRepo, mediaRepo, thumbProc, faceProc, videoMetaProc, media.ID.String(), force)
		if err != nil {
			mlog.Info("[ERROR] Failed to process media %s: %v", media.ID, err)
			continue
		}
		processed++
	}

	mlog.Info("Processed %d out of %d media items for user %s", processed, len(materials), userEmail)
	return nil

}

// processGlobalForce regenerates thumbnails for ALL media across ALL users
func processGlobalForce(ctx context.Context, jobRepo *database.PostgresJobRepository, mediaRepo *database.PostgresMediaRepository, thumbProc *processor.ThumbnailProcessor, faceProc *processor.FaceDetectionProcessor, videoMetaProc *VideoMetadataProcessor) error {
	// Get all media across all users
	mlog.Info("[GLOBAL FORCE MODE] Listing all media across all users...")
	materials, _, err := mediaRepo.ListAll(ctx, 100000, 0)
	if err != nil {
		return fmt.Errorf("failed to list all media: %w", err)
	}

	mlog.Info("[GLOBAL FORCE MODE] Found %d media items across all users", len(materials))

	// Reset ALL jobs across all users to pending
	mlog.Info("[GLOBAL FORCE MODE] Resetting all jobs to pending...")
	resetCount, err := jobRepo.ResetJobsAll(ctx)
	if err != nil {
		mlog.Info("[WARN] Failed to reset jobs: %v", err)
	} else {
		mlog.Info("[GLOBAL FORCE MODE] Reset %d existing jobs to pending", resetCount)
	}

	// Delete ALL existing thumbnail files
	mlog.Info("[GLOBAL FORCE MODE] Deleting all existing thumbnails...")
	thumbCount := 0
	for _, media := range materials {
		thumbAbsPath, err := thumbProc.GetThumbnailAbsPath(media.Path)
		if err != nil {
			continue
		}
		if _, statErr := os.Stat(thumbAbsPath); statErr == nil {
			if err := os.Remove(thumbAbsPath); err == nil {
				thumbCount++
				mlog.Info("  Deleted existing thumbnail: %s", thumbAbsPath)
			}
		}
	}
	mlog.Info("[GLOBAL FORCE MODE] Deleted %d existing thumbnail file(s)", thumbCount)

	// Process each media item with force mode
	processed := 0
	for _, media := range materials {
		err = processSingleMedia(ctx, jobRepo, mediaRepo, thumbProc, faceProc, videoMetaProc, media.ID.String(), true)
		if err != nil {
			mlog.Info("[ERROR] Failed to process media %s: %v", media.ID, err)
			continue
		}
		processed++
	}

	mlog.Info("[GLOBAL FORCE MODE] Processed %d out of %d media items", processed, len(materials))
	return nil
}

// findUserIDByEmail queries the users table to get the ID for a given email
func findUserIDByEmail(ctx context.Context, mediaRepo *database.PostgresMediaRepository, userEmail string) (uuid.UUID, error) {
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
func processSingleMedia(ctx context.Context, jobRepo *database.PostgresJobRepository, mediaRepo *database.PostgresMediaRepository, thumbProc *processor.ThumbnailProcessor, faceProc *processor.FaceDetectionProcessor, videoMetaProc *VideoMetadataProcessor, mediaIDStr string, force bool) error {
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

	mlog.Info("Found media: ID=%s Path=%s Type=%s UserID=%s",
		media.ID, media.Path, media.MediaType, media.UserID)

	// 2. Check for existing jobs for this media
	existingJobs, err := jobRepo.GetJobsByMediaID(ctx, mediaID)
	if err != nil {
		return fmt.Errorf("failed to get existing jobs for media %s: %w", mediaID, err)
	}

	var job *domain.Job
	if len(existingJobs) > 0 {
		mlog.Info("Found %d existing job(s) for this media:", len(existingJobs))
		for _, j := range existingJobs {
			mlog.Info("  Job ID=%s Status=%s Type=%s CreatedAt=%s",
				j.ID.String(), j.Status, j.Type, j.CreatedAt.Format(time.RFC3339))
		}

		// Force mode: skip existing thumbnail check, always regenerate
		if force {
			mlog.Info("[FORCE MODE] Deleting existing thumbnail and regenerating.")
			thumbAbsPath, err := thumbProc.GetThumbnailAbsPath(media.Path)
			if err == nil {
				os.Remove(thumbAbsPath)
			}
			// Mark existing jobs as completed, then we'll create a new one
			for _, j := range existingJobs {
				jobRepo.UpdateStatus(ctx, j.ID, domain.JobStatusCompleted, "")
			}
			// Don't reuse - we want a fresh job for the regeneration
			job = nil
		} else {
			// Check if thumbnail exists on disk and is valid (>0 bytes)
			thumbAbsPath, err := thumbProc.GetThumbnailAbsPath(media.Path)
			if err != nil {
				mlog.Info("[WARN] Could not determine thumbnail path: %v", err)
			} else {
				if _, statErr := os.Stat(thumbAbsPath); statErr == nil {
					info, _ := os.Stat(thumbAbsPath)
					if info.Size() > 0 {
						mlog.Info("[EXISTING] Thumbnail already exists on disk: %s (%d bytes)", thumbAbsPath, info.Size())
						// Reuse the most recent job and mark it completed
						job = existingJobs[0]
						mlog.Info("Reusing existing job: %s (marking as completed)", job.ID.String())

						// Update job status to completed if not already
						if job.Status != domain.JobStatusCompleted {
							err := jobRepo.UpdateStatus(ctx, job.ID, domain.JobStatusCompleted, "")
							if err != nil {
								return fmt.Errorf("failed to update job status: %w", err)
							}
						}
						return nil // Done - no need to generate thumbnail
					} else {
						mlog.Info("[EMPTY] Thumbnail exists but is empty (0 bytes). Will regenerate.")
					}
				} else {
					mlog.Info("[NEW] No existing thumbnail found. Will generate new one.")
				}
			}
		}
	} else {
		mlog.Info("No existing jobs found for this media.")
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
		mlog.Info("Created new job: %s", job.ID.String())
	}

	// Process the job (generate thumbnail)
	mlog.Info("Processing thumbnail for media: %s", media.Path)
	processErr := thumbProc.ProcessJob(ctx, job, media)
	if processErr != nil {
		// Update job status to failed
		mlog.Info("Thumbnail generation failed: %v", processErr)
		return jobRepo.UpdateStatus(ctx, job.ID, domain.JobStatusFailed, processErr.Error())
	}

	// 4. Update job status to completed (this is critical - was missing before!)
	mlog.Info("Thumbnail generated successfully. Updating job status to completed.")
	err = jobRepo.UpdateStatus(ctx, job.ID, domain.JobStatusCompleted, "")
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	return nil
}

// processAllJobs processes all pending jobs one at a time until the queue is empty.
func processAllJobs(ctx context.Context, jobRepo *database.PostgresJobRepository, mediaRepo *database.PostgresMediaRepository, thumbProc *processor.ThumbnailProcessor, faceProc *processor.FaceDetectionProcessor, videoMetaProc *VideoMetadataProcessor) (int, error) {
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
		mlog.Info("Processing job ID=%s type=%s mediaID=%s", job.ID.String(), job.Type, job.MediaID)

		processErr := func() error {
			media, err := mediaRepo.GetByID(ctx, job.MediaID, nil)
			if err != nil {
				return logError(jobRepo, job.ID, "failed to get media: "+err.Error())
			}
			if media == nil {
				return logError(jobRepo, job.ID, "media not found")
			}
			mlog.Info("  [DEBUG] Media: ID=%s Path=%s Type=%s UserID=%s CapturedAt=%s",
				media.ID, media.Path, media.MediaType, media.UserID, media.CapturedAt)

			switch job.Type {
			case domain.JobTypeThumbnail:
				mlog.Info("  [DEBUG] Generating thumbnail for: %s", media.Path)
				err = thumbProc.ProcessJob(ctx, job, media)
				if err != nil {
					return logError(jobRepo, job.ID, "thumbnail generation failed: "+err.Error())
				}
				mlog.Info("  [DEBUG] Thumbnail job %s completed successfully", job.ID)
			case domain.JobTypeFaceDetection:
				mlog.Info("  [DEBUG] Running face detection for: %s", media.Path)
				err = faceProc.ProcessJob(ctx, job, media)
				if err != nil {
					return logError(jobRepo, job.ID, "face detection failed: "+err.Error())
				}
				mlog.Info("  [DEBUG] Face detection job %s completed successfully", job.ID)
			case domain.JobTypeVideoMetadata:
				mlog.Info("  [DEBUG] Extracting video metadata for: %s", media.Path)
				err = videoMetaProc.ProcessJob(ctx, job, media)
				if err != nil {
					return logError(jobRepo, job.ID, "video metadata extraction failed: "+err.Error())
				}
				mlog.Info("  [DEBUG] Video metadata job %s completed successfully", job.ID)
			default:
				return logError(jobRepo, job.ID, "unknown job type: "+string(job.Type))
			}

			mlog.Info("job %s completed successfully", job.ID)
			return jobRepo.UpdateStatus(ctx, job.ID, domain.JobStatusCompleted, "")
		}()

		if processErr != nil {
			mlog.Info("error processing job %s: %v", job.ID, processErr)
			skippedErrors++
		}

		processed++
	}

	if skippedErrors > 0 {
		mlog.Info("[WARNING] %d jobs had errors during processing", skippedErrors)
	}

	return processed, nil
}

// logError updates a job's status to failed and logs the error.
func logError(jobRepo *database.PostgresJobRepository, jobID uuid.UUID, errMsg string) error {
	mlog.Info("job %s failed: %s", jobID, errMsg)
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
