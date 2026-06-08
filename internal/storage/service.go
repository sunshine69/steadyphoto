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

// GetAbsolutePath returns the absolute path for a given relative path without checking if it exists.
func (s *StorageService) GetAbsolutePath(relativePath string) string {
	// To prevent path traversal and absolute path escapes, we treat all paths as 
	// being relative to the baseDir by cleaning them against a virtual root ("/")
	// before joining with our actual storage root.
	rel := filepath.Clean("/" + relativePath)

	// Remove the leading slash so that filepath.Join treats it as a relative path
	// rather than an absolute one (which would otherwise escape s.baseDir).
	safeRel := strings.TrimPrefix(rel, string(os.PathSeparator))

	return filepath.Join(s.baseDir, safeRel)
}

// ResolvePath takes a path (either relative to baseDir or absolute) 
// and returns the full, absolute path on the local filesystem.
func (s *StorageService) ResolvePath(relativePath string) (string, error) {
	fullPath := s.GetAbsolutePath(relativePath)

	// Verify the file exists
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

// DeleteFile removes a file from the filesystem. It takes either an absolute path 
// or a relative path and resolves it before deletion. Returns any error encountered.
func (s *StorageService) DeleteFile(path string) error {
	fullPath, err := s.ResolvePath(path)
	if err != nil {
		return fmt.Errorf("could not resolve file for deletion: %w", err)
	}

	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file at %s: %w", fullPath, err)
	}

	return nil
}

// DeleteFileSilently removes a file from the filesystem without returning an error if it doesn't exist.
func (s *StorageService) DeleteFileSilently(path string) {
	fullPath := s.GetAbsolutePath(path)
	os.Remove(fullPath) // Ignore errors - we don't care if file didn't exist
}

// DeleteMediaFiles deletes both the main media file and its thumbnail from storage.
func (s *StorageService) DeleteMediaFiles(mediaPath string, thumbRelPath string) error {
	// Try to delete main file
	if err := s.DeleteFile(mediaPath); err != nil {
		return fmt.Errorf("failed to delete media file: %w", err)
	}

	// Try to delete thumbnail (if it exists and is different from the original path)
	if thumbRelPath != "" && thumbRelPath != mediaPath {
		s.DeleteFileSilently(thumbRelPath) // Ignore error for optional thumbnails
	}

	return nil
}

// GetUploadTempDir returns the directory where temporary upload files are stored.
func (s *StorageService) GetUploadTempDir() string {
	tempDir := filepath.Join(s.baseDir, ".upload-temp")
	os.MkdirAll(tempDir, 0755) // Create if it doesn't exist
	return tempDir
}

// EnsureDirForPath creates the directory structure for a given relative path (without user scoping).
func (s *StorageService) EnsureDirForPath(relTimePath string) error {
	fullDirPath := s.GetAbsolutePath(filepath.Dir(relTimePath))
	if err := os.MkdirAll(fullDirPath, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory %s: %w", fullDirPath, err)
	}
	return nil
}

// GetThumbnailRelativePath returns the relative path for a thumbnail based on media type.
func (s *StorageService) GetThumbnailRelativePath(mediaType string, filename string, ext string) string {
	if strings.ToLower(mediaType) == "video" {
		cleanPath := strings.TrimPrefix(filename, "storage/")
		basePart := strings.TrimSuffix(cleanPath, ext)
		return basePart + ".webp"
	}

	cleanPath := strings.TrimPrefix(filename, "storage/")
	basePart := strings.TrimSuffix(cleanPath, ext)
	return basePart + "_thumb.webp"
}
