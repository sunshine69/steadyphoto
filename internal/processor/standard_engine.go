package processor

import (
	"context"
	"fmt"
	"image"
	"image/jpeg"
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

// Resize resizes the image at inputPath and saves it to outputPath.
func (e *StandardImageEngine) Resize(ctx context.Context, inputPath string, outputPath string, width int) error {
	// 1. Open the file
	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer file.Close()

	// 2. Decode the image
	img, format, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode image (%s): %w", format, err)
	}

	// 3. Calculate new dimensions
	bounds := img.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()

	if origWidth <= width {
		// If image is smaller than target width, just copy it (or we could upscale, but let's stay small)
		width = origWidth
	}

	ratio := float64(width) / float64(origWidth)
	height := int(float64(origHeight) * ratio)

	// 4. Create destination image
	dst := image.NewRGBA(image.Rect(0, 0, width, height))

	// 5. Perform scaling
	// We use BiLinear interpolation for a good balance of speed and quality
	draw.BiLinear.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	// 6. Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// 7. Encode the image (using JPEG for simplicity in this prototype)
	// Note: In a real production environment, we'd want to support WebP
	err = jpeg.Encode(outFile, dst, &jpeg.Options{Quality: e.Quality})
	if err != nil {
		return fmt.Errorf("failed to encode jpeg: %w", err)
	}

	return nil
}
