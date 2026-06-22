package database

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"steadyphoto/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func TestPostgresMediaRepository(t *testing.T) {
	// Setup database connection using environment variable
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://steadyphoto:password@localhost:5432/steadyphoto?sslmode=disable"
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v. Ensure DATABASE_URL is set and Postgres is running.", err)
	}
	defer db.Close()

	// --- IDEMPOTENCY STEP: Ensure schema exists and wipe data ---
	
	// 1. Check if 'media' table exists
	var exists bool
	err = db.QueryRow("SELECT EXISTS (SELECT FROM pg_tables WHERE tablename = 'media')").Scan(&exists)
	if err != nil {
		t.Fatalf("Failed to check if 'media' table exists: %v", err)
	}

	if !exists {
		t.Log("Tables not found, attempting to apply initial schema...")
		// In a real test environment, migrations should be handled by a migration runner.
		// For this test, we manually run the init migration to ensure the test can proceed.
		// This addresses the "relation does not exist" error.
		initSchema := `
		CREATE EXTENSION IF NOT EXISTS vector;
		CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

		CREATE TABLE IF NOT EXISTS media (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			path TEXT NOT NULL UNIQUE,
			filename TEXT NOT NULL,
			hash TEXT NOT NULL UNIQUE,
			size_bytes BIGINT NOT NULL,
			width INT,
			height INT,
			captured_at TIMESTAMPTZ,
			media_type VARCHAR(20) DEFAULT 'photo' CHECK (media_type IN ('photo', 'video')),
			metadata JSONB DEFAULT '{}',
			video_metadata JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS faces (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			media_id UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
			bounding_box JSONB NOT NULL,
			embedding VECTOR(512),
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS albums (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			name TEXT NOT NULL,
			description TEXT,
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS album_photos (
			album_id UUID REFERENCES albums(id) ON DELETE CASCADE,
			media_id UUID REFERENCES media(id) ON DELETE CASCADE,
			PRIMARY KEY (album_id, media_id)
		);
		`
		_, err = db.Exec(initSchema)
		if err != nil {
			t.Fatalf("Failed to apply initial schema: %v", err)
		}
	} else {
		// 2. If tables exist, truncate them to ensure a clean state
		_, err = db.Exec("TRUNCATE TABLE media, faces, albums, album_photos RESTART IDENTITY CASCADE")
		if err != nil {
			t.Fatalf("Failed to clean up database for idempotent testing: %v", err)
		}
	}

	repo := NewPostgresMediaRepository(db)

	t.Run("Create_and_GetByID", func(t *testing.T) {
		userID := uuid.New()
		media := &domain.Media{
			ID:         uuid.New(),
			Path:       "/tmp/test/img.jpg",
			Filename:   "img.jpg",
			Hash:       "hash123",
			SizeBytes:  1024,
			Width:      1920,
			Height:     1080,
			MediaType:  domain.MediaTypePhoto,
			CapturedAt: time.Now().Truncate(time.Microsecond),
			Metadata:   domain.Metadata{"camera": "sony"},
			CreatedAt:  time.Now().Truncate(time.Microsecond),
			UpdatedAt:  time.Now().Truncate(time.Microsecond),
			UserID:     userID,
		}

		err := repo.Create(context.Background(), media)
		if err != nil {
			t.Fatalf("Failed to create media: %v", err)
		}

		retrieved, err := repo.GetByID(context.Background(), media.ID, &userID)
		if err != nil {
			t.Fatalf("Failed to get media: %v", err)
		}

		if retrieved.Hash != media.Hash {
			t.Errorf("Expected hash %s, got %s", media.Hash, retrieved.Hash)
		}
		if retrieved.SizeBytes != media.SizeBytes {
			t.Errorf("Expected size %d, got %d", media.SizeBytes, retrieved.SizeBytes)
		}
		if retrieved.Metadata["camera"] != "sony" {
			t.Errorf("Expected metadata camera=sony, got %s", retrieved.Metadata["camera"])
		}
	})

	t.Run("GetByHash", func(t *testing.T) {
		hash := "unique_hash_999"
		media := &domain.Media{
			ID:         uuid.New(),
			Path:       "/tmp/test/img2.jpg",
			Filename:   "img2.jpg",
			Hash:       hash,
			SizeBytes:  2048,
			MediaType:  domain.MediaTypePhoto,
			CapturedAt: time.Now().Truncate(time.Microsecond),
		}

		err := repo.Create(context.Background(), media)
		if err != nil {
			t.Fatalf("Failed to create media: %v", err)
		}

		retrieved, err := repo.GetByHash(context.Background(), hash)
		if err != nil {
			t.Fatalf("Failed to get media by hash: %v", err)
		}

		if retrieved.ID != media.ID {
			t.Errorf("Expected ID %s, got %s", media.ID, retrieved.ID)
		}
	})

	t.Run("List", func(t *testing.T) {
		// Create multiple media
		count := 5
		for i := 0; i < count; i++ {
			m := &domain.Media{
				ID:         uuid.New(),
				Path:       fmt.Sprintf("/tmp/test/list_%d.jpg", i),
				Filename:   fmt.Sprintf("list_%d.jpg", i),
				Hash:       uuid.New().String(),
				SizeBytes:  100,
				MediaType:  domain.MediaTypePhoto,
				CapturedAt: time.Now().Add(time.Duration(i) * time.Second),
			}
			if err := repo.Create(context.Background(), m); err != nil {
				t.Fatalf("Failed to create media in List test index %d: %v", i, err)
			}
		}

		mediaList, total, err := repo.List(context.Background(), 2, 0, nil)
		if err != nil {
			t.Fatalf("Failed to list media: %v", err)
		}

		if len(mediaList) != 2 {
			t.Errorf("Expected 2 media, got %d", len(mediaList))
		}
		if total < count {
			t.Errorf("Expected total at least %d, got %d", count, total)
		}
	})

	t.Run("Update", func(t *testing.T) {
		media := &domain.Media{
			ID:         uuid.New(),
			Path:       "/tmp/test/upd.jpg",
			Filename:   "upd.jpg",
			Hash:       "hash_upd",
			SizeBytes:  500,
			MediaType:  domain.MediaTypePhoto,
			CapturedAt: time.Now().Truncate(time.Microsecond),
		}
		if err := repo.Create(context.Background(), media); err != nil {
			t.Fatalf("Failed to create media for update test: %v", err)
		}

		media.SizeBytes = 9999
		media.Metadata = domain.Metadata{"updated": "true"}
		err := repo.Update(context.Background(), media)
		if err != nil {
			t.Fatalf("Failed to update media: %v", err)
		}

		userID := uuid.New()
		retrieved, _ := repo.GetByID(context.Background(), media.ID, &userID)
		if retrieved.SizeBytes != 9999 {
			t.Errorf("Expected size 9999, got %d", retrieved.SizeBytes)
		}
		if retrieved.Metadata["updated"] != "true" {
			t.Errorf("Expected metadata updated=true, got %s", retrieved.Metadata["updated"])
		}
	})

	t.Run("Delete", func(t *testing.T) {
		userID := uuid.New()
		media := &domain.Media{
			ID:         uuid.New(),
			Path:       "/tmp/test/del.jpg",
			Filename:   "del.jpg",
			Hash:       "hash_del",
			MediaType:  domain.MediaTypePhoto,
			CapturedAt: time.Now(),
			UserID:     userID,
		}
		if err := repo.Create(context.Background(), media); err != nil {
			t.Fatalf("Failed to create media for delete test: %v", err)
		}

		err := repo.Delete(context.Background(), media.ID, &userID)
		if err != nil {
			t.Fatalf("Failed to delete media: %v", err)
		}

		_, err = repo.GetByID(context.Background(), media.ID, &userID)
		if err == nil {
			t.Error("Expected error getting deleted media, got nil")
		}
	})
}
