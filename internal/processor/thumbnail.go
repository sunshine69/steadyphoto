package processor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"steadyphoto/internal/domain"
)

// ThumbnailProcessor handles the generation of image previews.
type ThumbnailProcessor struct {
	engine      ImageEngine
	storageRoot string // The root directory of the photo storage
	thumbRoot   string // The root directory where thumbnails are stored (e.g. storage/.thumbnails)
}

// NewThumbnailProcessor creates a new instance of the processor.
func NewThumbnailProcessor(engine ImageEngine, storageRoot string, thumbRoot string) *ThumbnailProcessor {
	return &ThumbnailProcessor{
		engine:      engine,
		storageRoot: storageRoot,
		thumbRoot:   thumbRoot,
	}
}

// ProcessJob takes a background job and performs the thumbnail generation.
func (p *ThumbnailProcessor) ProcessJob(ctx context.Context, job *domain.Job, photo *domain.Media) error {
	if job.Type != domain.JobTypeThumbnail {
		return fmt.Errorf("invalid job type: %s", job.Type)
	}

	if photo.Path == "" {
		return fmt.Errorf("photo path is empty")
	}

	// Check if this is a video file
	// Check MediaType first, then fallback to file extension if MediaType is empty/unknown
	isVideo := photo.MediaType == domain.MediaTypeVideo
	if !isVideo && photo.Path != "" {
		ext := strings.ToLower(filepath.Ext(photo.Path))
		isVideo = ext == ".mp4" || ext == ".mov" || ext == ".avi" || ext == ".mkv" || ext == ".webm" || ext == ".flv"
	}
	
	// For videos, generate an SVG placeholder with the filename
	if isVideo {
		return p.generateVideoPlaceholder(ctx, job, photo)
	}

	// 1. Resolve the absolute path of the original photo
	inputAbsPath := filepath.Join(p.storageRoot, photo.Path)

	// 2. Calculate the destination path
	// If original is "2023/01/01/photo.jpg"
	// Thumbnail will be "storage/.thumbnails/2023/01/01/photo_thumb.webp"
	thumbAbsPath, err := p.getThumbnailAbsPath(photo.Path)
	if err != nil {
		return fmt.Errorf("failed to calculate thumbnail path: %w", err)
	}

	// Ensure the thumbnail subdirectories exist
	err = os.MkdirAll(filepath.Dir(thumbAbsPath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create thumbnail subdirectories: %w", err)
	}

	// 3. Execute the resize via the engine
	// We target a standard width of 800px for previews
	err = p.engine.Resize(ctx, inputAbsPath, thumbAbsPath, 800)
	if err != nil {
		return fmt.Errorf("engine resize failed: %w", err)
	}

	return nil
}

// getThumbnailAbsPath maps an original relative photo path to its absolute thumbnail path.
func (p *ThumbnailProcessor) getThumbnailAbsPath(originalRelPath string) (string, error) {
	// Clean the path
	relPath := filepath.Clean(originalRelPath)
	
	// Remove the original extension and add thumb suffix
	ext := filepath.Ext(relPath)
	base := strings.TrimSuffix(relPath, ext)
	
	// We'll use .webp for thumbnails as it's efficient
	// The relative path inside thumbRoot will be ".thumbnails/YYYY/MM/DD/filename_thumb.webp"
	// but since thumbRoot is already "storage/.thumbnails", we just need "YYYY/MM/DD/..."
	
	// However, the original relPath might start with "storage/" depending on how it was stored.
	// The design doc says: "Stores relative paths in the database."
	// So if photo.Path is "2023/01/01/img.jpg", then thumbRel is "2023/01/01/img_thumb.webp"
	
	thumbRelPath := filepath.Join(base+"_thumb.webp")
	
	// Join with the thumb root directory
	fullThumbPath := filepath.Join(p.thumbRoot, thumbRelPath)
	
	return fullThumbPath, nil
}

// generateVideoPlaceholder creates an SVG placeholder for video files.
// This allows the UI to show a thumbnail instead of falling back to loading the whole video.
func (p *ThumbnailProcessor) generateVideoPlaceholder(ctx context.Context, job *domain.Job, photo *domain.Media) error {
	// Extract the filename from the path
	filename := filepath.Base(photo.Path)
	// Remove extension
	nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))
	// Truncate if too long
	if len(nameWithoutExt) > 40 {
		nameWithoutExt = nameWithoutExt[:37] + "..."
	}

	// SVG content with video icon and filename
	svgContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg width="400" height="300" xmlns="http://www.w3.org/2000/svg">
  <defs>
    <linearGradient id="bgGrad" x1="0%%" y1="0%%" x2="100%%" y2="100%%">
      <stop offset="0%%" style="stop-color:#1a1a1a;stop-opacity:1" />
      <stop offset="100%%" style="stop-color:#2d2d2d;stop-opacity:1" />
    </linearGradient>
  </defs>
  
  <!-- Background -->
  <rect width="400" height="300" fill="url(#bgGrad)" rx="8"/>
  
  <!-- Video icon -->
  <g transform="translate(200, 110)">
    <polygon points="0,-40 35,20 -35,20" fill="#ffffff" opacity="0.9"/>
  </g>
  
  <!-- Label -->
  <text x="200" y="180" text-anchor="middle" fill="#ffffff" font-family="Arial, sans-serif" font-size="14" font-weight="bold">
    VIDEO FILE
  </text>
  
  <!-- Filename -->
  <text x="200" y="220" text-anchor="middle" fill="#cccccc" font-family="Arial, sans-serif" font-size="12">
    %s
  </text>
</svg>`, nameWithoutExt)

	// Calculate thumbnail path
	thumbAbsPath, err := p.getVideoPlaceholderPath(photo.Path)
	if err != nil {
		return fmt.Errorf("failed to calculate placeholder path: %w", err)
	}

	// Ensure directory exists
	err = os.MkdirAll(filepath.Dir(thumbAbsPath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create placeholder directory: %w", err)
	}

	// Write SVG file
	err = os.WriteFile(thumbAbsPath, []byte(svgContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write placeholder SVG: %w", err)
	}

	return nil
}

// getVideoPlaceholderPath generates the path for the video placeholder thumbnail
func (p *ThumbnailProcessor) getVideoPlaceholderPath(originalRelPath string) (string, error) {
	// Clean the path
	relPath := filepath.Clean(originalRelPath)
	
	// Remove the original extension and add svg suffix for placeholder
	ext := filepath.Ext(relPath)
	base := strings.TrimSuffix(relPath, ext)
	
	// Use .svg extension for the placeholder
	thumbRelPath := filepath.Join(base + "_placeholder.svg")
	
	// Join with the thumb root directory
	fullThumbPath := filepath.Join(p.thumbRoot, thumbRelPath)
	
	return fullThumbPath, nil
}
