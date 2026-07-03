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

	// 3. Calculate new dimensions
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

	// 4. Create destination image
	dst := image.NewRGBA(image.Rect(0, 0, width, height))

	// 5. Perform scaling
	draw.BiLinear.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	// 6. Create output file
	log.Printf("  [ENGINE] Creating output file: %s", outputPath)
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputPath, err)
	}
	defer outFile.Close()

	// 7. Encode the image
	log.Printf("  [ENGINE] Encoding JPEG with quality: %d", e.Quality)
	err = jpeg.Encode(outFile, dst, &jpeg.Options{Quality: e.Quality})
	if err != nil {
		return fmt.Errorf("failed to encode jpeg for %s: %w", outputPath, err)
	}

	// 8. Validate the output file was created
	info, err := outFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat output file %s: %w", outputPath, err)
	}
	log.Printf("  [ENGINE] Output file created: %s (%d bytes)", outputPath, info.Size())

	return nil
}
