# EXIF Extraction Feature

**Status:** 🚧 Planned  
**Created:** July 10, 2026  
**Related:** Upload pipeline (metadata extraction was stubbed at upload time), Worker pipeline (similar CLI pattern)

---

## 1. What This Feature Does

Extracts embedded metadata from photo and video files (EXIF for photos, container metadata for videos) and writes it back into the database so the media items have rich, search/displayable metadata — camera model, GPS coordinates, capture time, lens info, video duration, codecs, etc.

**Key distinction:** This is NOT run at upload time. It is a **standalone post-processing CLI** that runs against existing media already in the database. Think of it as a one-time or on-demand backfill tool for the metadata that the upload handler only stubbed (see `Documentation/readme-upload.md` section "⚠️ Metadata Extraction — Stubbed").

---

## 2. Existing Codebase Context

### Project Structure (Critical Path)

```
cmd/
  exif/main.go              ← NEW: This CLI binary
internal/
  processor/
    exif.go                 ← NEW: Extraction logic
    thumbnail.go            ← Existing pattern to follow (ThumbnailProcessor)
    face_detection.go       ← Existing pattern to follow (FaceDetectionProcessor)
  domain/
    media.go                ← Media struct with Metadata (JSONB) + VideoMetadata (JSONB)
    job.go                  ← Job type system (not used by this CLI, but related)
  database/
    media_repository.go     ← PostgresMediaRepository (already implements domain.MediaRepository)
    user_repository.go      ← PostgresUserRepository
go.mod                      ← imagemeta v0.15.0+ (new, direct)
```

### Existing Media Domain (`internal/domain/media.go`)

```go
type Metadata map[string]string   // JSONB in DB — photo EXIF tags

type VideoMetadata struct {         // JSONB in DB — video container properties
    Duration      float64   `json:"duration"`
    Width         int       `json:"width"`
    Height        int       `json:"height"`
    Bitrate       int64     `json:"bitrate"`
    VideoCodec    string    `json:"video_codec"`
    AudioCodec    string    `json:"audio_codec"`
    FrameRate     float64   `json:"frame_rate"`
    CreatedAt     time.Time `json:"created_at"`
    ModifiedAt    time.Time `json:"modified_at"`
}

type Media struct {
    ID            uuid.UUID     `db:"id"`
    Path          string        `db:"path"`            // relative path under storage root
    Filename      string        `db:"filename"`
    Hash          string        `db:"hash"`
    SizeBytes     int64         `db:"size_bytes"`
    Width         int           `db:"width"`           // may be overwritten by EXIF
    Height        int           `db:"height"`          // may be overwritten by EXIF
    CapturedAt    time.Time     `db:"captured_at"`     // may be overwritten by EXIF
    MediaType     MediaType     `db:"media_type"`      // "photo" or "video"
    Metadata      Metadata      `db:"metadata"`        // JSONB map[string]string
    VideoMetadata VideoMetadata `db:"video_metadata"`  // JSONB struct
    CreatedAt     time.Time     `db:"created_at"`
    UpdatedAt     time.Time     `db:"updated_at"`
    Tags          string        `db:"tags"`
    UserID        uuid.UUID     `db:"user_id"`
    ClientSource  ClientSource  `db:"client_source"`
    DeletedAt     *time.Time    `db:"deleted_at"`
}
```

**Important:** `Metadata` is `map[string]string` (JSONB). The existing thumbnail and face detection processors do NOT populate it. EXIF extraction is the first thing that fills this field.

### Existing CLI Pattern (Follow This)

`cmd/worker/main.go` is the best reference for the new CLI. It shows:

1. **Flag parsing** with `flag` package
2. **.env loading** via `joho/godotenv`
3. **DB connection** via `sqlx.Connect("postgres", dbURL)`
4. **Mode selection** based on flags (single item / user-scoped / global)
5. **Error handling** with structured `log.Printf` output

### Existing Database Repositories (`internal/database/`)

All repositories use `sqlx` and implement domain interfaces. Key ones:

| Repository | Interface | Key Methods for EXIF CLI |
|-----------|-----------|--------------------------|
| `PostgresMediaRepository` | `domain.MediaRepository` | `GetByID(ctx, id, userID)`, `List(ctx, limit, offset, userID)`, `Update(ctx, media)` |
| `PostgresUserRepository` | `domain.UserRepository` | `GetByEmail(ctx, email)` |

The `MediaRepository.List()` method takes `userID *uuid.UUID` — pass `nil` for all users. The `Update()` method writes the modified `Media` struct back.

---

## 3. CLI Specification

### Command

```bash
go run cmd/exif/main.go [flags]
```

### Flags

| Flag | Short | Type | Description | Required? |
|------|-------|------|-------------|-----------|
| `-media-id` | — | string | UUID of a single media item to process | No |
| `-user` | — | string | Email of a user — processes ALL media for that user | No |
| `-force` | — | bool | Re-extract EXIF even if metadata already exists | No |

### Behavior Matrix

| Flags Provided | Scope | Force Behavior | Skip Logic |
|----------------|-------|---------------|------------|
| *(none)* | **ALL** media across ALL users | N/A | Skip media where `metadata` is already populated (non-empty map) |
| `-media-id <id>` | Single media item | Re-extracts regardless | N/A |
| `-user <email>` | All media owned by that user | Re-extracts regardless | Skip media where `metadata` is already populated |
| `-media-id <id> -force` | Single media item | Always re-extracts | N/A |
| `-user <email> -force` | All media for user | Always re-extracts | N/A |

**Default behavior (no flags):** One-shot scan — iterate all media, extract EXIF only for items that have empty metadata. Does NOT create background jobs. Just processes and exits (similar to worker's `processAllJobs` one-shot mode).

**Important:** This CLI does NOT use the `Job` system. It processes synchronously and writes results directly. The Job system exists for thumbnail/face-detection processing; EXIF extraction is simpler and doesn't need async queueing.

---

## 4. Extraction Logic

### 4.1 Photo Extraction (JPEG, HEIC, etc.)

**Library:** `github.com/bep/imagemeta` — modern, actively maintained image metadata library

**Advantages over goexif:**
- Supports JPEG, TIFF, PNG, WebP, HEIF, HEIC, AVIF, DNG, CR2, NEF, ARW, PEF (broader format support)
- Reads EXIF + IPTC + XMP in one pass
- Built-in `GetDateTime()` returns `time.Time` directly (no manual string parsing)
- Built-in `GetLatLong()` returns `(lat, lon, error)` directly (no manual rational conversion)
- Timeout and tag size limits for safety
- Active maintenance by bep (Hugo author)

**Process:**
1. Open the file from disk at `{STORAGE_ROOT}/{media.Path}`
2. Call `imagemeta.Decode()` with callback to collect tags into `Tags` struct
3. Use `tags.GetDateTime()` for capture time (auto-detects DateTimeOriginal, DateTime, XMP DateTimeOriginal, etc.)
4. Use `tags.GetLatLong()` for GPS coordinates (auto-detects EXIF, falls back to XMP)
5. Iterate remaining tags for camera info, lens, etc.

### 4.2 Video Extraction (MP4, MOV, etc.)

**Library:** `github.com/nareln/gomp4` — pure Go MP4 container parser

**Advantages over ffprobe:**
- No external dependencies — no ffmpeg/ffprobe installation needed
- Pure Go implementation — works everywhere Go compiles
- Direct memory parsing — faster than subprocess spawning
- Cross-platform — no platform-specific binary issues
- Simpler error handling — Go errors vs. parsing JSON from CLI output

**Process:**
1. Open the file from disk at `{STORAGE_ROOT}/{media.Path}`
2. Parse MP4 structure with `gomp4.ReadMoov(file)`
3. Extract duration, creation_time from `moov.mvhd` (movie header)
4. Extract track duration, time scale from `moov.trak.mdia.mdhd`
5. Extract codec information from `moov.trak.mdia.minf.stbl.stsd`
6. Populate `VideoMetadata` struct from parsed data

**Fallback:** If `gomp4` returns an error (corrupt file or unsupported format), log a warning and skip video metadata. Do NOT fail the entire operation.

---

## 5. EXIF Tag → Metadata Key Mapping

### Photo Tags (bep/imagemeta)

**Note:** `bep/imagemeta` uses callback-based tag collection. Each tag is passed to `HandleTag` as a `TagInfo` struct with `Source`, `Tag`, `Namespace`, and `Value` fields.

| TagInfo.Tag | Metadata Key | Value Format | Notes |
|-------------|-------------|--------------|-------|
| DateTimeOriginal | `datetime_original` | RFC3339 string | **Also populates `Media.CapturedAt` via `GetDateTime()`** |
| DateTime | `datetime` | RFC3339 string | Modified time |
| DateTimeDigitized | `datetime_digitized` | RFC3339 string | When digitized |
| Make | `make` | string | e.g. "Apple", "Canon", "NIKON" |
| Model | `model` | string | e.g. "iPhone 15 Pro" |
| LensModel | `lens_model` | string | |
| FocalLength | `focal_length` | string (e.g. "85/1") | Millimeters |
| FNumber | `fnumber` | string (e.g. "2.8") | Aperture (now decimal, not rational) |
| ExposureTime | `exposure_time` | string (e.g. "1/250") | Shutter speed |
| ISOSpeedRatings | `iso` | string | ISO value |
| ExposureProgram | `exposure_program` | string | |
| LensMake | `lens_make` | string | |
| WhiteBalance | `white_balance` | string | |
| Flash | `flash` | string | |
| ColorSpace | `color_space` | string | |
| ImageWidth | `image_width` | string | |
| ImageLength | `image_length` | string | |
| Software | `software` | string | Processing software |
| Artist | `artist` | string | Photographer credit |
| ImageDescription | `image_description` | string | Caption/Description |
| UserComment | `user_comment` | string | |
| Orientation | `orientation` | string | |
| GPSLatitude | `gps_latitude` | decimal string | Use `tags.GetLatLong()` |
| GPSLongitude | `gps_longitude` | decimal string | Use `tags.GetLatLong()` |
| GPSAltitude | `gps_altitude` | string | Meters above sea level |
| GPSDateTime | `gps_datetime` | RFC3339 | GPS timestamp |
| GPSLatitudeRef | `gps_latitude_ref` | string | N/S |
| GPSLongitudeRef | `gps_longitude_ref` | string | E/W |

**Key difference from goexif:**
- Tag names are slightly different (e.g., `ExifExposureTime` → `ExposureTime`, `ExifImageWidth` → `ImageWidth`)
- Tag values are already typed as `any` — rational numbers are returned as `Rat` interface (call `.Float64()`)
- GPS coordinates should use `tags.GetLatLong()` instead of manual rational conversion
- DateTime should use `tags.GetDateTime()` instead of manual string parsing

### Video Tags (gomp4)

| gomp4 Field | VideoMetadata Key | Type | Source |
|-------------|-------------------|------|--------|
| `moov.Mvhd.Duration` (divided by time scale) | `duration` | float64 | Movie header duration |
| `moov.Mvhd.Timescale` | — | — | Used for duration calculation |
| `moov.Mvhd.CreationTime` | `created_at` | time.Time | Movie header creation time |
| `moov.Mvhd.ModificationTime` | `modified_at` | time.Time | Movie header modification time |
| `moov.Traks[0].Tkhd.Width` (divided by 65536) | `width` | int | Track header (fixed-point 16.16) |
| `moov.Traks[0].Tkhd.Height` (divided by 65536) | `height` | int | Track header (fixed-point 16.16) |
| `moov.Traks[0].Mdia.Minf.Stbl.Stsd.CodecName` | `video_codec` | string | Sample description |
| `moov.Traks[1].Mdia.Minf.Stbl.Stsd.CodecName` | `audio_codec` | string | Second track if audio |
| `moov.Traks[0].Mdia.Mdhd.Duration` | — | — | Track duration (for reference) |

**Key difference from ffprobe:**
- No JSON parsing needed — direct Go struct access
- Track dimensions are in fixed-point 16.16 format — divide by 65536 to get actual pixels
- Duration is in time scale units from `mvhd` — divide by timescale for seconds
- Creation/modification times are in Apple QuickTime time format (seconds since 1904-01-01)
- Codec information comes from `stsd` (sample description) boxes, not stream metadata

---

## 6. File Structure

### NEW: `cmd/exif/main.go`

```go
package main

// Flags:
//   -media-id <uuid>    → single media mode
//   -user <email>       → user-scoped mode
//   -force              → re-extract even if metadata exists
//   (none)              → global scan, skip items with existing metadata

// Flow:
// 1. Parse flags
// 2. Load .env from project root (joho/godotenv)
// 3. Connect to Postgres (sqlx)
// 4. Create ExifProcessor
// 5. Choose processing mode based on flags
// 6. For each media: call ExtractExif(media), update DB via mediaRepo.Update()
// 7. Print summary: processed, skipped, errors
```

### NEW: `internal/processor/exif.go`

```go
package processor

// ExifProcessor orchestrates EXIF extraction for photos and videos.
type ExifProcessor struct {
    storageRoot string
}

// NewExifProcessor creates a new ExifProcessor.
func NewExifProcessor(storageRoot string) *ExifProcessor

// ExtractExif reads metadata from a media file and returns
// a modified *domain.Media with populated Metadata and/or VideoMetadata.
// Does NOT write to DB — caller decides.
func (p *ExifProcessor) ExtractExif(ctx context.Context, media *domain.Media) error
```

### NEW: `internal/processor/exif_test.go`

Tests covering:
- Photo with full EXIF (camera, GPS, capture time)
- Photo with no EXIF (e.g. PNG without chunks)
- Photo with invalid format (e.g., .txt file)
- Video with gomp4 parsing
- Video with corrupt MP4 data
- Video with no audio stream
- File not found on disk
- InvalidFormatError handling

---

## 7. Implementation Steps (For the AI Developer)

### Step 1: Update `go.mod`

Add `bep/imagemeta` and `gomp4` as direct dependencies, remove `goexif`:
```bash
go get github.com/bep/imagemeta
go get github.com/nareln/gomp4
go mod tidy  # removes goexif if not used elsewhere
```

### Step 2: Create `internal/processor/exif.go`

- Implement `ExifProcessor` struct with `storageRoot` field
- Implement `ExtractExif()` method that:
  - Builds the absolute file path: `filepath.Join(p.storageRoot, media.Path)`
  - Determines if it's a photo or video based on `media.MediaType` and file extension
  - For photos: opens file, calls `imagemeta.Decode()` with callback, uses `tags.GetDateTime()` and `tags.GetLatLong()`, maps remaining tags to `Metadata` map
  - For videos: opens file, calls `gomp4.ReadMoov()`, extracts duration/resolution/codecs from atoms, populates `VideoMetadata`
  - Returns populated `*domain.Media` (caller writes to DB)
- Add proper error handling and logging

### Step 3: Create `cmd/exif/main.go`

- Parse CLI flags (`-media-id`, `-user`, `-force`)
- Load `.env` using `joho/godotenv`
- Connect to Postgres using `sqlx.Connect("postgres", dbURL)`
- Resolve user ID from email if `-user` flag is set
- Iterate over media items based on scope:
  - No flags → `mediaRepo.List(ctx, 10000, 0, nil)` (all users, all media)
  - `-user` → `mediaRepo.List(ctx, 10000, 0, &userID)`
  - `-media-id` → `mediaRepo.GetByID(ctx, id, nil)`
- For each item: check if metadata exists, if not (or force=true), call `ExtractExif()`, then `mediaRepo.Update(ctx, media)`
- Print summary

### Step 4: Create tests (`internal/processor/exif_test.go`)

### Step 5: Build and verify

```bash
go build ./cmd/exif/
./exif -media-id <uuid> -force
```

---

## 8. Storage Root Configuration

Same pattern as `cmd/worker/main.go`:

```go
storageRoot := getEnv("STORAGE_ROOT", "storage")
```

The `STORAGE_ROOT` env var is already used by the upload handler and worker. Files are stored at `{STORAGE_ROOT}/{user_id}/{YYYY/MM/DD}/{filename}`.

---

## 9. Error Handling Strategy

| Scenario | Behavior |
|----------|----------|
| File missing on disk | Log `[WARN] File not found: {path}`, skip, continue to next |
| File cannot be opened | Log `[WARN] Cannot open file: {path} - {err}`, skip |
| imagemeta returns no tags (e.g., PNG) | Log `[INFO] No EXIF data in: {path}`, skip (no DB update) |
| imagemeta returns InvalidFormatError | Log `[WARN] Unsupported format: {path}`, skip |
| gomp4 fails to parse MP4 | Log `[WARN] Corrupt video: {path} - {err}`, save whatever partial data was extracted |
| gomp4 returns InvalidFormatError | Log `[WARN] Not an MP4/MOV: {path}`, skip video extraction |
| DB update fails | Log `[ERROR] Failed to update media {id}: {err}`, continue to next |
| Invalid UUID in `-media-id` | `log.Fatal("invalid media-id: ...")` — exit immediately |

---

## 10. Logging Convention

Follow the existing pattern from `cmd/worker/main.go`:

```go
log.Printf("[USER MODE] Processing media for user: %s", email)
log.Printf("[FORCE MODE] Re-extracting metadata for media: %s", mediaID)
log.Printf("Found media: ID=%s Path=%s Type=%s UserID=%s", media.ID, media.Path, media.MediaType, media.UserID)
log.Printf("Extracted %d EXIF tags from: %s", len(tags), media.Path)
log.Printf("  GPS: lat=%s lon=%s", lat, lon)
log.Printf("  Captured: %s", capturedAt)
log.Printf("Updated media %s metadata in DB", media.ID)
log.Printf("Summary: processed=%d skipped=%d errors=%d", processed, skipped, errors)
```

---

## 11. Dependencies Summary

| Dependency | Status | Action Needed |
|-----------|--------|---------------|
| `github.com/bep/imagemeta` | 🆕 New | `go get github.com/bep/imagemeta` |
| `github.com/nareln/gomp4` | 🆕 New | `go get github.com/nareln/gomp4` |
| `github.com/rwcarlsen/goexif/exif` | ❌ Remove | `go mod tidy` removes it |
| `github.com/joho/godotenv` | Likely already present (used by worker) | Verify |
| `ffprobe` (external tool) | ❌ **Removed** | No longer needed |
| `github.com/google/uuid` | Already in `go.mod` | Already available |
| `github.com/jmoiron/sqlx` | Already in `go.mod` | Already available |
| `github.com/lib/pq` | Already in `go.mod` | Already available |

---

## 12. Docker Considerations

The production Docker image is based on `alpine:3.19` and does NOT include `ffmpeg`/`ffprobe`. **Neither is needed anymore** — both `imagemeta` and `gomp4` are 100% pure Go libraries with zero external dependencies.

```dockerfile
# No changes needed to Dockerfile.
# No ffmpeg, ffprobe, or any system packages required.
```

The EXIF CLI has **zero runtime dependencies** beyond standard Go libraries. No format detection, no external tools, no platform-specific binaries.

---

## 13. Testing Strategy

### Unit Tests (`internal/processor/exif_test.go`)

| Test Name | Input | Expected Output |
|-----------|-------|-----------------|
| `TestExtractExifPhotoFull` | JPEG with full EXIF (GPS, camera, time) | Metadata populated with all tags, CapturedAt set, GPS as decimal |
| `TestExtractExifPhotoNoData` | PNG without EXIF chunks | No error, Metadata remains nil/empty |
| `TestExtractExifPhotoCorrupt` | File with truncated EXIF | Partial data saved, no panic |
| `TestExtractExifPhotoInvalidFormat` | Invalid file extension (e.g., .txt) | InvalidFormatError, no panic |
| `TestExtractExifVideo` | MP4 file (gomp4 parsed) | VideoMetadata populated (duration, codec, resolution) |
| `TestExtractExifVideoCorrupt` | Corrupt MP4 file | Warning logged, VideoMetadata remains zero-value |
| `TestExtractExifVideoNoAudio` | MP4 with video only (no audio stream) | audio_codec is empty string, no crash |
| `TestExtractExifFileNotFound` | Path doesn't exist on disk | Returns error, caller can skip |

### Integration Test

1. Insert a media record into test DB with a real test photo (JPEG with EXIF)
2. Run `ExtractExif()` against it
3. Call `mediaRepo.Update()`
4. Query DB and verify `metadata` JSONB field contains expected EXIF tags
5. Verify `captured_at` was updated from EXIF DateTimeOriginal
6. Verify `gps_latitude` and `gps_longitude` are decimal strings (not rationals)

---

## 14. GPS Coordinate Extraction

`bep/imagemeta` handles GPS extraction automatically. Use `tags.GetLatLong()`:

```go
lat, lon, err := tags.GetLatLong()
if err == nil {
    media.Metadata["gps_latitude"] = fmt.Sprintf("%.6f", lat)
    media.Metadata["gps_longitude"] = fmt.Sprintf("%.6f", lon)
    media.Metadata["gps_latitude_ref"] = ... // from raw tags if needed
    media.Metadata["gps_longitude_ref"] = ...
}
```

Returns decimal degrees directly (no manual rational conversion). Falls back to XMP if EXIF GPS tags are missing.

For altitude: read `tags.All()["gps_altitude"]` from the collected tags.

---

## 15. DateTime Extraction

`bep/imagemeta` handles DateTime extraction automatically with `tags.GetDateTime()`:

```go
capturedAt, err := tags.GetDateTime()
if err == nil {
    media.CapturedAt = capturedAt
    media.Metadata["datetime_original"] = capturedAt.Format(time.RFC3339)
}
```

**How it works:**
1. Checks EXIF `DateTimeOriginal` first
2. Falls back to EXIF `DateTime` if no original
3. Then checks XMP `DateTimeOriginal`, `CreateDate`, `DateCreated`
4. Finally checks IPTC `DateCreated` + `TimeCreated`

Returns a `time.Time` directly — no manual string parsing needed. The order of precedence matches what photographers and archivists expect.

Also stores the original string in `Metadata["datetime_original"]` for reference.

---

## 16. What NOT to Change

- **Do NOT modify** `internal/domain/media.go` — the existing types are sufficient
- **Do NOT use** the Job system — this is synchronous processing
- **Do NOT modify** `cmd/worker/main.go` — keep it separate
- **Do NOT modify** the upload handler — this CLI backfills existing data only
- **Do NOT add** new database migrations — existing `metadata` and `video_metadata` columns are already there

---

## 17. Example Run Output

```
$ ./exif -user admin@steadyphoto.com -force
Loading .env from: /app
DB URL: postgres://****@localhost:5432/steadyphoto
Storage root: storage

[USER MODE] Processing media for user: admin@steadyphoto.com (ID: a1b2c3d4-...)
[FORCE MODE] Re-extracting metadata for all items

Found 50 media items for user

[1/50] Processing IMG_0001.jpg...
  EXIF: Make=Apple, Model=iPhone 15 Pro, ISO=32, FNumber=1.78, FocalLength=6mm
  GPS: lat=37.7749 lon=-122.4194
  Captured: 2024-06-15 14:23:01
  Updated 15 EXIF tags in DB

[2/50] Processing video_001.mp4...
  Video: duration=45.2s, width=1920, height=1080, codec=h264
  Updated VideoMetadata in DB

[3/50] Processing photo_no_exif.png...
  No EXIF data found, skipping

...

Summary: processed=42 skipped=7 errors=1
```

---

## 18. Related Files to Read Before Implementing

| File | Why |
|------|-----|
| `cmd/worker/main.go` | CLI pattern reference (flags, .env, DB, mode selection) |
| `internal/processor/thumbnail.go` | Processor pattern reference (struct, method signature) |
| `internal/domain/media.go` | Media struct, Metadata type, VideoMetadata struct |
| `internal/database/media_repository.go` | How to query and update media in DB |
| `internal/database/user_repository.go` | How to look up user by email |
| `Documentation/readme-upload.md` | Context: metadata extraction was stubbed at upload time |
| `go.mod` | Verify imagemeta is available |
