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

---

## 4. Implementation Roadmap

### Phase 1: Core Foundation [COMPLETED]
* [x] **Domain Models**: Defined `Photo`, `Face`, and `Album` entities.
* [x] **Database Schema**: PostgreSQL with UUIDs, JSONB, and `pgvector` support.
* [x] **Scanner Engine**: Symlink-aware, deduplicating, EXIF-extracting scanner.
* [x] **Storage Service**: Robust path resolution (handles relative/absolute/duplicate prefixes).
* [x] **REST API**: 
    * `GET /api/v1/photos` (List/Paginate)
    * `GET /api/v1/photos/{id}` (Metadata)
    * `GET /api/v1/photos/{id}/original` (File Stream)

### Phase 2: Media Optimization & AI [NEXT]
* [ ] **Thumbnail Engine**: Integration of `libvips` to generate preview sizes.
* [ ] **AI Face Detection**: ONNX-based worker to find faces and store bounding boxes.
* [ ] **Semantic Search**: CLIP embedding generation for "search by description."

### Phase 3: Mobile & Sync [UPCOMING]
* [ ] **Android Client**: Kotlin/Jetpack Compose app for background uploads.
* [ ] **Sync Protocol**: Efficient delta-based file uploading.

---

## 5. Current Milestone Summary
**Status:** The "Full Loop" is functional.
**Verified Workflow:**
1. `Scanner` $\rightarrow$ Finds file $\rightarrow$ Extracts EXIF $\rightarrow$ Copies to `storage/YYYY/MM/DD/` $\rightarrow$ Saves **relative path** to DB.
2. `API` $\rightarrow$ Fetches relative path $\rightarrow$ `StorageService` resolves absolute path $\rightarrow$ `http.ServeFile` streams the file to the client.
**Result:** System is stable, predictable, and ready for media processing.
