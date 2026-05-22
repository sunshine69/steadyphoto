package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"steadyphoto/internal/database"
	"steadyphoto/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	dbURL := flag.String("db", "", "PostgreSQL connection URL")
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

	fmt.Println("🔍 Scanning database for missing thumbnail jobs...")

	ctx := context.Background()

	// 3. Get all photos
	// Note: For very large libraries, we'd need to paginate this.
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
	skippedCount := 0

	for _, photo := range photos {
		// Check if a thumbnail job already exists for this photo
		// We check for 'pending' or 'processing' jobs to avoid duplicates
		// In a real system, we might also check 'completed'
		
		// Since our JobRepository doesn't have a 'GetJobsByMediaID', 
		// we'll do a quick check via a raw query or assume we need to check status.
		// For simplicity in this repair script, we'll query the jobs table directly.
		
		var exists bool
		query := `SELECT EXISTS(SELECT 1 FROM jobs WHERE media_id = $1 AND job_type = $2 AND status IN ('pending', 'processing'))`
		err := db.GetContext(ctx, &exists, query, photo.ID, string(domain.JobTypeThumbnail))
		if err != nil {
			log.Printf("Error checking job status for photo %s: %v", photo.ID, err)
			continue
		}

		if !exists {
			// Create the missing job
			job := &domain.Job{
				ID:        uuid.New(),
				Type:      domain.JobTypeThumbnail,
				Status:    domain.JobStatusPending,
				MediaID:   photo.ID,
				CreatedAt: time.Now(),
			}

			if err := jobRepo.Create(ctx, job); err != nil {
				log.Printf("Failed to create job for photo %s: %v", photo.ID, err)
			} else {
				createdCount++
			}
		} else {
			skippedCount++
		}
	}

	fmt.Printf("\n✅ Repair Complete!\n")
	fmt.Printf("New jobs created: %d\n", createdCount)
	fmt.Printf("Jobs already existing: %d\n", skippedCount)
}
