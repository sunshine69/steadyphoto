package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"steadyphoto/internal/ai"
	"steadyphoto/internal/database"
	"steadyphoto/internal/domain"
	"steadyphoto/internal/processor"
)

const (
	defaultDBURL         = "postgres://localhost:5432/steadyphoto?sslmode=disable"
	defaultJobPollPeriod = 10 * time.Second
	defaultStorageRoot   = "/data/photos/storage"
	defaultThumbRoot     = "/data/photos/.thumbnails"
)

func main() {
	dbURL := getEnv("DB_URL", defaultDBURL)
	pollPeriod, err := parseDuration(getEnv("JOB_POLL_PERIOD", fmt.Sprintf("%d", defaultJobPollPeriod)))
	if err != nil {
		log.Fatalf("invalid JOB_POLL_PERIOD: %v", err)
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	jobRepo := database.NewPostgresJobRepository(db)
	mediaRepo := database.NewPostgresMediaRepository(db)
	faceRepo := database.NewPostgresFaceRepository(db)

	detector := &ai.NoopFaceDetector{}
	engine := processor.NewStandardImageEngine(85)
	thumbProcessor := processor.NewThumbnailProcessor(engine, defaultStorageRoot, defaultThumbRoot)
	faceProc := processor.NewFaceDetectionProcessor(detector, faceRepo, mediaRepo, defaultStorageRoot)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-sigCh:
			fmt.Println("\nShutting down worker...")
			cancel()
		}
	}()

	log.Printf("Worker started - polling for jobs every %s", pollPeriod)

	for {
		if err := processNextJob(ctx, jobRepo, mediaRepo, thumbProcessor, faceProc); err != nil {
			log.Printf("error processing job: %v", err)
		}
		time.Sleep(pollPeriod)
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func parseDuration(s string) (time.Duration, error) {
	d, err := time.ParseDuration(s)
	if err == nil {
		return d, nil
	}
	n, e := time.ParseDuration(fmt.Sprintf("%ss", s))
	if e != nil {
		return 0, fmt.Errorf("invalid duration %q: expected Go duration or number of seconds", s)
	}
	return n, nil
}

func processNextJob(ctx context.Context, jobRepo *database.PostgresJobRepository, mediaRepo *database.PostgresMediaRepository, thumbProc *processor.ThumbnailProcessor, faceProc *processor.FaceDetectionProcessor) error {
	jobs, err := jobRepo.GetPending(ctx, 1)
	if err != nil {
		return fmt.Errorf("failed to get pending jobs: %w", err)
	}
	if len(jobs) == 0 {
		return nil // No jobs available
	}

	job := jobs[0]
	log.Printf("Processing job ID=%s type=%s mediaID=%s", job.ID, job.Type, job.MediaID)

	err = func() error {
		media, err := mediaRepo.GetByID(ctx, job.MediaID, nil)
		if err != nil {
			return fmt.Errorf("failed to get media: %w", err)
		}
		if media == nil {
			return fmt.Errorf("media not found for ID=%s", job.MediaID)
		}

		switch job.Type {
		case domain.JobTypeThumbnail:
			err = thumbProc.ProcessJob(ctx, job, media)
		case domain.JobTypeFaceDetection:
			err = faceProc.ProcessJob(ctx, job, media)
		default:
			return fmt.Errorf("unknown job type: %s", job.Type)
		}

		if err != nil {
			log.Printf("job %s failed: %v - marking as error", job.ID, err)
			err2 := jobRepo.UpdateStatus(ctx, job.ID, domain.JobStatusFailed, err.Error())
			if err2 != nil {
				return fmt.Errorf("failed to mark job as error: %w", err2)
			}
			return nil // Don't return the original error; we handled it
		}

		log.Printf("job %s completed successfully", job.ID)
		err = jobRepo.UpdateStatus(ctx, job.ID, domain.JobStatusCompleted, "")
		if err != nil {
			return fmt.Errorf("failed to mark job as completed: %w", err)
		}
		return nil
	}()

	return err
}
