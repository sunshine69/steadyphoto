package processor

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/bep/imagemeta"
)

// Orientation represents EXIF orientation values (1-8).
type Orientation int

const (
	OrientationNormal           Orientation = 1
	OrientationFlipHorizontal   Orientation = 2
	OrientationRotate180        Orientation = 3
	OrientationFlipVertical     Orientation = 4
	OrientationRotate90CWFlipH  Orientation = 5
	OrientationRotate90CW       Orientation = 6
	OrientationRotate90CCWFlipH Orientation = 7
	OrientationRotate90CCW      Orientation = 8
)

// ExifInfo contains extracted EXIF information.
type ExifInfo struct {
	Orientation Orientation
	Tags        []ExifTag
}

// ExifTag represents a single EXIF/IPTC/XMP tag.
type ExifTag struct {
	Source    string `json:"source"`
	Tag       string `json:"tag"`
	Namespace string `json:"namespace,omitempty"`
	Value     string `json:"value"`
}

// ExifReader reads EXIF metadata using the bep/imagemeta library.
type ExifReader struct{}

// NewExifReader creates a new ExifReader.
func NewExifReader() *ExifReader {
	return &ExifReader{}
}

// ReadOrientation reads the EXIF Orientation tag from the given file.
// Returns the orientation value (1-8), or 1 (normal) if no orientation tag is found.
func (r *ExifReader) ReadOrientation(file *os.File) (Orientation, error) {
	if _, err := file.Seek(0, 0); err != nil {
		return OrientationNormal, fmt.Errorf("failed to seek file: %w", err)
	}

	format, err := detectImageFormat(file)
	if err != nil {
		return OrientationNormal, err
	}

	if _, err := file.Seek(0, 0); err != nil {
		return OrientationNormal, fmt.Errorf("failed to seek file: %w", err)
	}

	// Use a small struct to capture the orientation from the callback
	type orientationCapture struct {
		value int
		found bool
	}
	capture := orientationCapture{}

	opts := imagemeta.Options{
		R:           file,
		ImageFormat: convertImageFormat(format),
		Sources:     imagemeta.EXIF,
		HandleTag: func(info imagemeta.TagInfo) error {
			if strings.EqualFold(info.Tag, "Orientation") {
				capture.value = extractIntValue(info.Value)
				capture.found = true
			}
			return nil
		},
	}

	if _, err := imagemeta.Decode(opts); err != nil {
		// If imagemeta fails to parse (e.g. no EXIF), treat as normal orientation
		log.Printf("[EXIF] Failed to decode metadata: %v", err)
		return OrientationNormal, nil
	}

	if !capture.found {
		return OrientationNormal, nil
	}

	if capture.value < 1 || capture.value > 8 {
		log.Printf("[EXIF] Invalid orientation value: %d", capture.value)
		return OrientationNormal, nil
	}

	return Orientation(capture.value), nil
}

// ReadExif reads full EXIF info from the given file.
func (r *ExifReader) ReadExif(file *os.File) (*ExifInfo, error) {
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek file: %w", err)
	}

	format, err := detectImageFormat(file)
	if err != nil {
		return nil, err
	}

	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek file: %w", err)
	}

	var (
		orientationVal int
		orientationOK  bool
		tags           []ExifTag
	)

	opts := imagemeta.Options{
		R:           file,
		ImageFormat: convertImageFormat(format),
		Sources:     imagemeta.EXIF | imagemeta.XMP | imagemeta.IPTC,
		HandleTag: func(info imagemeta.TagInfo) error {
			tags = append(tags, ExifTag{
				Source:    info.Source.String(),
				Tag:       info.Tag,
				Namespace: info.Namespace,
				Value:     fmt.Sprintf("%v", info.Value),
			})

			if strings.EqualFold(info.Tag, "Orientation") {
				orientationVal = extractIntValue(info.Value)
				orientationOK = true
			}
			return nil
		},
	}

	_, decodeErr := imagemeta.Decode(opts)
	if decodeErr != nil {
		log.Printf("[EXIF] Decode returned error: %v", decodeErr)
		// If we already collected some tags, still return them (partial data is better than nothing)
		if len(tags) == 0 {
			return nil, nil
		}
	}

	info := &ExifInfo{
		Tags: tags,
	}
	if orientationOK {
		info.Orientation = Orientation(orientationVal)
	}
	return info, nil
}

// extractIntValue converts an unknown numeric type to an int.
// imagemeta stores numeric tag values as various Go types depending on the encoding.
func extractIntValue(v any) int {
	switch val := v.(type) {
	case int:
		return val
	case int8:
		return int(val)
	case int16:
		return int(val)
	case int32:
		return int(val)
	case int64:
		return int(val)
	case uint:
		return int(val)
	case uint8:
		return int(val)
	case uint16:
		return int(val)
	case uint32:
		return int(val)
	case uint64:
		return int(val)
	case float32:
		return int(val)
	case float64:
		return int(val)
	default:
		log.Printf("[EXIF] Orientation value type %T is not supported", v)
		return 0
	}
}

// ImageFormat represents supported image formats.
type ImageFormat int

const (
	FormatJPEG ImageFormat = iota
	FormatPNG
	FormatGIF
	FormatWebP
	FormatTIFF
	FormatHEIF
	FormatAVIF
	FormatDNG
	FormatCR2
	FormatNEF
	FormatARW
	FormatPEF
)

func convertImageFormat(f ImageFormat) imagemeta.ImageFormat {
	switch f {
	case FormatJPEG:
		return imagemeta.JPEG
	case FormatPNG:
		return imagemeta.PNG
	case FormatGIF:
		return imagemeta.ImageFormatAuto
	case FormatWebP:
		return imagemeta.WebP
	case FormatTIFF:
		return imagemeta.TIFF
	case FormatHEIF:
		return imagemeta.HEIF
	case FormatAVIF:
		return imagemeta.AVIF
	case FormatDNG:
		return imagemeta.DNG
	case FormatCR2:
		return imagemeta.CR2
	case FormatNEF:
		return imagemeta.NEF
	case FormatARW:
		return imagemeta.ARW
	case FormatPEF:
		return imagemeta.PEF
	default:
		return imagemeta.ImageFormatAuto
	}
}

// detectImageFormat detects the image format from the file header using magic bytes.
func detectImageFormat(file *os.File) (ImageFormat, error) {
	// Save current position
	pos, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return FormatJPEG, fmt.Errorf("failed to get file position: %w", err)
	}

	// Read first 12 bytes for magic byte detection
	buf := make([]byte, 12)
	n, readErr := file.Read(buf)
	if readErr != nil && readErr != io.EOF {
		return FormatJPEG, fmt.Errorf("failed to read file header: %w", readErr)
	}

	// Restore position
	if _, seekErr := file.Seek(pos, io.SeekStart); seekErr != nil {
		return FormatJPEG, fmt.Errorf("failed to restore file position: %w", seekErr)
	}

	if n < 4 {
		return FormatJPEG, nil
	}

	// JPEG: FF D8 FF
	if buf[0] == 0xFF && buf[1] == 0xD8 && buf[2] == 0xFF {
		return FormatJPEG, nil
	}

	// PNG: 89 50 4E 47
	if buf[0] == 0x89 && buf[1] == 0x50 && buf[2] == 0x4E && buf[3] == 0x47 {
		return FormatPNG, nil
	}

	// GIF: GIF87a or GIF89a
	if n >= 6 && buf[0] == 'G' && buf[1] == 'I' && buf[2] == 'F' {
		if (buf[3] == '8' && buf[4] == '7' && buf[5] == 'a') ||
			(buf[3] == '8' && buf[4] == '9' && buf[5] == 'a') {
			return FormatGIF, nil
		}
	}

	// WebP: RIFF....WEBP
	if n >= 12 && buf[0] == 'R' && buf[1] == 'I' && buf[2] == 'F' && buf[3] == 'F' &&
		buf[8] == 'W' && buf[9] == 'E' && buf[10] == 'B' && buf[11] == 'P' {
		return FormatWebP, nil
	}

	// TIFF: II (little-endian) or MM (big-endian) followed by 42
	if (buf[0] == 'I' && buf[1] == 'I' && buf[2] == 0x2A && buf[3] == 0x00) ||
		(buf[0] == 'M' && buf[1] == 'M' && buf[2] == 0x00 && buf[3] == 0x2A) {
		return FormatTIFF, nil
	}

	return FormatJPEG, nil
}
