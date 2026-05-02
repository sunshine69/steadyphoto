# SteadyPhoto Design Document

## 1. Core Philosophy
SteadyPhoto is built on the principle of **"Filesystem as the Source of Truth."** 
- **Human Readable Storage:** Photos are stored in a hierarchical structure (`/YYYY/MM/DD/filename.ext`) to ensure they are accessible without any software.
- **Database as an Index:** The database (PostgreSQL) is a high-performance cache and index. If lost, the entire state can be rebuilt by scanning the filesystem.
- **Strict Versioning:** The API must be versioned (e.g., `/v1/`) to ensure long-term compatibility for clients (Android, Web, CLI).

## 2. Architecture Overview
The system follows a decoupled, modular architecture designed for concurrent development and scalability.

### 2.1 Components
1.  **`cmd/server` (API Server):** A Go-based RESTful API that serves metadata, handles uploads, and provides image streaming.
2.  **`cmd/worker` (Background Processor):** A Go-based worker that listens to a Redis queue to perform heavy tasks:
    *   Thumbnail generation (via `libvips`).
    *   AI Face Detection (via `ONNX Runtime`).
    *   Metadata extraction.
3.  **`cmd/scanner` (Filesystem Crawler):** A utility to scan existing directories and sync them with the database.
4.  **`cmd/cli` (Management Tool):** A command-line interface for administrators to manage the system.

### 2.2 Modular Design (Internal Packages)
To allow parallel development, the system is split into discrete internal modules:

*   **`internal/domain`**: Contains the core business logic, entities (Photo, Album, Face), and interface definitions. This is the "glue" that all other modules depend on.
*   **`internal/storage`**: Handles the physical movement and organization of files on the disk.
*   **`internal/database`**: Implementation of the persistence layer (PostgreSQL).
*   **`internal/api`**: HTTP handlers and routing logic.
*   **`internal/processor`**: Logic for image manipulation and heavy lifting.
*   **`internal/scanner`**: Logic for traversing filesystems and detecting changes.
*   **`internal/ai`**: Interface and implementation for ONNX-based AI models.
*   **`internal/config`**: Centralized configuration management.

## 3. Data Model (High-Level)
- **Photo**: `id (UUID)`, `original_path`, `checksum (sha256)`, `filename`, `mime_type`, `exif_data (JSONB)`, `created_at`.
- **Face**: `id (UUID)`, `photo_id`, `bounding_box (JSONB)`, `embedding (vector)`, `is_person_identified (bool)`.
- **Album**: `id`, `name`, `photo_ids (array/join table)`.

## 4. Implementation Roadmap (Phased Approach)

### Phase 1: The Core Engine (Foundation)
- [ ] Define `domain` entities and interfaces.
- [ ] Implement `storage` module (File mover + YYYY/MM/DD logic).
- [ ] Implement `database` module (PostgreSQL schema).
- [ ] Implement `scanner` (The "Rebuild from Filesystem" logic).

### Phase 2: API & Viewing
- [ ] Implement `api` (v1) with basic CRUD for photos.
- [ ] Implement `processor` for thumbnail generation using `libvips`.
- [ ] Create the `server` entry point.

### Phase 3: Background Tasks & AI
- [ ] Setup Redis and the `worker` module.
- [ ] Implement the job queue for image processing.
- [ ] Integrate `ai` module using ONNX for face detection.

### Phase 4: Client Ecosystem
- [ ] Android Client (Kotlin).
- [ ] Web Gallery (React/Vue).

## 5. Development Guidelines for Agents
1.  **Dependency Rule:** Modules should depend on `internal/domain` interfaces, not on each other's concrete implementations. This allows mocking during testing.
2.  **Error Handling:** Always return wrapped errors to provide context in the logs.
3.  **Concurrency:** Use Go channels and context for all long-running processes (scanner, worker).
4.  **Testing:** Every module in `internal/` must have a corresponding `_test.go` file.
