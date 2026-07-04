package main

import (
	"encoding/json"
	"fmt"
	"steadyphoto/internal/domain"
	"steadyphoto/internal/processor"
)

func main() {
	// Simulate what buildMetadataFromExif would produce
	exifInfo := &processor.ExifInfo{
		Orientation: processor.OrientationRotate90CW, // 6
		Tags: []processor.ExifTag{
			{Source: "EXIF", Tag: "Make", Value: "Canon"},
			{Source: "EXIF", Tag: "Model", Value: "EOS R5"},
			{Source: "EXIF", Tag: "ExposureTime", Value: "1/250"},
			{Source: "EXIF", Tag: "FNumber", Value: "2.8"},
			{Source: "EXIF", Tag: "ISOSpeedRatings", Value: "100"},
			{Source: "EXIF", Tag: "FocalLength", Value: "35"},
			{Source: "EXIF", Tag: "DateTimeOriginal", Value: "2024:01:15 10:30:00"},
			{Source: "XMP", Tag: "LensModel", Value: "RF35mm f/1.8 Macro IS STM"},
			{Source: "EXIF", Tag: "Orientation", Value: "6"}, // This gets skipped in buildMetadataFromExif
		},
	}

	// This is what buildMetadataFromExif returns
	metadata := buildMetadataFromExif(exifInfo)

	// Convert to JSON (what gets stored in the database)
	jsonBytes, _ := json.MarshalIndent(metadata, "", "  ")
	
	fmt.Println("=== JSON stored in media.metadata JSONB field ===")
	fmt.Println(string(jsonBytes))
}

// buildMetadataFromExif converts ExifInfo to Metadata map
func buildMetadataFromExif(info *processor.ExifInfo) domain.Metadata {
	metadata := domain.Metadata{}

	// Add orientation if not normal
	if info.Orientation != processor.OrientationNormal {
		metadata["orientation"] = fmt.Sprintf("%d", int(info.Orientation))
	}

	// Add other EXIF tags
	for _, tag := range info.Tags {
		// Skip orientation as it's already handled
		if tag.Tag == "Orientation" {
			continue
		}

		// Add tag to metadata (use tag name as key)
		if _, exists := metadata[tag.Tag]; !exists {
			metadata[tag.Tag] = tag.Value
		}
	}

	return metadata
}
