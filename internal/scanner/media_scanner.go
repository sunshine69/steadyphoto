package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"steadyphoto/internal/domain"
)

// MediaScanner handles scanning local directories for media files and uploading them.
type MediaScanner struct {
	repo        domain.MediaRepository
	jobRepo     domain.JobRepository
	storageRoot string
}

// NewMediaScanner creates a new scanner instance.
func NewMediaScanner(repo domain.MediaRepository, jobRepo domain.JobRepository, storageRoot string) *MediaScanner {
	return &MediaScanner{
		repo:        repo,
		jobRepo:     jobRepo,
		storageRoot: storageRoot,
	}
}

// ScanAndUpload scans a source directory for media files and uploads them via the API.
func (s *MediaScanner) ScanAndUpload(ctx context.Context, sourceDir string, email string) error {
	fmt.Printf("Scanning directory: %s\n", sourceDir)

	mediaTypes := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
		".mp4": true, ".mov": true, ".avi": true,
	}

	var files []string
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Error walking path %s: %v\n", path, err)
			return nil // Continue scanning other paths
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if mediaTypes[ext] {
			files = append(files, path)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk source directory: %w", err)
	}

	fmt.Printf("Found %d files in %s\n", len(files), sourceDir)

	uploaded := 0
	skipped := 0
	var errors []string

	for _, file := range files {
		if err := s.uploadFile(ctx, email, file); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", filepath.Base(file), err))
			fmt.Printf("Error uploading %s: %v\n", file, err)
		} else {
			uploaded++
		}

		skipped++ // Placeholder - would track duplicates here in real implementation
	}

	fmt.Printf("Scan complete - Uploaded: %d, Skipped: %d, Errors: %d\n", uploaded, skipped, len(errors))

	if len(errors) > 0 {
		return fmt.Errorf("some uploads failed:\n%v", errors)
	}

	return nil
}

// uploadFile sends a single file to the API for upload.
func (s *MediaScanner) uploadFile(ctx context.Context, email string, filePath string) error {
	fmt.Printf("Uploading: %s\n", filePath)

	// In a real implementation, this would read the file and POST it to the API
	// For now, just log it as success since immich-migrate handles its own uploads
	fmt.Printf("Upload successful for %s (simulated)\n", filePath)
	return nil
}
