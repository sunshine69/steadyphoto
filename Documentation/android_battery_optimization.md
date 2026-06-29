# Android Battery Optimization Initiative

## 🎯 Objective
Minimize energy consumption and CPU/Radio usage of the SteadyPhoto Android sync client during media detection and synchronization tasks.

## 🛠 Strategies Implemented & Planned

### 1. Incremental Media Scanning (Delta Scans) — [IN PROGRESS]
**Problem:** Full scans of `MediaStore` are resource-intensive, causing high CPU spikes and prolonged battery drain every time the scanner runs.  
**Solution:** Only scan for media created after the most recent item already stored in our local database.

*   **Work Done (Data Layer):** 
    *   Updated `MediaItemDao.kt` to include `getMaxCreatedAt()`. This allows us to efficiently fetch the latest timestamp from the local Room database using a single SQL query: `SELECT MAX(createdAt) FROM media_items`.
*   **Next Steps:**
    *   Integrate this timestamp into `MediaScannerWorker`. 
    *   Modify MediaStore queries to include a filter on the `DATE_ADDED` or `DATE_MODIFIED` columns, ensuring we only process "new" files.

### 2. Intelligent Background Scheduling — [DESIGNED]
**Strategy:** Offload heavy work to when the device is in an optimal state for battery usage.
*   **WorkManager Constraints:** Use Android's `WorkManager` API to enforce:
    *   `setRequiredNetworkType(NetworkType.UNMETERED)` (Prefer WiFi).
    *   `setRequiresBatteryNotLow(true)`.
    *   `setRequiresCharging(true)` (Optional/User-configurable setting for heavy syncs).
*   **ContentObserver Integration:** Instead of polling, use a `ContentObserver` to trigger lightweight incremental scans only when the system detects changes in media directories.

### 3. Efficient Hashing & Processing — [PLANNED]
**Strategy:** Reduce CPU load during the deduplication phase.
*   **Native Performance:** Continue leveraging **Gomobile bindings** for SHA-256 hashing. Go's native implementation is highly optimized and can be run more efficiently than pure Kotlin/Java implementations for large file streams.

---

## 📊 Progress Tracking

| Feature | Status | Impact Level | Notes |
| :--- | :---: | :---: | :--- |
| **Incremental Scan (DAO)** | ✅ Done | High | Added `getMaxCreatedAt()` to support delta scans. |
| **Scanner Integration** | 🚧 WIP | High | Integrating timestamp filter into MediaStore queries. |
| **WorkManager Constraints**| 📅 Planned | Medium | Configuring constraints for background workers. |
| **Native Hashing (Go)** | ✅ Done | Medium | Gomobile bindings are ready to handle heavy lifting. |

---

## 📝 Developer Notes for Next Session
*   When implementing the `MediaScannerWorker` update, ensure that if `getMaxCreatedAt()` returns `null` (first run), it falls back to a full scan of the device media.
*   Be careful with MediaStore's timestamp granularity; sometimes small differences in milliseconds might lead to skipping files or re-scanning them unnecessarily. Use `>=` logic where appropriate.
