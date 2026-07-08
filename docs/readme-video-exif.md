# Phone Video EXIF Data Extraction — Status & Handoff

**Last Updated:** July 10, 2026
**Current Status:** Phase 2 Complete, Phase 3 In Progress
**Next Agent Task:** Fix QuickTime parser bug, then complete Phase 3 + remaining phases

---

## Quick Status Summary

| Phase | Description | Status | Files |
|-------|------------|--------|-------|
| Phase 1 | Video Format Detection | ✅ Complete | `video_format.go` (130 lines, 4 funcs) |
| Phase 2 | FFProbe Wrapper | ✅ Complete | `ffprobe.go` (322 lines, 14 funcs) |
| Phase 3 | QuickTime Metadata Parser | 🟡 In Progress — BUG to fix | `qt_atoms.go` (128 lines) + `qt_parser.go` (868 lines, 29 funcs) + `qt_parser_test.go` (831 lines) |
| Phase 4 | Combined Extraction | ❌ Not Started | — |
| Phase 5 | Integration | ❌ Not Started | — |
| Phase 6 | Real-World Testing | ❌ Not Started | — |

**Test Results (current):**
```
=== PASS: TestExifReader (3 tests)
=== PASS: TestConvertImageFormat (11 subtests)
=== PASS: TestExtractIntValue (8 subtests)
=== PASS: TestDetectImageFormat (6 subtests)
=== PASS: TestFaceDetection (2 tests)
=== PASS: TestFFProbe (all 60+ subtests)
=== FAIL: TestParseQuickTimeFile_SimpleMP4  ← BUG — see below
=== RUN:   ~30 more Qt parser tests (will fail after SimpleMP4)
```

---

## CRITICAL BUG — Fix Before Continuing

**File:** `internal/processor/qt_parser.go`, line 285-289
**Error:** `panic: runtime error: index out of range [3] with length 3`
**Root Cause:** `binary.BigEndian.Uint32(data[1:4])` reads 4 bytes from a slice that may only have 3 bytes. The check `dataSize >= 4` is wrong — to read `data[1:5]` you need 5 total bytes.

```go
// CURRENT (buggy) — line 285-289:
if dataSize >= 4 && !isContainerAtom(atomType) {
    version = data[0]
    flags = binary.BigEndian.Uint32(data[1:4]) & 0x00FFFFFF
    payloadStart = 4
}

// FIX:
if dataSize >= 5 && !isContainerAtom(atomType) {
    version = data[0]
    flags = binary.BigEndian.Uint32(data[1:5]) & 0x00FFFFFF
    payloadStart = 5
}
```

**Why this matters:** When an atom has exactly 3 bytes of data (size 11: 8 header + 3 data), `dataSize >= 4` is false so the bug is avoided. But `isContainerAtom` returns `true` for `AtomMvhd` (which is a container), so when parsing the `mvhd` child atom inside `moov`, it enters the container branch and doesn't hit this code. The panic actually occurs on the **ftyp** atom which has only 3 bytes of payload data (major_brand=4 + minor_version=4 — wait, that's 8 bytes). Let me re-examine: the ftyp atom in the test has 12 bytes of data, and `data[1:4]` needs indices 1,2,3,4 which means 5 total bytes — 12 >= 5 so it shouldn't fail there either. The actual panic happens because the `moov` atom's `mvhd` child has 100 bytes of data but the issue is in parsing the **ftyp** atom which has `dataSize = 12` but the check `dataSize >= 4` is true. Then `binary.BigEndian.Uint32(data[1:4])` reads bytes at indices 1,2,3,4 — wait, `data[1:4]` is a 3-byte slice. **`binary.BigEndian.Uint32` requires exactly 4 bytes.** `data[1:4]` gives bytes at indices 1,2,3 — only 3 bytes. This will always panic when `dataSize >= 5` because `data[1:4]` is always 3 bytes (indices 1,2,3). **The bug is actually: `data[1:4]` must be `data[1:5]`.**

This is a fundamental bug — `binary.BigEndian.Uint32` always requires 4 bytes, and `data[1:4]` is a 3-byte slice. The correct code is `data[1:5]`. This affects ALL non-container atoms with ≥5 bytes of data (ftyp, mvhd, tkhd, mdhd, hdlr, stsd, etc.).

**Fix it, then run all Qt tests — they should mostly pass after this single change.**

---

## Phase 3 — QuickTime Metadata Parser (In Progress)

### What's Been Built

#### `internal/processor/qt_atoms.go` (128 lines)
- All QuickTime atom type constants (`AtomFTyp`, `AtomMoov`, `AtomTrak`, `AtomMdia`, `AtomMinf`, `AtomStbl`, `AtomStsd`, `AtomMvhd`, `AtomTkhd`, `AtomMdhd`, `AtomHdlr`, `AtomFree`, `AtomMdat`, `AtomMeta`, `AtomMdta`, `AtomKeys`, `AtomData`, `AtomDref`, `AtomEdts`, `AtomIpco`, `AtomMvex`, `AtomAvcC`, `AtomHvcC`, `AtomEsds`)
- Reference to ISO 14496-12 spec

#### `internal/processor/qt_parser.go` (868 lines, 29 functions)
**Type definitions:**
- `QuickTimeAtom` — generic atom/box representation
- `QuickTimeFile` — top-level container (FTyp + Moov)
- `FTypBox` — file type (major brand, minor version, compatible brands)
- `MoovBox` — movie metadata (Mvhd, Traks, Meta)
- `MovieHeaderAtom` — mvhd (creation time, time scale, duration)
- `TrackBox` — trak (Tkhd, Mdia)
- `TrackHeaderAtom` — tkhd (version, flags, enabled, width, height, matrix)
- `MediaBox` — mdia (Mdhd, Hdlr, Minf)
- `MediaHeaderAtom` — mdhd (creation time, time scale, duration, language)
- `HandlerBox` — hdlr (handler type, description)
- `MediaInformationBox` — minf (Stbl)
- `SampleTableBox` — stbl (Stsd)
- `SampleDescriptionBox` — stsd (codec info)
- `CodecConfig` — codec config (avcC, hvcC, av1C, esds)
- `MetaBox` — meta (mdta)
- `MdtoBox` — mdta (key-value metadata items)
- `PhoneMetadata` — extracted phone metadata (device, GPS, stabilization, HDR, etc.)

**Core parser functions:**
- `ParseQuickTimeFile(filePath)` — parse from file path
- `ParseQuickTimeFromReader(r)` — parse from io.Reader
- `parseAtoms(r, depth)` — read atoms at a nesting level (max depth 20)
- `parseAtom(r, depth)` — read a single atom with 8-byte or 64-bit extended header
- `isContainerAtom(type)` — check if atom has children
- `parseFTypBox(atom)` — parse file type
- `parseMoovBox(atom)` — parse movie metadata
- `parseMovieHeaderAtom(atom)` — parse mvhd (v0 and v1)
- `parseTrackBox(atom)` — parse track container
- `parseTrackHeaderAtom(atom)` — parse tkhd (v0 and v1)
- `parseMediaBox(atom)` — parse media container
- `parseMediaHeaderAtom(atom)` — parse mdhd (v0 and v1)
- `parseHandlerBox(atom)` — parse hdlr
- `parseMediaInformationBox(atom)` — parse minf
- `parseSampleTableBox(atom)` — parse stbl
- `parseSampleDescriptionBox(atom)` — parse stsd
- `parseCodecConfigs(data, descType)` — find codec configs (avcC, hvcC, av1C, esds)
- `parseMetaBox(atom)` — parse meta
- `parseMdtoBox(atom)` — parse mdta
- `parseKeysAtom(atom)` — parse keys
- `parseDataAtom(atom)` — parse data entries (stub)

**Extraction functions:**
- `ExtractPhoneMetadata(filePath)` — high-level extraction
- `extractPhoneMetadataFromContainer(qtf)` — extract from parsed container
- `extractFromMdtoBox(mdta, metadata)` — extract Apple metadata keys
- `parseGPSCoordinates(location, metadata)` — parse ISO 6709
- `splitLocation(location)` — split ISO 6709 string
- `parseCoordinate(s)` — parse coordinate float

**Helpers:**
- `bytesReader` — io.Reader wrapper for byte slices

**Key mapping (mdta keys → PhoneMetadata fields):**
| mdta Key | PhoneMetadata Field |
|----------|-------------------|
| `mdta.1001`, `mdta.2000` | DeviceModel |
| `mdta.1002` | DeviceSoftware |
| `mdta.1000` | CreationDate |
| `mdta.3004` | GPS (ISO 6709) |
| `mdta.3007` | Stabilization (bool) |
| `mdta.3008` | VideoEncoder |
| `mdta.4016` | AudioCodec |

#### `internal/processor/qt_parser_test.go` (831 lines, ~30 tests)
- `TestParseQuickTimeFile_SimpleMP4` — minimal ftyp + moov/mvhd
- `TestParseQuickTimeFile_TrackWithMedia` — full trak/mdia/hdlr/stbl tree
- `TestParseQuickTimeFile_64BitAtom` — 64-bit extended size
- `TestParseQuickTimeFile_EmptyMoov` — error when moov missing
- `TestParseQuickTimeFile_DeepNesting` — depth limit enforcement
- `TestParseMvhdVersion0/Version1` — mvhd v0 and v1
- `TestParseTkhdVersion0` — track header (width/height fixed-point)
- `TestParseMdhdVersion0` — media header
- `TestParseHdlr` — handler type
- `TestParseFTyp` / `TestParseFTyp_AppleBrand` — file type
- `TestExtractPhoneMetadata_MvhdCreationDate` — timestamp conversion (QuickTime 1904 → Unix 1970)
- `TestParseGPSCoordinates_*` — GPS coordinate parsing (SF, NY, London, with altitude)
- `TestSplitLocation` — ISO 6709 splitter
- `TestExtractFromMdtoBox*` — mdta key extraction
- `TestParseAtom_EmptyData/TruncatedData` — edge cases
- `TestBytesReader` — bytesReader helper
- `TestIsContainerAtom` / `TestFindAtomType` — utility functions
- `TestParseQuickTimeFile_NonExistentFile/EmptyReader/JustFtyp` — error cases
- `TestExtractPhoneMetadata_EmptyContainer/WithMdtoBox` — extraction
- `TestParseQuickTimeFile_RealTestVideo/RealHDRTVideo/AllTestVideos` — real file tests (uses testdata/)

### Known Limitations in Current Implementation
1. **`parseDataAtom` is a stub** — Apple metadata data atom parsing not fully implemented
2. **`parseMdtoBox` is simplified** — key-to-value matching is not correct (real implementation needs keys + data alignment)
3. **GPS coordinate parser is basic** — handles standard ISO 6709 format but edge cases may exist
4. **No udta box parsing** — some phone metadata is in udta, not meta/mdta
5. **No iprp/ipco parsing** — newer iOS metadata format not handled
6. **`parseCodecConfigs` is simplistic** — finds atom type strings but doesn't fully decode codec config

---

## Phase 4 — Combined Extraction (Not Started)

### What's Needed
Create a unified extractor that combines ffprobe + QuickTime parser output into a single `VideoMetadata` domain model.

### Design
```go
type VideoMetadataExtractor struct {
    ffprobePath string  // path to ffprobe binary
}

func NewVideoMetadataExtractor() *VideoMetadataExtractor

func (e *VideoMetadataExtractor) ExtractVideoMetadata(ctx context.Context, filePath string) (*VideoMetadata, error)
```

### Merged Data Model
The `VideoMetadata` struct needs to be populated from both sources:

| Source | Fields Populated |
|--------|-----------------|
| **ffprobe** | Duration, Width, Height, FrameRate, BitRate, VideoCodec, AudioCodec, AudioSampleRate, AudioChannels, PixelFormat, ColorSpace, CreationTime (from format tags) |
| **Qt Parser** | DeviceMake, DeviceModel, DeviceSoftware, GPS coordinates, Stabilization, VideoEncoder, VideoLevel, HDR, Caption, Title, CreationDate (from mvhd/mdta) |

### Implementation Plan
1. Create `internal/processor/video_extractor.go`
2. Implement `ExtractVideoMetadata(ctx, filePath)` that:
   - Calls `ffprobe.ExtractStreamProperties(filePath)` for stream data
   - Calls `ExtractPhoneMetadata(filePath)` for container data
   - Merges into single `VideoMetadata` struct
   - Handles conflicts (e.g., both sources report creation time — Qt parser takes priority if available)
3. Write tests for the combined extraction
4. Handle edge cases: missing ffprobe, corrupted files, unusual codecs

---

## Phase 5 — Integration (Not Started)

### What's Needed
Wire the extraction into the existing codebase:
1. Update `cmd/worker` to call the new extractor for video files
2. Update `cmd/exif` CLI to support video files
3. Update upload handler to extract metadata at upload time
4. Update DB model if `VideoMetadata` struct doesn't match existing schema

### Key Integration Points
- **Worker (`cmd/worker`):** Find the video processing job handler and add extraction step
- **EXIF CLI (`cmd/exif`):** Add video file scanning alongside image EXIF
- **Upload handler:** Extract metadata synchronously or queue as background job

### Files to Modify
- `cmd/worker/main.go` or worker job handlers
- `cmd/exif/main.go`
- Any domain model files for `VideoMetadata`
- Upload handler code

---

## Phase 6 — Real-World Testing (Not Started)

### What's Needed
1. Test with real iPhone videos (MOV/M4V) — need Apple-specific test data
2. Test with real Android videos (MP4) — need Android-specific test data
3. Test with HDR videos (Dolby Vision, HDR10)
4. Test with 4K/8K videos
5. Test with videos that have no metadata
6. Performance benchmarking

### Test Data
- `testdata/ten_second.mp4` — basic test video (may exist from Phase 2)
- `testdata/hdr_like.mp4` — HDR test video (may exist)
- Real phone videos would need to be obtained from actual devices

---

## File Inventory

| File | Lines | Functions | Purpose |
|------|-------|-----------|---------|
| `internal/processor/exif_reader.go` | 527 | 10 | Image EXIF extraction (Phase 1 — complete) |
| `internal/processor/exif_reader_test.go` | — | — | Image EXIF tests |
| `internal/processor/face_detection.go` | — | — | Face detection processor |
| `internal/processor/face_detection_test.go` | — | — | Face detection tests |
| `internal/processor/ffprobe.go` | 322 | 14 | FFProbe wrapper for stream properties (Phase 2 — complete) |
| `internal/processor/ffprobe_test.go` | — | — | FFProbe tests (all passing) |
| `internal/processor/video_format.go` | 130 | 4 | Video format detection (Phase 1 — complete) |
| `internal/processor/video_format_test.go` | — | — | Format detection tests |
| `internal/processor/qt_atoms.go` | 128 | 0 | QuickTime atom type constants (Phase 3 — complete) |
| `internal/processor/qt_parser.go` | 868 | 29 | QuickTime atom parser (Phase 3 — has known bug) |
| `internal/processor/qt_parser_test.go` | 831 | ~30 | QuickTime parser tests (has known bug) |
| `internal/processor/geocode.go` | — | — | Geocoding utility |
| `internal/processor/image_engine.go` | — | — | Image processing engine |
| `internal/processor/standard_engine.go` | — | — | Standard engine |
| `internal/processor/thumbnail.go` | — | — | Thumbnail generation |
| `internal/processor/mock_engine_test.go` | — | — | Mock engine tests |

**Total Phase 3 code:** ~1,927 lines (qt_atoms.go + qt_parser.go + qt_parser_test.go)
**Total processor package:** ~2,806 lines across the 6 main files

---

## Architecture Reference

```
┌──────────────────────────────────────────────────────────────────┐
│                    VideoMetadataExtractor                         │
│                                                                    │
│  Phase 1: Format Detection (✅ done)                              │
│  ┌─────────────────────┐                                          │
│  │ VideoFormat (enum)  │  MP4, MOV, M4V, AVI, WebM               │
│  │ DetectFormat(path)  │                                          │
│  └─────────────────────┘                                          │
│                          │                                        │
│  Phase 2: FFProbe (✅ done)                                       │
│  ┌─────────────────────┐                                          │
│  │ FFProbeExtractor    │                                          │
│  │ ProbeStreamProps()  │  duration, width, height, fps,           │
│  │ ProbeVideoStream()  │  bitrate, codecs, pixel format           │
│  │ ProbeCreationTime() │                                          │
│  └─────────────────────┘                                          │
│                          │                                        │
│  Phase 3: Qt Parser (🟡 in progress — BUG)                        │
│  ┌─────────────────────┐                                          │
│  │ ParseQuickTimeFile  │  Parse MP4/MOV binary structure          │
│  │ ExtractPhoneMeta()  │  device, GPS, stabilization, HDR, etc.   │
│  └─────────────────────┘                                          │
│                          │                                        │
│  Phase 4: Combined (❌ TODO)                                      │
│  ┌─────────────────────┐                                          │
│  │ Combine ffprobe + Qt│  Single VideoMetadata struct             │
│  └─────────────────────┘                                          │
│                          │                                        │
│  Phase 5: Integration (❌ TODO)                                   │
│  ┌─────────────────────┐                                          │
│  │ Wire into worker/CLI│  cmd/worker, cmd/exif, upload handler    │
│  └─────────────────────┘                                          │
│                          │                                        │
│  Phase 6: Real Testing (❌ TODO)                                  │
│  ┌─────────────────────┐                                          │
│  │ Real phone videos   │  iPhone MOV, Android MP4, HDR, 4K/8K     │
│  └─────────────────────┘                                          │
└──────────────────────────────────────────────────────────────────┘
```

---

## Dependencies

| Dependency | Purpose | Status |
|-----------|---------|--------|
| FFmpeg (system binary) | ffprobe for stream properties | ✅ Installed at `/usr/bin/ffprobe` |
| Go 1.x | Language | ✅ |
| No third-party Go deps for Qt parser | Pure Go implementation | ✅ |

---

## Environment

- **OS:** Ubuntu (Linux)
- **FFmpeg/ffprobe:** Installed at `/usr/bin/ffprobe`
- **Go:** Available
- **Test data directory:** `./testdata/` (may contain `ten_second.mp4`, `hdr_like.mp4`)

---

## Recommended Order of Operations

1. **FIX THE BUG** — `qt_parser.go` line 285-289 (change `data[1:4]` to `data[1:5]`, `dataSize >= 4` to `dataSize >= 5`, `payloadStart = 4` to `payloadStart = 5`)
2. **Run all Qt parser tests** — Most should pass after the fix
3. **Fix any remaining test failures** — May need to adjust `parseAtom` for edge cases
4. **Add missing metadata key mappings** — Some phone-specific keys not yet mapped (camera type, frame rate, color primaries, transfer function, HDR info, audio layout, title, caption)
5. **Implement Phase 4** — Combined extraction with `video_extractor.go`
6. **Implement Phase 5** — Integration into worker, CLI, upload
7. **Implement Phase 6** — Real-world testing with actual phone videos

---

## Key Design Decisions Made

1. **Hybrid approach (ffprobe + Qt parser):** ffprobe handles stream-level properties it knows best; Qt parser handles phone-specific metadata ffprobe can't see
2. **Pure Go Qt parser:** No external dependencies for the binary parser; avoids needing MP4 parsing libraries
3. **Depth-limited nesting:** Max 20 levels to prevent infinite recursion on malformed files
4. **64-bit atom size support:** Required for large files
5. **Version 0/1 support:** Both 32-bit and 64-bit time formats in mvhd and tkhd
6. **QuickTime→Unix timestamp conversion:** QuickTime epoch is 1904, Unix is 1970 (offset 2082844800 seconds)
7. **ISO 6709 GPS parsing:** Apple stores GPS in ISO 6709 format (+DD.DDDD±DDD.DDDD/±alt/)
