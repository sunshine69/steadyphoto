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
| HomeScreen.kt | ✅ Complete | Stats cards, status indicator, Start/Stop buttons, permission rationale UI. Now uses `UploadManager.uploadMedia()` directly for consistent streaming uploads. |
| SetupScreen.kt | ✅ Complete | API URL configuration with validation and example URLs |
| SettingsScreen.kt | ⚠️ Partially implemented | **FIXED**: Auto-sync/WiFi-only toggles are now persisted via DataStore-backed `SettingsRepository`. Logout button is a stub. Storage info is placeholder text only. |

#### Login/Auth — ✅ Complete (Previously marked as missing)
- ✅ `AuthViewModel` with proper loading/error/success states
- ✅ JWT token storage via `ApiClient.storeAuthToken()` / `storeRefreshToken()`
- ✅ **FIXED**: Token refresh flow implemented in `authInterceptor` — when a 401 is received and a refresh token exists, it calls `/api/v1/auth/refresh`, stores the new access token, and retries the original request. If refresh fails or there's no refresh token, it clears auth state and dispatches an `onAuthFailure` callback to show the login screen.
- ✅ **FIXED**: Persistent session across app restarts — `getAuthToken()` reads from SharedPreferences which persists across app lifecycle

#### MediaStore Queries
| Scanner | Status | Notes |
|---------|--------|-------|
| `MediaScanner.kt` | ✅ Complete | Three-tier scan: type-specific (Images/Video) → broad Files provider fallback. Hash dedup via SHA-256. |
| `MediaContentObserver.kt` | ✅ Complete | Debounced ContentObserver on Images + Video URIs with notifyForDescendants=true |
| `DirectoryFileObserver.kt` | ⚠️ Partially implemented | FileObserver for real-time filesystem detection, but only watches 3 hardcoded directories. Needs dynamic directory support from setup screen (user selects which folders to monitor). **FIXED**: Duplicate const val declarations across FileObserver files causing compilation errors — consolidated into shared `FileObserverConstants.kt` object with `val` instead of `private const val`. |

#### Permissions — ✅ Complete (Previously marked as missing)
- ✅ `PermissionHelper` utility class exists with proper Android permission checking
- ✅ Runtime permission request via ActivityResultLauncher in HomeScreen
- ✅ Permission rationale UI shown when denied
- ✅ **FIXED**: Handles Android 13+ granular media permissions (`READ_MEDIA_IMAGES`, `READ_MEDIA_VIDEO`) — uses version-conditional logic via `Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU`

#### Foreground Service (`SyncService.kt`)
- ✅ Proper foreground notification with channel creation
- ✅ FileObserver + ContentObserver dual detection (real-time primary, content observer secondary)
- ✅ Periodic fallback sync every 5 minutes as safety net
- ⚠️ **Missing**: Doze mode / battery optimization handling — WorkManager handles this but the service itself doesn't account for it

#### WorkManager Scheduling (`SyncManager.kt`) — ✅ Complete (Previously marked as missing)
- ✅ `UploadWorker` — one-time upload work request
- ✅ **FIXED**: Network constraints implemented via `getNetworkConstraintsFromSettings()` — sets `UNMETERED` when WiFi-only is enabled or sync on metered connection is disabled, otherwise uses `CONNECTED`. Applied to both scan and upload workers.
- ✅ **FIXED**: WiFi-only enforcement wired through SettingsViewModel → settingsRepository → SyncManager → WorkManager constraints

#### Battery/Network Constraints — ✅ Complete (Previously marked as missing)
- ✅ `NetworkConnectivityMonitor` with Flow-based network status observation
- ✅ **FIXED**: WiFi-only toggle now affects WorkManager constraints via `getNetworkConstraintsFromSettings()` and triggers `syncManager.rescheduleWithCurrentSettings()` when changed

### ⚠️ Go Sync Engine (gomobile) — Partially Implemented

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
- ⚠️ **Still missing**: Capture time, camera model, GPS fields present but **never populated** — EXIF parsing is missing from both the Go and Kotlin layers

### Repository Layer
| Component | Status | Notes |
|-----------|--------|-------|
| `SyncRepositoryImpl` | ✅ Fixed (Previously marked as problematic) | Upload path now delegates to `UploadManager.uploadMedia()` internally. Marked as deprecated with clear warning about OOM crashes — should not be used for new code. MainViewModel and UploadWorker already migrated to use UploadManager directly via AppContainer. |
| `UploadManager` | ✅ Well implemented | Has chunked upload with streaming, batch uploads, progress tracking, network monitoring, retry logic. Now the primary upload path used by both HomeScreen (via MainViewModel) and UploadWorker. |

### DTOs
- ✅ All required DTOs present: LoginRequest/Response, RefreshRequest/Response, UploadResponse, SyncStatusResponse, ChunkUploadResponse, etc.
- ⚠️ **Missing**: Error handling DTO — no structured error response parsing (e.g., `ErrorResponse` with code/message)

---

## 3. Critical Gaps to Address

### 🔴 High Priority

1. **EXIF Metadata Not Populated**
   - The entity has fields for captureTime, cameraModel, gpsLatitude, gpsLongitude — but no code populates them. Need Go EXIF parsing (e.g., via `github.com/xor-gate/go-exifdecode`) or Kotlin-based extraction.

2. **FileObserver Directory Configuration**
   - Monitors hardcoded directories only in `MultiDirectoryFileObserver.OBSERVED_DIRS`. Should read configured directories from settings so users can choose which folders to sync. Consider adding a SettingsRepository entry for custom watch directories and updating the FileObserver files accordingly.

3. **No Download/Deletion Sync Support**
   - `deleteMedia` in the repository just deletes local records — doesn't call the server endpoint to actually delete from cloud. No download sync exists at all.

### 🟡 Medium Priority

4. **Settings Screen Features Not Wired Up**
   - Logout button is a stub (`/* TODO: Logout */`). Storage info shows static placeholder text ("Local storage: 0 MB used", "Synced items: 0") instead of querying the database for actual counts.

5. **gomobile Chunk Upload Dead Code**
   - The Go chunk upload functions are never called from Kotlin. Either integrate them or remove them to reduce confusion.

6. **Deprecation Warnings — FileObserver(String) Constructor**
   - All three FileObserver files now show deprecation warnings for the `FileObserver(String)` constructor (deprecated in Android API 26+). The `@Suppress("DEPRECATION")` annotations have been removed per user request to see these warnings. No good alternative exists: ContentObserver doesn't provide real-time filesystem events, and periodic scanning defeats the purpose of immediate detection. Consider migrating to a more modern approach when Android provides one.

7. **Unused Variables in ForceMediaIndexer.kt**
   - `uri` variable (line 74), `path` parameter (lines 79, 146), `scannedUri` parameter (line 146) — all unused but kept for API compatibility with callbacks that may be filled in later.

8. **Unused Variables in UploadManager.kt**
   - Several parameters (`token`, `callback`) and variables (`lastError`) are declared but never used, likely from incomplete refactoring or future use placeholders.

### 🟢 Low Priority / Nice-to-Have

9. **No structured error response parsing** — No `ErrorResponse` DTO for API errors with code/message fields.

10. **Doze mode handling in foreground service** — WorkManager handles this but the SyncService itself doesn't account for battery optimization limits.

---

## 4. Architecture Alignment Scorecard

| Design Requirement | Implementation Status | Gap |
|-------------------|----------------------|-----|
| UI layer (Compose) | ✅ Implemented | Logout button stub, storage info placeholder |
| Login/Auth | ✅ Implemented | Token refresh now implemented and working |
| MediaStore queries | ✅ Implemented | Three-tier scan working well; FileObserver dirs hardcoded |
| Permissions | ✅ Implemented | Android 13+ granular permissions handled |
| Foreground service | ✅ Implemented | Doze mode handling missing in service itself |
| WorkManager scheduling | ✅ Implemented | Network constraints now properly configured |
| Battery/network constraints | ✅ Implemented | WiFi-only toggle wired to WorkManager constraints |
| Go file scanning | ❌ Not implemented | Gomobile has hash/metadata but no scan function exposed — relies entirely on Kotlin MediaScanner |
| Go hashing | ✅ Implemented | SHA256 works via gomobile |
| Go metadata extraction | ⚠️ Partially implemented | MIME type only, EXIF missing |
| Go upload (single file) | ✅ Implemented | Multipart with retry |
| Go chunked upload | ⚠️ Partially implemented | Exists in Go but unused — Android uses OkHttp instead |
| File descriptor / path passing | ⚠️ Mixed | Gomobile takes file paths, Kotlin MediaScanner works via ContentResolver URIs |

---

## 5. Recommendations for Next Steps

### Phase 1: Critical Fixes (Pre-Release)
1. Implement EXIF parsing in Go for capture time, camera model, GPS coordinates
2. Add dynamic FileObserver directory configuration from user preferences
3. Implement actual server-side delete in `deleteMedia` — call the API endpoint before deleting local records

### Phase 2: Feature Completeness
4. Wire up Settings screen logout button to clear auth tokens and navigate back to login
5. Replace placeholder storage info with real database counts (pending, uploaded items)
6. Clean up unused variables in ForceMediaIndexer.kt and UploadManager.kt — either remove or use them

### Phase 3: Polish
7. Remove dead gomobile chunk upload code or integrate it properly as a fallback
8. Add structured error response parsing (`ErrorResponse` DTO with code/message)
9. Consider doze mode handling in the foreground service for battery optimization compatibility
