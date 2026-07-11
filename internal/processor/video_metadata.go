package processor

import (
	"context"
	"fmt"
	"github.com/jbrodriguez/mlog"
	"strings"
	"time"

	"steadyphoto/internal/domain"
)

// ExtractVideoMetadata extracts comprehensive metadata from a video file using ffprobe.
// It returns a populated domain.VideoMetadata or an error if ffprobe fails or no video stream is found.
func ExtractVideoMetadata(ctx context.Context, filePath string) (*domain.VideoMetadata, error) {
	extractor := NewFFProbeExtractor()

	// Get full stream properties from ffprobe
	result, err := extractor.ProbeStreamProperties(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed for %s: %w", filePath, err)
	}

	// Find video and audio streams
	var (
		videoStream *FFProbeStream
		audioStream *FFProbeStream
	)
	for i := range result.Streams {
		s := &result.Streams[i]
		if s.CodecType == "video" && videoStream == nil {
			videoStream = s
		}
		if s.CodecType == "audio" && audioStream == nil {
			audioStream = s
		}
	}

	if videoStream == nil {
		return nil, fmt.Errorf("no video stream found in %s", filePath)
	}

	// Parse duration
	duration := ParseDurationString(result.Format.Duration)
	if duration == 0 && videoStream.Duration != "" {
		duration = ParseDurationString(videoStream.Duration)
	}

	// Parse frame rate
	frameRate := ParseFrameRate(videoStream.RFrameRate)
	if frameRate == 0 {
		frameRate = ParseFrameRate(videoStream.AvgFrameRate)
	}

	// Parse bitrate
	var bitrate int64
	if result.Format.BitRate != "" {
		bitrate, _ = FormatBitRate(result.Format.BitRate)
	}

	// Parse creation/modification times
	creationTime := ""
	modifiedTime := ""
	if result.Format.Tag != nil {
		creationTime = result.Format.Tag["creation_time"]
		if creationTime == "" {
			creationTime = result.Format.Tag["creationtime"]
		}
		modifiedTime = result.Format.Tag["modification_time"]
		if modifiedTime == "" {
			modifiedTime = result.Format.Tag["modifytime"]
		}
	}

	vm := &domain.VideoMetadata{
		Duration:   duration,
		Width:      videoStream.Width,
		Height:     videoStream.Height,
		Bitrate:    bitrate,
		VideoCodec: VideoCodecName(videoStream.CodecName),
		FrameRate:  frameRate,
	}

	if audioStream != nil {
		vm.AudioCodec = AudioCodecName(audioStream.CodecName)
	}

	// Parse creation time
	if creationTime != "" {
		for _, layout := range []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02T15:04:05.000000Z",
			"2006-01-02 15:04:05",
		} {
			if t, err := time.Parse(layout, creationTime); err == nil {
				vm.CreatedAt = t
				break
			}
		}
	}

	// Parse modification time
	if modifiedTime != "" {
		for _, layout := range []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02T15:04:05.000000Z",
			"2006-01-02 15:04:05",
		} {
			if t, err := time.Parse(layout, modifiedTime); err == nil {
				vm.ModifiedAt = t
				break
			}
		}
	}

	mlog.Info("[VIDEO_META] Extracted from %s: codec=%s audio=%s res=%dx%d dur=%.2fs fps=%.2f",
		filePath, vm.VideoCodec, vm.AudioCodec, vm.Width, vm.Height, vm.Duration, vm.FrameRate)

	return vm, nil
}

// IsVideoFile returns true if the file path has a video extension.
func IsVideoFile(path string) bool {
	ext := strings.ToLower(strings.TrimSpace(path))
	videoExts := map[string]bool{
		".mp4": true, ".mov": true, ".avi": true, ".mkv": true,
		".webm": true, ".flv": true, ".wmv": true, ".m4v": true, ".3gp": true,
	}
	for extPath := range videoExts {
		if strings.HasSuffix(ext, extPath) {
			return true
		}
	}
	return false
}
