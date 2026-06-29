# Media Upload Feature

**Status:** ✅ Backend Completed — Multiple upload modes verified  
**Frontend UI Polish:** 🚧 Remaining work on mobile devices  
**Last Updated:** June 13, 2026

---

## Overview & Goals

- Allow users to upload photos (JPG, PNG) and videos (MP4, MOV, AVI) via the web interface.
- Enforce strict user ownership (`user_id` from JWT context). ✅ Verified — `GetUserIDFromContext()` is called at entry point of all handlers.
- Automatically organize files into `storage/{user_id}/YYYY/MM/DD/`. ✅ Verified in upload_handler.go: ```go dir := fmt.Sprintf("storage/%s/%s", userID, time.Now().Format("2006/01/02"))```
- Deduplicate uploads using SHA256 hashing to prevent redundant storage. ✅ Implemented — hash computed during upload stream copy, checked against DB before saving duplicate file data.

---

## Configurable Upload Limits ⚠️ **Updated from documentation**

- **Environment Variable**: `MAX_UPLOAD_SIZE` (in bytes, configurable)
- ⚠️ **Actual Default Value**: `512 << 20 = 536870912` (**~512 MB**) — *not* 10MB as previously documented. This is set in `internal/api/middleware.go`:
```go
var MaxUploadSizeBytes int64 = 512 << 20 // Default: 512MB (supports large single-file uploads)
```
- **Rationale**: Supports large 4K video clips while preventing accidental abuse or network timeouts. Files exceeding this limit will be rejected with a `413 Request Entity Too Large` error before consuming server resources.

---

## API Specifications *(Updated paths and methods)*

| Method | Path | Description | Auth Required |
|--------|------|-------------|---------------|
| `POST` | `/api/v1/media/upload` | Upload one or more media files (multipart/form-data) for Web & Mobile clients | ✅ JWT/Session |
| `POST` | `/api/v1/media/upload/single` | Single file upload endpoint for mobile clients — increased memory limit to 1GB | ✅ JWT/Session |
| `POST` | `/api/v1/media/upload/chunk` | Chunked upload endpoint for resumable uploads (mobile) — 128MB per chunk | ✅ JWT/Session |
| `GET` | `/api/v1/media/upload/status` | Upload status endpoint (mobile client) | ✅ JWT/Session |
| `POST` | `/api/v1/media/upload/abort` | Abort upload endpoint (mobile client) — small limit is fine for abort requests | ✅ JWT/Session |
| `POST` | `/api/v1/media/upload/complete` | Complete resumable upload — assemble chunks into final file | ✅ JWT/Session |

**Request Format**: `multipart/form-data`
- ⚠️ **Note**: The original design specified a single endpoint with an optional `albumId` parameter. However, the actual implementation does NOT include this feature yet — uploaded files are not automatically added to albums. This is a gap between design and implementation that should be addressed.

**Response Format**: `application/json`
```json
{
  "uploaded": [
    {
      "id": "uuid-...",
      "filename": "IMG_1234.jpg",
      "mediaType": "photo",
      "path": "/storage/{user_id}/YYYY/MM/DD/filename_ext", // Actual filesystem path, not API URL
      "size": 5248096,
      "captured_at": "2024-12-17T10:30:00Z"
    }
  ],
  "skipped_duplicates": [
    {
      "filename": "IMG_1235.jpg",
      "id": "uuid-dup1"
    }
  ]
}
```

---

## Backend Design (Go) ✅ Implemented

### Handler
`upload_handler.go` registered in `server.go`. Designed to be client-agnostic (Web, Android, iOS).

### Validation
✅ **File type validation** now includes both extension checking AND MIME-type detection via `http.DetectContentType()` which reads the first 512 bytes of uploaded file content. The `isValidMediaType()` function validates that detected MIME types (`image/jpeg`, `image/png`, `video/mp4`, etc.) match expected formats for given extensions. Rejects uploads where MIME type doesn't match (e.g., `.jpg` extension with executable content).

### Storage Path Generation
✅ Verified in upload_handler.go:
```go
dateDir := time.Now().Format("2006/01/02")
relTimePath := filepath.Join(dateDir, newFilename)
relPathFromRoot := filepath.Join(userID.String(), relTimePath)
// Result: "storage/{user_id}/YYYY/MM/DD/{uuid}.{ext}"
```

### Deduplication Logic ✅ Implemented
1. Compute SHA256 hash of the uploaded file stream (during `io.Copy(io.MultiWriter(tempFile, hasher), file)`).
2. Query DB: `SELECT id FROM media WHERE sha256 = ? LIMIT 1`
3. If exists → skip copy to disk, return existing ID in `skipped_duplicates`. File is deleted from temp storage.
4. If new → save file to disk (via temp file), proceed with metadata extraction.

---

## ⚠️ Metadata Extraction — Stubbed (Planned for Future Phase)

- **Current Behavior**: Defaults capture timestamp (`captured_at`) to upload time (`time.Now()`). Actual EXIF/video property parsing is not yet implemented in `upload_handler.go`.
- **Images** (Planned): Parse EXIF data (camera model, ISO, aperture, GPS, capture time).
- **Videos** (Planned): Extract duration, resolution, and codecs via `ffprobe` or `ffmpeg-go`.

---

## Frontend Design — Remaining Work 🚧

- Drag & Drop zone with overlay feedback.
- Pre-upload preview grid (images show thumbnails, videos show first frame or native player).
- Per-file and overall progress tracking.
- Batch handling queue (max 3 concurrent uploads to prevent browser limits).

---

## Android App Upload Integration

### Chunked Upload via OkHttp (Not Go gomobile)
**✅ Confirmed in use by Android app**: The `UploadManager.kt` uses the `/api/v1/media/upload/chunk` endpoint for chunked resumable uploads when file size >5MB and `enableChunkedUpload=true`. 

- 5MB chunks on client side, 64MB per chunk on server side
- Session management with persistence to disk for crash recovery
- The old inconsistent upload path in `SyncRepositoryImpl.uploadMedia()` (which loaded entire file into memory via `it.readBytes()`) is now **dead code** — MainViewModel no longer calls it. However, that method still exists as a risk: future developers adding new upload paths through SyncRepository could accidentally reintroduce the OOM issue. **Recommendation**: delete `SyncRepositoryImpl.uploadMedia()` entirely or mark it as deprecated with a clear warning comment.

### Go gomobile Bindings
- `UploadSingleFile` (multipart upload with retry logic) — complete but never called from Kotlin — Android uses OkHttp for chunked uploads instead.
- **⚠️ Okio deprecation fixes applied**: Replaced deprecated `Okio.buffer(Sink)` static method calls in ProgressRequestBody.kt and StreamingUploadHelper with extension function patterns (`sink.buffer()`, `countingSink.buffer()`).
- **⚠️ OkHttp deprecation fix applied**: Replaced deprecated `RequestBody.create(mediaType, content)` calls in UploadManager.kt with the extension function pattern `content.toRequestBody(mediaType)`. Both were compilation errors — the old APIs no longer compile.

---

## Related Documents

- [Architecture Overview](readme-arch.md) — High-level system architecture
- [Sharing Feature](readme-sharing.md) — Public share links for uploaded media (In Progress)
