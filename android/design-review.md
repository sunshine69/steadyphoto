# SteadyPhoto Android Sync Client — Design Review

## Executive Summary

The Android Kotlin + Go (gomobile) sync client is **substantially implemented** with ~80-90% of planned features in place. The architecture follows the design document closely, but there are several gaps and inconsistencies that need attention before production readiness.

---

## 1. Architecture Layer Review

### ✅ Android Kotlin Layer — Well Implemented

#### UI (Compose)
| Component | Status | Notes |
|-----------|--------|-------|
| LoginScreen.kt | ✅ Complete | Has loading/error/success states, proper error handling |
| HomeScreen.kt | ✅ Complete | Stats cards, status indicator, Start/Stop buttons, permission rationale UI |
| SetupScreen.kt | ✅ Complete | API URL configuration with validation and example URLs |
| SettingsScreen.kt | ⚠️ Partially implemented | API URL config works, but auto-sync/WiFi-only toggles are **not persisted** — they reset on every screen recreation. Logout button is a stub. Storage info is placeholder text only. |

#### Login/Auth
- ✅ `AuthViewModel` with proper loading/error/success states
- ✅ JWT token storage via `ApiClient.storeAuthToken()` / `storeRefreshToken()`
- ⚠️ **Missing**: Token refresh flow — while the API endpoint exists (`refreshSync`), there's no implementation of automatic token refresh when access tokens expire. The interceptor is a placeholder.
- ⚠️ **Missing**: Persistent session across app restarts — unclear if `ApiClient.getAuthToken()` persists to disk

#### MediaStore Queries
| Scanner | Status | Notes |
|---------|--------|-------|
| `MediaScanner.kt` | ✅ Complete | Three-tier scan: type-specific (Images/Video) → broad Files provider fallback. Hash dedup via SHA-256. |
| `MediaContentObserver.kt` | ✅ Complete | Debounced ContentObserver on Images + Video URIs with notifyForDescendants=true |
| `DirectoryFileObserver.kt` | ⚠️ Partially implemented | FileObserver for real-time filesystem detection, but only watches 3 hardcoded directories. Needs dynamic directory support from setup screen (user selects which folders to monitor). |

#### Permissions
- ✅ `PermissionHelper` utility class exists with proper Android permission checking
- ✅ Runtime permission request via ActivityResultLauncher in HomeScreen
- ✅ Permission rationale UI shown when denied
- ⚠️ **Missing**: Handling of Android 13+ granular media permissions (READ_MEDIA_IMAGES, READ_MEDIA_VIDEO) — the code uses `READ_EXTERNAL_STORAGE` which is deprecated on API 33+. Need to check for permission version and request appropriate ones.

#### Foreground Service (`SyncService.kt`)
- ✅ Proper foreground notification with channel creation
- ✅ FileObserver + ContentObserver dual detection (real-time primary, content observer secondary)
- ✅ Periodic fallback sync every 5 minutes as safety net
- ⚠️ **Missing**: Doze mode / battery optimization handling — WorkManager handles this but the service itself doesn't account for it

#### WorkManager Scheduling (`SyncManager.kt`)
- ✅ `UploadWorker` — one-time upload work request
- ✅ `MediaScannerWorker` — periodic media scan (5 min interval)
- ⚠️ **Missing**: Network constraint on workers — no `requireNetwork()` or `setRequiredNetworkType(TRANSPORT_ANY)` configured. Workers will run even when offline, wasting battery and failing uploads.

#### Battery/Network Constraints
- ✅ `NetworkConnectivityMonitor` with Flow-based network status observation
- ⚠️ **Missing**: WiFi-only enforcement — SettingsScreen has a toggle but it's not wired to WorkManager constraints or the UploadWorker
- ⚠️ **Missing**: Doze mode / background execution limits handling

### ✅ Go Sync Engine (gomobile) — Partially Implemented

| Component | Status | Notes |
|-----------|--------|-------|
| `MediaProcessor.MediaHasher` | ✅ Complete | SHA256 file hashing via Go's crypto/sha256 |
| `MediaProcessor.GetFileMetadata` | ⚠️ Partially implemented | Only handles basic MIME type from extension. **Missing**: EXIF parsing (capture time, camera model, GPS coordinates) which the MediaItemEntity supports as fields |
| `UploadSingleFile` | ✅ Complete | Multipart upload with retry logic, streaming copy |
| Chunked Upload (`CreateChunkedUploadSession`, `UploadChunk`, `GetChunkedUploadStatus`, `CompleteChunkedUpload`) | ⚠️ Partially implemented | All four endpoints exist in Go but **the Android UploadManager uses OkHttp for chunked uploads**, not the gomobile bindings. The gomobile chunk upload functions are never called from Kotlin code — they appear to be dead code or fallback implementations that were superseded by the OkHttp approach. |

---

## 2. Data Layer Review

### Room Database (`MediaItemEntity`)
- ✅ Proper entity with hash-based deduplication (unique index on `hash` field)
- ✅ Upload status enum with all states: PENDING, UPLOADING, UPLOADED, FAILED, SKIPPED_DUPLICATE, CANCELLED, DELETED
- ✅ Capture time, camera model, GPS fields present but **never populated** — EXIF parsing is missing from both the Go and Kotlin layers

### Repository Layer
| Component | Status | Notes |
|-----------|--------|-------|
| `SyncRepositoryImpl` | ⚠️ Partially implemented | Uses Retrofit for uploads (not gomobile). No retry logic in this path. Uploads entire file into memory via `it.readBytes()` — problematic for large files (>100MB) that could cause OOM on Android. |
| `UploadManager` | ✅ Well implemented | Has chunked upload with streaming, batch uploads, progress tracking, network monitoring, retry logic |
| ⚠️ **Inconsistency**: Two upload paths exist — SyncRepositoryImpl (single-threaded, no chunking, loads whole file into memory) and UploadManager (chunked, streaming). MainViewModel uses `container.repository.uploadMedia()` which calls the weaker path. The UploadManager is used by UploadWorker but NOT by HomeScreen's direct sync. |

### DTOs
- ✅ All required DTOs present: LoginRequest/Response, RefreshRequest/Response, UploadResponse, SyncStatusResponse, ChunkUploadResponse, etc.
- ⚠️ **Missing**: Error handling DTO — no structured error response parsing (e.g., `ErrorResponse` with code/message)

---

## 3. Critical Gaps to Address

### 🔴 High Priority

1. **Token Refresh Flow Missing**
   - The API endpoint exists but is never used. When the access token expires, all subsequent requests will fail silently. Need interceptor implementation that calls `/api/v1/auth/refresh` and retries the original request with a new token.

2. **Android 13+ Permission Handling**
   - Uses `READ_EXTERNAL_STORAGE` which requires manifest permission on API 30 but is unnecessary on API 33+. Need version-conditional permission requests using `READ_MEDIA_IMAGES` / `READ_MEDIA_VIDEO`.

3. **Two Upload Paths — Inconsistent Behavior**
   - HomeScreen → MainViewModel → SyncRepositoryImpl (loads file into memory, no chunking)
   - UploadWorker → UploadManager (chunked upload, streaming, retry logic)
   - Fix: Have MainViewModel use the same `UploadManager` that UploadWorker uses for consistency.

4. **EXIF Metadata Not Populated**
   - The entity has fields for captureTime, cameraModel, gpsLatitude, gpsLongitude — but no code populates them. Need Go EXIF parsing (e.g., via `github.com/xor-gate/go-exifdecode`) or Kotlin-based extraction.

### 🟡 Medium Priority

5. **WorkManager Network Constraints Missing**
   - Workers should have `setRequiredNetworkType(TRANSPORT_ANY)` at minimum, and ideally `requireNetwork()` to prevent offline uploads.

6. **Settings Persistence Broken**
   - AutoSyncEnabled, WiFiOnly toggles in SettingsScreen are not persisted (SharedPreferences/DataStore). They reset on every screen recreation.

7. **FileObserver Directory Configuration**
   - Monitors hardcoded directories only. Should read configured directories from setup/preferences so users can choose which folders to sync.

8. **ApiClient Token Persistence**
   - Unclear if `getAuthToken()` persists across app restarts. If not, user must log in every time the app is opened.

9. **SyncRepositoryImpl Memory Issue**
   - `it.readBytes()` loads entire file into memory for upload — crashes on large videos. Should use streaming approach like UploadManager does.

### 🟢 Low Priority / Nice-to-Have

10. **Settings Screen Features Not Wired Up**
    - Logout button is a stub, storage info is static text, auto-sync/WiFi-only toggles don't affect behavior.

11. **gomobile Chunk Upload Dead Code**
    - The Go chunk upload functions are never called from Kotlin. Either integrate them or remove them to reduce confusion.

12. **No Download/Deletion Sync Support**
    - `deleteMedia` in the repository just deletes local records — doesn't call the server endpoint to actually delete from cloud. No download sync exists at all.

---

## 4. Architecture Alignment Scorecard

| Design Requirement | Implementation Status | Gap |
|-------------------|----------------------|-----|
| UI layer (Compose) | ✅ Implemented | Settings not fully wired up |
| Login/Auth | ⚠️ Partially implemented | Token refresh missing |
| MediaStore queries | ✅ Implemented | Three-tier scan working well |
| Permissions | ⚠️ Partially implemented | Android 13+ permissions not handled |
| Foreground service | ✅ Implemented | Doze mode handling missing |
| WorkManager scheduling | ⚠️ Partially implemented | No network constraints configured |
| Battery/network constraints | ⚠️ Partially implemented | WiFi-only toggle not wired, no doze handling |
| Go file scanning | ❌ Not implemented | Gomobile has hash/metadata but no scan function exposed to Android — relies entirely on Kotlin MediaScanner |
| Go hashing | ✅ Implemented | SHA256 works via gomobile |
| Go metadata extraction | ⚠️ Partially implemented | MIME type only, EXIF missing |
| Go upload (single file) | ✅ Implemented | Multipart with retry |
| Go chunked upload | ⚠️ Partially implemented | Exists in Go but unused — Android uses OkHttp instead |
| File descriptor / path passing | ⚠️ Mixed | Gomobile takes file paths, Kotlin MediaScanner works via ContentResolver URIs |

---

## 5. Recommendations for Next Steps

### Phase 1: Critical Fixes (Pre-Release)
1. Implement token refresh interceptor in `ApiClient`
2. Fix Android 13+ permission handling with version-conditional requests
3. Consolidate upload path — have MainViewModel use `UploadManager.uploadMedia()` instead of `SyncRepositoryImpl.uploadMedia()`
4. Add WorkManager network constraints to workers

### Phase 2: Feature Completeness
5. Implement EXIF parsing in Go for capture time, camera model, GPS coordinates
6. Wire up Settings screen (persist toggles via DataStore/SharedPreferences)
7. Implement actual token persistence across app restarts
8. Fix SyncRepositoryImpl memory issue — use streaming upload

### Phase 3: Polish
9. Clean up dead gomobile chunk upload code or integrate it properly
10. Add download sync support (server → device)
11. Implement actual server-side delete in `deleteMedia`
12. Add dynamic FileObserver directory configuration from user preferences

