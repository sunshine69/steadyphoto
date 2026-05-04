package processor_test

import (
	"context"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"steadyphoto/internal/domain"
	"steadyphoto/internal/processor"
)

// TestThumbnailProcessor_ProcessJob tests the high-level orchestration of the thumbnail processor.
func TestThumbnailProcessor_ProcessJob(t *testing.T) {
	ctx := context.Background()
	tmpDir, err := os.MkdirTemp("", "thumb_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// The storage root is the tmpDir
	storageRoot := tmpDir
	thumbRoot := filepath.Join(tmpDir, ".thumbnails")

	// 1. Setup Input Image
	inputRelPath := filepath.Join("2023", "01", "01", "test.jpg")
	inputAbsPath := filepath.Join(storageRoot, inputRelPath)
	err = os.MkdirAll(filepath.Dir(inputAbsPath), 0755)
	if err != nil {
		t.Fatal(err)
	}

	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{0, 255, 0, 255}}, image.Point{}, draw.Src)
	f, err := os.Create(inputAbsPath)
	if err != nil {
		t.Fatal(err)
	}
	err = jpeg.Encode(f, img, nil)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	// 2. Setup Processor and Mock Engine
	mockEngine := &processor.MockImageEngine{}
	thumbProc := processor.NewThumbnailProcessor(mockEngine, storageRoot, thumbRoot)

	photoID := uuid.New()
	photo := &domain.Photo{
		ID:   photoID,
		Path: inputRelPath,
	}

	job := &domain.Job{
		ID:      uuid.New(),
		Type:    domain.JobTypeThumbnail,
		PhotoID: photoID,
	}

	// 3. Execute
	err = thumbProc.ProcessJob(ctx, job, photo)
	if err != nil {
		t.Fatalf("ProcessJob failed: %v", err)
	}

	// 4. Verify
	// Check if engine was called correctly
	if mockEngine.LastInputPath != inputAbsPath {
		t.Errorf("Expected input path %s, got %s", inputAbsPath, mockEngine.LastInputPath)
	}
	if mockEngine.LastWidth != 800 {
		t.Errorf("Expected width 800, got %d", mockEngine.LastWidth)
	}

	// Check if file exists at expected location
	expectedThumbPath := filepath.Join(thumbRoot, "2023/01/01/test_thumb.webp")
	if _, err := os.Stat(expectedThumbPath); os.IsNotExist(err) {
		t.Errorf("Expected thumbnail file not found at %s", expectedThumbPath)
	}
}
