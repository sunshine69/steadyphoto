package processor

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"log"
	"os"

	"github.com/rwcarlsen/goexif/exif"
	"golang.org/x/image/draw"
)

// StandardImageEngine implements ImageEngine using Go's standard library.
// It is slower than libvips but requires no C dependencies.
type StandardImageEngine struct {
	Quality int
}

// NewStandardImageEngine creates a new StandardImageEngine.
func NewStandardImageEngine(quality int) *StandardImageEngine {
	return &StandardImageEngine{
		Quality: quality,
	}
}

// applyExifOrientation reads the EXIF orientation tag and applies the corresponding
// rotation/flipping to the image so the thumbnail is displayed upright on all devices.
func applyExifOrientation(file *os.File, img image.Image) (image.Image, error) {
	// Seek to beginning to read EXIF data
	_, err := file.Seek(0, 0)
	if err != nil {
		return img, fmt.Errorf("failed to seek file: %w", err)
	}

	exifData, err := exif.Decode(file)
	if err != nil {
		// No EXIF data — return image as-is
		return img, nil
	}

	orientation, err := exifData.Get(exif.Orientation)
	if err != nil {
		// No orientation tag — return image as-is
		return img, nil
	}

	orientVal, err := orientation.Int(0)
	if err != nil {
		return img, fmt.Errorf("failed to parse orientation value: %w", err)
	}

	if orientVal == 1 {
		// Orientation 1 = normal, no rotation needed
		return img, nil
	}

	origBounds := img.Bounds()
	origWidth := origBounds.Dx()
	origHeight := origBounds.Dy()

	switch orientVal {
	case 2:
		// Flip horizontally
		return flipHorizontal(img, origWidth, origHeight), nil
	case 3:
		// Rotate 180°
		return rotate180(img, origWidth, origHeight), nil
	case 4:
		// Flip vertically
		return flipVertical(img, origWidth, origHeight), nil
	case 5:
		// Rotate 90° CW + flip horizontally
		return rotate90CWThenFlipH(img, origWidth, origHeight), nil
	case 6:
		// Rotate 90° CW — standard "portrait" after camera orientation
		return rotate90CW(img, origWidth, origHeight), nil
	case 7:
		// Rotate 90° CCW + flip horizontally
		return rotate90CCWThenFlipH(img, origWidth, origHeight), nil
	case 8:
		// Rotate 90° CCW — standard for phones
		return rotate90CCW(img, origWidth, origHeight), nil
	default:
		return img, fmt.Errorf("unknown orientation value: %d", orientVal)
	}
}

// flipHorizontal flips the image horizontally (mirror)
func flipHorizontal(img image.Image, w, h int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx := w - 1 - x
			p := img.At(dx, y)
			dst.Set(x, y, p)
		}
	}
	return dst
}

// flipVertical flips the image vertically
func flipVertical(img image.Image, w, h int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dy := h - 1 - y
			p := img.At(x, dy)
			dst.Set(x, y, p)
		}
	}
	return dst
}

// rotate90CW rotates the image 90° clockwise
// New dimensions: width becomes height, height becomes width
func rotate90CW(img image.Image, w, h int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, h, w))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			p := img.At(x, y)
			dst.Set(y, w-1-x, p)
		}
	}
	return dst
}

// rotate90CCW rotates the image 90° counter-clockwise
// New dimensions: width becomes height, height becomes width
func rotate90CCW(img image.Image, w, h int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, h, w))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			p := img.At(x, y)
			dst.Set(h-1-y, x, p)
		}
	}
	return dst
}

// rotate180 rotates the image 180°
func rotate180(img image.Image, w, h int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx := w - 1 - x
			dy := h - 1 - y
			p := img.At(dx, dy)
			dst.Set(x, y, p)
		}
	}
	return dst
}

// rotate90CWThenFlipH rotates 90° CW then flips horizontally
func rotate90CWThenFlipH(img image.Image, w, h int) image.Image {
	rotated := rotate90CW(img, w, h)
	return flipHorizontal(rotated, h, w)
}

// rotate90CCWThenFlipH rotates 90° CCW then flips horizontally
func rotate90CCWThenFlipH(img image.Image, w, h int) image.Image {
	rotated := rotate90CCW(img, w, h)
	return flipHorizontal(rotated, h, w)
}

// Resize resizes the image at inputPath and saves it to outputPath.
// It first reads EXIF orientation and applies the correct rotation so that
// the thumbnail is always displayed upright regardless of device orientation.
func (e *StandardImageEngine) Resize(ctx context.Context, inputPath string, outputPath string, width int) error {
	// 1. Open the file
	log.Printf("  [ENGINE] Opening input file: %s", inputPath)
	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file %s: %w", inputPath, err)
	}
	defer file.Close()

	// 2. Decode the image
	img, format, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode image %s (%s): %w", inputPath, format, err)
	}
	log.Printf("  [ENGINE] Decoded image: format=%s", format)

	// 3. Apply EXIF orientation (rotate/flip if needed) so the thumbnail
	//    appears upright on all devices, not just landscape mode.
	img, err = applyExifOrientation(file, img)
	if err != nil {
		log.Printf("  [ENGINE] Warning: could not apply EXIF orientation: %v", err)
		// Continue without rotation if EXIF fails
	}

	// 4. Calculate new dimensions based on the (possibly rotated) image
	bounds := img.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()
	log.Printf("  [ENGINE] Original dimensions: %dx%d, target width: %d", origWidth, origHeight, width)

	if origWidth <= width {
		width = origWidth
		log.Printf("  [ENGINE] Image smaller than target, using original width: %d", width)
	}

	ratio := float64(width) / float64(origWidth)
	height := int(float64(origHeight) * ratio)
	log.Printf("  [ENGINE] New dimensions: %dx%d (ratio: %.2f)", width, height, ratio)

	// 5. Create destination image
	dst := image.NewRGBA(image.Rect(0, 0, width, height))

	// 6. Perform scaling
	draw.BiLinear.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	// 7. Create output file
	log.Printf("  [ENGINE] Creating output file: %s", outputPath)
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputPath, err)
	}
	defer outFile.Close()

	// 8. Encode the image
	log.Printf("  [ENGINE] Encoding JPEG with quality: %d", e.Quality)
	err = jpeg.Encode(outFile, dst, &jpeg.Options{Quality: e.Quality})
	if err != nil {
		return fmt.Errorf("failed to encode jpeg for %s: %w", outputPath, err)
	}

	// 9. Validate the output file was created
	info, err := outFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat output file %s: %w", outputPath, err)
	}
	log.Printf("  [ENGINE] Output file created: %s (%d bytes)", outputPath, info.Size())

	return nil
}
