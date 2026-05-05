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
* **Frontend:** **Next.js (React) + Tailwind CSS** (Robust, SSR/ISR capabilities, excellent developer experience).

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

### Phase 2: Modern Web Interface [IN PROGRESS]
* [ ] **Next.js Scaffolding**: Initialize project with Tailwind CSS and TypeScript.
* [ ] **API Integration Layer**: Robust fetching service using TanStack Query (React Query).
* [ ] **Photo Grid**: High-performance, infinite-scroll masonry grid.
* [ ] **Advanced Photo Viewer**: Full-screen immersive viewer with EXIF details.
* [ ] **Timeline/Calendar Navigation**: Interactive time-based browsing.
* [ ] **Search Interface**: Real-time search for filenames and metadata.

### Phase 3: Media Optimization & AI [UPCOMING]
* [x] **Thumbnail Engine**: Decoupled architecture with `ImageEngine` interface.
* [ ] **AI Face Detection**: ONNX-based worker to find faces and store bounding boxes.
* [ ] **Semantic Search**: CLIP embedding generation for "search by description."

### Phase 4: Mobile & Sync [UPCOMING]
* [ ] **Mobile Client**: Cross-platform app (React Native or Flutter).
* [ ] **Sync Protocol**: Efficient delta-based file uploading.

---

## 5. Current Milestone Summary
**Status:** Pivoting Frontend from Vue/Vite to **Next.js/Tailwind** for better reliability and scale.

**Next Immediate Goal:** Scaffolding the Next.js application.

**Verified Workflow (Media Optimization):**
1. `Scanner` $\rightarrow$ Finds file $\rightarrow$ Extracts EXIF $\rightarrow$ Copies to `storage/YYYY/MM/DD/` $\rightarrow$ Saves **relative path** to DB.
2. `Worker` $\rightarrow$ Polls `pending` job $\rightarrow$ `ThumbnailProcessor` resolves paths $\rightarrow$ `ImageEngine` resizes $\rightarrow$ Saves to `storage/.thumbnails/YYYY/MM/DD/`.
3. `API` $\rightarrow$ Fetches relative path $\rightarrow$ `StorageService` resolves absolute path $\rightarrow$ `http.ServeFile` streams the file to the client.

**Result:** The background processing pipeline is stable, testable, and extensible.

## 🚀 Next.js Frontend Implementation Plan

Since we are starting fresh, we will follow a structured approach to ensure stability and high performance (essential for a photo app).

### **Step 1: Scaffolding & Environment (Immediate)**
*   **Action:** Initialize a new Next.js project using `npx create-next-app@latest`.
*   **Config:** TypeScript, Tailwind CSS, ESLint, and `src/` directory.
*   **Clean up:** Remove the old `web/` directory (once we are sure we don't need anything from it).
*   **Env Setup:** Create `.env.local` with `NEXT_PUBLIC_API_BASE_URL=http://localhost:8081/api/v1`.

### **Step 2: The Data Layer (Reliability)**
*   **Tooling:** Install `@tanstack/react-query` (React Query). This is non-negotiable for a photo app to handle caching, background refetching, and loading states without manual headache.
*   **Service Layer:** Create a `lib/api.ts` file that mimics the logic of the previous service but with strict TypeScript types.
*   **Types:** Define `Photo` and `ListPhotosResponse` interfaces to ensure end-to-end type safety.

### **Step 3: Core UI Components (The "Wow" Factor)**
*   **Masonry Grid:** Implement a high-performance grid using `react-plock` or a similar lightweight masonry library. This is much better than a standard CSS grid for photos of varying aspect ratios.
*   **Image Optimization:** Utilize Next.js `<Image />` component for smart resizing and lazy loading (though for external API images, we'll use standard `<img>` with `loading="lazy"` or a custom implementation to avoid complex domain configuration).
*   **The "Lightbox":** A high-quality, full-screen modal using `framer-motion` for smooth transitions (the "pop" effect when clicking a photo).

### **Step 4: Advanced Features (Phase 2 Completion)**
*   **Infinite Scroll:** Integrate `react-intersection-observer` to trigger the next page fetch automatically.
*   **Search/Filter:** A command-palette style search (like macOS Spotlight/Raycast) for finding photos by date or name.

