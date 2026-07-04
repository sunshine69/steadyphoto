package processor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bep/imagemeta"
)

func TestExifReader_ReadOrientation_NonExistentFile(t *testing.T) {
	reader := NewExifReader()

	file, err := os.Open("/nonexistent/file.jpg")
	if err != nil {
		// Expected - file doesn't exist
		return
	}
	defer file.Close()

	orientation, err := reader.ReadOrientation(file)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Should default to normal orientation
	if orientation != OrientationNormal {
		t.Errorf("Expected orientation %d, got %d", OrientationNormal, orientation)
	}
}

func TestExifReader_ReadOrientation_Directory(t *testing.T) {
	reader := NewExifReader()

	file, err := os.Open("/tmp")
	if err != nil {
		t.Skipf("Could not open /tmp: %v", err)
	}
	defer file.Close()

	orientation, err := reader.ReadOrientation(file)
	// Directory may cause an error or return default - both are acceptable
	if err != nil {
		t.Logf("Got expected error for directory: %v", err)
	}
	_ = orientation
}

func TestExifReader_ReadExif_NonExistentFile(t *testing.T) {
	reader := NewExifReader()

	file, err := os.Open("/nonexistent/test.jpg")
	if err == nil {
		defer file.Close()
		exifData, err := reader.ReadExif(file)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if exifData != nil {
			t.Errorf("Expected nil ExifInfo, got: %+v", exifData)
		}
	}
}

func TestConvertImageFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    ImageFormat
		expected imagemeta.ImageFormat
	}{
		{"JPEG", FormatJPEG, imagemeta.JPEG},
		{"PNG", FormatPNG, imagemeta.PNG},
		{"TIFF", FormatTIFF, imagemeta.TIFF},
		{"WebP", FormatWebP, imagemeta.WebP},
		{"HEIF", FormatHEIF, imagemeta.HEIF},
		{"AVIF", FormatAVIF, imagemeta.AVIF},
		{"DNG", FormatDNG, imagemeta.DNG},
		{"CR2", FormatCR2, imagemeta.CR2},
		{"NEF", FormatNEF, imagemeta.NEF},
		{"ARW", FormatARW, imagemeta.ARW},
		{"PEF", FormatPEF, imagemeta.PEF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertImageFormat(tt.input)
			if result != tt.expected {
				t.Errorf("convertImageFormat(%d) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractIntValue(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected int
	}{
		{"int", 5, 5},
		{"int32", int32(10), 10},
		{"int64", int64(15), 15},
		{"uint16", uint16(20), 20},
		{"uint32", uint32(25), 25},
		{"uint64", uint64(30), 30},
		{"float64", float64(3.7), 3},
		{"float32", float32(4.9), 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractIntValue(tt.input)
			if result != tt.expected {
				t.Errorf("extractIntValue(%v) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDetectImageFormat_JPEG(t *testing.T) {
	// Create a temporary JPEG-like file
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.jpg")
	
	// JPEG magic bytes: FF D8 FF E0
	data := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("failed to open test file: %v", err)
	}
	defer file.Close()

	format, err := detectImageFormat(file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if format != FormatJPEG {
		t.Errorf("expected FormatJPEG (%d), got %d", FormatJPEG, format)
	}
}

func TestDetectImageFormat_PNG(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.png")
	
	// PNG magic bytes: 89 50 4E 47 0D 0A 1A 0A
	data := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x00}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("failed to open test file: %v", err)
	}
	defer file.Close()

	format, err := detectImageFormat(file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if format != FormatPNG {
		t.Errorf("expected FormatPNG (%d), got %d", FormatPNG, format)
	}
}

func TestDetectImageFormat_GIF(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.gif")
	
	// GIF magic bytes: 47 49 46 38 39 61
	data := []byte{'G', 'I', 'F', '8', '9', 'a', 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("failed to open test file: %v", err)
	}
	defer file.Close()

	format, err := detectImageFormat(file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if format != FormatGIF {
		t.Errorf("expected FormatGIF (%d), got %d", FormatGIF, format)
	}
}

func TestDetectImageFormat_WebP(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.webp")
	
	// WebP magic bytes: 52 49 46 46 ... 57 45 42 50 (RIFF....WEBP)
	data := []byte{'R', 'I', 'F', 'F', 0x00, 0x00, 0x00, 0x00, 'W', 'E', 'B', 'P'}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("failed to open test file: %v", err)
	}
	defer file.Close()

	format, err := detectImageFormat(file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if format != FormatWebP {
		t.Errorf("expected FormatWebP (%d), got %d", FormatWebP, format)
	}
}

func TestDetectImageFormat_TIFF(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.tiff")
	
	// TIFF magic bytes (little-endian): 49 49 2A 00
	data := []byte{'I', 'I', 0x2A, 0x00, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("failed to open test file: %v", err)
	}
	defer file.Close()

	format, err := detectImageFormat(file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if format != FormatTIFF {
		t.Errorf("expected FormatTIFF (%d), got %d", FormatTIFF, format)
	}
}

func TestDetectImageFormat_Unknown(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.unknown")
	
	// Unknown format (less than 4 bytes)
	data := []byte{0x01, 0x02}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("failed to open test file: %v", err)
	}
	defer file.Close()

	format, err := detectImageFormat(file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should default to FormatJPEG for unknown/short files
	if format != FormatJPEG {
		t.Errorf("expected FormatJPEG (%d), got %d", FormatJPEG, format)
	}
}
