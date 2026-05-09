package processor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"steadyphoto/internal/domain"
)

// ThumbnailProcessor handles the generation of image previews.
type ThumbnailProcessor struct {
	engine      ImageEngine
	storageRoot string // The root directory of the photo storage
	thumbRoot   string // The root directory where thumbnails are stored (e.g. storage/.thumbnails)
}

// NewThumbnailProcessor creates a new instance of the processor.
func NewThumbnailProcessor(engine ImageEngine, storageRoot string, thumbRoot string) *ThumbnailProcessor {
	return &ThumbnailProcessor{
		engine:      engine,
		storageRoot: storageRoot,
		thumbRoot:   thumbRoot,
	}
}

// ProcessJob takes a background job and performs the thumbnail generation.
func (p *ThumbnailProcessor) ProcessJob(ctx context.Context, job *domain.Job, photo *domain.Media) error {
	if job.Type != domain.JobTypeThumbnail {
		return fmt.Errorf("invalid job type: %s", job.Type)
	}

	if photo.Path == "" {
		return fmt.Errorf("photo path is empty")
	}

	// 1. Resolve the absolute path of the original photo
	inputAbsPath := filepath.Join(p.storageRoot, photo.Path)

	// 2. Calculate the destination path
	// If original is "2023/01/01/photo.jpg"
	// Thumbnail will be "storage/.thumbnails/2023/01/01/photo_thumb.webp"
	thumbAbsPath, err := p.getThumbnailAbsPath(photo.Path)
	if err != nil {
		return fmt.Errorf("failed to calculate thumbnail path: %w", err)
	}

	// Ensure the thumbnail subdirectories exist
	err = os.MkdirAll(filepath.Dir(thumbAbsPath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create thumbnail subdirectories: %w", err)
	}

	// 3. Execute the resize via the engine
	// We target a standard width of 800px for previews
	err = p.engine.Resize(ctx, inputAbsPath, thumbAbsPath, 800)
	if err != nil {
		return fmt.Errorf("engine resize failed: %w", err)
	}

	return nil
}

// getThumbnailAbsPath maps an original relative photo path to its absolute thumbnail path.
func (p *ThumbnailProcessor) getThumbnailAbsPath(originalRelPath string) (string, error) {
	// Clean the path
	relPath := filepath.Clean(originalRelPath)
	
	// Remove the original extension and add thumb suffix
	ext := filepath.Ext(relPath)
	base := strings.TrimSuffix(relPath, ext)
	
	// We'll use .webp for thumbnails as it's efficient
	// The relative path inside thumbRoot will be ".thumbnails/YYYY/MM/DD/filename_thumb.webp"
	// but since thumbRoot is already "storage/.thumbnails", we just need "YYYY/MM/DD/..."
	
	// However, the original relPath might start with "storage/" depending on how it was stored.
	// The design doc says: "Stores relative paths in the database."
	// So if photo.Path is "2023/01/01/img.jpg", then thumbRel is "2023/01/01/img_thumb.webp"
	
	thumbRelPath := filepath.Join(base+"_thumb.webp")
	
	// Join with the thumb root directory
	fullThumbPath := filepath.Join(p.thumbRoot, thumbRelPath)
	
	return fullThumbPath, nil
}
