# SteadyPhoto Design Document

**Goal:** A reliable, long-term, "Filesystem-First" photo management system. The database is an index; the filesystem is the source of truth.

---

## 1. Core Philosophy
* **Filesystem as Source of Truth:** Files are organized in a human-readable hierarchy: `/YYYY/MM/DD/filename.ext`.
* **Database as Index:** The database is a cache of metadata. If the database is lost, the system can be fully reconstructed by scanning the filesystem.
* **Predictability & Stability:** Use versioned APIs and avoid breaking changes.
* **No Hash-Based Naming:** Files keep their original names for easy backup and manual browsing.

---

## 2. Technology Stack
* **Language:** Golang (High performance, single binary).
* **Database:** PostgreSQL (Reliable, JSONB for metadata, `pgvector` for AI).
* **Storage Access:** `sqlx` (Transparent SQL) + `pgx` (High-performance driver).
* **API:** RESTful with versioning (`/api/v1/...`).
* **Image Processing:** `libvips` (Fast, low memory).
* **AI Engine:** ONNX Runtime (Local, high-performance AI inference).
* **Frontend:** **Angular** (Current Implementation) / **Next.js (Planned Pivot)**.

---

## 3. System Architecture

### A. Ingestion (The Scanner)
* Scans source directories.
* Resolves symlinks.
* Deduplicates via SHA256.
* Extracts EXIF metadata.
* Moves/Copies files to organized `storage/YYYY/MM/DD/` structure.
* Stores **relative paths** in the database.

### B. Storage Service
* Translates relative paths from the DB into absolute system paths.
* Handles path normalization to prevent directory traversal and duplication.

### C. API Layer
* Provides metadata and file streaming.
* Supports `Range` requests (crucial for video seeking).
* Provides search capabilities (Text/Metadata/Date).

---

## 4. Implementation Roadmap

### Phase 1: Core Foundation [COMPLETED]
* [x] **Domain Models**: Defined `Photo`, `Face`, and `Album` entities.
* [x] **Database Schema**: PostgreSQL with UUIDs, JSONB, and `pgvector` support.
* [x] **Scanner Engine**: Symlink-aware, deduplicating, EXIF-extracting scanner.
* [x] **Storage Service**: Robust path resolution.
* [x] **REST API**: Basic photo listing and streaming.

### Phase 2: Web Interface [IN PROGRESS]
* [x] **Angular Scaffolding**: Initial project setup.
* [x] **Photo Grid**: Responsive grid layout.
* [x] **Navigation**: Top navbar with search and main content routing.
* [x] **Search**: Client-side filename search implemented.
* [x] **Pagination**: Bottom bar with "Jump to Page" functionality.
* [x] **Scroll Restoration**: Maintains position when returning from photo details.
* [ ] **Advanced Photo Viewer**: Full-screen immersive viewer with EXIF details.
* [ ] **Timeline/Calendar Navigation**: Interactive time-based browsing.

### Phase 3: Media Optimization & AI [UPCOMING]
* [x] **Thumbnail Engine**: Decoupled architecture with `ImageEngine` interface.
* [ ] **AI Face Detection**: ONNX-based worker to find faces and store bounding boxes.
* [ ] **Semantic Search**: CLIP embedding generation for "search by description."

### Phase 4: Mobile & Sync [UPCOMING]
* [ ] **Mobile Client**: Cross-platform app (React Native or Flutter).
* [ ] **Sync Protocol**: Efficient delta-based file uploading.

---

## 5. Current Milestone Summary
**Status:** Basic Web UI is functional and implemented using **Angular**.

**Key Accomplishments:**
* Stable background processing pipeline (Scanner $\rightarrow$ Worker $\rightarrow$ API).
* Functional photo gallery with pagination and jump-to-page.
* Real-time filename search.
* Smooth user experience with scroll position preservation.

**Next Immediate Goal:** Implement the Advanced Photo Viewer (Lightbox) with EXIF metadata display.

**Verified Workflow (Media Optimization):**
1. `Scanner` $\rightarrow$ Finds file $\rightarrow$ Extracts EXIF $\rightarrow$ Copies to `storage/YYYY/MM/DD/` $\rightarrow$ Saves **relative path** to DB.
2. `Worker` $\rightarrow$ Polls `pending` job $\rightarrow$ `ThumbnailProcessor` resolves paths $\rightarrow$ `ImageEngine` resizes $\rightarrow$ Saves to `storage/.thumbnails/YYYY/MM/DD/`.
3. `API` $\rightarrow$ Fetches relative path $\rightarrow$ `StorageService` resolves absolute path $\rightarrow$ `http.ServeFile` streams the file to the client.
