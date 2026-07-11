package processor

import (
	"fmt"
	"io"
	"github.com/jbrodriguez/mlog"
	"os"
	"strconv"
	"strings"
	"time"

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
	Orientation    Orientation
	Tags           []ExifTag
	GPSLatitude    float64
	GPSLongitude   float64
	GPSAltitude    float64
	CapturedAt     time.Time // DateTimeOriginal from EXIF (zero value if not found)
}

// ExifTag represents a single EXIF/IPTC/XMP tag.
type ExifTag struct {
	Source    string `json:"source"`
	Tag       string `json:"tag"`
	Namespace string `json:"namespace,omitempty"`
	Value     string `json:"value"`
}

// GPSMetadata represents GPS coordinates as decimal degrees.
type GPSMetadata struct {
	Latitude  float64
	Longitude float64
	Altitude  float64
	HasGPS    bool
}

// ReadGPS reads only the GPS coordinates from the given file.
// Returns a GPSMetadata struct with HasGPS=true if coordinates were found.
func (r *ExifReader) ReadGPS(file *os.File) (*GPSMetadata, error) {
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek file: %w", err)
	}

	format, err := detectImageFormat(file)
	if err != nil {
		return &GPSMetadata{}, nil
	}

	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek file: %w", err)
	}

	var (
		latRef     string
		lonRef     string
		latRaw     string
		lonRaw     string
		altRaw     string
	)

	opts := imagemeta.Options{
		R:           file,
		ImageFormat: convertImageFormat(format),
		Sources:     imagemeta.EXIF | imagemeta.XMP,
		HandleTag: func(info imagemeta.TagInfo) error {
			if strings.EqualFold(info.Tag, "GPSLatitudeRef") {
				latRef = fmt.Sprintf("%v", info.Value)
			} else if strings.EqualFold(info.Tag, "GPSLatitude") {
				latRaw = fmt.Sprintf("%v", info.Value)
			} else if strings.EqualFold(info.Tag, "GPSLongitudeRef") {
				lonRef = fmt.Sprintf("%v", info.Value)
			} else if strings.EqualFold(info.Tag, "GPSLongitude") {
				lonRaw = fmt.Sprintf("%v", info.Value)
			} else if strings.EqualFold(info.Tag, "GPSAltitude") {
				altRaw = fmt.Sprintf("%v", info.Value)
			}
			return nil
		},
	}

	_, decodeErr := imagemeta.Decode(opts)
	if decodeErr != nil {
		mlog.Info("[EXIF] Decode returned error: %v", decodeErr)
	}

	gps := &GPSMetadata{}

	// Only report GPS if we have both lat and lon
	if latRaw != "" && lonRaw != "" {
		latitude, err := parseGPSCoordinate(latRaw, latRef)
		if err == nil {
			gps.Latitude = latitude
			longitude, err := parseGPSCoordinate(lonRaw, lonRef)
			if err == nil {
				gps.Longitude = longitude
				gps.HasGPS = true
			}
		}
	}

	if altRaw != "" && gps.HasGPS {
		gps.Altitude = parseAltitude(altRaw)
	}

	return gps, nil
}

// GPSMetadataToMap converts GPS data into a map suitable for storing in Metadata JSONB.
func GPSMetadataToMap(gps *GPSMetadata) map[string]string {
	if gps == nil || !gps.HasGPS {
		return nil
	}
	result := make(map[string]string)
	result["gps_latitude"] = fmt.Sprintf("%.6f", gps.Latitude)
	result["gps_longitude"] = fmt.Sprintf("%.6f", gps.Longitude)
	result["gps_altitude"] = fmt.Sprintf("%.1f", gps.Altitude)
	return result
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
		mlog.Info("[EXIF] Failed to decode metadata: %v", err)
		return OrientationNormal, nil
	}

	if !capture.found {
		return OrientationNormal, nil
	}

	if capture.value < 1 || capture.value > 8 {
		mlog.Info("[EXIF] Invalid orientation value: %d", capture.value)
		return OrientationNormal, nil
	}

	return Orientation(capture.value), nil
}

// ReadDateTimeOriginal extracts the DateTimeOriginal EXIF tag from the file.
// EXIF stores dates as "2023:05:14 10:30:00" (with colons, not slashes).
// Returns (time.Time, true) if found and parseable, or (zero_time, false) if not available.
// Also checks DateTimeDigitized and DateTime as fallbacks.
func (r *ExifReader) ReadDateTimeOriginal(file *os.File) (time.Time, error) {
	if _, err := file.Seek(0, 0); err != nil {
		return time.Time{}, fmt.Errorf("failed to seek file: %w", err)
	}

	format, err := detectImageFormat(file)
	if err != nil {
		return time.Time{}, err
	}

	if _, err := file.Seek(0, 0); err != nil {
		return time.Time{}, fmt.Errorf("failed to seek file: %w", err)
	}

	var capturedAtStr string

	opts := imagemeta.Options{
		R:           file,
		ImageFormat: convertImageFormat(format),
		Sources:     imagemeta.EXIF | imagemeta.XMP | imagemeta.IPTC,
		HandleTag: func(info imagemeta.TagInfo) error {
			// Priority order: DateTimeOriginal > DateTimeDigitized > DateTime
			if capturedAtStr == "" {
				if strings.EqualFold(info.Tag, "DateTimeOriginal") {
					capturedAtStr = fmt.Sprintf("%v", info.Value)
				}
			}
			if capturedAtStr == "" && strings.EqualFold(info.Tag, "DateTimeDigitized") {
				capturedAtStr = fmt.Sprintf("%v", info.Value)
			}
			if capturedAtStr == "" && strings.EqualFold(info.Tag, "DateTime") {
				capturedAtStr = fmt.Sprintf("%v", info.Value)
			}
			return nil
		},
	}

	_, decodeErr := imagemeta.Decode(opts)
	if decodeErr != nil {
		mlog.Info("[EXIF] Failed to decode for DateTimeOriginal: %v", decodeErr)
	}

	if capturedAtStr == "" {
		return time.Time{}, nil
	}

	return parseDateTimeOriginal(capturedAtStr)
}

// parseDateTimeOriginal parses EXIF date strings.
// EXIF DateTimeOriginal format: "2023:05:14 10:30:00" (colons in date)
// Fallback formats: "2023/05/14 10:30:00", "2023-05-14 10:30:00", "2023:05:14", etc.
func parseDateTimeOriginal(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return time.Time{}, fmt.Errorf("empty date string")
	}

	// EXIF standard format uses colons: "2023:05:14 10:30:00"
	// Try the EXIF format first
	exifFormats := []string{
		"2006:01:02 15:04:05",
		"2006/01/02 15:04:05",
		"2006-01-02 15:04:05",
		"2006:01:02",
		"2006/01/02",
		"2006-01-02",
	}

	for _, layout := range exifFormats {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse EXIF date: %s", dateStr)
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
		// GPS raw values
		latRef     string
		lonRef     string
		latRaw     string
		lonRaw     string
		altRaw     string
		// DateTimeOriginal capture
		capturedAtStr string
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

			// Capture GPS raw values for later processing
			if strings.EqualFold(info.Tag, "GPSLatitudeRef") {
				latRef = fmt.Sprintf("%v", info.Value)
			} else if strings.EqualFold(info.Tag, "GPSLatitude") {
				latRaw = fmt.Sprintf("%v", info.Value)
			} else if strings.EqualFold(info.Tag, "GPSLongitudeRef") {
				lonRef = fmt.Sprintf("%v", info.Value)
			} else if strings.EqualFold(info.Tag, "GPSLongitude") {
				lonRaw = fmt.Sprintf("%v", info.Value)
			} else if strings.EqualFold(info.Tag, "GPSAltitude") {
				altRaw = fmt.Sprintf("%v", info.Value)
			}

			// Capture DateTimeOriginal for CapturedAt (only if not already captured)
			if capturedAtStr == "" {
				if strings.EqualFold(info.Tag, "DateTimeOriginal") ||
					strings.EqualFold(info.Tag, "DateTimeDigitized") ||
					strings.EqualFold(info.Tag, "DateTime") {
					capturedAtStr = fmt.Sprintf("%v", info.Value)
				}
			}

			return nil
		},
	}

	_, decodeErr := imagemeta.Decode(opts)
	if decodeErr != nil {
		mlog.Info("[EXIF] Decode returned error: %v", decodeErr)
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

	// Process GPS coordinates if found
	if latRaw != "" && lonRaw != "" {
		latitude, err := parseGPSCoordinate(latRaw, latRef)
		if err == nil {
			info.GPSLatitude = latitude
		}
	}
	if lonRaw != "" {
		longitude, err := parseGPSCoordinate(lonRaw, lonRef)
		if err == nil {
			info.GPSLongitude = longitude
		}
	}
	if altRaw != "" {
		info.GPSAltitude = parseAltitude(altRaw)
	}


	// Set CapturedAt from EXIF DateTimeOriginal if found
	if capturedAtStr != "" {
		if capturedAt, err := parseDateTimeOriginal(capturedAtStr); err == nil {
			info.CapturedAt = capturedAt
		}
	}

	return info, nil
}

// parseGPSCoordinate converts GPS coordinate string to decimal degrees.
// Accepts both EXIF "degrees minutes/1 seconds/1" format (e.g., "40/1 25/1 47/1")
// AND decimal degrees format (e.g., "27.578691666666668").
func parseGPSCoordinate(coordStr, ref string) (float64, error) {
	coordStr = strings.TrimSpace(coordStr)
	if coordStr == "" {
		return 0, fmt.Errorf("empty coordinate string")
	}

	// Check if the coordinate is already in decimal degrees format.
	// Decimal degrees are a single number like "27.578691666666668",
	// whereas EXIF format uses spaces between degrees/minutes/seconds.
	if len(strings.Fields(coordStr)) == 1 {
		// Single value — treat as decimal degrees
		decimal, err := strconv.ParseFloat(coordStr, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse decimal degrees: %v", err)
		}
		// Apply reference direction
		if ref == "S" || ref == "W" {
			decimal = -decimal
		}
		return decimal, nil
	}

	// Multiple parts — EXIF DMS format: "degrees minutes/1 seconds/1"
	parts := strings.Fields(coordStr)

	// Try to parse degrees, minutes, seconds
	var degrees, minutes, seconds float64
	var err error

	if len(parts) >= 3 {
		// Format with fractions: "40/1 25/1 47/1"
		if _, err = fmt.Sscanf(parts[0], "%f/%f", &degrees, &minutes); err != nil {
			if _, err = fmt.Sscanf(parts[0], "%f", &degrees); err != nil {
				return 0, fmt.Errorf("failed to parse degrees: %v", err)
			}
			minutes = 0
		}
		if _, err = fmt.Sscanf(parts[1], "%f/%f", &minutes, &seconds); err != nil {
			if _, err = fmt.Sscanf(parts[1], "%f", &minutes); err != nil {
				return 0, fmt.Errorf("failed to parse minutes: %v", err)
			}
			seconds = 0
		}
		if len(parts) >= 3 {
			if _, err = fmt.Sscanf(parts[2], "%f/%f", &seconds, &seconds); err != nil {
				if _, err = fmt.Sscanf(parts[2], "%f", &seconds); err != nil {
					return 0, fmt.Errorf("failed to parse seconds: %v", err)
				}
			}
		}
	} else {
		// Simple format: "40 25 47"
		if _, err = fmt.Sscanf(parts[0], "%f", &degrees); err != nil {
			return 0, fmt.Errorf("failed to parse degrees: %v", err)
		}
		if _, err = fmt.Sscanf(parts[1], "%f", &minutes); err != nil {
			return 0, fmt.Errorf("failed to parse minutes: %v", err)
		}
		seconds = 0
		if len(parts) >= 3 {
			if _, err = fmt.Sscanf(parts[2], "%f", &seconds); err != nil {
				return 0, fmt.Errorf("failed to parse seconds: %v", err)
			}
		}
	}

	// Convert to decimal degrees
	decimal := degrees + minutes/60.0 + seconds/3600.0

	// Apply reference direction
	if ref == "S" || ref == "W" {
		decimal = -decimal
	}

	return decimal, nil
}

// parseAltitude converts altitude string to float64 (meters)
func parseAltitude(altStr string) float64 {
	altStr = strings.TrimSpace(altStr)
	if altStr == "" {
		return 0
	}

	var altitude float64
	if _, err := fmt.Sscanf(altStr, "%f", &altitude); err != nil {
		return 0
	}
	return altitude
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
		mlog.Info("[EXIF] Orientation value type %T is not supported", v)
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
