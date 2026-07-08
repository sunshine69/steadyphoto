package processor

import (
	"fmt"
	"io"
	"os"
)

// VideoFormat represents a detected video container type
type VideoFormat int

const (
	VideoFormatUnknown VideoFormat = iota
	VideoFormatMP4      // Android default (.mp4)
	VideoFormatMOV      // iPhone default (.mov)
	VideoFormatM4V      // Apple iTunes (.m4v)
	VideoFormatAVI      // Legacy (.avi)
	VideoFormatWebM     // Rare on phones (.webm)
)

// String returns the human-readable name of the video format
func (f VideoFormat) String() string {
	switch f {
	case VideoFormatMP4:
		return "MP4"
	case VideoFormatMOV:
		return "MOV"
	case VideoFormatM4V:
		return "M4V"
	case VideoFormatAVI:
		return "AVI"
	case VideoFormatWebM:
		return "WebM"
	default:
		return "Unknown"
	}
}

// IsSupported returns true if the video format is supported by the video metadata extractor
func (f VideoFormat) IsSupported() bool {
	return f != VideoFormatUnknown
}

// DetectVideoFormat detects the video container type from the file's magic bytes.
// It reads the first 16 bytes to identify the container without loading the entire file.
// Returns the detected format or VideoFormatUnknown if unable to identify.
func DetectVideoFormat(file *os.File) (VideoFormat, error) {
	// Save current position
	pos, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return VideoFormatUnknown, fmt.Errorf("failed to get file position: %w", err)
	}

	// Read first 16 bytes for magic byte detection
	buf := make([]byte, 16)
	n, readErr := file.Read(buf)
	if readErr != nil && readErr != io.EOF {
		return VideoFormatUnknown, fmt.Errorf("failed to read file header: %w", readErr)
	}

	// Restore position
	if _, seekErr := file.Seek(pos, io.SeekStart); seekErr != nil {
		return VideoFormatUnknown, fmt.Errorf("failed to restore file position: %w", seekErr)
	}

	if n < 4 {
		return VideoFormatUnknown, fmt.Errorf("file too small for format detection (%d bytes)", n)
	}

	// Try to detect each format in order of likelihood for phone videos

	// MP4: 'ftyp' box at offset 4
	if n >= 8 && buf[4] == 'f' && buf[5] == 't' && buf[6] == 'y' && buf[7] == 'p' {
		// Check major brand for more specific detection
		if n >= 12 {
			majorBrand := string(buf[8:12])
			switch majorBrand {
			case "qt  ":
				// QuickTime MOV with ftyp box
				return VideoFormatMOV, nil
			case "M4V ":
				return VideoFormatM4V, nil
			case "M4P ":
				return VideoFormatM4V, nil // iTunes Plus
			case "M4A ":
				return VideoFormatM4V, nil // Audio only, but could be video
			}
		}
		return VideoFormatMP4, nil
	}

	// MOV: 'moov' atom at offset 8
	if n >= 12 && buf[8] == 'm' && buf[9] == 'o' && buf[10] == 'o' && buf[11] == 'v' {
		return VideoFormatMOV, nil
	}

	// AVI: 'RIFF'... 'AVI ' at offset 8
	if n >= 12 && buf[0] == 'R' && buf[1] == 'I' && buf[2] == 'F' && buf[3] == 'F' {
		if buf[8] == 'A' && buf[9] == 'V' && buf[10] == 'I' && buf[11] == ' ' {
			return VideoFormatAVI, nil
		}
	}

	// WebM: EBML header (1A 45 DF A3) at offset 0
	if buf[0] == 0x1A && buf[1] == 0x45 && buf[2] == 0xDF && buf[3] == 0xA3 {
		return VideoFormatWebM, nil
	}

	return VideoFormatUnknown, nil
}

// DetectVideoFormatByExtension detects the expected video format from the file extension
// This is a fallback when magic byte detection fails
func DetectVideoFormatByExtension(ext string) VideoFormat {
	ext = fmt.Sprintf("%v", ext) // Ensure string conversion
	switch ext {
	case ".mp4":
		return VideoFormatMP4
	case ".mov":
		return VideoFormatMOV
	case ".m4v":
		return VideoFormatM4V
	case ".avi":
		return VideoFormatAVI
	case ".webm":
		return VideoFormatWebM
	default:
		return VideoFormatUnknown
	}
}
