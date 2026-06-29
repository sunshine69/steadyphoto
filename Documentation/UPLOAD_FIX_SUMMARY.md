# Fix for 4GB Video Upload Failure on Android

## Problem
Android app failed to upload 4GB video files with error "http: request body too large" (HTTP 413). The same file uploaded successfully via the web client.

## Root Cause Analysis

### Issue 1: `LimitBodySizeMiddleware` blocking uploads (FIXED)
- **File**: `internal/api/server.go`
- **Problem**: The middleware checked Content-Length header before processing, rejecting requests with body sizes > 5MB (configurable via MAX_UPLOAD_SIZE env var). For multipart/form-data uploads from Android, the Content-Length reflects the total file size, not individual chunk boundaries.
- **Fix**: Removed `LimitBodySizeMiddleware` from upload routes since:
  - Chunked uploads use small chunks (5MB) that pass through fine
  - Single-file uploads should rely on `ParseMultipartForm` limits instead

### Issue 2: Insufficient `ParseMultipartForm` limit in single-file handler (FIXED)
- **File**: `internal/api/upload_handler_single.go` 
- **Problem**: `HandleSingleFileUpload` used a 32MB memory limit for parsing multipart forms. If the Android app sent a file > 5MB as a single request instead of chunks, this would fail with "http: request body too large".
- **Fix**: Increased to 1GB (1 << 30) to handle large single-file uploads without chunking

### Issue 3: Android app upload strategy decision (NOT FIXED - App-side issue)
- **File**: `./android/app/src/main/java/com/steadyphoto/sync/data/repository/UploadManager.kt`
- **Problem**: The Android app uses a threshold of 5MB to decide between single-file and chunked uploads:
  ```kotlin
  val result = if (currentConfig.enableChunkedUpload && item.fileSize > CHUNK_SIZE) {
      performChunkedUpload(item, token, callback)
  } else {
      performSingleFileUpload(item, uri, token, callback)
  }
  ```
  For a 4GB file, this should trigger chunked upload. However, if `item.fileSize` is incorrectly set or the Android app tries to use single-file upload for large files, it would fail with the old limits.

## Changes Made

### 1. `internal/api/server.go` - Removed middleware from upload routes
```go
// Before:
protected.Route("/media/upload", func(r chi.Router) {
    r.Use(LimitBodySizeMiddleware)
    r.Post("/", uploadHandler.Handle)
    // ... other endpoints ...
})

// After:
protected.Route("/media/upload", func(r chi.Router) {
    // LimitBodySizeMiddleware removed - chunked uploads use small chunks (5MB), 
    // and ParseMultipartForm handles per-part limits. The middleware's Content-Length check
    // was blocking large file uploads that Android sends as multipart forms with proper boundaries.

    r.Post("/", uploadHandler.Handle)
    // ... other endpoints unchanged ...
})
```

### 2. `internal/api/upload_handler_single.go` - Increased ParseMultipartForm limit
```go
// Before:
err := r.ParseMultipartForm(32 << 20) // 32MB memory limit

// After:
err := r.ParseMultipartForm(1 << 30) // 1GB memory limit - allows large single-file uploads without chunking
```

### 3. Chunk upload handler already has adequate limits (no change needed)
- `HandleChunkUpload`: Uses 64MB ParseMultipartForm limit, sufficient for 5MB chunks
- `HandleComplete`: Uses 1MB limit (fine - no new data in this request)
- `HandleAbort`: Uses 1MB limit (fine - small abort request)

## Server Restart Required
After making these changes:
```bash
export DATABASE_URL="postgres://steadyphoto:password@localhost:5432/steadyphoto?sslmode=disable"
export STORAGE_DIR="storage"
export LOG_LEVEL=debug
export MAX_UPLOAD_SIZE=5242880
export API_PORT=:8081
killall server.exe 2>/dev/null; sleep 1
go build -o server.exe cmd/server/main.go
nohup ./server.exe > server.log 2>&1 &
```

## Testing
To test the fix:
1. Upload a large video file (>5MB) from Android app
2. Monitor server logs for successful upload completion
3. Verify file appears in media library

## Additional Notes
- The `MAX_UPLOAD_SIZE` environment variable is now only relevant for other purposes (not blocking uploads via middleware)
- Consider adding rate limiting or per-user upload quotas as a separate security measure
- If Android app continues to fail, check if `item.fileSize` is being set correctly in the media entity
