package processor

import (
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStandardImageEngine_Resize(t *testing.T) {
	ctx := context.Background()
	tmpDir, err := os.MkdirTemp("", "engine_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "input.jpg")
	outputPath := filepath.Join(tmpDir, "output.jpg")

	// 1. Create a dummy input image (200x100)
	inputImg := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			inputImg.Set(x, y, color.RGBA{uint8(x % 255), uint8(y % 255), 0, 255})
		}
	}

	f, err := os.Create(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	err = jpeg.Encode(f, inputImg, nil)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	// 2. Test Resize to 100 width (should result in 100x50)
	engine := NewStandardImageEngine(80)
	err = engine.Resize(ctx, inputPath, outputPath, 100)
	assert.NoError(t, err)

	// 3. Verify output file exists
	_, err = os.Stat(outputPath)
	assert.NoError(t, err)

	// 4. Verify output dimensions
	outFile, err := os.Open(outputPath)
	assert.NoError(t, err)
	defer outFile.Close()

	decodedImg, _, err := image.Decode(outFile)
	assert.NoError(t, err)
	assert.Equal(t, 100, decodedImg.Bounds().Dx())
	assert.Equal(t, 50, decodedImg.Bounds().Dy())
}

func TestStandardImageEngine_Resize_FileNotFound(t *testing.T) {
	ctx := context.Background()
	engine := NewStandardImageEngine(80)
	err := engine.Resize(ctx, "non_existent.jpg", "out.jpg", 100)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open input file")
}

func TestStandardImageEngine_Resize_InvalidImage(t *testing.T) {
	ctx := context.Background()
	tmpDir, err := os.MkdirTemp("", "engine_invalid_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	invalidPath := filepath.Join(tmpDir, "not_an_image.txt")
	outputPath := filepath.Join(tmpDir, "out.jpg")

	err = os.WriteFile(invalidPath, []byte("this is not an image"), 0644)
	assert.NoError(t, err)

	engine := NewStandardImageEngine(80)
	err = engine.Resize(ctx, invalidPath, outputPath, 100)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode image")
}
