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

// MediaProcessor provides media-related operations for Android.
type MediaProcessor struct{}

// FileMetadata holds the metadata returned by GetFileMetadata.
type FileMetadata struct {
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
}

// MediaHasher computes SHA256 hash of a file at the given path.
func (mp MediaProcessor) MediaHasher(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to hash file: %v", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// GetFileMetadata returns metadata for a given file path.
func (mp MediaProcessor) GetFileMetadata(filePath string) (*FileMetadata, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %v", err)
	}

	size := info.Size()

	// Determine MIME type from extension
	ext := strings.ToLower(filepath.Ext(filePath))
	mimeType := "application/octet-stream"
	switch ext {
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".png":
		mimeType = "image/png"
	case ".gif":
		mimeType = "image/gif"
	case ".bmp":
		mimeType = "image/bmp"
	case ".webp":
		mimeType = "image/webp"
	case ".mp4", ".m4v":
		mimeType = "video/mp4"
	case ".mov":
		mimeType = "video/quicktime"
	case ".avi":
		mimeType = "video/x-msvideo"
	case ".mkv":
		mimeType = "video/x-matroska"
	}

	return &FileMetadata{Size: size, MimeType: mimeType}, nil
}
