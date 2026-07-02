package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

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

// processAllJobs processes all pending jobs one at a time until the queue is empty.
func processAllJobs(ctx context.Context, jobRepo *database.PostgresJobRepository, mediaRepo *database.PostgresMediaRepository, thumbProc *processor.ThumbnailProcessor, faceProc *processor.FaceDetectionProcessor) (int, error) {
	processed := 0

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
				return err
			}
			if media == nil {
				return logError(jobRepo, job.ID, "media not found")
			}

			switch job.Type {
			case domain.JobTypeThumbnail:
				err = thumbProc.ProcessJob(ctx, job, media)
			case domain.JobTypeFaceDetection:
				err = faceProc.ProcessJob(ctx, job, media)
			default:
				return logError(jobRepo, job.ID, "unknown job type")
			}

			if err != nil {
				return logError(jobRepo, job.ID, err.Error())
			}

			log.Printf("job %s completed successfully", job.ID)
			return jobRepo.UpdateStatus(ctx, job.ID, domain.JobStatusCompleted, "")
		}()

		if processErr != nil {
			log.Printf("error processing job %s: %v", job.ID, processErr)
		}

		processed++
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
