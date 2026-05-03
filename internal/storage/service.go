package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StorageService manages access to the physical media files.
type StorageService struct {
	baseDir string
}

// NewStorageService creates a new service with a base directory.
func NewStorageService(baseDir string) *StorageService {
	return &StorageService{
		baseDir: filepath.Clean(baseDir),
	}
}

// ResolvePath takes a path (either relative to baseDir or absolute) 
// and returns the full, absolute path on the local filesystem.
func (s *StorageService) ResolvePath(relativePath string) (string, error) {
	// 1. Clean the input
	relativePath = filepath.Clean(relativePath)

	// 2. If the path is already absolute, just verify it exists
	if filepath.IsAbs(relativePath) {
		if _, err := os.Stat(relativePath); err != nil {
			return "", fmt.Errorf("file not found at absolute path: %w", err)
		}
		return relativePath, nil
	}

	// 3. Normalize the relative path for comparison:
	// Strip leading slashes and current-directory dots (e.g., "/storage/..." -> "storage/...")
	normalizedRel := strings.TrimLeft(relativePath, "/\\.")

	// 4. Handle the "Duplicate BaseDir" edge case.
	// We check if the normalized path starts with our base directory's name.
	baseName := filepath.Base(s.baseDir)
	
	// Check if the path starts with "storage/" or just "storage"
	if strings.HasPrefix(normalizedRel, baseName+string(os.PathSeparator)) || normalizedRel == baseName {
		// Strip the base name and any following separator
		normalizedRel = strings.TrimPrefix(normalizedRel, baseName)
		normalizedRel = strings.TrimLeft(normalizedRel, string(os.PathSeparator))
		relativePath = filepath.Clean(normalizedRel)
	}

	// 5. Join the cleaned relative path with the base directory
	fullPath := filepath.Join(s.baseDir, relativePath)

	// 6. Verify the file exists
	if _, err := os.Stat(fullPath); err != nil {
		return "", fmt.Errorf("file not found at resolved path: %w", err)
	}

	return fullPath, nil
}
