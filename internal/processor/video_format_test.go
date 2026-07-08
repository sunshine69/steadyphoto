package processor

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectVideoFormatMP4(t *testing.T) {
	// Create a minimal MP4 file with ftyp box
	// ftyp box: [size=8][type='ftyp'][major_brand=mp41][compat=mp41]
	mp4Data := []byte{
		0x00, 0x00, 0x00, 0x18, // size: 24 bytes
		'f', 't', 'y', 'p',     // type: "ftyp"
		'm', 'p', '4', '1',     // major brand: mp41
		0x00, 0x00, 0x00, 0x00, // minor version
		'm', 'p', '4', '1',     // compatible brand: mp41
	}

	tmpFile := filepath.Join(t.TempDir(), "test.mp4")
	if err := os.WriteFile(tmpFile, mp4Data, 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	file, err := os.Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open temp file: %v", err)
	}
	defer file.Close()

	format, err := DetectVideoFormat(file)
	if err != nil {
		t.Fatalf("DetectVideoFormat returned error: %v", err)
	}

	if format != VideoFormatMP4 {
		t.Errorf("expected VideoFormatMP4, got %v", format)
	}
}

func TestDetectVideoFormatMOV(t *testing.T) {
	// Create a minimal MOV file with moov box
	// moov box: [size=8][type='moov']
	movData := []byte{
		0x00, 0x00, 0x00, 0x08, // size: 8 bytes
		'm', 'o', 'o', 'v',     // type: "moov"
	}

	tmpFile := filepath.Join(t.TempDir(), "test.mov")
	if err := os.WriteFile(tmpFile, movData, 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	file, err := os.Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open temp file: %v", err)
	}
	defer file.Close()

	format, err := DetectVideoFormat(file)
	if err != nil {
		t.Fatalf("DetectVideoFormat returned error: %v", err)
	}

	if format != VideoFormatMOV {
		t.Errorf("expected VideoFormatMOV, got %v", format)
	}
}

func TestDetectVideoFormatMOVWithFtyp(t *testing.T) {
	// Create a MOV file with QuickTime ftyp box
	// ftyp box with major brand "qt  "
	ftypData := []byte{
		0x00, 0x00, 0x00, 0x18, // size: 24 bytes
		'f', 't', 'y', 'p',     // type: "ftyp"
		'q', 't', ' ', ' ',     // major brand: "qt  " (QuickTime)
		0x00, 0x00, 0x00, 0x00, // minor version
		'q', 't', ' ', ' ',     // compatible brand
	}

	tmpFile := filepath.Join(t.TempDir(), "test_qt.mov")
	if err := os.WriteFile(tmpFile, ftypData, 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	file, err := os.Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open temp file: %v", err)
	}
	defer file.Close()

	format, err := DetectVideoFormat(file)
	if err != nil {
		t.Fatalf("DetectVideoFormat returned error: %v", err)
	}

	if format != VideoFormatMOV {
		t.Errorf("expected VideoFormatMOV (QuickTime ftyp), got %v", format)
	}
}

func TestDetectVideoFormatM4V(t *testing.T) {
	// Create a minimal M4V file with ftyp box and M4V major brand
	m4vData := []byte{
		0x00, 0x00, 0x00, 0x18, // size: 24 bytes
		'f', 't', 'y', 'p',     // type: "ftyp"
		'M', '4', 'V', ' ',     // major brand: "M4V "
		0x00, 0x00, 0x00, 0x00, // minor version
		'M', '4', 'V', ' ',     // compatible brand
	}

	tmpFile := filepath.Join(t.TempDir(), "test.m4v")
	if err := os.WriteFile(tmpFile, m4vData, 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	file, err := os.Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open temp file: %v", err)
	}
	defer file.Close()

	format, err := DetectVideoFormat(file)
	if err != nil {
		t.Fatalf("DetectVideoFormat returned error: %v", err)
	}

	if format != VideoFormatM4V {
		t.Errorf("expected VideoFormatM4V, got %v", format)
	}
}

func TestDetectVideoFormatM4P(t *testing.T) {
	// Create a minimal M4P file with ftyp box and M4P major brand (iTunes Plus)
	m4pData := []byte{
		0x00, 0x00, 0x00, 0x18, // size: 24 bytes
		'f', 't', 'y', 'p',     // type: "ftyp"
		'M', '4', 'P', ' ',     // major brand: "M4P "
		0x00, 0x00, 0x00, 0x00, // minor version
		'M', '4', 'P', ' ',     // compatible brand
	}

	tmpFile := filepath.Join(t.TempDir(), "test.m4p")
	if err := os.WriteFile(tmpFile, m4pData, 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	file, err := os.Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open temp file: %v", err)
	}
	defer file.Close()

	format, err := DetectVideoFormat(file)
	if err != nil {
		t.Fatalf("DetectVideoFormat returned error: %v", err)
	}

	if format != VideoFormatM4V {
		t.Errorf("expected VideoFormatM4V (M4P is treated as M4V), got %v", format)
	}
}

func TestDetectVideoFormatAVI(t *testing.T) {
	// Create a minimal AVI file
	// RIFF header followed by 'AVI '
	aviData := []byte{
		'R', 'I', 'F', 'F',     // RIFF marker
		0x00, 0x00, 0x00, 0x00, // file size placeholder
		'A', 'V', 'I', ' ',     // AVI marker
	}

	tmpFile := filepath.Join(t.TempDir(), "test.avi")
	if err := os.WriteFile(tmpFile, aviData, 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	file, err := os.Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open temp file: %v", err)
	}
	defer file.Close()

	format, err := DetectVideoFormat(file)
	if err != nil {
		t.Fatalf("DetectVideoFormat returned error: %v", err)
	}

	if format != VideoFormatAVI {
		t.Errorf("expected VideoFormatAVI, got %v", format)
	}
}

func TestDetectVideoFormatWebM(t *testing.T) {
	// Create a minimal WebM file with EBML header
	// EBML header: 1A 45 DF A3
	webmData := []byte{
		0x1A, 0x45, 0xDF, 0xA3, // EBML header magic
		0x01, 0x00, 0x00, 0x00, // version
		0x00, 0x00, 0x00, 0x00, // read version
		0x00, 0x00, 0x00, 0x00, // max ID length
		0x00, 0x00, 0x00, 0x00, // max size length
	}

	tmpFile := filepath.Join(t.TempDir(), "test.webm")
	if err := os.WriteFile(tmpFile, webmData, 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	file, err := os.Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open temp file: %v", err)
	}
	defer file.Close()

	format, err := DetectVideoFormat(file)
	if err != nil {
		t.Fatalf("DetectVideoFormat returned error: %v", err)
	}

	if format != VideoFormatWebM {
		t.Errorf("expected VideoFormatWebM, got %v", format)
	}
}

func TestDetectVideoFormatUnknown(t *testing.T) {
	// Create a file with random data that doesn't match any format
	unknownData := []byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F,
	}

	tmpFile := filepath.Join(t.TempDir(), "test.unknown")
	if err := os.WriteFile(tmpFile, unknownData, 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	file, err := os.Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open temp file: %v", err)
	}
	defer file.Close()

	format, err := DetectVideoFormat(file)
	if err != nil {
		t.Fatalf("DetectVideoFormat returned error: %v", err)
	}

	if format != VideoFormatUnknown {
		t.Errorf("expected VideoFormatUnknown, got %v", format)
	}
}

func TestDetectVideoFormatTooSmall(t *testing.T) {
	// Create a file that's too small for format detection
	smallData := []byte{0x00, 0x01}

	tmpFile := filepath.Join(t.TempDir(), "test_small")
	if err := os.WriteFile(tmpFile, smallData, 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	file, err := os.Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open temp file: %v", err)
	}
	defer file.Close()

	format, err := DetectVideoFormat(file)
	if err == nil {
		t.Errorf("expected error for too-small file, got %v", format)
	}
}

func TestDetectVideoFormatFileNotFound(t *testing.T) {
	// Try to detect format on a non-existent file
	file, err := os.Open("/nonexistent/path/test.mp4")
	if err == nil {
		file.Close()
		t.Fatal("expected error opening non-existent file")
	}

	// The function should handle this gracefully
	if file != nil {
		format, err := DetectVideoFormat(file)
		if err == nil {
			t.Errorf("expected error for non-existent file, got %v", format)
		}
	}
}

func TestVideoFormatString(t *testing.T) {
	tests := []struct {
		format VideoFormat
		want   string
	}{
		{VideoFormatMP4, "MP4"},
		{VideoFormatMOV, "MOV"},
		{VideoFormatM4V, "M4V"},
		{VideoFormatAVI, "AVI"},
		{VideoFormatWebM, "WebM"},
		{VideoFormatUnknown, "Unknown"},
		{VideoFormat(99), "Unknown"}, // Invalid value
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.format.String(); got != tt.want {
				t.Errorf("VideoFormat(%d).String() = %q, want %q", tt.format, got, tt.want)
			}
		})
	}
}

func TestVideoFormatIsSupported(t *testing.T) {
	tests := []struct {
		format VideoFormat
		want   bool
	}{
		{VideoFormatMP4, true},
		{VideoFormatMOV, true},
		{VideoFormatM4V, true},
		{VideoFormatAVI, true},
		{VideoFormatWebM, true},
		{VideoFormatUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.format.String(), func(t *testing.T) {
			if got := tt.format.IsSupported(); got != tt.want {
				t.Errorf("VideoFormat(%d).IsSupported() = %v, want %v", tt.format, got, tt.want)
			}
		})
	}
}

func TestDetectVideoFormatByExtension(t *testing.T) {
	tests := []struct {
		extension string
		want      VideoFormat
	}{
		{".mp4", VideoFormatMP4},
		{".mov", VideoFormatMOV},
		{".m4v", VideoFormatM4V},
		{".avi", VideoFormatAVI},
		{".webm", VideoFormatWebM},
		{".mpg", VideoFormatUnknown},
		{".mkv", VideoFormatUnknown},
		{"", VideoFormatUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.extension, func(t *testing.T) {
			got := DetectVideoFormatByExtension(tt.extension)
			if got != tt.want {
				t.Errorf("DetectVideoFormatByExtension(%q) = %v, want %v", tt.extension, got, tt.want)
			}
		})
	}
}

func TestDetectVideoFormatMultipleFTypes(t *testing.T) {
	// Test different ftyp major brands
	tests := []struct {
		name    string
		data    []byte
		want    VideoFormat
	}{
		{
			name: "mp41 major brand",
			data: []byte{
				0x00, 0x00, 0x00, 0x18,
				'f', 't', 'y', 'p',
				'm', 'p', '4', '1',
				0x00, 0x00, 0x00, 0x00,
				'm', 'p', '4', '1',
			},
			want: VideoFormatMP4,
		},
		{
			name: "isom major brand",
			data: []byte{
				0x00, 0x00, 0x00, 0x18,
				'f', 't', 'y', 'p',
				'i', 's', 'o', 'm',
				0x00, 0x00, 0x00, 0x00,
				'i', 's', 'o', 'm',
			},
			want: VideoFormatMP4,
		},
		{
			name: "avci major brand",
			data: []byte{
				0x00, 0x00, 0x00, 0x18,
				'f', 't', 'y', 'p',
				'a', 'v', 'c', '1',
				0x00, 0x00, 0x00, 0x00,
				'a', 'v', 'c', '1',
			},
			want: VideoFormatMP4,
		},
		{
			name: "M4A major brand (audio)",
			data: []byte{
				0x00, 0x00, 0x00, 0x18,
				'f', 't', 'y', 'p',
				'M', '4', 'A', ' ',
				0x00, 0x00, 0x00, 0x00,
				'M', '4', 'A', ' ',
			},
			want: VideoFormatM4V, // Treated as M4V for compatibility
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), "test")
			if err := os.WriteFile(tmpFile, tt.data, 0644); err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}

			file, err := os.Open(tmpFile)
			if err != nil {
				t.Fatalf("failed to open temp file: %v", err)
			}
			defer file.Close()

			format, err := DetectVideoFormat(file)
			if err != nil {
				t.Fatalf("DetectVideoFormat returned error: %v", err)
			}

			if format != tt.want {
				t.Errorf("DetectVideoFormat() = %v, want %v", format, tt.want)
			}
		})
	}
}

func TestDetectVideoFormatPositionRestore(t *testing.T) {
	// Test that the file position is restored after detection
	mp4Data := []byte{
		0x00, 0x00, 0x00, 0x18,
		'f', 't', 'y', 'p',
		'm', 'p', '4', '1',
		0x00, 0x00, 0x00, 0x00,
		'm', 'p', '4', '1',
	}

	tmpFile := filepath.Join(t.TempDir(), "test_position.mp4")
	if err := os.WriteFile(tmpFile, mp4Data, 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	file, err := os.Open(tmpFile)
	if err != nil {
		t.Fatalf("failed to open temp file: %v", err)
	}
	defer file.Close()

	// Read some data first
	buf := make([]byte, 4)
	file.Read(buf)

	// Detect format
	format, err := DetectVideoFormat(file)
	if err != nil {
		t.Fatalf("DetectVideoFormat returned error: %v", err)
	}

	if format != VideoFormatMP4 {
		t.Errorf("expected VideoFormatMP4, got %v", format)
	}

	// Verify we can still read from the file
	pos, _ := file.Seek(0, io.SeekCurrent)
	if pos != 4 {
		t.Errorf("expected file position to be 4 after detection, got %d", pos)
	}
}
