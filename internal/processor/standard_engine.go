package processor

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"github.com/jbrodriguez/mlog"
	"os"

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
	// Use the new ExifReader instead of goexif
	reader := NewExifReader()
	orientation, err := reader.ReadOrientation(file)
	if err != nil {
		return img, fmt.Errorf("failed to read EXIF orientation: %w", err)
	}

	return applyOrientationFromValue(img, orientation)
}

// applyOrientationFromValue applies rotation/flip based on the given EXIF orientation value.
// This is used when the orientation is already known (e.g., read before image.Decode).
func applyOrientationFromValue(img image.Image, orientation Orientation) (image.Image, error) {
	origBounds := img.Bounds()
	origWidth := origBounds.Dx()
	origHeight := origBounds.Dy()

	switch orientation {
	case OrientationFlipHorizontal:
		return flipHorizontal(img, origWidth, origHeight), nil
	case OrientationRotate180:
		return rotate180(img, origWidth, origHeight), nil
	case OrientationFlipVertical:
		return flipVertical(img, origWidth, origHeight), nil
	case OrientationRotate90CWFlipH:
		return rotate90CWThenFlipH(img, origWidth, origHeight), nil
	case OrientationRotate90CW:
		return rotate90CW(img, origWidth, origHeight), nil
	case OrientationRotate90CCWFlipH:
		return rotate90CCWThenFlipH(img, origWidth, origHeight), nil
	case OrientationRotate90CCW:
		return rotate90CCW(img, origWidth, origHeight), nil
	default:
		return img, nil
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
// Pixel (x, y) → (h-1-y, x)
func rotate90CW(img image.Image, w, h int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, h, w))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			p := img.At(x, y)
			dst.Set(h-1-y, x, p)
		}
	}
	return dst
}

// rotate90CCW rotates the image 90° counter-clockwise
// New dimensions: width becomes height, height becomes width
// Pixel (x, y) → (y, w-1-x)
func rotate90CCW(img image.Image, w, h int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, h, w))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			p := img.At(x, y)
			dst.Set(y, w-1-x, p)
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
	mlog.Info("  [ENGINE] Opening input file: %s", inputPath)
	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file %s: %w", inputPath, err)
	}
	defer file.Close()

	// 2. Read EXIF orientation BEFORE decoding (decode consumes EXIF data)
	orientation, err := NewExifReader().ReadOrientation(file)
	if err != nil {
		mlog.Info("  [ENGINE] Warning: could not read EXIF orientation: %v", err)
		orientation = OrientationNormal
	}
	mlog.Info("  [ENGINE] EXIF orientation: %d", orientation)

	// 3. Decode the image
	_, err = file.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("failed to seek file: %w", err)
	}

	img, format, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode image %s (%s): %w", inputPath, format, err)
	}
	mlog.Info("  [ENGINE] Decoded image: format=%s", format)

	// 4. Apply rotation/flip based on EXIF orientation (already read in step 2)
	if orientation != OrientationNormal {
		img, err = applyOrientationFromValue(img, orientation)
		if err != nil {
			mlog.Info("  [ENGINE] Warning: could not apply orientation: %v", err)
		}
	}

	// 5. Calculate new dimensions based on the (possibly rotated) image
	bounds := img.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()
	mlog.Info("  [ENGINE] Original dimensions: %dx%d, target width: %d", origWidth, origHeight, width)

	if origWidth <= width {
		width = origWidth
		mlog.Info("  [ENGINE] Image smaller than target, using original width: %d", width)
	}

	ratio := float64(width) / float64(origWidth)
	height := int(float64(origHeight) * ratio)
	mlog.Info("  [ENGINE] New dimensions: %dx%d (ratio: %.2f)", width, height, ratio)

	// 6. Create destination image
	dst := image.NewRGBA(image.Rect(0, 0, width, height))

	// 7. Perform scaling
	draw.BiLinear.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	// 8. Create output file
	mlog.Info("  [ENGINE] Creating output file: %s", outputPath)
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputPath, err)
	}
	defer outFile.Close()

	// 9. Encode the image
	mlog.Info("  [ENGINE] Encoding JPEG with quality: %d", e.Quality)
	err = jpeg.Encode(outFile, dst, &jpeg.Options{Quality: e.Quality})
	if err != nil {
		return fmt.Errorf("failed to encode jpeg for %s: %w", outputPath, err)
	}

	// 10. Validate the output file was created
	info, err := outFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat output file %s: %w", outputPath, err)
	}
	mlog.Info("  [ENGINE] Output file created: %s (%d bytes)", outputPath, info.Size())

	return nil
}
