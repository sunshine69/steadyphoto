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

func TestPostgresPhotoRepository(t *testing.T) {
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

	// --- IDEMPOTENCY STEP: Wipe the database before starting the tests ---
	// We use CASCADE to ensure that all related records in faces and albums are also removed.
	_, err = db.Exec("TRUNCATE TABLE photos, faces, albums RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("Failed to clean up database for idempotent testing: %v", err)
	}

	repo := NewPostgresPhotoRepository(db)

	t.Run("Create_and_GetByID", func(t *testing.T) {
		photo := &domain.Photo{
			ID:         uuid.New(),
			Path:       "/tmp/test/img.jpg",
			Filename:   "img.jpg",
			Hash:       "hash123",
			SizeBytes:  1024,
			Width:      1920,
			Height:    1080,
			CapturedAt: time.Now().Truncate(time.Microsecond),
			Metadata:   domain.Metadata{"camera": "sony"},
			CreatedAt:  time.Now().Truncate(time.Microsecond),
			UpdatedAt:  time.Now().Truncate(time.Microsecond),
		}

		err := repo.Create(context.Background(), photo)
		if err != nil {
			t.Fatalf("Failed to create photo: %v", err)
		}

		retrieved, err := repo.GetByID(context.Background(), photo.ID)
		if err != nil {
			t.Fatalf("Failed to get photo: %v", err)
		}

		if retrieved.Hash != photo.Hash {
			t.Errorf("Expected hash %s, got %s", photo.Hash, retrieved.Hash)
		}
		if retrieved.SizeBytes != photo.SizeBytes {
			t.Errorf("Expected size %d, got %d", photo.SizeBytes, retrieved.SizeBytes)
		}
		if retrieved.Metadata["camera"] != "sony" {
			t.Errorf("Expected metadata camera=sony, got %s", retrieved.Metadata["camera"])
		}
	})

	t.Run("GetByHash", func(t *testing.T) {
		hash := "unique_hash_999"
		photo := &domain.Photo{
			ID:         uuid.New(),
			Path:       "/tmp/test/img2.jpg",
			Filename:   "img2.jpg",
			Hash:       hash,
			SizeBytes:  2048,
			CapturedAt: time.Now().Truncate(time.Microsecond),
		}

		err := repo.Create(context.Background(), photo)
		if err != nil {
			t.Fatalf("Failed to create photo: %v", err)
		}

		retrieved, err := repo.GetByHash(context.Background(), hash)
		if err != nil {
			t.Fatalf("Failed to get photo by hash: %v", err)
		}

		if retrieved.ID != photo.ID {
			t.Errorf("Expected ID %s, got %s", photo.ID, retrieved.ID)
		}
	})

	t.Run("List", func(t *testing.T) {
		// Create multiple photos
		count := 5
		for i := 0; i < count; i++ {
			p := &domain.Photo{
				ID:         uuid.New(),
				Path:       fmt.Sprintf("/tmp/test/list_%d.jpg", i),
				Filename:   fmt.Sprintf("list_%d.jpg", i),
				Hash:       uuid.New().String(),
				SizeBytes:  100,
				CapturedAt: time.Now().Add(time.Duration(i) * time.Second),
			}
			if err := repo.Create(context.Background(), p); err != nil {
				t.Fatalf("Failed to create photo in List test index %d: %v", i, err)
			}
		}

		photos, total, err := repo.List(context.Background(), 2, 0)
		if err != nil {
			t.Fatalf("Failed to list photos: %v", err)
		}

		if len(photos) != 2 {
			t.Errorf("Expected 2 photos, got %d", len(photos))
		}
		if total < count {
			t.Errorf("Expected total at least %d, got %d", count, total)
		}
	})

	t.Run("Update", func(t *testing.T) {
		photo := &domain.Photo{
			ID:         uuid.New(),
			Path:       "/tmp/test/upd.jpg",
			Filename:   "upd.jpg",
			Hash:       "hash_upd",
			SizeBytes:  500,
			CapturedAt: time.Now().Truncate(time.Microsecond),
		}
		if err := repo.Create(context.Background(), photo); err != nil {
			t.Fatalf("Failed to create photo for update test: %v", err)
		}

		photo.SizeBytes = 9999
		photo.Metadata = domain.Metadata{"updated": "true"}
		err := repo.Update(context.Background(), photo)
		if err != nil {
			t.Fatalf("Failed to update photo: %v", err)
		}

		retrieved, _ := repo.GetByID(context.Background(), photo.ID)
		if retrieved.SizeBytes != 9999 {
			t.Errorf("Expected size 9999, got %d", retrieved.SizeBytes)
		}
		if retrieved.Metadata["updated"] != "true" {
			t.Errorf("Expected metadata updated=true, got %s", retrieved.Metadata["updated"])
		}
	})

	t.Run("Delete", func(t *testing.T) {
		photo := &domain.Photo{
			ID:         uuid.New(),
			Path:       "/tmp/test/del.jpg",
			Filename:   "del.jpg",
			Hash:       "hash_del",
			CapturedAt: time.Now(),
		}
		if err := repo.Create(context.Background(), photo); err != nil {
			t.Fatalf("Failed to create photo for delete test: %v", err)
		}

		err := repo.Delete(context.Background(), photo.ID)
		if err != nil {
			t.Fatalf("Failed to delete photo: %v", err)
		}

		_, err = repo.GetByID(context.Background(), photo.ID)
		if err == nil {
			t.Error("Expected error getting deleted photo, got nil")
		}
	})
}
