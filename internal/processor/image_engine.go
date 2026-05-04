package processor

import (
	"context"
)

// ImageEngine defines the interface for low-level image manipulation.
// This allows us to swap between the standard library and libvips.
type ImageEngine interface {
	// Resize returns a resized version of the image at the given width,
	// maintaining the aspect ratio.
	Resize(ctx context.Context, inputPath string, outputPath string, width int) error
}
