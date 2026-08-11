# Upload Silent Failure Analysis

## Problem Statement
Some uploads fail silently — the server doesn't print any error log, but the Android client never receives the file on the server.

---

## Root Causes Found

### 1. 🔴 HEIC Files — MIME Type Mismatch (Most Common)

**Android Client sends:**
- `file.mime` = `image/heic` (MediaStore reports this as the actual MIME type)
- But `http.DetectContentType()` on the server reads the first 512 bytes and often detects HEIC files as `image/jpeg` or `image/x-adobe-dng` (a known Go bug).

**Server side:**
```go
// upload_handler.go line 186-191
detectedType := http.DetectContentType(buf[:nRead])
if !isValidMediaType(detectedType, extLower) {
    mlog.Info("[ERROR] UploadHandler: MIME type mismatch...")  // <-- ONLY logged as INFO, not ERROR
    os.Remove(tempFile.Name())
    continue  // <-- SILENTLY SKIPPED
}
```

**What happens:** The HEIC file is silently dropped with no visible error.

**Fix:** Add `"image/x-adobe-dng"` and `"application/octet-stream"` with `.heic`/`.heif` extension to the valid MIME list. Also add `http.DetectContentType` alternatives for HEIC files.

---

### 2. 🔴 MIME Type from MediaStore Doesn't Match Server Sniffing

**Android:** `MediaStore` reports MIME types that differ from `http.DetectContentType()`:
- `.heic` → MediaStore says `image/heic` → Server detects as `application/octet-stream` or `image/jpeg`
- `.dng` (Adobe raw) → Server detects as `image/x-adobe-dng` or `image/tiff`

**Fix:** The server's `isValidMediaType` must accept more MIME types that are legitimate for the given extension.

---

### 3. 🔴 `openInputStream()` Returns Partial Data (Truncated File)

**UploadManager.kt:**
```kotlin
override fun contentLength(): Long = item.fileSize  // DB size, WRONG if file changed
override fun writeTo(sink: BufferedSink) {
    val inputStream = context.contentResolver.openInputStream(uri)  // May return less data
    ...
    while (true) {
        val bytesRead = inputStream.read(buffer)  // Could return -1 before fileSize
        if (bytesRead == -1) break
        sink.write(buffer, 0, bytesRead)
    }
}
```

**What happens:** If MediaStore file was truncated or deleted between scan and upload, `contentLength()` reports a larger size than actual bytes written. Server receives fewer bytes → may silently fail or corrupt.

**Fix:** Log actual bytes sent, compare with expected. Abort if mismatch.

---

### 4. 🔴 Chunk Upload Error Silently Swallowed

**UploadManager.kt:**
```kotlin
val response = apiService.uploadChunk(...)
if (response.success) return true
else lastError = Exception("Server returned failure for chunk $chunkIndex")  // Not logged with enough detail
```

**What happens:** If chunk upload fails (network blip, 500 error), only the `lastError.message` is used in retry. No HTTP status code or response body is logged.

**Fix:** Log HTTP status code and response body from chunk failures.

---

### 5. 🔴 `contentLength()` in Chunk Upload Returns DB `fileSize` (Wrong)

**UploadManager.kt line ~512:**
```kotlin
override fun contentLength(): Long = item.fileSize  // May differ from actual chunk data
```

**What happens:** Content-Length mismatch → server might silently drop the request or return 400.

---

### 6. 🔴 Missing Retry Details for HTTP Errors

**UploadManager.kt:**
```kotlin
} catch (e: Exception) {
    lastError = e
    Log.w(TAG, "Upload attempt $attempt failed for ${item.fileName}: ${e.message}")  // Just e.message, no stack trace
    ...
}
```

**What happens:** Server returns 500, 503, or timeout — only the message is logged. Stack trace and URL not logged.

**Fix:** Log full exception, status code, and response body.

---

### 7. 🔴 Server Doesn't Log Content-Length of Incoming Requests

When a chunk is uploaded, the server receives the raw bytes but doesn't log:
- What `Content-Length` was in the HTTP header
- How many bytes were actually written to disk
- Whether the written size matches what was expected

**Fix:** Log `Content-Length` header vs bytes written per chunk.

---

### 8. 🔴 `isValidMediaType` Only Rejects Non-Image/Non-Video (Not Robust)

The current check:
```go
case strings.HasPrefix(detectedType, "image/"):
    return true
case strings.HasPrefix(detectedType, "video/"):
    return true
default:
    return false
```

This rejects:
- `image/x-adobe-dng` (Adobe RAW) → rejected even though `.dng` is in valid extensions
- `image/x-panasonic-rw2` (Panasonic RAW) → rejected
- `application/octet-stream` with known image extension → rejected

**Fix:** Also allow `application/octet-stream` when the extension is a known media format.

---

### 9. 🔴 Server Response to Client Not Logged

When the server returns an error or a "duplicate" response, it's not always logged which response was sent.

**Fix:** Log the final JSON response sent to the client.

---

### 10. 🔴 `handleComplete()` Fails Silently After Chunk Upload

When all chunks are uploaded, `handleComplete()` calls:
```go
assembledFile, err := assembleChunks(ctx, session, r)
```

If `assembleChunks` returns `nil`, `err != nil`, or the file is corrupted, the response is:
```go
w.WriteHeader(http.StatusInternalServerError)
json.NewEncoder(w).Encode(map[string]interface{}{
    "success": false,
    "error":   "Failed to assemble file",
})
```

But `assembleChunks` doesn't return a meaningful error message — it returns `nil, nil` if everything looks good but some chunk file is missing.

**Fix:** Return meaningful errors from `assembleChunks`.

---

## Files to Fix

1. **`internal/api/upload_handler.go`** — Fix `isValidMediaType` for HEIC, add detailed logging
2. **`internal/api/upload_handler_single.go`** — Fix `isValidMediaType` for HEIC, add detailed logging
3. **`android/app/src/main/java/com/steadyphoto/sync/data/repository/UploadManager.kt`** — Add detailed error logging, fix content length mismatch

---
