# SteadyPhoto Date/Time Handling Documentation

## Overview

SteadyPhoto captures and stores datetime metadata for media files from multiple sources depending on the upload method. This document explains the sources, formats, and known issues with datetime handling across the codebase.

---

## Datetime Sources by Upload Method

### 1. Web Upload (Angular)

**Location:** `angular-app/src/app/services/upload.service.ts`

The web upload client sends the filesystem timestamp as `fileCreatedAt`.

- **Source:** `file.lastModified` (DOM File API timestamp, in milliseconds since epoch)
- **Format:** `YYYY/MM/DD HH:MM:SS`
- **Example:** `2024/06/14 09:30:45`
- **Code:**
  ```typescript
  if (file.lastModified) {
    const date = new Date(file.lastModified);
    const formatted = `${date.getFullYear()}/${String(date.getMonth() + 1).padStart(2, '0')}/${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}:${String(date.getSeconds()).padStart(2, '0')}`;
    formData.append('fileCreatedAt', formatted);
  }
  ```
- **Issue:** Only captures the filesystem creation time, **not** the camera capture time for photos/videos. On desktop, `file.lastModified` is often the same as creation time, but for photos taken with a camera, it will be the time the file was copied to disk, not when the photo was taken.

---

### 2. Android Upload (Kotlin)

**Location:** `android/app/src/main/java/com/steadyphoto/sync/worker/UploadWorker.kt`

The Android upload client sends the MediaStore `DATE_ADDED` column as `fileCreatedAt`.

- **Source:** `MediaItemEntity.fileCreatedAt` — set from MediaStore `DATE_ADDED` (epoch seconds)
- **Format:** Raw epoch seconds as a string (e.g., `1716369600`)
- **Example:** `1716369600`
- **Code:**
  ```kotlin
  apiService.uploadSingleFile(
    ...
    fileCreatedAt = (item.fileCreatedAt?.toString() ?: "").toRequestBody("text/plain".toMediaType())
  )
  ```
- **Issue:** Sends raw epoch seconds instead of converting to `YYYY/MM/DD HH:MM:SS` format. The Go server expects the `2006/01/02 15:04:05` (Go time format) string.
- **Also:** The `MediaItemEntity` has a `captureTime` field (the camera capture time from EXIF), but this is **not** sent to the server during upload.

**Note on `captureTime` field:**
```kotlin
@Entity(tableName = "media_items")
data class MediaItemEntity(
    ...
    val captureTime: Long? = null,          // Camera capture time from EXIF
    val fileCreatedAt: Long? = null,        // MediaStore DATE_ADDED (epoch seconds)
    ...
)
```

---

### 3. Scanner CLI

**Location:** `cmd/scanner/main.go`

The scanner tool uses a priority-based approach to extract the date:

1. **Filename date extraction** (highest priority)
2. **File metadata** (EXIF for photos, ffprobe for videos)
3. **Fallback: `time.Now()`** (local timezone, NOT UTC)

- **Filename extraction:** Uses `processor.ParseDateFromFilename()` which supports formats like `DSC_20230514_103000.JPG`
- **EXIF extraction:** Uses `processor.ExtractDateFromExif()` for photos, extracts `DateTimeOriginal`, `DateTimeDigitized`, `CreateDate`, `TrackCreateDate`
- **Video extraction:** Uses `processor.ExtractVideoMetadata()` which extracts `creation_time` from MP4 metadata
- **Fallback:** If no date is found anywhere, uses `time.Now()` — **this is a bug** because it uses the local timezone instead of UTC

---

### 4. Update-Capture-Date CLI

**Location:** `cmd/update-capture-date/main.go`

This tool updates existing database records where `capture_date` is null or equals `file_created_at`.

- **Purpose:** Backfills the `capture_date` column for existing records that only have `file_created_at`
- **Format:** `YYYY/MM/DD HH:MM:SS` (Go time format)
- **Behavior:** Sets `capture_date = file_created_at` for records where `capture_date IS NULL OR capture_date = file_created_at`

---

### 5. Go Server Upload Handlers

**Web Upload Handler:** `internal/api/upload_handler_single.go`  
**Android Upload Handler:** `internal/api/upload_handler.go`

Both handlers accept the `fileCreatedAt` parameter from the client and store it in the `file_created_at` column.

- **Format accepted:** `2006/01/02 15:04:05` (Go time layout, equivalent to `YYYY/MM/DD HH:MM:SS`)
- **Parsing code:**
  ```go
  var fileCreatedAt time.Time
  if fileCreatedAtStr != "" {
      parsed, err := time.Parse("2006/01/02 15:04:05", fileCreatedAtStr)
      if err == nil {
          fileCreatedAt = parsed
      }
  }
  ```
- **Critical issue:** The Go server **never reads EXIF** from the uploaded files. It relies entirely on the client-provided `fileCreatedAt` parameter. This means:
  - Web uploads always get the filesystem creation time, not capture time
  - Android uploads get the MediaStore `DATE_ADDED` time, not capture time
  - Photos taken with a camera will have incorrect timestamps

---

### 6. EXIF Reader

**Location:** `internal/processor/exif_reader.go`

The EXIF reader can extract dates from photos and videos, but it is only used by:
- The Scanner CLI
- The Update-Capture-Date CLI

It is **not** called during web or Android uploads.

- **Photo date extraction:** Reads `DateTimeOriginal`, `DateTimeDigitized`, `CreateDate`, `TrackCreateDate`
- **EXIF date format:** These EXIF fields are in `YYYY:MM:DD HH:MM:SS` format (with colons, not slashes), and the reader converts them to Go time objects
- **Video date extraction:** Extracts `creation_time` from MP4 metadata

---

### 7. Filename Date Extraction

**Location:** `internal/processor/filename_date.go`

The scanner uses this to extract dates from filenames like `DSC_20230514_103000.JPG`.

**Supported formats:**

| Format | Example |
|--------|---------|
| `YYYYMMDD` | `DSC_20230514_103000.JPG` |
| `YYYY-MM-DD` | `2023-05-14_103000.jpg` |
| `YYYYMMDD-HHMMSS` | `20230514-103000.jpg` |
| `YYYY-MM-DDTHHMMSS` | `2023-05-14T103000.jpg` |
| `YYYYMMDD_HHMMSS` | `20230514_103000.jpg` |
| `YYYY-MM-DD_HHMMSS` | `2023-05-14_103000.jpg` |

- **Timezone:** Defaults to UTC
- **Used by:** Scanner CLI only

---

## Database Schema

The `media_items` table has two datetime-related columns:

```sql
file_created_at   VARCHAR(20)   -- YYYY/MM/DD HH:MM:SS format
capture_date      VARCHAR(20)   -- YYYY/MM/DD HH:MM:SS format (nullable)
```

- **`file_created_at`:** The filesystem creation time, provided by the upload client
- **`capture_date`:** The camera capture time, populated by the Scanner CLI from EXIF/filename, and the Update-Capture-Date CLI for backfilling

---

## Known Issues

1. **Web upload uses filesystem timestamp, not capture time** — photos taken with a camera will have timestamps reflecting when the file was saved to disk, not when the photo was taken.

2. **Android upload sends epoch seconds instead of formatted date** — the Android client sends `fileCreatedAt` as a raw epoch seconds string (e.g., `1716369600`) but the Go server expects `YYYY/MM/DD HH:MM:SS`. The Android `captureTime` field (which contains the camera capture time) is not sent to the server.

3. **Go server never reads EXIF from uploaded files** — whether uploading from web or Android, the Go server never parses EXIF metadata from the uploaded file. The only date source is the client-provided `fileCreatedAt` parameter.

4. **Scanner fallback uses local timezone** — if no date is found in the filename or EXIF, the Scanner CLI falls back to `time.Now()`, which uses the local timezone instead of UTC. This could cause timezone inconsistencies.

5. **Android `fileCreatedAt` is MediaStore `DATE_ADDED`, not file creation time** — `DATE_ADDED` represents when the file was added to the MediaStore (which could be when the photo was first captured if it was already on the device), not necessarily the filesystem creation time.

---

## Recommended Fixes

1. **Android upload:** Convert `captureTime` (camera capture time from MediaStore/EXIF) to `YYYY/MM/DD HH:MM:SS` format and send it as `fileCreatedAt` instead of raw epoch seconds.

2. **Web upload:** Consider using the browser API to extract EXIF data from photos before upload, then send the capture time instead of the filesystem timestamp.

3. **Go server:** Implement EXIF reading during upload (both single and batch) to extract the camera capture time and populate the `capture_date` column.

4. **Scanner CLI:** Change the fallback from `time.Now()` to `time.Now().UTC()` to ensure consistency.

5. **Android upload:** Also send `captureTime` as a separate parameter to the server so it can be stored in the `capture_date` column.
