package storage

import (
	"encoding/json"
	"fmt"
	"github.com/jbrodriguez/mlog"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// StorageService manages access to the physical media files, 
// providing isolation between different users at the filesystem level.
type StorageService struct {
	baseDir string // The root directory for media files (e.g., "/mnt/data/storage")
	thumbDir string // The root directory for thumbnails (e.g., "/mnt/data/storage/.thumbnails")
}

// NewStorageService creates a new service with a base directory for media and an optional separate
// directory for thumbnails. Pass an empty string for thumbDir to co-locate thumbnails with media.
func NewStorageService(baseDir string, thumbDir string) *StorageService {
	return &StorageService{
		baseDir:  filepath.Clean(baseDir),
		thumbDir: filepath.Clean(thumbDir),
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

// GetAbsolutePath returns the absolute path for a given relative path within the media base directory.
// It does NOT check if the file exists.
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

// GetThumbnailAbsolutePath returns the absolute path for a given thumbnail relative path.
// Thumbnails are stored in a separate root directory (thumbDir) with the same relative structure.
// E.g., if thumbRelPath is "user_id/2024/01/01/photo_thumb.webp" and thumbDir is "storage/.thumbnails",
// this returns "storage/.thumbnails/user_id/2024/01/01/photo_thumb.webp".
func (s *StorageService) GetThumbnailAbsolutePath(thumbRelPath string) string {
	rel := filepath.Clean("/" + thumbRelPath)
	safeRel := strings.TrimPrefix(rel, string(os.PathSeparator))
	return filepath.Join(s.thumbDir, safeRel)
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

// DeleteThumbnailSilently removes a thumbnail file from the thumbnail storage directory
// without returning an error if it doesn't exist. This is the correct method for deleting
// thumbnails because they live in a separate directory (thumbDir), not the media base directory.
func (s *StorageService) DeleteThumbnailSilently(thumbRelPath string) {
	fullPath := s.GetThumbnailAbsolutePath(thumbRelPath)
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
		s.DeleteThumbnailSilently(thumbRelPath) // Use correct thumbDir, not baseDir
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

// CleanupOrphanedChunks safely removes chunk temp files that don't belong to any active session.
// This function loads the session registry and only deletes files that are truly orphaned.
// Returns the number of files removed.
func (s *StorageService) CleanupOrphanedChunks() int {
	tempDir := s.GetUploadTempDir()

	// Load active sessions to know which chunks are still in use
	activeSessions := s.loadActiveSessions(tempDir)

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		mlog.Info("[ERROR] StorageService: Failed to read upload temp dir %s: %v", tempDir, err)
		return 0
	}

	removed := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".tmp") {
			// Extract session ID from filename (format: {sessionID}.tmp or {sessionID}_{chunkIndex}.tmp)
			sessionID := extractSessionIDFromFilename(entry.Name())
			
			// Skip if this file belongs to an active session
			if activeSessions[sessionID] {
				mlog.Info("[INFO] StorageService: Skipping chunk file %s (belongs to active session %s)", entry.Name(), sessionID)
				continue
			}

			filePath := filepath.Join(tempDir, entry.Name())
			if err := os.Remove(filePath); err != nil {
				mlog.Info("[WARN] StorageService: Failed to remove orphaned chunk %s: %v", filePath, err)
			} else {
				mlog.Info("[INFO] StorageService: Removed orphaned chunk file: %s", filePath)
				removed++
			}
		}
	}

	return removed
}

// loadActiveSessions loads session IDs from the .sessions.json file in the storage root directory
func (s *StorageService) loadActiveSessions(tempDir string) map[string]bool {
	activeSessions := make(map[string]bool)
	// Sessions are stored in the storage root, not in the upload temp directory
	sessionFile := filepath.Join(s.baseDir, ".sessions.json")

	data, err := os.ReadFile(sessionFile)
	if err != nil {
		mlog.Info("[INFO] StorageService: No session file found at %s: %v", sessionFile, err)
		return activeSessions
	}

	// Parse sessions JSON (format: map[string]*UploadSession)
	var sessions map[string]map[string]interface{}
	if err := json.Unmarshal(data, &sessions); err != nil {
		mlog.Info("[ERROR] StorageService: Failed to parse session file: %v", err)
		return activeSessions
	}

	// Extract session IDs
	for sessionID := range sessions {
		activeSessions[sessionID] = true
	}

	mlog.Info("[INFO] StorageService: Loaded %d active sessions from %s", len(activeSessions), sessionFile)
	return activeSessions
}

// extractSessionIDFromFilename extracts the session ID from a .tmp filename
// Format: {sessionID}.tmp or {sessionID}_{chunkIndex}.tmp
func extractSessionIDFromFilename(filename string) string {
	// Remove .tmp suffix
	name := strings.TrimSuffix(filename, ".tmp")
	
	// Check if it has chunk index (contains underscore)
	if idx := strings.LastIndex(name, "_"); idx != -1 {
		return name[:idx]
	}
	
	// No chunk index, the whole name is the session ID
	return name
}
