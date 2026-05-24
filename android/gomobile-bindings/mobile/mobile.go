package mobile

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	exif "github.com/xor-gate/go-exif/v3"
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

// ExifMetadata holds extracted EXIF data from an image file.
type ExifMetadata struct {
	CameraModel string    `json:"camera_model"`
	ISO         int       `json:"iso"`
	GPSLatitude float64   `json:"gps_latitude"`
	GPSLongitude float64  `json:"gps_longitude"`
	CaptureTime time.Time `json:"capture_time"`
	ImageWidth  int       `json:"image_width"`
	ImageHeight int       `json:"image_height"`
}

// ExifParser extracts metadata from an image file.
// Returns a map of key-value pairs for simple access, or structured ExifMetadata.
func ExifParser(filePath string) (ExifMetadata, error) {
	var meta ExifMetadata

	f, err := os.Open(filePath)
	if err != nil {
		return meta, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	e, err := exif.Decode(f)
	if err != nil {
		// Not all files have EXIF data; return zero values without error for non-image files
		if strings.Contains(err.Error(), "not an image") || strings.Contains(err.Error(), "unknown format") {
			return meta, nil
		}
		return meta, fmt.Errorf("failed to decode exif: %w", err)
	}

	// Camera Model
	model, _ := e.Get(exif.Model)
	if model != nil {
		meta.CameraModel = model.StringVal()
	}

	// ISO Speed
	iso, _ := e.Get(exif.ISOSpeedRatings)
	if iso != nil {
		vals, _ := iso.Values(0)
		if len(vals) > 0 {
			meta.ISO = int(vals[0].Int64())
		}
	}

	// GPS Coordinates
	lat, lon, err := e.LatLong()
	if err == nil {
		meta.GPSLatitude = lat
		meta.GPSLongitude = lon
	}

	// Capture Time
	timestamp, _ := e.Get(exif.DateTimeOriginal)
	if timestamp != nil {
		meta.CaptureTime, _ = timestamp.Time()
	}

	// Image Dimensions
	width, _ := e.Get(exif.ImageWidth)
	height, _ := e.Get(exif.ImageLength)
	if width != nil {
		meta.ImageWidth = int(width.Values(0).Int64())
	}
	if height != nil {
		meta.ImageHeight = int(height.Values(0).Int64())
	}

	return meta, nil
}

// FileMetadata returns basic file info (size, modification time).
type FileMetadata struct {
	Size         int64     `json:"size"`
	ModificationTime time.Time `json:"modification_time"`
	MimeType     string    `json:"mime_type"`
}

// GetFileMetadata returns metadata for a given file path.
func GetFileMetadata(filePath string) (FileMetadata, error) {
	var meta FileMetadata

	info, err := os.Stat(filePath)
	if err != nil {
		return meta, fmt.Errorf("failed to stat file: %w", err)
	}

	meta.Size = info.Size()
	meta.ModificationTime = info.ModTime()

	// Determine MIME type from extension
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".jpg", ".jpeg":
		meta.MimeType = "image/jpeg"
	case ".png":
		meta.MimeType = "image/png"
	case ".mp4":
		meta.MimeType = "video/mp4"
	case ".mov":
		meta.MimeType = "video/quicktime"
	case ".avi":
		meta.MimeType = "video/x-msvideo"
	default:
		meta.MimeType = "application/octet-stream"
	}

	return meta, nil
}
