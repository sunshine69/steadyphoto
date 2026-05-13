package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// StorageService manages access to the physical media files, 
// providing isolation between different users at the filesystem level.
type StorageService struct {
	baseDir string // The root directory (e.g., "/mnt/data/storage")
}

// NewStorageService creates a new service with a base directory.
func NewStorageService(baseDir string) *StorageService {
	return &StorageService{
		baseDir: filepath.Clean(baseDir),
	}
}

// GetUserRelativePath constructs the logical path used for storage organization.
// It returns a path that includes the user's ID to ensure physical isolation.
// Example input: userID="abc-123", relTimePath="2024/05/13/photo.jpg"
// Returns: "abc-123/2024/05/13/photo.jpg"
func (s *StorageService) GetUserRelativePath(userID uuid.UUID, relTimePath string) string {
	return filepath.Join(userID.String(), filepath.Clean(relTimePath))
}

// ResolveUserFile takes a userID and the relative path stored in the database 
// and returns the absolute path on the local filesystem inside that user's folder.
func (s *StorageService) ResolveUserFile(userID uuid.UUID, dbRelPath string) (string, error) {
	userScopedPath := s.GetUserRelativePath(userID, dbRelPath)
	return s.ResolvePath(userScopedPath)
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

	// 3. Normalize the relative path for comparison
	normalizedRel := strings.TrimLeft(relativePath, "/\\.")

	// 4. Handle the "Duplicate BaseDir" edge case.
	baseName := filepath.Base(s.baseDir)
	if strings.HasPrefix(normalizedRel, baseName+string(os.PathSeparator)) || normalizedRel == baseName {
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

// EnsureDir creates the directory structure for a given user-scoped relative path.
func (s *StorageService) EnsureDir(userID uuid.UUID, relTimePath string) error {
	userScopedDir := filepath.Join(s.GetUserRelativePath(userID, ""), filepath.Dir(relTimePath))
	fullDirPath := filepath.Join(s.baseDir, userScopedDir)

	if err := os.MkdirAll(fullDirPath, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory %s: %w", fullDirPath, err)
	}
	return nil
}
