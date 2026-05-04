package processor

import (
	"context"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"os"
	"path/filepath"
)

// MockImageEngine implements ImageEngine for testing purposes.
type MockImageEngine struct {
	LastInputPath  string
	LastOutputPath string
	LastWidth      int
}

func (m *MockImageEngine) Resize(ctx context.Context, inputPath string, outputPath string, width int) error {
	m.LastInputPath = inputPath
	m.LastOutputPath = outputPath
	m.LastWidth = width

	// Create a dummy image file at outputPath
	err := os.MkdirAll(filepath.Dir(outputPath), 0755)
	if err != nil {
		return err
	}

	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	// Fill with a color
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{255, 0, 0, 255}}, image.Point{}, draw.Src)

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return jpeg.Encode(f, img, nil)
}
