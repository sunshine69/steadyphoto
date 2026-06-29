# SteadyPhoto Android Sync Strategy & Roadmap

## Executive Summary
The project is transitioning from a **"Client-Driven Utility Model"** to an **"Engine-Oriented Orchestration Model."** 

Currently, your Android app acts as the "brain," using Go only as a set of helper functions (hashing/EXIF) while Kotlin manages the database, scheduling, and networking. The new design flips this: the **Go Sync Engine becomes the brain**, managing its own state, concurrency, and sync logic via a single orchestration call from Android. This is significantly more robust and efficient for large-scale media synchronization.

---

## 1. Gap Analysis (Current vs. Proposed)

| Feature | Current Implementation (`readme-android.md`) | New Proposed Design (`readme-new-android.md` & Image) | Impact / Required Change |
| :--- | :--- | :--- | :--- |
| **Android Control Flow** | Periodic `WorkManager` tasks & manual repository calls via Kotlin. | Foreground Service with persistent notification + real-time `ContentObserver`. | **Major Refactor**: Move logic from Workers to a long-running service that handles lifecycle/notifications. |
| **Gomobile Interface** | Granular, low-level utilities (`MediaHasher`, `ExifParser`) called by Kotlin code. | A single high-level orchestration call: `StartSync(...)` with progress callbacks. | **Architecture Shift**: Redesign bindings from "Toolbox" to "Engine." Most existing Kotlin sync logic will be replaced by a callback listener. |
| **Local State (DB)** | Android Room Database (Kotlin side). | SQLite within Go (`modernc.org/sqlite`) managed inside the engine. | **Data Migration**: The source of truth for what is synced moves from the Android app to the Go binary. |
| **Sync Strategy** | Sequential: Scan $\rightarrow$ Check DB $\rightarrow$ Upload one-by-one via Multipart. | Diffing Engine: Local Index vs. Remote Manifest (JSON) comparison. | **Complexity Increase**: Requires implementing a "manifest" protocol and efficient diffing logic in Go to handle thousands of files. |
| **Upload Protocol** | Standard HTTP `multipart/form-data`. | **TUS Protocol** (Resumable, byte-range based uploads). | **Protocol Change**: Replace the existing multipart upload implementation with TUS client support (Go side) and ensure backend compatibility. |
| **Concurrency** | Managed by Android (one file at a time per worker typically). | Go goroutines + buffered channels for high-performance concurrent uploading. | **Performance Boost**: The engine will natively handle parallel uploads without taxing the JVM thread pool as heavily. |

---

## 2. Development Roadmap

### Phase 1: Backend & Protocol Foundation (Prerequisites)
*   **TUS Support**: Update/verify the backend can handle TUS protocol requests (handling `PATCH` with offsets).
*   **Manifest Endpoint**: Implement a lightweight API endpoint (`GET /api/v1/sync/manifest`) that returns a compressed JSON list of current file IDs and hashes. This is essential for the "diffing" strategy to work without downloading everything.

### Phase 2: Go Sync Engine Overhaul (The Core)
*   **Internal State**: Integrate `modernc.org/sqlite` into the Go module to manage the local index internally.
*   **Diffing Logic**: Implement the logic that compares the Local SQLite DB against the Remote Manifest JSON.
*   **Concurrency Model**: Build the buffered channel architecture and worker pool for concurrent uploads within Go.
*   **TUS Client**: Write or integrate a TUS client in Go to handle resumable byte-range transfers.

### Phase 3: Gomobile Bridge Redesign (The Interface)
*   **API Refactor**: Define the new `SyncCallback` interface and wrap it into the `.aar`.
*   **Orchestration Function**: Implement the primary `StartSync(rootPath, token, callback)` function that kicks off the internal Go pipeline.

### Phase 4: Android Layer Transformation (The Shell)
*   **Foreground Service**: Build a robust service to manage the lifecycle of the sync process and maintain the "Syncing X/Y..." notification.
*   **ContentObserver**: Implement real-time monitoring for media changes in `DCIM` to trigger immediate, low-priority syncs via WorkManager or direct Engine calls.
*   **Callback UI**: Connect the Gomobile progress callbacks to Jetpack Compose components for a reactive status dashboard.

---

## 3. Critical Missing Pieces (Immediate Priorities)

1.  **The "Brain" Migration Logic**: You currently have significant logic in `MediaScanner.kt` and various Android repositories. **None of this will be needed once the Go Engine is implemented.** Your first task should be deciding when to stop building features on the Kotlin side and start rebuilding them in Go.
2.  **CGO-Free SQLite**: Since you are using Gomobile, standard `go-sqlite3` (which uses CGO) might cause build headaches or increase binary size significantly for Android. Implementing/configuring **pure Go SQLite (`modernc.org/sqlite`)** is a high priority to ensure smooth cross-compilation and stability on mobile devices.
3.  **TUS Compatibility Check**: Before writing the client, confirm your backend can support `PATCH` requests with specific offsets—this is the core requirement for resumable uploads that differentiates TUS from standard multipart/form-data.
