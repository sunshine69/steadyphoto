package mobile

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// MediaHasher computes SHA256 hash of a file at the given path.
// Returns hex-encoded hash string or error.
func MediaHasher(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to hash file: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// FileMetadata holds basic file information.
type FileMetadata struct {
	Size         int64  `json:"size"`
	MimeType     string `json:"mime_type"`
}

// GetFileMetadata returns metadata for a given file path.
func GetFileMetadata(filePath string) (FileMetadata, error) {
	var meta FileMetadata

	info, err := os.Stat(filePath)
	if err != nil {
		return meta, fmt.Errorf("failed to stat file: %w", err)
	}

	meta.Size = info.Size()

	// Determine MIME type from extension
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".jpg", ".jpeg":
		meta.MimeType = "image/jpeg"
	case ".png":
		meta.MimeType = "image/png"
	case ".gif":
		meta.MimeType = "image/gif"
	case ".bmp":
		meta.MimeType = "image/bmp"
	case ".webp":
		meta.MimeType = "image/webp"
	case ".mp4", ".m4v":
		meta.MimeType = "video/mp4"
	case ".mov":
		meta.MimeType = "video/quicktime"
	case ".avi":
		meta.MimeType = "video/x-msvideo"
	case ".mkv":
		meta.MimeType = "video/x-matroska"
	default:
		meta.MimeType = "application/octet-stream"
	}

	return meta, nil
}
