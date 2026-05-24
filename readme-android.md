# Android App Development Plan: SteadyPhoto Mobile

## 1. Architecture Overview

SteadyPhoto Mobile will follow a **Local-First** architecture, leveraging Kotlin for the UI layer and Gomobile to wrap high-performance Go libraries for on-device backend tasks. This approach ensures fast media processing (hashing, EXIF parsing, encryption) without network latency, while keeping networking separate for eventual cloud sync with the existing web backend.

### Core Principles
- **Local-First**: All metadata, thumbnails, and user sessions are cached locally via SQLite/Room.
- **Gomobile Backend**: CPU-intensive operations (SHA256 deduplication, EXIF extraction, crypto) run natively via Go bindings.
- **Kotlin + Compose UI**: Modern, declarative Android UI with Material 3 design system.
- **Sync-Ready**: Networking layer abstracted to easily connect with the existing `/api/v1/media/upload` and admin endpoints when cloud sync is enabled.

---

## 2. Tech Stack & Dependencies

| Layer | Technology | Purpose |
|-------|------------|---------|
| **Language** | Kotlin (JVM 17+) | Primary app language |
| **UI Framework** | Jetpack Compose + Material 3 | Declarative, responsive UI |
| **Local Database** | Room (SQLite) | Cache media metadata, sync state, user sessions |
| **Networking** | Ktor Client / Retrofit | HTTP requests for auth & cloud sync |
| **Gomobile Bindings** | Go 1.22+ → `.aar` library | Native SHA256, EXIF parsing, crypto, file I/O |
| **Image/Video Loading** | Coil + ExoPlayer | Efficient media rendering & playback |
| **State Management** | ViewModel + Kotlin Flow | Reactive UI state handling |

---

## 3. Gomobile Integration Strategy

Gomobile will wrap specific Go packages into a Java/Kotlin-accessible `.aar` library. This avoids JNI boilerplate and provides type-safe bindings.

### What to Wrap in Go (`mobile/` package)
```go
package mobile

import (
    "crypto/sha256"
    "encoding/hex"
    "io"
    "os"
    
    "github.com/xor-gate/go-exif/v3"
)

// MediaHasher computes SHA256 hash of a file path
func MediaHasher(filePath string) (string, error) {
    f, err := os.Open(filePath)
    if err != nil { return "", err }
    defer f.Close()
    
    h := sha256.New()
    if _, err := io.Copy(h, f); err != nil { return "", err }
    return hex.EncodeToString(h.Sum(nil)), nil
}

// ExifParser extracts basic metadata from image/video files
func ExifParser(filePath string) (map[string]string, error) {
    // Use go-exif or mp4ff for video duration/resolution
    // Returns: map of key-value pairs (camera, iso, gps, duration, etc.)
}

// CryptoUtils handles JWT token signing/verification locally
func GenerateSecureToken(length int) (string, error) { ... }
```

### Build & Integration Process
1. **Generate AAR**: `gomobile bind -target=android -o mobile-bindings.aar ./mobile`
2. **Android Studio Setup**: Drop `.aar` into `app/libs/`, add to `build.gradle.kts`:
   ```kotlin
   dependencies {
       implementation(files("libs/mobile-bindings.aar"))
   }
   ```
3. **Kotlin Usage**: Direct method calls via generated Java bridge:
   ```kotlin
   val hash = MobileBindings.MediaHasher(filePath)
   val exifData = MobileBindings.ExifParser(filePath)
   ```

### Performance & Limitations Notes
- ✅ Excellent for CPU-bound tasks (hashing, parsing, encryption)
- ⚠️ No direct access to Android `Context` or UI threads from Go (use Kotlin bridge for threading/callbacks)
- ✅ Gomobile 1.20+ supports most standard library packages (`os`, `io`, `crypto`)

---

## 4. Core Feature Implementation Plan

### Phase 1: Local Media Management (Weeks 1-4)
| Task | Description | Tech/Component |
|------|-------------|----------------|
| **Device Scanning** | Background service to scan `DCIM/`, `Pictures/`, etc. for photos/videos | `MediaScannerConnection` + Kotlin Coroutines |
| **SHA256 Deduplication** | Compute hashes via Gomobile, store in Room to prevent duplicate uploads | `MobileBindings.MediaHasher()` → Room DAO |
| **EXIF Extraction** | Parse camera model, ISO, GPS, capture time using Go bindings | `MobileBindings.ExifParser()` → Map to `MediaEntity` |
| **Local Media Grid** | Compose-based responsive grid with lazy loading & pagination | `LazyVerticalGrid`, Coil for thumbnails |
| **Video Playback** | Native seekable playback with range-request simulation (local) | ExoPlayer + Media3 |

### Phase 2: Cloud Sync Integration (Weeks 5-8)
| Task | Description | Tech/Component |
|------|-------------|----------------|
| **Authentication** | Login screen, JWT storage, session refresh logic | Ktor Client + Retrofit Auth Interceptor |
| **Upload Queue** | Batch uploads with progress tracking & retry logic | WorkManager + `POST /api/v1/media/upload` |
| **Sync State Management** | Track local vs remote IDs, conflict resolution | Room `SyncStatus` enum + timestamp reconciliation |
| **Album Sync** | Fetch remote albums, map to local UI filters | REST API → ViewModel state flow |
| **Admin Panel (Optional)** | Read-only user list & status view for admin accounts | Reuse web API endpoints, Compose tables |

---

## 5. Project Structure

```
android/
├── app/
│   ├── src/main/java/com/steadyphoto/mobile/
│   │   ├── di/              # Hilt/Koin dependency injection
│   │   ├── data/            # Room DAOs, DTOs, Repository interfaces
│   │   ├── local/           # Room database setup & migrations
│   │   ├── remote/          # Ktor/Retrofit API clients
│   │   ├── ui/              # Compose screens (Login, Grid, Player, Settings)
│   │   ├── viewmodel/       # State holders + Flow emitters
│   │   └── worker/          # Background scanning & upload workers
│   ├── libs/                # gomobile-generated .aar files
│   └── build.gradle.kts     # Dependencies & Gomobile plugin config
├── gomobile-bindings/       # Go source code for mobile bindings
│   ├── go.mod
│   └── mobile/              # Exported functions/classes
├── gradle/                  # Wrapper & version catalogs
└── build.gradle.kts         # Root project config
```

---

## 6. Build & CI/CD Pipeline

### Local Development
- **Android Studio**: Standard Gradle sync → Run on emulator/device
- **Gomobile Regeneration**: `./gradlew :app:generateMobileBindings` (custom task wrapping `gomobile bind`)
- **Hot Reload**: Compose preview + Android Emulator live reload

### Automated CI/CD (GitHub Actions)
```yaml
name: Android Build & Test
on: [push, pull_request]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - run: cd gomobile-bindings && gomobile bind -target=android -o ../app/libs/mobile.aar ./mobile
      - uses: gradle/gradle-build-action@v3
      - run: ./gradlew assembleDebug test
```

---

## 7. Roadmap & Milestones

| Phase | Timeline | Deliverables | Success Criteria |
|-------|----------|--------------|------------------|
| **1. Foundation** | Week 1-2 | Gomobile AAR generation, Room setup, Compose navigation shell | App launches, binds Go functions, DB initializes |
| **2. Local Media Core** | Week 3-4 | Device scanner, SHA256 dedup UI, EXIF parsing, local grid view | Scans 10k+ files in <30s, deduplicates correctly |
| **3. Playback & UX** | Week 5-6 | Video player, image viewer, settings screen, dark mode polish | Smooth scrolling, zero ANRs on mid-range devices |
| **4. Cloud Sync** | Week 7-8 | Auth flow, upload queue with progress, remote album fetch | Uploads complete without crashes, sync state persists |
| **5. Beta Release** | Week 9-10 | Internal testing, performance profiling, Play Store listing prep | Passes CTS, <2% crash rate, ready for closed testing |

---

## 8. Risk Mitigation & Considerations

| Risk | Impact | Mitigation Strategy |
|------|--------|---------------------|
| **Gomobile AAR size** | Increases APK footprint (~5-10MB) | Use `arm64-v8a` only initially, strip unused Go symbols |
| **Threading mismatches** | Go blocks Android main thread | Wrap all Gomobile calls in `Dispatchers.IO` or background workers |
| **SQLite vs PostgreSQL schema drift** | Sync conflicts later | Design Room entities with `remote_id` UUID columns & sync timestamps from day 1 |
| **Battery drain during scanning** | Poor UX on large libraries | Implement adaptive scanning (only scan new/modified files), throttle I/O |

---

## Next Steps
1. Initialize Android Studio project with Kotlin + Compose template
2. Set up `gomobile-bindings/` Go module & export `MediaHasher`, `ExifParser`
3. Generate `.aar` and integrate into Android app dependencies
4. Scaffold Room database schema matching SteadyPhoto's `media` table structure
5. Begin Phase 1 implementation (local scanning + deduplication UI)
