package processor

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"steadyphoto/internal/domain"
)

// ThumbnailProcessor handles the generation of image previews.
type ThumbnailProcessor struct {
	engine      ImageEngine
	storageRoot string
	thumbRoot   string
}

// NewThumbnailProcessor creates a new instance of the processor.
func NewThumbnailProcessor(engine ImageEngine, storageRoot string, thumbRoot string) *ThumbnailProcessor {
	return &ThumbnailProcessor{
		engine:      engine,
		storageRoot: storageRoot,
		thumbRoot:   thumbRoot,
	}
}

// GetThumbRoot returns the thumbnail root directory.
func (p *ThumbnailProcessor) GetThumbRoot() string {
	return p.thumbRoot
}

// GetThumbnailAbsPath returns the absolute thumbnail path for a given media relative path.
func (p *ThumbnailProcessor) GetThumbnailAbsPath(originalRelPath string) (string, error) {
	return p.getThumbnailAbsPath(originalRelPath)
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
	isVideo := photo.MediaType == domain.MediaTypeVideo
	if !isVideo && photo.Path != "" {
		ext := strings.ToLower(filepath.Ext(photo.Path))
		isVideo = ext == ".mp4" || ext == ".mov" || ext == ".avi" || ext == ".mkv" || ext == ".webm" || ext == ".flv"
	}

	// For videos, generate an SVG placeholder with the filename
	if isVideo {
		return p.generateVideoThumbnail(ctx, job, photo)
	}

	// 1. Resolve the absolute path of the original photo
	inputAbsPath := filepath.Join(p.storageRoot, photo.Path)
	log.Printf("  [DEBUG] Input path: %s", inputAbsPath)

	// 2. Calculate the destination path
	thumbAbsPath, err := p.getThumbnailAbsPath(photo.Path)
	if err != nil {
		return fmt.Errorf("failed to calculate thumbnail path: %w", err)
	}
	log.Printf("  [DEBUG] Thumbnail path: %s", thumbAbsPath)

	// Ensure the thumbnail subdirectories exist
	err = os.MkdirAll(filepath.Dir(thumbAbsPath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create thumbnail subdirectories: %w", err)
	}

	// 3. Execute the resize via the engine
	err = p.engine.Resize(ctx, inputAbsPath, thumbAbsPath, 800)
	if err != nil {
		return fmt.Errorf("engine resize failed: %w", err)
	}

	// 4. VALIDATE: Check that the file actually exists on disk after write
	if _, statErr := os.Stat(thumbAbsPath); statErr != nil {
		return fmt.Errorf("VALIDATION FAILED: thumbnail file does not exist at %s (stat error: %v)", thumbAbsPath, statErr)
	}

	info, err := os.Stat(thumbAbsPath)
	if err != nil {
		return fmt.Errorf("VALIDATION FAILED: cannot stat file %s: %v", thumbAbsPath, err)
	}

	log.Printf("  [DEBUG] Thumbnail file created successfully: %s (%d bytes)", thumbAbsPath, info.Size())

	if info.Size() == 0 {
		return fmt.Errorf("VALIDATION FAILED: thumbnail file is empty (0 bytes) at %s", thumbAbsPath)
	}

	return nil
}

// getThumbnailAbsPath maps an original relative photo path to its absolute thumbnail path.
func (p *ThumbnailProcessor) getThumbnailAbsPath(originalRelPath string) (string, error) {
	relPath := filepath.Clean(originalRelPath)

	ext := filepath.Ext(relPath)
	base := strings.TrimSuffix(relPath, ext)

	thumbRelPath := filepath.Join(base + "_thumb.webp")

	fullThumbPath := filepath.Join(p.thumbRoot, thumbRelPath)

	return fullThumbPath, nil
}

// generateVideoThumbnail extracts a real frame from the video using ffmpeg and writes it as webp.
// It captures the frame at ~5 seconds into the video and outputs base.webp to match
// the path expected by ServeThumbnailFile for video media.
func (p *ThumbnailProcessor) generateVideoThumbnail(ctx context.Context, job *domain.Job, photo *domain.Media) error {
	// 1. Resolve input path
	inputAbsPath := filepath.Join(p.storageRoot, photo.Path)
	log.Printf("  [DEBUG] Video input path: %s", inputAbsPath)

	// 2. Calculate destination path: base.webp (matching ServeThumbnailFile expectation for videos)
	thumbAbsPath, err := p.getVideoThumbnailAbsPath(photo.Path)
	if err != nil {
		return fmt.Errorf("failed to calculate video thumbnail path: %w", err)
	}
	log.Printf("  [DEBUG] Video thumbnail path: %s", thumbAbsPath)

	// Ensure the thumbnail subdirectories exist
	err = os.MkdirAll(filepath.Dir(thumbAbsPath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create thumbnail subdirectories: %w", err)
	}

	// 3. Use ffmpeg to extract a frame at 5 seconds and output as webp
	//    ffmpeg -ss 00:00:05 -i <input> -frames:v 1 -q:v 3 <output.webp>
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-ss", "00:00:05",
		"-i", inputAbsPath,
		"-frames:v", "1",
		"-q:v", "3",
		thumbAbsPath,
	)

	// CombinedOutput captures both stdout and stderr together, so no need to set cmd.Stderr separately
	log.Printf("  [DEBUG] Running ffmpeg: %s", cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[WARN] ThumbnailProcessor: ffmpeg frame extraction failed for '%s': %v (output: %s)", photo.Path, err, string(out))

		// Fallback: generate SVG placeholder at the same path
		return p.generateSVGFallback(ctx, job, photo, thumbAbsPath)
	}

	// VALIDATE: check file exists and has size > 0
	info, err := os.Stat(thumbAbsPath)
	if err != nil {
		return fmt.Errorf("VALIDATION FAILED: video thumbnail does not exist at %s: %v", thumbAbsPath, err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("VALIDATION FAILED: video thumbnail is empty (0 bytes) at %s", thumbAbsPath)
	}

	log.Printf("  [DEBUG] Video thumbnail created via ffmpeg: %s (%d bytes)", thumbAbsPath, info.Size())
	return nil
}

// getVideoThumbnailAbsPath generates the absolute thumbnail path for a video file.
// Videos use base.webp (not base_thumb.webp) to match ServeThumbnailFile.
// Preserves full directory structure so it matches the handler's lookup.
// e.g. "storage/9a362745-.../2024/05/13/video.mp4" -> ".thumbnails/storage/9a362745-.../2024/05/13/video.webp"
func (p *ThumbnailProcessor) getVideoThumbnailAbsPath(originalRelPath string) (string, error) {
	relPath := filepath.Clean(originalRelPath)
	extClean := filepath.Ext(relPath)
	basePart := strings.TrimSuffix(relPath, extClean)
	thumbRelPath := basePart + ".webp"
	fullThumbPath := filepath.Join(p.thumbRoot, thumbRelPath)
	return fullThumbPath, nil
}

// generateSVGFallback creates a simple SVG placeholder when ffmpeg fails.
func (p *ThumbnailProcessor) generateSVGFallback(ctx context.Context, job *domain.Job, photo *domain.Media, thumbAbsPath string) error {
	filename := filepath.Base(photo.Path)
	nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))
	if len(nameWithoutExt) > 40 {
		nameWithoutExt = nameWithoutExt[:37] + "..."
	}

	svgContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg width="400" height="300" xmlns="http://www.w3.org/2000/svg">
  <defs>
    <linearGradient id="bgGrad" x1="0%%" y1="0%%" x2="100%%" y2="100%%">
      <stop offset="0%%" style="stop-color:#1a1a1a;stop-opacity:1" />
      <stop offset="100%%" style="stop-color:#2d2d2d;stop-opacity:1" />
    </linearGradient>
  </defs>
  <rect width="400" height="300" fill="url(#bgGrad)" rx="8"/>
  <g transform="translate(200, 110)">
    <polygon points="0,-40 35,20 -35,20" fill="#ffffff" opacity="0.9"/>
  </g>
  <text x="200" y="180" text-anchor="middle" fill="#ffffff" font-family="Arial, sans-serif" font-size="14" font-weight="bold">VIDEO FILE</text>
  <text x="200" y="220" text-anchor="middle" fill="#cccccc" font-family="Arial, sans-serif" font-size="12">%s</text>
</svg>`, nameWithoutExt)

	err := os.WriteFile(thumbAbsPath, []byte(svgContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write fallback SVG: %w", err)
	}
	log.Printf("  [DEBUG] Fallback SVG created for video: %s (%d bytes)", thumbAbsPath, 1024)
	return nil
}
