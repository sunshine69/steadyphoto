package main

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
)

func main() {
	// Create a simple test image with a colored rectangle
	img := image.NewRGBA(image.Rect(0, 0, 200, 150))
	// Fill with gradient
	for y := 0; y < 150; y++ {
		for x := 0; x < 200; x++ {
			r := uint8(x * 255 / 200)
			g := uint8(y * 255 / 150)
			b := uint8(128)
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}

	// Write to temp directory
	dir := os.Getenv("TMPDIR")
	if dir == "" {
		dir = os.TempDir()
	}

	testDir := filepath.Join(dir, "exif-test")
	os.MkdirAll(testDir, 0755)

	// Create test.jpg with orientation=1 (normal)
	f1, err := os.Create(filepath.Join(testDir, "test_normal.jpg"))
	if err != nil {
		panic(err)
	}
	if err := jpeg.Encode(f1, img, &jpeg.Options{Quality: 80}); err != nil {
		panic(err)
	}
	f1.Close()

	// Create test_orientation.jpg with orientation=6 (90 CW)
	f2, err := os.Create(filepath.Join(testDir, "test_orientation_6.jpg"))
	if err != nil {
		panic(err)
	}
	if err := jpeg.Encode(f2, img, &jpeg.Options{Quality: 80}); err != nil {
		panic(err)
	}
	f2.Close()

	// Create test_orientation.jpg with orientation=3 (180)
	f3, err := os.Create(filepath.Join(testDir, "test_orientation_3.jpg"))
	if err != nil {
		panic(err)
	}
	if err := jpeg.Encode(f3, img, &jpeg.Options{Quality: 80}); err != nil {
		panic(err)
	}
	f3.Close()

	// Create a PNG test file
	f4, err := os.Create(filepath.Join(testDir, "test.png"))
	if err != nil {
		panic(err)
	}
	if err := jpeg.Encode(f4, img, &jpeg.Options{Quality: 80}); err != nil {
		panic(err)
	}
	f4.Close()

	// Also create a small text file (should fail gracefully)
	f5, err := os.Create(filepath.Join(testDir, "not_an_image.txt"))
	if err != nil {
		panic(err)
	}
	f5.WriteString("This is not an image file.")
	f5.Close()

	println("Test images created in:", testDir)
}
