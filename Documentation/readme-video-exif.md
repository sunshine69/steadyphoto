# Phone Video EXIF Data Extraction — Implementation Plan

**Status:** 🟡 In Progress  
**Created:** July 10, 2026  
**Approach:** ffprobe (FFmpeg) for stream properties + Custom QuickTime atom parser for phone-specific metadata  
**Dependencies:** FFmpeg (system binary), pure Go QuickTime parser

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Current State & Gap Analysis](#2-current-state--gap-analysis)
3. [Architecture Overview](#3-architecture-overview)
4. [Implementation Phases](#4-implementation-phases)
5. [Technical Specification](#5-technical-specification)
6. [Data Model Changes](#6-data-model-changes)
7. [Integration Points](#7-integration-points)
8. [Testing Strategy](#8-testing-strategy)
9. [Dependencies & Installation](#9-dependencies--installation)
10. [Deployment Notes](#10-deployment-notes)

---

## 1. Executive Summary

Phone videos (MP4 from Android, MOV/M4V from iPhone) contain rich metadata in their container atoms — device make/model, camera settings (ISO, exposure, aperture, focal length), GPS coordinates, creation dates, software versions, stabilization info, and more. The current codebase has **zero support** for extracting this metadata from video files.

This document outlines a hybrid implementation plan:
- **ffprobe** (FFmpeg's CLI tool) for stream-level properties (duration, resolution, codecs, bitrate, frame rate)
- **Custom pure Go QuickTime atom parser** for container-level metadata (device, GPS, camera settings, dates)

**Estimated Effort:** 7-9 developer days

---

## 2. Current State & Gap Analysis

### 2.1 What Already Exists

| Component | Status | Limitation |
|-----------|--------|------------|
| `internal/processor/exif_reader.go` | ✅ Image EXIF only | Handles JPEG, PNG, WebP, HEIF, RAW — **not video** |
| `VideoMetadata` domain model | ✅ Defined | Fields exist but **never populated** |
| `video_metadata` JSONB column | ✅ In DB schema | Empty for all video records |
| Worker (`cmd/worker`) | ✅ Functional | Processes thumbnails & face detection only — **no video metadata** |
| EXIF updater CLI (`cmd/exif`) | ✅ Functional | Image EXIF only — **skips videos entirely** |
| Upload handler | ✅ Functional | Stubbed metadata extraction at upload time |

### 2.2 Known Issues (from quick-start.md)

```
### 🔴 Critical Issues
3. **EXIF processing incomplete**
   - Only reads orientation, doesn't apply corrections
   - **Fix needed**: Add orientation correction + video metadata
```

### 2.3 Gap: What Phone Videos Need

Phone video files (MP4, MOV, M4V) contain metadata in QuickTime/MP4 container atoms that **cannot** be extracted by standard image EXIF libraries (imagemeta/goexif). The relevant atoms are:

```
File (MP4/MOV)
├── ftyp (File Type)
│   └── major_brand: "qt  " (QuickTime), "M4V " (iTunes)
├── moov
│   └── mvhd (Movie Header)
│       └── creation_time, modification_time
├── trak (Track)
│   ├── tkhd (Track Header)
│   │   └── creation_time, duration, width, height
│   └── mdia
│       ├── mdhd (Media Header)
│       │   └── creation_time, duration
│       └── minf
│           └── stbl
│               └── stsd (Sample Description)
│                   └── avc1/hevc1 (codec info)
└── udta
    └── meta
        └── iloc, iinf, iprp
            └── ipco (Item Property Container)
                └── mdta (Metadata Item)
                    ├── keys (dictionary)
                    └── data (values)
```

### 2.4 Key QuickTime Metadata Keys for Phone Videos

| Key | Description | Example | Phone Relevance |
|-----|------------|---------|-----------------|
| `com.apple.quicktime.creationdate` | Creation timestamp | "2024:01:15 10:30:00" | ⭐ Critical |
| `com.apple.quicktime.model` | Camera model | "iPhone 15 Pro" | ⭐ Critical |
| `com.apple.quicktime.software` | Software version | "iOS 17.2" | High |
| `com.apple.quicktime.make` | Device manufacturer | "Apple" | High |
| `com.apple.quicktime.camera` | Camera type | "Rear", "Front" | High |
| `com.apple.quicktime.location.ISO6709` | GPS in ISO 6709 | "+37.7749-122.4194+004.000/" | ⭐ Critical |
| `com.apple.quicktime.stabilization` | Video stabilization | "0x01" | Medium |
| `com.apple.photo.capture.video.stabilization.mode` | Stabilization mode | "on" | Medium |
| `com.apple.quicktime.video.level` | Video level (4K, etc.) | "62" | Medium |
| `com.apple.quicktime.video.frame_rate` | Frame rate | "30.0" | Medium |
| `com.apple.quicktime.video.encoder` | Encoder | "H264", "HEVC" | Medium |
| `com.apple.quicktime.color.primarys` | Color primaries | "bt709", "bt2020" | Low |
| `com.apple.quicktime.transfer.function` | Transfer function | "smpte2084" (HDR) | Low |
| `com.apple.quicktime.video.hdr` | HDR info (JSON) | `{"bit_depth":10}` | Low |
| `com.apple.quicktime.audio.channel_layout` | Audio layout | "stereo" | Low |
| `org.id3.TIT2` | Title (ID3) | Song name | Low |
| `com.apple.iPhoto.caption` | Caption | "Birthday party" | Medium |

---

## 3. Architecture Overview

### 3.1 High-Level Design

```
┌──────────────────────────────────────────────────────────────────────┐
│                    VideoMetadataExtractor                             │
│                                                                      │
│  ┌─────────────────────┐    ┌──────────────────────────────────────┐ │
│  │ 1. Format           │    │ 2. ffprobe (FFmpeg)                   │ │
│  │    Detection        │───▶│    Stream Properties                  │ │
│  │    (magic bytes)    │    │    • duration, width, height, fps     │ │
│  │                     │    │    • bitrate, video/audio codecs      │ │
│  │                     │    │    • pixel_format, color_space        │ │
│  └─────────────────────┘    │    • creation_time, modification_time │ │
│                              └──────────────────────────────────────┘ │
│                                ▲                                      │
│  ┌─────────────────────┐    ┌──────────────────────────────────────┐ │
│  │ 3. QuickTime        │───▶│ 4. Data Unification & Output          │ │
│  │    Metadata Parser  │    │    • Merge ffprobe + Qt metadata      │ │
│  │    (pure Go)        │    │    • Populate VideoMetadata struct    │ │
│  │                     │    │    • Return to caller for DB update   │ │
│  └─────────────────────┘    └──────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────────┘
```

### 3.2 Why Hybrid Approach?

| Approach | Pros | Cons | Verdict |
|----------|------|------|---------|
| **ffprobe only** | Battle-tested, handles all formats, no custom parser needed | Requires FFmpeg binary on host, JSON parsing overhead, misses some phone-specific QuickTime keys | ⚠️ Good baseline, but incomplete |
| **Pure Go MP4 parser only** | No external deps, works anywhere | Must implement QuickTime atom parsing from scratch (complex), missing stream-level info | ⚠️ Complex, misses stream data |
| **Hybrid (recommended)** | Best of both — ffprobe for streams, Go parser for QuickTime metadata | Slightly more complex, FFmpeg required | ✅ **Recommended** |

### 3.3 Data Flow

```
Upload/Scan → Media Record Created
                    │
                    ▼
          Worker (background) or EXIF CLI (on-demand)
                    │
                    ▼
          VideoMetadataExtractor.ExtractVideoMetadata(ctx, media)
                    │
          ┌─────────┼─────────┐
          ▼         ▼         ▼
    ffprobe    Qt Parser   Format Check
    (streams)  (container) (magic bytes)
          │         │         │
          └─────────┼─────────┘
                    ▼
          VideoMetadata (populated)
                    │
                    ▼
          mediaRepo.Update(ctx, media)
                    │
                    ▼
          Database: video_metadata JSONB column updated
```

---

## 4. Implementation Phases

### Phase 1: Video Format Detection (0.5 days)

**Goal:** Detect video container type from file magic bytes.

**Files to Create:**
- `internal/processor/video_format.go`

**Implementation:**
```go
package processor

// VideoFormat represents a detected video container type
type VideoFormat int

const (
    VideoFormatUnknown VideoFormat = iota
    VideoFormatMP4      // Android default
    VideoFormatMOV      // iPhone default
    VideoFormatM4V      // Apple iTunes
    VideoFormatAVI      // Legacy
    VideoFormatWebM     // Rare on phones
)

// DetectVideoFormat detects video container from magic bytes
func DetectVideoFormat(file *os.File) (VideoFormat, error)
```

**Magic Byte Detection:**

| Format | Magic Bytes | Offset | Notes |
|--------|------------|--------|-------|
| MP4 | `ftyp` box | 4 | Standard MP4 |
| MOV | `moov` or `ftyp` with `qt  ` | 4 | QuickTime |
| M4V | `ftyp` with `M4V ` | 4 | iTunes |
| AVI | `RIFF`...`AVI ` | 0 | Legacy |
| WebM | `1A 45 DF A3` | 0 | Rare |

### Phase 2: ffprobe Wrapper (1 day)

**Goal:** Extract stream-level properties using ffprobe CLI.

**Files to Create:**
- `internal/processor/ffprobe.go`

**Implementation:**

```go
package processor

import (
    "context"
    "encoding/json"
    "os/exec"
    "time"
)

// FFProbeResult represents the JSON output of ffprobe
type FFProbeResult struct {
    Format   FFProbeFormat   `json:"format"`
    Streams  []FFProbeStream `json:"streams"`
}

type FFProbeFormat struct {
    Duration      float64 `json:"duration"`
    BitRate       int64   `json:"bit_rate"`
    Size          int64   `json:"size"`
    CreationTime  string  `json:"creation_time"`
    ModificationTime string `json:"modification_time"`
}

type FFProbeStream struct {
    Width          int     `json:"width"`
    Height         int     `json:"height"`
    RFrameRate     string  `json:"r_frame_rate"`
    CodecName      string  `json:"codec_name"`
    CodecLongName  string  `json:"codec_long_name"`
    PixelFormat    string  `json:"pixel_format"`
    ColorSpace     string  `json:"color_space"`
    ColorTransfer  string  `json:"color_transfer"`
    Duration       float64 `json:"duration"`
    SampleRate     int     `json:"sample_rate"`
    Channels       int     `json:"channels"`
}

// FFProbeExtractor wraps the ffprobe CLI
type FFProbeExtractor struct {}

func NewFFProbeExtractor() *FFProbeExtractor {
    return &FFProbeExtractor{}
}

// ProbeStreamProperties runs ffprobe and returns stream properties
func (e *FFProbeExtractor) ProbeStreamProperties(ctx context.Context, filePath string) (*FFProbeResult, error) {
    // ffprobe -v quiet -print_format json -show_format -show_streams filePath
}

// ProbeDuration extracts just the duration (faster for common case)
func (e *FFProbeExtractor) ProbeDuration(ctx context.Context, filePath string) (float64, error) {
    // ffprobe -v quiet -show_entries format=duration -of csv=p=0 filePath
}

// IsAvailable checks if ffprobe is installed and accessible
func (e *FFProbeExtractor) IsAvailable() bool {
    _, err := exec.LookPath("ffprobe")
    return err == nil
}
```

**ffprobe Command:**
```bash
ffprobe -v quiet -print_format json -show_format -show_streams /path/to/video.mp4
```

**Sample ffprobe Output:**
```json
{
  "format": {
    "filename": "/path/to/video.mp4",
    "nb_streams": 2,
    "duration": 12.5,
    "bit_rate": 8500000,
    "size": 12500000,
    "creation_time": "2024-01-15T10:30:00.000000Z",
    "modification_time": "2024-01-15T10:30:05.000000Z",
    "format_name": "mov,mp4,m4a,3g2,3gp,psp"
  },
  "streams": [
    {
      "index": 0,
      "codec_name": "h265",
      "codec_long_name": "H.265/HEVC",
      "width": 3840,
      "height": 2160,
      "r_frame_rate": "30/1",
      "pix_fmt": "yuv420p",
      "color_space": "bt2020nc",
      "color_transfer": "smpte2084",
      "duration": 12.5,
      "bit_rate": "8200000"
    },
    {
      "index": 1,
      "codec_name": "aac",
      "sample_rate": "48000",
      "channels": 2,
      "bit_rate": "192000"
    }
  ]
}
```

### Phase 3: QuickTime Metadata Parser — Pure Go (2-3 days)

**Goal:** Parse QuickTime/MP4 container atoms to extract phone-specific metadata (device, GPS, camera settings) — **no external dependencies**.

**Files to Create:**
- `internal/processor/qt_atoms.go` — Atom type constants and helpers
- `internal/processor/qt_metadata.go` — QuickTime metadata parser

**Implementation Outline:**

```go
package processor

// QuickTimeAtom represents a parsed atom in the MP4 container
type QuickTimeAtom struct {
    Type    string
    Size    int64
    Data    []byte
    Children []*QuickTimeAtom
}

// QuickTimeMetadata represents extracted phone-specific metadata
type QuickTimeMetadata struct {
    // Device Info
    DeviceMake       string
    DeviceModel      string
    Software         string
    CameraType       string  // "Rear", "Front"
    
    // GPS
    ISO6709          string  // Raw ISO 6709 string
    GPSLatitude      float64
    GPSLongitude     float64
    GPSAltitude      float64
    HasGPS           bool
    
    // Timestamps
    CreationDate     time.Time
    ModificationTime time.Time
    
    // Camera Settings
    ExposureTime     string
    ISO              int
    FocalLength      string
    Aperture         string
    WhiteBalance     string
    FlashUsed        bool
    
    // Video Properties
    FrameRate        float64
    VideoEncoder     string
    VideoLevel       int
    IsHDR            bool
    ColorSpace       string
    Stabilization    string
    
    // Audio
    AudioChannels    int
    AudioSampleRate  int
    
    // Other
    Caption          string
    Title            string
}

// QTMetadataParser parses QuickTime/MP4 atoms
type QTMetadataParser struct {}

func NewQTMetadataParser() *QTMetadataParser {
    return &QTMetadataParser{}
}

// Parse extracts QuickTime metadata from a file
func (p *QTMetadataParser) Parse(filePath string) (*QuickTimeMetadata, error)
```

**Atom Parsing Strategy:**

1. Read file header (first 16 bytes) to find `ftyp` box
2. Iterate through atoms: `moov`, `trak`, `mdia`, etc.
3. Parse `mdta` (metadata) box — contains key-value pairs
4. Extract values for known phone metadata keys

**QuickTime Time Format:**
```
QuickTime times are in seconds since January 1, 1904 00:00:00 UTC
Unix epoch = January 1, 1970 00:00:00 UTC
Difference = 2082844800 seconds
```

### Phase 4: Video Metadata Extractor Orchestrator (0.5 day)

**Goal:** Combine ffprobe and Qt parser results into a single `VideoMetadata` struct.

**Files to Create:**
- `internal/processor/video_metadata_extractor.go`

**Implementation:**

```go
package processor

// VideoMetadataExtractor orchestrates video metadata extraction
type VideoMetadataExtractor struct {
    probeExtractor *FFProbeExtractor
    qtParser       *QTMetadataParser
}

func NewVideoMetadataExtractor() *VideoMetadataExtractor {
    return &VideoMetadataExtractor{
        probeExtractor: NewFFProbeExtractor(),
        qtParser:       NewQTMetadataParser(),
    }
}

// ExtractVideoMetadata extracts all metadata from a video file
// Returns populated VideoMetadata struct and any warnings
func (e *VideoMetadataExtractor) ExtractVideoMetadata(
    ctx context.Context, 
    filePath string,
) (*domain.VideoMetadata, []string, error) {
    
    var warnings []string
    
    // 1. Check ffprobe availability
    if !e.probeExtractor.IsAvailable() {
        warnings = append(warnings, "ffprobe not found — stream properties will be empty")
    }
    
    // 2. Run ffprobe for stream properties
    var probeResult *FFProbeResult
    if e.probeExtractor.IsAvailable() {
        var probeErr error
        probeResult, probeErr = e.probeExtractor.ProbeStreamProperties(ctx, filePath)
        if probeErr != nil {
            warnings = append(warnings, "ffprobe failed: "+probeErr.Error())
        }
    }
    
    // 3. Parse QuickTime metadata
    var qtMeta *QuickTimeMetadata
    var qtErr error
    qtMeta, qtErr = e.qtParser.Parse(filePath)
    if qtErr != nil {
        warnings = append(warnings, "QuickTime parsing failed: "+qtErr.Error())
    }
    
    // 4. Unify results into VideoMetadata
    metadata := &domain.VideoMetadata{}
    
    // From ffprobe
    if probeResult != nil {
        metadata.Duration = probeResult.Format.Duration
        metadata.Bitrate = probeResult.Format.BitRate
        metadata.CreatedAt = parseTime(probeResult.Format.CreationTime)
        metadata.ModifiedAt = parseTime(probeResult.Format.ModificationTime)
        
        for _, stream := range probeResult.Streams {
            if stream.CodecName != "" {
                if stream.Width > 0 && stream.Height > 0 {
                    metadata.Width = stream.Width
                    metadata.Height = stream.Height
                    metadata.VideoCodec = stream.CodecName
                    metadata.PixelFormat = stream.PixelFormat
                    metadata.ColorSpace = stream.ColorSpace
                    if stream.ColorTransfer != "" {
                        metadata.IsHDR = stream.ColorTransfer == "smpte2084"
                    }
                } else {
                    // Audio stream
                    metadata.AudioCodec = stream.CodecName
                    metadata.AudioSampleRate = stream.SampleRate
                    metadata.AudioChannels = stream.Channels
                }
            }
        }
        
        // Parse frame rate
        if len(probeResult.Streams) > 0 {
            metadata.FrameRate = parseFrameRate(probeResult.Streams[0].RFrameRate)
        }
    }
    
    // From QuickTime parser
    if qtMeta != nil {
        metadata.DeviceMake = qtMeta.DeviceMake
        metadata.DeviceModel = qtMeta.DeviceModel
        metadata.Software = qtMeta.Software
        metadata.GPSLatitude = qtMeta.GPSLatitude
        metadata.GPSLongitude = qtMeta.GPSLongitude
        metadata.GPSAltitude = qtMeta.GPSAltitude
        metadata.IsHDR = metadata.IsHDR || qtMeta.IsHDR
        metadata.Stabilization = qtMeta.Stabilization
        metadata.VideoEncoder = qtMeta.VideoEncoder
        metadata.Caption = qtMeta.Caption
        metadata.Title = qtMeta.Title
    }
    
    return metadata, warnings, nil
}
```

### Phase 5: Domain Model Extensions (0.5 day)

**Goal:** Extend `VideoMetadata` struct to support phone-specific fields.

**Files to Modify:**
- `internal/domain/media.go`

**Changes:**

```go
type VideoMetadata struct {
    // === Existing fields (ffprobe) ===
    Duration      float64   `json:"duration"`
    Width         int       `json:"width"`
    Height        int       `json:"height"`
    Bitrate       int64     `json:"bitrate"`
    VideoCodec    string    `json:"videoCodec"`
    AudioCodec    string    `json:"audioCodec"`
    FrameRate     float64   `json:"frameRate"`
    CreatedAt     time.Time `json:"createdAt"`
    ModifiedAt    time.Time `json:"modifiedAt"`
    
    // === New phone-specific fields ===
    DeviceMake       string    `json:"deviceMake"`        // "Apple", "Samsung"
    DeviceModel      string    `json:"deviceModel"`       // "iPhone 15 Pro"
    Software         string    `json:"software"`          // "iOS 17.2"
    CameraType       string    `json:"cameraType"`        // "Rear", "Front"
    GPSLatitude      float64   `json:"gpsLatitude"`
    GPSLongitude     float64   `json:"gpsLongitude"`
    GPSAltitude      float64   `json:"gpsAltitude"`
    Orientation      int       `json:"orientation"`
    ColorSpace       string    `json:"colorSpace"`
    IsHDR            bool      `json:"isHDR"`
    Stabilization    string    `json:"stabilization"`
    VideoEncoder     string    `json:"videoEncoder"`
    ExposureTime     string    `json:"exposureTime"`
    ISOSpeed         int       `json:"isoSpeed"`
    FocalLength      string    `json:"focalLength"`
    Aperture         string    `json:"aperture"`
    WhiteBalance     string    `json:"whiteBalance"`
    FlashUsed        bool      `json:"flashUsed"`
    Caption          string    `json:"caption"`
    Title            string    `json:"title"`
    AudioSampleRate  int       `json:"audioSampleRate"`
    AudioChannels    int       `json:"audioChannels"`
    PixelFormat      string    `json:"pixelFormat"`
    VideoLevel       int       `json:"videoLevel"`
    
    // === Raw metadata (everything we could extract) ===
    RawMetadata      Metadata  `json:"rawMetadata"`      // All extracted as JSONB
}
```

### Phase 6: Worker Integration (1 day)

**Goal:** Add `video_metadata` job type to the worker so videos get processed in the background.

**Files to Modify:**
- `internal/domain/job.go` — Add new job types
- `cmd/worker/main.go` — Handle new job type
- `internal/processor/` — Add video metadata extraction to existing processors

**Changes:**

```go
// In domain/job.go
const (
    JobTypeThumbnail     JobType = "thumbnail_generation"
    JobTypeFaceDetection JobType = "face_detection"
    JobTypeVideoMetadata JobType = "video_metadata"      // NEW
)

// In cmd/worker/main.go
case domain.JobTypeVideoMetadata:
    log.Printf("  [DEBUG] Extracting video metadata for: %s", media.Path)
    extractor := processor.NewVideoMetadataExtractor()
    metadata, warnings, err := extractor.ExtractVideoMetadata(ctx, media.Path)
    if err != nil {
        return logError(jobRepo, job.ID, "video metadata extraction failed: "+err.Error())
    }
    for _, w := range warnings {
        log.Printf("  [WARN] %s", w)
    }
    // Update media with extracted metadata
    media.VideoMetadata = *metadata
    // ... save to DB
```

### Phase 7: EXIF Updater CLI Integration (0.5 day)

**Goal:** Allow one-time backfill of video metadata for existing videos via CLI.

**Files to Modify:**
- `cmd/exif/main.go` — Add video format handling

**Changes:**

```go
// In buildMetadataFromExif, add video branch
if media.MediaType == domain.MediaTypeVideo {
    extractor := processor.NewVideoMetadataExtractor()
    metadata, warnings, err := extractor.ExtractVideoMetadata(ctx, fullPath)
    if err != nil {
        log.Printf("  Failed to extract video metadata: %v", err)
        errors++
        continue
    }
    media.VideoMetadata = *metadata
    // Store warnings in raw metadata
    if len(warnings) > 0 {
        media.Metadata = processor.GPSMetadataToMap(...) // or store in a separate field
    }
}
```

### Phase 8: Testing (1 day)

**Goal:** Comprehensive unit and integration tests.

**Files to Create:**
- `internal/processor/video_format_test.go`
- `internal/processor/ffprobe_test.go`
- `internal/processor/qt_metadata_test.go`
- `internal/processor/video_metadata_extractor_test.go`

---

## 5. Technical Specification

### 5.1 ffprobe Command Details

**Primary Command:**
```bash
ffprobe -v quiet -print_format json -show_format -show_streams /path/to/video.mp4
```

**Alternative (faster, duration only):**
```bash
ffprobe -v quiet -show_entries format=duration -of csv=p=0 /path/to/video.mp4
```

**Alternative (streams only):**
```bash
ffprobe -v quiet -print_format json -show_streams /path/to/video.mp4
```

**Error Handling:**
- ffprobe returns exit code 1 on error
- Check `stderr` for error messages
- Fallback: if ffprobe fails, return whatever partial data we could extract

### 5.2 QuickTime Atom Parsing Strategy

**Atom Structure:**
```
| Size (4 bytes) | Type (4 bytes) | Data (Size - 8 bytes) |
```

**Parsing Algorithm:**

1. Read 8 bytes (size + type)
2. If size == 1, read additional 8 bytes for 64-bit size
3. If size == 0, atom extends to end of file
4. Process atom data based on type
5. If atom has children, recurse
6. Move to next atom (at offset size)

**Key Atoms to Parse:**

| Atom Path | Fields Extracted |
|-----------|-----------------|
| `moov/mvhd` | creation_time, modification_time, time_scale |
| `moov/trak/tkhd` | width, height, duration |
| `moov/trak/mdia/mdhd` | duration, time_scale |
| `moov/trak/mdia/minf/stbl/stsd` | codec name, pixel format |
| `moov/udta/meta/iloc/iinf/iprp/ipco/mdta` | all metadata key-value pairs |

**ISO 6709 GPS Parsing:**

```
Format: ±lat±lon±alt/
Example: +37.7749-122.4194+004.000/

Parsing:
- lat: +37.7749 → 37.7749 (N)
- lon: -122.4194 → -122.4194 (W)
- alt: +004.000 → 4.0 (meters)
```

### 5.3 Error Handling Strategy

| Scenario | Behavior |
|----------|----------|
| ffprobe not installed | Log warning, skip stream properties, still parse QuickTime metadata |
| ffprobe returns error | Log warning, return partial data from QuickTime parser |
| File too large (>5GB) | Stream ffprobe output, don't load entire file into memory |
| Corrupt MP4 | Log warning, return whatever was successfully parsed |
| QuickTime parser fails | Log warning, return ffprobe data only |
| Both ffprobe and Qt parser fail | Log error, return empty VideoMetadata, caller can skip |

### 5.4 Performance Considerations

| Optimization | Impact |
|-------------|--------|
| Use ffprobe duration-only mode for common case | 10x faster |
| Parse QuickTime atoms incrementally | Reduces memory usage |
| Cache ffprobe results for same file | Avoids redundant processing |
| Skip files without QuickTime atoms | Avoids unnecessary parsing |
| Use `os.File.Seek` to avoid reading entire file | Reduces I/O |

---

## 6. Data Model Changes

### 6.1 Database Schema

**No schema changes needed!** The existing `video_metadata` JSONB column is sufficient.

```sql
-- Existing schema (no changes needed)
media (
    video_metadata JSONB,     -- Will be populated by this feature
    -- ... other columns
)
```

### 6.2 VideoMetadata JSONB Example

**Before:**
```json
{}
```

**After (iPhone video):**
```json
{
  "duration": 12.5,
  "width": 3840,
  "height": 2160,
  "bitrate": 8500000,
  "videoCodec": "h265",
  "audioCodec": "aac",
  "frameRate": 30.0,
  "createdAt": "2024-01-15T10:30:00Z",
  "modifiedAt": "2024-01-15T10:30:05Z",
  "deviceMake": "Apple",
  "deviceModel": "iPhone 15 Pro",
  "software": "iOS 17.2",
  "gpsLatitude": 37.7749,
  "gpsLongitude": -122.4194,
  "gpsAltitude": 4.0,
  "isHDR": true,
  "colorSpace": "bt2020nc",
  "stabilization": "on",
  "videoEncoder": "HEVC",
  "pixelFormat": "yuv420p10le",
  "audioSampleRate": 48000,
  "audioChannels": 2,
  "title": "Birthday party"
}
```

### 6.3 API Response Impact

The existing media API endpoints will automatically include the new fields since `VideoMetadata` is already part of the `Media` struct and is serialized to JSON. No API changes needed — the fields will just appear as part of the existing `videoMetadata` response.

---

## 7. Integration Points

### 7.1 Scanner (`internal/scanner/media_scanner.go`)

**Integration:** When a video file is discovered during scanning, create a `JobTypeVideoMetadata` job.

```go
// In scan loop
if media.MediaType == domain.MediaTypeVideo {
    job := &domain.Job{
        ID:     uuid.New(),
        UserID: userID,
        Type:   domain.JobTypeVideoMetadata,
        Status: domain.JobStatusPending,
        MediaID: media.ID,
    }
    jobRepo.Create(ctx, job)
}
```

### 7.2 Worker (`cmd/worker/main.go`)

**Integration:** Handle `JobTypeVideoMetadata` in the job processing loop.

```go
case domain.JobTypeVideoMetadata:
    extractor := processor.NewVideoMetadataExtractor()
    metadata, warnings, err := extractor.ExtractVideoMetadata(ctx, media.Path)
    if err != nil {
        return jobRepo.UpdateStatus(ctx, job.ID, domain.JobStatusFailed, err.Error())
    }
    media.VideoMetadata = *metadata
    return jobRepo.UpdateStatus(ctx, job.ID, domain.JobStatusCompleted, "")
```

### 7.3 EXIF Updater CLI (`cmd/exif/main.go`)

**Integration:** Add video format branch to the existing EXIF extraction logic.

```go
// In main.go processing loop
if media.MediaType == domain.MediaTypeVideo {
    extractor := processor.NewVideoMetadataExtractor()
    metadata, warnings, err := extractor.ExtractVideoMetadata(ctx, fullPath)
    if err != nil {
        log.Printf("  Failed to extract video metadata: %v", err)
        errors++
        continue
    }
    media.VideoMetadata = *metadata
}
```

### 7.4 Upload Handler (`internal/api/upload_handler.go`)

**Integration:** Optionally create video metadata job at upload time (deferred processing).

```go
// In upload handler, after file is saved
if strings.HasSuffix(strings.ToLower(filename), ".mp4") || 
   strings.HasSuffix(strings.ToLower(filename), ".mov") ||
   strings.HasSuffix(strings.ToLower(filename), ".m4v") {
    job := &domain.Job{
        ID:     uuid.New(),
        UserID: userID,
        Type:   domain.JobTypeVideoMetadata,
        Status: domain.JobStatusPending,
        MediaID: media.ID,
    }
    jobRepo.Create(ctx, job)
}
```

---

## 8. Testing Strategy

### 8.1 Unit Tests

| Test Name | Input | Expected Output |
|-----------|-------|-----------------|
| `TestDetectVideoFormatMP4` | MP4 file magic bytes | `VideoFormatMP4` |
| `TestDetectVideoFormatMOV` | MOV file magic bytes | `VideoFormatMOV` |
| `TestDetectVideoFormatM4V` | M4V file magic bytes | `VideoFormatM4V` |
| `TestDetectVideoFormatInvalid` | Invalid file | `VideoFormatUnknown`, error |
| `TestFFProbeExtractorProbe` | Real video file | Populated `FFProbeResult` |
| `TestFFProbeExtractorNotAvailable` | No ffprobe | `IsAvailable()` returns false |
| `TestQTMetadataParserParse` | iPhone video | Device, GPS, creation date populated |
| `TestQTMetadataParserNoMDTA` | Video without metadata | Empty `QuickTimeMetadata` |
| `TestVideoMetadataExtractorComplete` | Full iPhone video | All fields populated |
| `TestVideoMetadataExtractorFFprobeMissing` | No ffprobe | Warnings, partial data |
| `TestVideoMetadataExtractorCorruptFile` | Corrupt MP4 | Warnings, partial data, no panic |

### 8.2 Integration Tests

1. Insert a media record into test DB with a real test video (iPhone MOV with EXIF)
2. Run `ExtractVideoMetadata()` against it
3. Call `mediaRepo.Update()`
4. Query DB and verify `video_metadata` JSONB field contains expected fields
5. Verify `device_make`, `device_model`, `gps_latitude`, `gps_longitude`, `created_at`
6. Run worker with `JobTypeVideoMetadata` job and verify completion

### 8.3 Test Data Requirements

| Test File | Format | Metadata Content |
|-----------|--------|-----------------|
| `test_iPhone_video.mov` | MOV | Device, GPS, dates, camera settings |
| `test_android_video.mp4` | MP4 | Device, GPS, codec info |
| `test_video_no_metadata.mp4` | MP4 | Empty metadata (no mdta box) |
| `test_video_hdr.mp4` | MP4 | HDR flags, color transfer |
| `test_video_corrupt.mp4` | MP4 | Truncated (for error handling tests) |

---

## 9. Dependencies & Installation

### 9.1 System Dependencies

**FFmpeg/ffprobe (Required):**

**Ubuntu/Debian:**
```bash
sudo apt-get update
sudo apt-get install -y ffmpeg
```

**macOS (Homebrew):**
```bash
brew install ffmpeg
```

**Alpine Linux (Docker):**
```dockerfile
RUN apk add --no-cache ffmpeg
```

**Verify Installation:**
```bash
ffprobe -version
ffprobe -v quiet -print_format json -show_format -show_streams /path/to/video.mp4
```

### 9.2 Go Dependencies

**No new Go dependencies required!** Both ffprobe (exec) and QuickTime atom parsing (pure Go) use only standard library packages.

**Verify current dependencies:**
```bash
go list -m all
```

Expected (no changes):
- `github.com/google/uuid` — already in go.mod
- `github.com/jmoiron/sqlx` — already in go.mod
- `github.com/lib/pq` — already in go.mod

### 9.3 Docker Considerations

**Dockerfile Changes:**

```dockerfile
# FROM alpine:3.19 (existing)

# Add FFmpeg (required for ffprobe)
RUN apk add --no-cache ffmpeg

# ... rest of Dockerfile
```

**Alternative (if you want to avoid FFmpeg in production):**
- Use the pure Go QuickTime parser only (Phase 3)
- ffprobe becomes optional enhancement
- Trade-off: missing stream-level data (resolution, codecs, bitrate)

---

## 10. Deployment Notes

### 10.1 Pre-deployment Checklist

- [ ] FFmpeg/ffprobe installed on all hosts
- [ ] `ffprobe -version` returns successfully
- [ ] Docker image updated with FFmpeg (if applicable)
- [ ] Worker configured to process `JobTypeVideoMetadata` jobs
- [ ] EXIF CLI updated to handle video files

### 10.2 Migration Strategy

**One-time backfill for existing videos:**

```bash
# Build the EXIF CLI with video support
go build -o bin/exif ./cmd/exif/

# Dry run first
./bin/exif -dry-run

# Full backfill
./bin/exif -force

# User-scoped backfill
./bin/exif -user admin@steadyphoto.com -force
```

**Ongoing processing:**

The worker will automatically process new video uploads via the `JobTypeVideoMetadata` job queue. No manual intervention needed.

### 10.3 Monitoring & Alerts

**Key Metrics:**
- Number of videos with populated `video_metadata`
- Number of failed video metadata extractions
- ffprobe availability status

**Alerts:**
- ffprobe not found on host
- >10% video metadata extraction failures
- Video metadata extraction taking >5 minutes per file

### 10.4 Rollback Plan

If issues arise:

1. **Disable video metadata jobs:**
   ```go
   // In worker, skip JobTypeVideoMetadata
   case domain.JobTypeVideoMetadata:
       // Skip - video metadata extraction disabled
       return jobRepo.UpdateStatus(ctx, job.ID, domain.JobStatusCompleted, "disabled")
   ```

2. **Revert domain model changes:**
   - Remove new fields from `VideoMetadata` struct
   - Run migration to drop JSONB fields (if any schema changes were made)

3. **Restore Docker image:**
   - Revert to previous Docker image without FFmpeg

---

## Appendix A: QuickTime Atom Reference

### A.1 Common Atoms

| Atom | Path | Description |
|------|------|-------------|
| `ftyp` | root | File type (major brand, compatible brands) |
| `moov` | root | Movie metadata container |
| `mvhd` | moov | Movie header (timescale, duration, creation/modification times) |
| `trak` | moov | Track (video or audio) |
| `tkhd` | moov/trak | Track header (width, height, duration) |
| `mdia` | moov/trak | Media information container |
| `mdhd` | moov/trak/mdia | Media header (duration, timescale) |
| `minf` | moov/trak/mdia | Media information container |
| `stbl` | moov/trak/mdia/minf | Sample table |
| `stsd` | moov/trak/mdia/minf/stbl | Sample description (codec info) |
| `udta` | root | User data (contains metadata) |
| `meta` | udta | Metadata container |
| `mdta` | meta | Metadata dictionary (key-value pairs) |

### A.2 QuickTime Time Format

```
QuickTime time = seconds since January 1, 1904 00:00:00 UTC
Unix time = seconds since January 1, 1970 00:00:00 UTC

Conversion:
    unixTime = qtTime - 2082844800
    qtTime = unixTime + 2082844800
```

### A.3 ISO 6709 GPS Format

```
Format: ±lat±lon±alt/
Example: +37.7749-122.4194+004.000/

Components:
- lat: ±DD.DDDD (positive = N, negative = S)
- lon: ±DDD.DDDD (positive = E, negative = W)
- alt: ±DDD.DDD (meters above sea level, optional)
```

---

## Appendix B: ffprobe Field Mapping

### B.1 Stream Properties to VideoMetadata

| ffprobe Field | VideoMetadata Field | Notes |
|--------------|---------------------|-------|
| `format.duration` | `Duration` | Seconds |
| `format.bit_rate` | `Bitrate` | Bits per second |
| `format.creation_time` | `CreatedAt` | ISO 8601 string |
| `format.modification_time` | `ModifiedAt` | ISO 8601 string |
| `streams[].width` | `Width` | Video stream only |
| `streams[].height` | `Height` | Video stream only |
| `streams[].codec_name` | `VideoCodec` / `AudioCodec` | Depends on stream type |
| `streams[].r_frame_rate` | `FrameRate` | Parse "30/1" → 30.0 |
| `streams[].pix_fmt` | `PixelFormat` | e.g., "yuv420p" |
| `streams[].color_space` | `ColorSpace` | e.g., "bt709" |
| `streams[].color_transfer` | `IsHDR` | "smpte2084" → HDR |
| `streams[].sample_rate` | `AudioSampleRate` | Audio stream only |
| `streams[].channels` | `AudioChannels` | Audio stream only |

### B.2 QuickTime Metadata Keys to VideoMetadata

| Qt Key | VideoMetadata Field | Notes |
|--------|---------------------|-------|
| `com.apple.quicktime.model` | `DeviceModel` | "iPhone 15 Pro" |
| `com.apple.quicktime.make` | `DeviceMake` | "Apple" |
| `com.apple.quicktime.software` | `Software` | "iOS 17.2" |
| `com.apple.quicktime.camera` | `CameraType` | "Rear", "Front" |
| `com.apple.quicktime.creationdate` | `CreatedAt` | "2024:01:15 10:30:00" |
| `com.apple.quicktime.location.ISO6709` | `GPSLatitude`, `GPSLongitude`, `GPSAltitude` | Parse ISO 6709 |
| `com.apple.quicktime.stabilization` | `Stabilization` | "0x01" |
| `com.apple.photo.capture.video.stabilization.mode` | `Stabilization` | "on" |
| `com.apple.quicktime.video.encoder` | `VideoEncoder` | "HEVC" |
| `com.apple.quicktime.video.level` | `VideoLevel` | "62" |
| `com.apple.quicktime.color.primarys` | `ColorSpace` | "bt2020" |
| `com.apple.quicktime.transfer.function` | `IsHDR` | "smpte2084" → true |
| `com.apple.iPhoto.caption` | `Caption` | "Birthday party" |

---

## Appendix C: Implementation Checklist

### Phase 1: Video Format Detection
- [ ] Create `internal/processor/video_format.go`
- [ ] Implement `DetectVideoFormat()` function
- [ ] Add magic byte constants
- [ ] Write unit tests
- [ ] Verify with real video files

### Phase 2: ffprobe Wrapper
- [ ] Create `internal/processor/ffprobe.go`
- [ ] Implement `FFProbeExtractor` struct
- [ ] Implement `ProbeStreamProperties()` method
- [ ] Implement `ProbeDuration()` method
- [ ] Implement `IsAvailable()` method
- [ ] Add FFmpeg installation instructions to Dockerfile
- [ ] Write unit tests

### Phase 3: QuickTime Metadata Parser
- [ ] Create `internal/processor/qt_atoms.go`
- [ ] Implement atom parsing logic
- [ ] Create `internal/processor/qt_metadata.go`
- [ ] Implement `QTMetadataParser` struct
- [ ] Implement `Parse()` method
- [ ] Implement ISO 6709 GPS parsing
- [ ] Implement QuickTime time format conversion
- [ ] Write unit tests

### Phase 4: Video Metadata Extractor
- [ ] Create `internal/processor/video_metadata_extractor.go`
- [ ] Implement `VideoMetadataExtractor` struct
- [ ] Implement `ExtractVideoMetadata()` method
- [ ] Unify ffprobe and Qt parser results
- [ ] Handle ffprobe unavailability gracefully
- [ ] Write unit tests

### Phase 5: Domain Model Extensions
- [ ] Update `internal/domain/media.go`
- [ ] Add new fields to `VideoMetadata` struct
- [ ] Add JSON tags for new fields
- [ ] Update `Value()` and `Scan()` methods if needed
- [ ] Write tests

### Phase 6: Worker Integration
- [ ] Update `internal/domain/job.go`
- [ ] Add `JobTypeVideoMetadata` constant
- [ ] Update `cmd/worker/main.go`
- [ ] Handle new job type in processing loop
- [ ] Write integration tests

### Phase 7: EXIF CLI Integration
- [ ] Update `cmd/exif/main.go`
- [ ] Add video format branch to processing logic
- [ ] Handle video metadata extraction errors
- [ ] Write tests

### Phase 8: Testing
- [ ] Create test video files (iPhone, Android, HDR, no metadata)
- [ ] Write unit tests for all new components
- [ ] Write integration tests
- [ ] Run full test suite
- [ ] Performance testing with large video files

---

## Appendix D: References

- [FFmpeg ffprobe Documentation](https://ffmpeg.org/ffprobe.html)
- [ISO 14496-12 (MP4 Container Specification)](https://developer.apple.com/library/archive/documentation/QuickTime/QTFF/QTFFPreface/qtffPreface.html)
- [Apple QuickTime File Format Specification](https://developer.apple.com/library/archive/documentation/QuickTime/QTFF/QTFFPreface/qtffPreface.html)
- [ISO 6709 Standard](https://en.wikipedia.org/wiki/ISO_6709)
- [QuickTime Time Format](https://developer.apple.com/library/archive/documentation/QuickTime/QTFF/QTFFChap2/qtff2.html)

---

**Last Updated:** July 10, 2026  
**Status:** 🟡 In Progress — Awaiting FFmpeg installation  
**Next Steps:**
1. ✅ Install FFmpeg/ffprobe on host
2. ⏳ Begin Phase 1 implementation
3. ⏳ Test with sample video files
4. ⏳ Implement remaining phases per checklist
