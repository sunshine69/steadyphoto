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
* **Frontend:** Web-based SPA (Single Page Application).

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

### Phase 2: Web Interface & UX [IN PROGRESS]
* [ ] **Web Dashboard**: Simple, responsive grid view for browsing photos.
* [ ] **Photo Viewer**: Lightbox component to view images and EXIF metadata.
* [ ] **Timeline Navigation**: Ability to browse by Year/Month/Day.
* [ ] **Basic Search**: Text-based search against filenames and metadata.

### Phase 3: Media Optimization & AI [UPCOMING]
* [x] **Thumbnail Engine**: Decoupled architecture with `ImageEngine` interface.
* [ ] **AI Face Detection**: ONNX-based worker to find faces and store bounding boxes.
* [ ] **Semantic Search**: CLIP embedding generation for "search by description."

### Phase 4: Mobile & Sync [UPCOMING]
* [ ] **Android/iOS Client**: Mobile apps for viewing and background uploads.
* [ ] **Sync Protocol**: Efficient delta-based file uploading.

---

## 5. Current Milestone Summary
**Status:** Transitioning from Backend Core to **Web User Interface**.

**Verified Workflow (Media Optimization):**
1. `Scanner` $\rightarrow$ Finds file $\rightarrow$ Extracts EXIF $\rightarrow$ Copies to `storage/YYYY/MM/DD/` $\rightarrow$ Saves **relative path** to DB.
2. `Worker` $\rightarrow$ Polls `pending` job $\rightarrow$ `ThumbnailProcessor` resolves paths $\rightarrow$ `ImageEngine` resizes $\rightarrow$ Saves to `storage/.thumbnails/YYYY/MM/DD/`.
3. `API` $\rightarrow$ Fetches relative path $\rightarrow$ `StorageService` resolves absolute path $\rightarrow$ `http.ServeFile` streams the file to the client.

**Result:** The background processing pipeline is stable, testable, and extensible.
