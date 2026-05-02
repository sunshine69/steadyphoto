package database

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stevek/steadyphoto/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresPhotoRepository_Integration(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("Skipping integration test: DATABASE_URL not set")
	}

	repo, err := NewPostgresPhotoRepository(dbURL)
	require.NoError(t, err)
	defer repo.Close()

	ctx := context.Background()

	// Test Data
	id := uuid.New()
	photo := &domain.Photo{
		ID:         id.String(),
		Path:       "/tmp/test/2023/01/01/test.jpg",
		Filename:   "test.jpg",
		Hash:       "abc123hash",
		Size:       1024,
		Width:      1920,
		Height:     1080,
		CapturedAt: uint64(1672531200), // 2023-01-01
		Metadata:   map[string]string{"camera": "test-cam"},
	}

	t.Run("Create and GetByID", func(t *testing.T) {
		err := repo.Create(ctx, photo)
		require.NoError(t, err)

		got, err := repo.GetByID(ctx, id.String())
		require.NoError(t, err)
		assert.Equal(t, photo.ID, got.ID)
		assert.Equal(t, photo.Hash, got.Hash)
		assert.Equal(t, photo.Metadata["camera"], got.Metadata["camera"])
	})

	t.Run("GetByHash", func(t *testing.T) {
		got, err := repo.GetByHash(ctx, "abc123hash")
		require.NoError(t, err)
		assert.Equal(t, photo.ID, got.ID)
	})

	t.Run("GetNonExistent", func(t *testing.T) {
		_, err := repo.GetByID(ctx, uuid.New().String())
		assert.Error(t, err)
	})
}
