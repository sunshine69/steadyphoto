package processor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ffprobe default timeout
const defaultFFProbeTimeout = 30 * time.Second

// FFProbeResult represents the full JSON output of ffprobe -show_format -show_streams
type FFProbeResult struct {
	Format   FFProbeFormat   `json:"format"`
	Streams  []FFProbeStream `json:"streams"`
}

// FFProbeExtractor wraps the ffprobe CLI for video metadata extraction
type FFProbeExtractor struct{}

// NewFFProbeExtractor creates a new FFProbeExtractor
func NewFFProbeExtractor() *FFProbeExtractor {
	return &FFProbeExtractor{}
}

// FFProbeFormat represents the "format" section of ffprobe output
type FFProbeFormat struct {
	FileName         string        `json:"filename"`
	NbStreams        int           `json:"nb_streams"`
	FormatName       string        `json:"format_name"`
	FormatLongName   string        `json:"format_long_name"`
	StartTime        string        `json:"start_time"`
	Duration         string        `json:"duration"`
	BitRate          string        `json:"bit_rate"`
	Size             string        `json:"size"`
	ProbeSize        string        `json:"probe_size"`
	Tag              FFProbeTags   `json:"tags,omitempty"`
}

// FFProbeStream represents a single stream entry from ffprobe output
type FFProbeStream struct {
	Index           int     `json:"index"`
	CodecName       string  `json:"codec_name"`
	CodecLongName   string  `json:"codec_long_name"`
	CodecType       string  `json:"codec_type"`
	Width           int     `json:"width"`
	Height          int     `json:"height"`
	RFrameRate      string  `json:"r_frame_rate"`
	AvgFrameRate    string  `json:"avg_frame_rate"`
	PixFmt          string  `json:"pix_fmt"`
	ColorSpace      string  `json:"color_space"`
	ColorTransfer   string  `json:"color_transfer"`
	ColorPrimaries  string  `json:"color_primaries"`
	Duration        string  `json:"duration"`
	BitRate         string  `json:"bit_rate"`
	SampleRate      string  `json:"sample_rate"`
	SampleFmt       string  `json:"sample_fmt"`
	Channels        int     `json:"channels"`
	ChannelLayout   string  `json:"channel_layout"`
	Disposition     *FFProbeDisposition `json:"disposition,omitempty"`
	Tag             FFProbeTags `json:"tags,omitempty"`
}

// FFProbeDisposition represents the "disposition" object in stream data
type FFProbeDisposition struct {
	Default         int `json:"default"`
	Dub             int `json:"dub"`
	Original        int `json:"original"`
	Comment         int `json:"comment"`
	Lyrics          int `json:"lyrics"`
	Karaoke         int `json:"karaoke"`
	Forced          int `json:"forced"`
	HearingImpaired int `json:"hearing_impaired"`
	VisualImpaired  int `json:"visual_impaired"`
	CleanEffects    int `json:"clean_effects"`
	AttachedPic     int `json:"attached_pic"`
	TimedThumbnails int `json:"timed_thumbnails"`
	Captions        int `json:"captions"`
	Description     int `json:"description"`
}

// FFProbeTags represents key-value metadata tags
type FFProbeTags map[string]string

// ProbeStreamProperties runs ffprobe and returns the full result
func (e *FFProbeExtractor) ProbeStreamProperties(ctx context.Context, filePath string) (*FFProbeResult, error) {
	if ctx.Done() != nil {
		// Use provided context with timeout
	}

	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	}

	output, err := e.runFFProbe(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	var result FFProbeResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	return &result, nil
}

// ProbeDuration extracts just the duration using the lightweight CSV output
func (e *FFProbeExtractor) ProbeDuration(ctx context.Context, filePath string) (float64, error) {
	args := []string{
		"-v", "quiet",
		"-show_entries", "format=duration",
		"-of", "csv=p=0",
		filePath,
	}

	output, err := e.runFFProbe(ctx, args)
	if err != nil {
		return 0, fmt.Errorf("ffprobe failed: %w", err)
	}

	durationStr := strings.TrimSpace(string(output))
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration %q: %w", durationStr, err)
	}

	return duration, nil
}

// ProbeVideoStream extracts only the first video stream's properties
func (e *FFProbeExtractor) ProbeVideoStream(ctx context.Context, filePath string) (*FFProbeStream, error) {
	result, err := e.ProbeStreamProperties(ctx, filePath)
	if err != nil {
		return nil, err
	}

	// Find first video stream
	for _, stream := range result.Streams {
		if stream.CodecType == "video" {
			return &stream, nil
		}
	}

	return nil, fmt.Errorf("no video stream found")
}

// ProbeAudioStream extracts only the first audio stream's properties
func (e *FFProbeExtractor) ProbeAudioStream(ctx context.Context, filePath string) (*FFProbeStream, error) {
	result, err := e.ProbeStreamProperties(ctx, filePath)
	if err != nil {
		return nil, err
	}

	// Find first audio stream
	for _, stream := range result.Streams {
		if stream.CodecType == "audio" {
			return &stream, nil
		}
	}

	return nil, fmt.Errorf("no audio stream found")
}

// ProbeCreationTime extracts the creation_time from format tags
func (e *FFProbeExtractor) ProbeCreationTime(ctx context.Context, filePath string) (string, error) {
	result, err := e.ProbeStreamProperties(ctx, filePath)
	if err != nil {
		return "", err
	}

	if result.Format.Tag != nil {
		if t, ok := result.Format.Tag["creation_time"]; ok {
			return t, nil
		}
		if t, ok := result.Format.Tag["creationtime"]; ok {
			return t, nil
		}
	}

	return "", nil
}

// IsAvailable checks if ffprobe is installed and accessible
func (e *FFProbeExtractor) IsAvailable() bool {
	_, err := exec.LookPath("ffprobe")
	return err == nil
}

// runFFProbe executes ffprobe with the given arguments and returns stdout
func (e *FFProbeExtractor) runFFProbe(ctx context.Context, args []string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultFFProbeTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffprobe", args...)

	// Capture both stdout and stderr
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("ffprobe exited with code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("ffprobe timed out after %v", defaultFFProbeTimeout)
		}
		return nil, err
	}

	return output, nil
}

// ParseFrameRate parses an ffprobe frame rate string like "30/1" or "24000/1001"
// Returns float64 fps value
func ParseFrameRate(rFrameRate string) float64 {
	rFrameRate = strings.TrimSpace(rFrameRate)
	if rFrameRate == "" {
		return 0
	}

	// Handle integer format (e.g. "30")
	if !strings.Contains(rFrameRate, "/") {
		fps, err := strconv.ParseFloat(rFrameRate, 64)
		if err != nil {
			return 0
		}
		return fps
	}

	// Handle fraction format (e.g. "30/1" or "24000/1001")
	parts := strings.SplitN(rFrameRate, "/", 2)
	numerator, err1 := strconv.ParseFloat(parts[0], 64)
	denominator, err2 := strconv.ParseFloat(parts[1], 64)

	if err1 != nil || err2 != nil || denominator == 0 {
		return 0
	}

	return numerator / denominator
}

// ParseDurationString converts an ffprobe duration string (seconds as float string) to float64
func ParseDurationString(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	d, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return d
}

// VideoCodecName returns a human-readable codec name from codec_name
func VideoCodecName(codecName string) string {
	switch strings.ToLower(codecName) {
	case "h264", "avc1":
		return "H.264/AVC"
	case "h265", "hevc", "hvc1", "hev1":
		return "H.265/HEVC"
	case "vp8", "vp9":
		return "WebM VP" + strings.TrimPrefix(codecName, "vp")
	case "av1":
		return "AV1"
	case "mpeg4", "mp4v":
		return "MPEG-4 Part 2"
	case "mpeg2video":
		return "MPEG-2 Video"
	default:
		return strings.ToUpper(codecName)
	}
}

// AudioCodecName returns a human-readable audio codec name
func AudioCodecName(codecName string) string {
	switch strings.ToLower(codecName) {
	case "aac":
		return "AAC"
	case "mp3":
		return "MP3"
	case "opus":
		return "Opus"
	case "vorbis":
		return "Vorbis"
	case "flac":
		return "FLAC"
	case "alac":
		return "ALAC (Apple Lossless)"
	case "ac3", "eac3":
		return strings.ToUpper(codecName)
	default:
		return strings.ToUpper(codecName)
	}
}

// FormatBitRate converts an ffprobe bit_rate string to int64 bits per second
func FormatBitRate(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	return strconv.ParseInt(s, 10, 64)
}

// FormatSize converts an ffprobe size string to int64 bytes
func FormatSize(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	return strconv.ParseInt(s, 10, 64)
}
