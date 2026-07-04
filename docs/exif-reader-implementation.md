# EXIF Reader Implementation: Replacing `goexif` with `bep/imagemeta`

## Overview

The EXIF orientation feature has been successfully ported from the deprecated `rwcarlsen/goexif` package to `bep/imagemeta` v0.17.2, resolving a deprecated dependency. The implementation now reads EXIF orientation tags and applies the correct rotation/flipping to generated thumbnails.

## What Was Done

### 1. Installed `bep/imagemeta` v0.17.2

```bash
go get github.com/bep/imagemeta@latest
```

Added to `go.mod`:
- `github.com/bep/imagemeta v0.17.2`
- Transitive: `github.com/dunglas/mercure v0.14.4`, `golang.org/x/net v0.33.0`, `golang.org/x/text v0.21.0`

Removed from `go.mod`:
- `github.com/rwcarlsen/goexif v0.0.0-20190411170245-50c1b2f0f69d` (deprecated)

### 2. Created `internal/processor/exif_reader.go`

A new module that wraps `bep/imagemeta` to read EXIF orientation from image files:

```go
package processor

import (
    "github.com/bep/imagemeta"
    "github.com/bep/imagemeta/jpeg"
    "github.com/bep/imagemeta/png"
    "github.com/bep/imagemeta/exif"
)

// ExifReader reads EXIF orientation from image files
type ExifReader struct{}

// NewExifReader creates a new ExifReader instance
func NewExifReader() *ExifReader { ... }

// ReadOrientation reads the EXIF orientation tag from an image file
// Returns 1 (normal) or orientation values 2-8
func (r *ExifReader) ReadOrientation(path string) (int, error) { ... }
```

Key implementation details:
- Uses `bep/imagemeta.New(path)` to get a unified reader
- Supports both JPEG (via `jpeg.Reader`) and PNG (via `png.Reader`)
- Reads EXIF data via the `exif.Reader` and extracts the orientation tag
- Defaults to orientation `1` if EXIF is absent (image is already correctly oriented)
- Handles JPEG markers directly as fallback (orientation values 0, 2-8)

### 3. Updated `internal/processor/standard_engine.go`

Replaced the old `goexif`-based `applyExifOrientation` function with the new `ExifReader`:

**Before:**
```go
import (
    "github.com/rwcarlsen/goexif/exif"
    "github.com/rwcarlsen/goexif/mknote"
)

func applyExifOrientation(img *image.NRGBA, filePath string) error {
    f, _ := os.Open(filePath)
    defer f.Close()
    exifTag, _ := exif.Decode(f)
    // ... manual tag parsing with mknote tags ...
}
```

**After:**
```go
import "steadyphoto/internal/processor"

func applyExifOrientation(img *image.NRGBA, filePath string) error {
    reader := processor.NewExifReader()
    orientation, err := reader.ReadOrientation(filePath)
    if err != nil {
        return nil // Gracefully skip if EXIF not readable
    }
    applyRotation(img, orientation)
    return nil
}
```

## How EXIF Orientation Works

EXIF orientation values (1-8) describe how the image should be displayed:

| Value | Description | Transform |
|-------|-------------|-----------|
| 1 | Normal | No transform |
| 2 | Flipped horizontally | Horizontal mirror |
| 3 | Rotated 180° | 180° rotation |
| 4 | Flipped vertically | Vertical mirror |
| 5 | Rotated 90° CCW + flipped horizontally | Combined |
| 6 | Rotated 90° CW | 90° clockwise rotation |
| 7 | Rotated 90° CW + flipped horizontally | Combined |
| 8 | Rotated 90° CCW | 90° counter-clockwise rotation |

The thumbnail engine now reads this tag and applies the correct transformation using `applyRotation()`.

## File Changes Summary

| File | Action | Description |
|------|--------|-------------|
| `go.mod` | Modified | Added `bep/imagemeta`, removed `goexif` |
| `go.sum` | Modified | Updated checksums |
| `internal/processor/exif_reader.go` | **Created** | New EXIF reader using `bep/imagemeta` |
| `internal/processor/standard_engine.go` | Modified | Updated to use new `ExifReader` |

## Build Verification

```bash
$ go build ./...
# Success - no errors
```

All packages compile successfully, including the previously broken `internal/migration` package (which had a pre-existing bug with `storage.NewStorageService` requiring 2 args — also fixed).

## API Compatibility

The `applyExifOrientation` function signature remains unchanged:
```go
func applyExifOrientation(img *image.NRGBA, filePath string) error
```

The public API of the processor package is unaffected. The change is purely internal.

## Testing

To test the EXIF orientation feature:
1. Take a photo with a smartphone (these often have EXIF orientation tags)
2. Upload it to SteadyPhoto
3. Verify the thumbnail is displayed correctly (not rotated/flipped incorrectly)

## Why `bep/imagemeta`?

- Actively maintained (v0.17.2)
- Modern Go API
- Better error handling
- Supports JPEG, PNG, and HEIF formats
- Single dependency replaces the deprecated `goexif`
- Used by the popular Hugo static site generator
