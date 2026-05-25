# SteadyPhoto Android Client - Testing Guide

This document provides step-by-step instructions for building, installing, and testing the Android Sync Client up to **Phase 1.5** (MediaStore Integration & Permissions) and early **Phase 2** (Scanning & Deduplication).

---

## 📋 Prerequisites

Before testing, ensure your development environment is correctly configured:
*   **JDK**: OpenJDK 17 installed at `/opt/jdk-17` (AGP 8.2 compatibility)
*   **Android Studio**: Hedgehog or Iguana+ with Android SDK 34
*   **Gradle Wrapper**: `./gradlew` executable in project root
*   **Test Device/Emulator**: 
    *   Physical device running Android 13 (API 33) or higher, OR
    *   Emulator with Google Play Services & API 34+

---

## 🛠️ 1. Build & Install the App

### Clean Build
```bash
cd /mnt/portdata/stevek-src/steadyphoto/android
./gradlew clean assembleDebug
```
*Expected Output*: `app-debug.apk` generated in `android/app/build/outputs/apk/debug/` (~30-35MB)

### Install via ADB
```bash
adb install -r app/build/outputs/apk/debug/app-debug.apk
```
*Verify Installation*: Check device for "SteadyPhoto Sync" launcher icon.

---

## 📱 2. Test: Permissions & MediaStore Scanning

This tests `PermissionHelper` and `MediaScannerWorker`.

### Step A: Grant Runtime Permissions
1. Open App → Navigate to **Settings** or trigger a sync from Home screen.
2. Observe permission dialog for `READ_MEDIA_IMAGES` and `READ_MEDIA_VIDEO`.
3. Tap **Allow**.
4. *Verification*: No "Permission Denied" toast/snackbar should appear.

### Step B: Populate Emulator with Test Media
Use ADB to push test files into the emulator's shared storage (simulating DCIM/Pictures):
```bash
# Create directories if they don't exist
adb shell mkdir -p /sdcard/DCIM/Camera
adb shell mkdir -p /sdcard/Pictures/TestUploads

# Push test images/videos (replace with actual files)
adb push ./test-assets/test_image_1.jpg /sdcard/DCIM/Camera/
adb push ./test-assets/test_video_1.mp4 /sdcard/Pictures/TestUploads/
```

### Step C: Trigger MediaScannerWorker
1. Open App → Go to **Settings** → Tap **"Scan Now"** (or trigger via UI button).
2. Monitor Logcat in Android Studio or terminal:
   ```bash
   adb logcat -s "MediaScannerWorker" "PermissionHelper" "SyncDatabase"
   ```
3. *Expected Logs*:
   ```
   D/MediaScannerWorker: Starting scan...
   I/PermissionHelper: Permissions granted successfully.
   D/MediaScannerWorker: Found 2 new items (1 image, 1 video)
   D/SyncDatabase: Inserted MediaItemEntity for /sdcard/DCIM/Camera/test_image_1.jpg
   ```

### Step D: Verify Scan Results
* Check UI status screen shows "Scan Complete: 2 items found".
* Check Room DB (see Section 3).

---

## 💾 3. Test: Database Integration (Room)

Verify that `SyncDatabase` and `MediaItemDao` are correctly persisting scanned media.

### Method A: Android Studio Database Inspector
1. Open **View → Tool Windows → App Inspection**.
2. Select the running SteadyPhoto process.
3. Navigate to **SQLite** tab.
4. Query: 
   ```sql
   SELECT * FROM MediaItemEntity;
   ```
5. *Expected*: Rows with `uri`, `localPath`, `hash` (placeholder), and `upload_status = 'PENDING'`.

### Method B: ADB Shell Direct Access
```bash
# Find database file path
adb shell run-as com.steadyphoto.sync ls -l /app_databases/

# Copy DB to host for inspection
adb pull /data/user/0/com.steadyphoto.sync/app_databases/sync_database.db ./test_sync.db

# Open with SQLite CLI or DB Browser
sqlite3 test_sync.db "SELECT * FROM MediaItemEntity;"
```

---

## 🔗 4. Test: Gomobile Bindings (.aar) Integration

Verify that the Go-generated `mobile-bindings.aar` is correctly linked and accessible from Kotlin.

### Step A: Verify Library Presence
```bash
ls -l android/app/libs/mobile-bindings.aar
```
*Expected*: File exists, ~5-10MB in size (arm64-v8a).

### Step B: Build Verification
Run a clean build to ensure Gradle resolves the `.aar`:
```bash
./gradlew :app:dependencies --configuration debugRuntimeClasspath | grep mobile-bindings
```
*Expected*: Output shows `mobile-bindings.aar` in dependencies tree.

### Step C: Unit Test Hashing Function (Placeholder)
If a test harness exists, run:
```kotlin
// In Android Studio Run Configuration → JUnit Test
@Test
fun testMediaHasher() {
    val hash = MobileBindings.MediaHasher("/sdcard/DCIM/Camera/test_image_1.jpg")
    assertNotNull(hash) // Should return SHA256 hex string or placeholder
}
```

---

## 🎨 5. Test: UI & Settings Flow (Jetpack Compose)

### Navigation Testing
1. **Home Screen**: Verify layout, status text ("Ready", "Scanning...", "Sync Complete").
2. **Settings Screen**: 
   * Toggle **"Upload over Wi-Fi only"** → Check DataStore persistence.
   * Toggle **"Auto-upload"** → Verify WorkManager constraints update.
3. **Error States**: Simulate permission denial (via device settings) and verify graceful fallback UI.

### Compose Preview Verification
In Android Studio:
* Open `SettingsScreen.kt` & `HomeScreen.kt`.
* Use **Compose Previews** to verify Light/Dark theme rendering.
* Check `MaterialTheme` typography & color tokens match design specs.

---

## 🐛 6. Debugging Tips

### Common Issues & Fixes
| Issue | Likely Cause | Fix |
|-------|-------------|-----|
| `Permission Denied` toast on launch | Android 13+ scoped storage not handled | Verify `READ_MEDIA_IMAGES/VIDEO` in manifest & runtime check |
| `ClassNotFoundException: MobileBindings` | `.aar` not synced or ABI mismatch | Run `./gradlew clean assembleDebug`, ensure `arm64-v8a` only if testing on emulator |
| DB empty after scan | Worker scheduled but not triggered | Manually trigger via Settings UI or `adb shell am broadcast -a android.intent.action.BOOT_COMPLETED` |
| Gradle build fails with JDK error | Wrong Java version in PATH | Export `JAVA_HOME=/opt/jdk-17` before running gradlew |

### Useful ADB Commands for Debugging
```bash
# Clear app data & restart fresh
adb shell pm clear com.steadyphoto.sync
adb install -r app/build/outputs/apk/debug/app-debug.apk

# Force stop & restart worker
adb shell cmd workmanager run-now com.steadyphoto.sync.worker.MediaScannerWorker

# Monitor real-time logs
adb logcat -c && adb logcat *:V | grep -E "MediaScanner|PermissionHelper|SyncDatabase|MobileBindings"
```

---

## ✅ Acceptance Criteria Checklist

- [ ] App installs & launches without crash on API 34 emulator/device
- [ ] Runtime permissions requested & granted smoothly
- [ ] Test media pushed via ADB is detected by MediaStore scan
- [ ] `MediaScannerWorker` logs show successful scan & DB insertion
- [ ] Room DB contains entries with correct URIs, paths, and PENDING status
- [ ] `mobile-bindings.aar` resolves in Gradle dependencies
- [ ] Settings toggles persist across app restarts (DataStore)
- [ ] UI handles permission denial gracefully without crashing

---
*Last Updated: Phase 2 Completion | Environment: JDK 17, AGP 8.2, Android SDK 34*
