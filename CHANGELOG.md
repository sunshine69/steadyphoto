# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Fixed

#### Date reporting broken on Android uploads

**Root cause:** Android app sends `fileCreatedAt` as epoch seconds (e.g., `"1720000000"`) from `MediaItemEntity.fileCreatedAt` (which reads `MediaStore.DATE_ADDED`), but the Go backend tried to parse it as a formatted date string using `time.Parse("2006/01/02 15:04:05", ...)`. This parsing always failed, so `meta.FileCreatedAt` was never set, breaking the entire CapturedAt fallback chain.

**Changes:**

##### Go backend (`internal/api/upload_handler_single.go`)

- **`fileCreatedAt` parsing fixed** — changed from `time.Parse("2006/01/02 15:04:05", fileCreatedAtStr)` to `strconv.ParseInt(fileCreatedAtStr, 10, 64)` + `time.Unix(epochSecs, 0)` in both `HandleSingleFileUpload` and `HandleComplete`. This correctly interprets the epoch seconds sent by Android.

- **Duplicate `fileCreatedAt` parsing block removed** — the parsing appeared twice in `HandleSingleFileUpload` (lines ~452 and ~482). Kept only one instance.

- **`captureTime` field now parsed** — Android already sends the EXIF capture time as `captureTime` (another epoch seconds field), but the Go backend ignored it. Added parsing for `captureTime` in both `HandleSingleFileUpload` and `HandleComplete`, and it is now the **highest-priority** source for `CapturedAt`.

- **CapturedAt fallback chain fixed and simplified** — the old logic had a dead branch (`if !capturedAt.IsZero()` after it was already checked) and wrong priority order. New priority order:
  1. Client `captureTime` (most accurate EXIF capture time)
  2. `VideoMetadata.CreatedAt` (for videos, from video analysis)
  3. EXIF `DateTimeOriginal` (for images)
  4. Filename date heuristic (e.g., `2024-07-15_10-30-00.jpg`)
  5. Client `fileCreatedAt` (filesystem creation time from `DATE_ADDED`)
  6. Upload time (fallback)

- **`HandleComplete` now includes `CreatedAt`/`UpdatedAt`** — the `HandleComplete` endpoint was missing these fields on the Media struct. Added them.

- **`HandleComplete` response now includes `file_created_at`** — the upload response now returns `file_created_at` in the response JSON for both endpoints.

- **`UploadCompleteResponse` now includes `file_created_at`** — the upload complete response now returns `file_created_at` in the response JSON.

##### Android app (`ApiService.kt` + `UploadManager.kt`)

- **`uploadSingleFile` now sends `captureTime`** — added `@Part("captureTime") captureTime: RequestBody? = null` parameter to `uploadSingleFile` in `ApiService.kt`, and pass `item.captureTime?.toString()` from `UploadManager.kt`. The `MediaItemEntity` already had this field populated from EXIF data.

##### Android app (`MediaScannerWorker.kt` + `MediaScanner.kt`)

- **Added `captureTime` to upload request** — the `uploadSingleFile` API endpoint now includes the `captureTime` parameter, allowing the Go backend to use the EXIF capture time as the highest-priority source for `CapturedAt`.

#### Fewer media items found after clearing app data

**Root cause:** When the user pressed "Start" from the HomeScreen, `MainViewModel.startSync()` called `container.repository.scanNewMedia()` without the `forceFullScan` parameter, which defaulted to `false`. This meant the broad "Files" MediaStore scan was skipped — only the narrow Images and Video type-specific scans ran. The broad Files scan catches additional file types (e.g., HEIF, AVI, MKV, WebM) that the type-specific scans miss.

**Fix:** Changed `MainViewModel.startSync()` to call `container.repository.scanNewMedia(forceFullScan = true)` so the broad Files scan always runs, catching all media file types.

#### No upload status / progress display during upload

**Root cause:** The `UploadProgressCallback.onProgressUpdated()` override in `MainViewModel` was empty, so upload progress was never communicated to the UI. The `SyncUiState.Uploading` state only displayed "Uploading..." without showing which file was being uploaded or how far along the process was.

**Changes:**

- **`MainUiState` now tracks current upload item and progress** — added `currentUploadItem: String?` and `currentUploadProgress: Float` fields to `MainUiState` so the UI can display real-time upload progress.

- **`onProgressUpdated` now updates UI state** — the callback now reads `progress.currentUpload` to get the current file name and calculates the upload percentage (`bytesUploaded / totalBytes * 100`), updating the `MainUiState` accordingly.

- **HomeScreen now shows upload progress bar** — when `SyncUiState.Uploading` is active, the status indicator now shows:
  - The name of the currently uploading file (truncated to one line)
  - A `LinearProgressIndicator` showing progress percentage
  - The percentage value displayed as text

---

## [Previous Versions]

### Fixed

- *(previous fixes documented here)*

### Added

- *(previous additions documented here)*

### Changed

- *(previous changes documented here)*
