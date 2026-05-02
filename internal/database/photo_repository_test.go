package database

import (
	"context"
	"os"
	"testing"
	"time"

	"steadyphoto/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func getTestDB(t *testing.T) *sqlx.DB {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Fatal("DATABASE_URL environment variable is not set")
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	// Clean up after tests
	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func TestPostgresPhotoRepository(t *testing.T) {
	db := getTestDB(t)
	ctx := context.Background()
	repo := NewPostgresPhotoRepository(db)

	// 1. Test Create
	p := &domain.Photo{
		Hash:       "test-hash-123",
		Filename:   "test.jpg",
		Path:       "/tmp/test/test.jpg",
		SizeBytes:  1024,
		Width:      800,
		Height:     600,
		CapturedAt: time.Now().Truncate(time.Second),
		Metadata:   map[string]string{"camera": "test-cam"},
	}

	err := repo.Create(ctx, p)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if p.ID == uuid.Nil {
		t.Error("expected photo ID to be populated")
	}

	// 2. Test GetByID
	found, err := repo.GetByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if found.Hash != p.Hash {
		t.Errorf("expected hash %s, got %s", p.Hash, found.Hash)
	}
	if found.SizeBytes != p.SizeBytes {
		t.Errorf("expected size %d, got %d", p.SizeBytes, found.SizeBytes)
	}
	if found.Metadata["camera"] != "test-cam" {
		t.Errorf("expected metadata camera=test-cam, got %s", found.Metadata["camera"])
	}

	// 3. Test GetByHash
	foundByHash, err := repo.GetByHash(ctx, p.Hash)
	if err != nil {
		t.Fatalf("GetByHash failed: %v", err)
	}
	if foundByHash.ID != p.ID {
		t.Errorf("expected ID %s, got %s", p.ID, foundByHash.ID)
	}

	// 4. Test List
	photos, err := repo.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(photos) == 0 {
		t.Error("expected at least one photo in list")
	}

	// 5. Test Update
	p.Filename = "updated.jpg"
	err = repo.Update(ctx, p)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	updated, err := repo.GetByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetByID after update failed: %v", err)
	}
	if updated.Filename != "updated.jpg" {
		t.Errorf("expected filename updated.jpg, got %s", updated.Filename)
	}
}

func TestPostgresFaceRepository(t *testing.T) {
	db := getTestDB(t)
	ctx := context.Background()
	photoRepo := NewPostgresPhotoRepository(db)
	faceRepo := NewPostgresFaceRepository(db)

	// Setup: Create a photo first
	p := &domain.Photo{
		Hash:      "face-test-hash",
		Filename:  "face.jpg",
		Path:      "/tmp/face.jpg",
		SizeBytes: 2048,
	}
	err := photoRepo.Create(ctx, p)
	if err != nil {
		t.Fatalf("failed to setup photo for face test: %v", err)
	}

	// 1. Test Create Face
	f := &domain.Face{
		PhotoID:     p.ID,
		BoundingBox: map[string]float64{"x": 10, "y": 20, "w": 100, "h": 100},
		Embedding:   make([]float64, 512), // zero vector
	}
	f.Embedding[0] = 0.5

	err = faceRepo.Create(ctx, f)
	if err != nil {
		t.Fatalf("Create face failed: %v", err)
	}

	// 2. Test GetByPhotoID
	faces, err := faceRepo.GetByPhotoID(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetByPhotoID failed: %v", err)
	}
	if len(faces) != 1 {
		t.Fatalf("expected 1 face, got %d", len(faces))
	}
	if faces[0].Embedding[0] != 0.5 {
		t.Errorf("expected embedding[0]=0.5, got %f", faces[0].Embedding[0])
	}
}

func TestPostgresJobRepository(t *testing.T) {
	db := getTestDB(t)
	ctx := context.Background()
	jobRepo := NewPostgresJobRepository(db)

	// Setup: Create a photo for the job
	photoRepo := NewPostgresPhotoRepository(db)
	p := &domain.Photo{Hash: "job-test-hash", Filename: "job.jpg", Path: "/job.jpg", SizeBytes: 100}
	photoRepo.Create(ctx, p)

	// 1. Test Create Job
	j := &domain.Job{
		PhotoID: p.ID,
		Type:    domain.JobTypeFaceDetection,
		Status:  domain.JobStatusPending,
	}
	err := jobRepo.Create(ctx, j)
	if err != nil {
		t.Fatalf("Create job failed: %v", err)
	}

	// 2. Test GetPending
	pending, err := jobRepo.GetPending(ctx, 10)
	if err != nil {
		t.Fatalf("GetPending failed: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending job, got %d", len(pending))
	}
	if pending[0].ID != j.ID {
		t.Errorf("expected job ID %s, got %s", j.ID, pending[0].ID)
	}

	// 3. Test UpdateStatus
	var errMsg *string = nil
	err = jobRepo.UpdateStatus(ctx, j.ID, domain.JobStatusCompleted, errMsg)
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	// 4. Verify status change
	updated, err := jobRepo.GetByID(ctx, j.ID)
	if err != nil {
		t.Fatalf("GetByID after update failed: %v", err)
	}
	if updated.Status != domain.JobStatusCompleted {
		t.Errorf("expected status completed, got %s", updated.Status)
	}
}
