## SteadyPhoto Architecture Analysis

Based on my review of the codebase, here's a comprehensive analysis of the project structure, technology choices, and areas for improvement.

### Project Overview

**SteadyPhoto** is a self-hosted photo and video management system written in Go, with a PostgreSQL backend and a REST API. The project follows a clean architecture pattern with clear separation between domain, infrastructure, and application layers.

### Technology Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.22+ |
| Database | PostgreSQL (via `sqlx`) |
| API Framework | Standard `net/http` with custom routing |
| Image Processing | `bep/imagemeta` for EXIF, custom for thumbnails |
| Scheduling | `robfig/cron` for worker execution |
| Authentication | Session-based with bcrypt |
| File Storage | Local filesystem (configurable path) |
| Mobile | Go Mobile bindings for Android |

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         cmd/                                    │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────────────┐   │
│  │  server  │  │  worker  │  │  scanner │  │   migrate      │   │
│  │          │  │          │  │          │  │   cleanup      │   │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────────┬───────┘   │
│       │              │              │                 │         │
└───────┼──────────────┼──────────────┼─────────────────┼─────────┘
        │              │              │                 │
        ▼              ▼              ▼                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                      internal/                                  │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────────────┐   │
│  │   api/   │  │ database/│  │ processor│  │   scanner/     │   │
│  │          │  │          │  │          │  │                │   │
│  │ - REST   │  │ - Repos  │  │ - EXIF   │  │ - Media scan   │   │
│  │ - Auth   │  │ - Migrate│  │ - Thumb  │  │ - Upload       │   │
│  │ - Upload │  │          │  │ - Face   │  │                │   │
│  └──────────┘  └──────────┘  └──────────┘  └────────────────┘   │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────────────┐   │
│  │  domain/ │  │ storage/ │  │   ai/    │  │   utils/       │   │
│  │          │  │          │  │          │  │                │   │
│  │ - Models │  │ - Files  │  │ - Engine │  │ - Helpers      │   │
│  │ - Intfs  │  │ - Thumb  │  │ - Mock   │  │                │   │
│  └──────────┘  └──────────┘  └──────────┘  └────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
        │              │              │                 │
        ▼              ▼              ▼                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                     PostgreSQL Database                          │
│  media  │  albums  │  users  │  sessions  │  jobs  │  faces     │
└─────────────────────────────────────────────────────────────────┘
```

### Key Components

#### 1. Server (`cmd/server/main.go`)

The main API server that:
- Listens on port 8081 (configurable)
- Supports TLS (direct or behind reverse proxy)
- Includes a **cron-based worker scheduler** that periodically runs the worker binary
- Seeds an admin user on startup
- Manages the entire API surface

**Worker Integration Issue**: The server doesn't have a proper background job processor. Instead, it shells out to an external `worker` binary via cron. This is a fragile approach that doesn't integrate well with the application lifecycle.

#### 2. Worker (`cmd/worker/main.go`)

Currently serves as an **EXIF metadata updater**:
- Takes a user email flag to target specific users
- Scans media files and updates EXIF orientation data
- Runs as an external process triggered by the server's cron

**Problems**:
- Not a true background worker — it's a CLI tool
- No proper job queue or task distribution
- Tightly coupled to EXIF processing
- No error handling for partial failures

#### 3. EXIF Processing (`internal/processor/exif_reader.go`)

Well-implemented EXIF reader using `bep/imagemeta`:
- Supports 12+ image formats (JPEG, PNG, GIF, WebP, TIFF, HEIF, AVIF, DNG, CR2, NEF, ARW, PEF)
- Detects format via magic bytes
- Reads orientation and other EXIF tags
- Handles various numeric types gracefully

**Improvements Needed**:
- Currently only **reads** orientation, doesn't **apply** it
- No video metadata extraction
- Error handling could be more robust

#### 4. Domain Layer (`internal/domain/`)

Clean domain models:
- **Media**: Photos/videos with metadata, dimensions, capture date
- **Face**: Detected faces with bounding boxes and embeddings
- **Album**: Logical groupings of media
- **Job**: Background processing tasks (defined but underutilized)
- **Share**: Shared media/albums

The **Job** model is defined but not properly utilized — the worker doesn't create or consume jobs from this queue.

#### 5. Database Layer (`internal/database/`)

PostgreSQL repositories for all domain entities:
- Media, Album, User, Session, Share, Job, Face repositories
- Proper abstraction via interfaces
- Migration support

#### 6. API Layer (`internal/api/`)

REST API with:
- Authentication (login, logout, session management)
- Media upload and management
- Album creation and management
- Search functionality
- Sharing features
- Rate limiting middleware

### Critical Issues & Recommendations

#### 🔴 **High Priority**

1. **Worker Architecture is Broken**
   - The "worker" is just an EXIF updater CLI, not a background job processor
   - The cron-based approach is fragile and doesn't integrate with the app lifecycle
   - **Recommendation**: Implement a proper in-process job queue (e.g., using channels or a library like `asynq`)

2. **No Real Background Processing**
   - Face detection, thumbnail generation, and EXIF updates all need a proper worker pool
   - **Recommendation**: Create a `worker/` package with:
     - Job queue (in-memory or Redis-backed)
     - Worker pool with configurable concurrency
     - Proper job lifecycle (pending → processing → completed/failed)

3. **EXIF Processing is Incomplete**
   - Only reads orientation, doesn't apply corrections
   - No video metadata extraction
   - **Recommendation**: 
     - Add orientation correction (rotate/flip images)
     - Extract video metadata (duration, codec, bitrate)
     - Add EXIF write-back capability

#### 🟡 **Medium Priority**

4. **Face Detection is a Stub**
   - The face detection code exists but appears to be a mock implementation
   - **Recommendation**: Implement actual face detection using a library like `github.com/nfityp/sfz` or integrate with an external service

5. **No Proper Testing Strategy**
   - Limited test coverage
   - **Recommendation**: Add integration tests for database operations, unit tests for processors

6. **Storage Service Needs Review**
   - Need to understand how files are organized and stored
   - **Recommendation**: Document the storage layout and add tests

7. **Migration from Immich**
   - The `immich-migrate` command exists but needs verification
   - **Recommendation**: Test and document the migration process

#### 🟢 **Low Priority**

8. **Documentation Gaps**
   - Some components lack documentation
   - **Recommendation**: Add README files for each `cmd/` binary

9. **Android Bindings**
   - Go Mobile bindings exist but need verification
   - **Recommendation**: Test and document the mobile integration

### Suggested Improvements to Worker System

```go
// Proposed worker architecture:

type Worker struct {
    queue    JobQueue
    pool     WorkerPool
    handlers map[JobType]JobHandler
}

type JobQueue interface {
    Enqueue(ctx context.Context, job *Job) error
    Dequeue(ctx context.Context) (*Job, error)
}

type WorkerPool interface {
    Start(ctx context.Context, numWorkers int)
    Stop()
}

type JobHandler interface {
    Handle(ctx context.Context, job *Job) error
}

// Handlers:
// - ExifHandler: Updates EXIF metadata
// - ThumbnailHandler: Generates thumbnails
// - FaceDetectionHandler: Detects faces
```

### Conclusion

SteadyPhoto has a solid foundation with clean architecture and good separation of concerns. However, the worker system is the most critical issue — it's not a true background worker but rather a CLI tool triggered by cron. Implementing a proper job queue and worker pool would significantly improve reliability and extensibility.

