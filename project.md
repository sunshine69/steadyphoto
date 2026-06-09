## **SteadyPhoto — Complete Codebase Audit**

### **1. Project Overview**

SteadyPhoto is an open-source self-hosted photo and video management system built with Go (backend) and Angular (frontend). It supports multi-tenancy, AI-powered face detection, background job processing, a
nd migration from Immich.

---

### **2. Architecture Summary**

```
┌──────────────┐    ┌─────────────────┐    ┌──────────────┐
│  Angular UI   │◄──►│  Go HTTP Server │◄──►│ PostgreSQL DB │
│  (Angular)    │    │  (chi router)   │    │ (+ pgvector)  │
└──────────────┘    └────────┬────────┘    └──────────────┘
                            │
                    ┌───────▼────────┐
                    │  Worker Process │◄── Jobs queuem(DB polling)
                    │  (Thumbnail +   │
                    │   Face Detect)  │
                    └────────────────┘

  CLI tools: scanner, migrate, immich-migrate, repair_jobs
```

---

### **3. Command Binaries (6 binaries)**

| Binary | Path | Purpose |
|--------|------|---------|
| `server` | `cmd/server/main.go` | Main HTTP API server + Angular static files |
| `worker` | `cmd/worker/main.go` | Background job processor (thumbnails, face detection) — polls DB every 10s |
| `scanner` | `cmd/scanner/main.go` | CLI tool to scan a local directory and upload via API (multipart POST) |
| `migrate` | `cmd/migrate/main.go` | Runs SQL migrations (`up` action only) from `./migrations/*.up.sql` |
| `immich-migrate` | `cmd/immich-migrate/main.go` | Migrates media from an Immich server via its API (supports parallel workers, dry-run mode) |
| `repair_jobs` | `cmd/repair_jobs/main.go` | Scans DB for photos missing thumbnail jobs and creates them |

---

### **4. Domain Model (`internal/domain`)**

| Entity | Key Fields | Purpose |
|--------|-----------|---------|
| `User` | ID, Email, PasswordHash, Status (active/disabled), Role (admin/user) | User accounts with status-based access control |
| `UserSession` | ID, UserID, RefreshTokenHash, ExpiresAt, IsRevoked | JWT refresh token persistence |
| `Media` | ID, UserID, Path, Filename, Hash, SizeBytes, Width, Height, CapturedAt, MediaType (Photo/Video), Metadata (JSONB), VideoMetadata (JSONB), Tags, DeletedAt (soft delete) | Core media entity with s
oft-delete support |
| `Album` | ID, Name, Description, CreatedAt, UpdatedAt | User-owned photo albums |
| `Face` | ID, MediaID, BoundingBox (JSONB), Embedding ([]float32 via pgvector) | AI-detected faces per photo |
| `Job` | ID, Type (Thumbnail/FaceDetection), Status (Pending/Processing/Completed/Failed), MediaID, ErrorMessage, CreatedAt, UpdatedAt | Background job queue entries |

**Types:**
- `MediaType`: `"photo"` or `"video"`
- `BoundingBox`: struct with X1, Y1, X2, Y2 coordinates
- `Metadata`: map[string]string (JSONB in DB)

---

### **5. API Layer (`internal/api`)**

#### **Server Structure**
```go
type Server struct {
    mediaRepo     domain.MediaRepository
    albumRepo     domain.AlbumRepository
    userRepo      domain.UserRepository
    sessionRepo   domain.SessionRepository
    storageSvc    *storage.StorageService
    thumbRoot     string  // e.g. "/app/storage/.thumbnails"
}
```

#### **Router (chi/v5)**
| Route | Handler | Auth Required? | Admin Only? | Method |
|-------|---------|----------------|-------------|--------|
| `/api/v1/auth/register` | RegisterHandler | No | — | POST |
| `/api/v1/auth/login` | LoginHandler | No | — | POST |
| `/api/v1/auth/logout` | LogoutHandler | Yes | — | POST |
| `/api/v1/auth/refresh` | RefreshTokenHandler | No (uses refresh token in body) | — | POST |
| `/api/v1/auth/profile` | GetProfileHandler | Yes | — | GET |
| `/api/v1/auth/profile/email` | UpdateEmailHandler | Yes | — | PATCH |
| `/api/v1/auth/profile/password` | ChangePasswordHandler | Yes | — | PATCH |
| `/api/v1/media/upload/single` | UploadSingleMediaHandler | Yes | — | POST (multipart) |
| `/api/v1/media` | ListMediaHandler | Yes | — | GET (paginated: ?page=&per_page=) |
| `/api/v1/media/{id}` | GetMediaByIDHandler | Yes | — | GET |
| `/api/v1/media/{id}/delete` | DeleteMediaHandler (soft delete to trash) | Yes | — | POST |
| `/api/v1/media/trashed` | ListTrashedMediaHandler | Yes | — | GET (paginated) |
| `/api/v1/media/trashed/{id}` | GetTrashedMediaHandler | Yes | — | GET |
| `/api/v1/media/trashed/{id}/restore` | RestoreMediaHandler | Yes | — | POST |
| `/api/v1/media/trashed/{id}/permanently-delete` | PermanentlyDeleteMediaHandler | Yes | — | DELETE |
| `/api/v1/albums` | ListAlbumsHandler / CreateAlbumHandler | Yes | — | GET / POST |
| `/api/v1/albums/{id}` | GetAlbumByIDHandler | Yes | — | GET |
| `/api/v1/albums/{id}/update` | UpdateAlbumHandler | Yes | — | PATCH |
| `/api/v1/albums/{id}/delete` | DeleteAlbumHandler | Yes | — | DELETE |
| `/api/v1/albums/{id}/media` | GetAlbumMediaHandler / AddMediaToAlbumHandler | Yes | — | GET / POST |
| `/api/v1/albums/{id}/media/remove` | RemoveMediaFromAlbumHandler | Yes | — | POST (bulk) |
| `/api/v1/admin/users` | ListUsersHandler | Yes | **Yes** | GET |
| `/api/v1/admin/users/bulk-status` | BulkUpdateUserStatusHandler | Yes | **Yes** | PATCH |
| `/api/v1/admin/users/bulk-delete` | BulkDeleteUsersHandler | Yes | **Yes** | DELETE |

#### **Middleware Chain**
```
Request → CORS → RequestID (X-Request-ID) → AuthToken → AdminCheck (if needed) → Handler
```

---

### **6. Security Layer (`internal/security`)**
| Function | Purpose |
|----------|---------|
| `GenerateAccessToken(userID, role)` | Creates JWT access token with 15-minute TTL |
| `ValidateAccessToken(token)` | Validates and extracts claims from JWT |
| `GetUserIDFromContext(ctx)` | Extracts user ID from request context (set by AuthToken middleware) |
| `HashPassword(password)` | Argon2id password hashing via `golang.org/x/crypto/argon2` |
| `CheckPasswordHash(password, hash)` | Verifies a plaintext password against its hash |

---

### **7. Database Repositories (`internal/database`)**

All use `sqlx` with PostgreSQL (`lib/pq`). Parameterized queries only (no string interpolation for user input).

#### **PostgresUserRepository**
- CRUD for users + bulk operations
- Bulk status updates via dynamic `$N` placeholders
- Soft-delete-like behavior via `status = 'disabled'`
- `GetByUsernameOrEmail`: searches email OR id::text (note: not truly "username")

#### **PostgresMediaRepository**
- Full CRUD with soft delete (`deleted_at IS NULL`)
- Hash-based deduplication (`GetByHash`)
- Tag search via LIKE query
- Trash operations: `ListTrashed`, `GetTrashedMedia`, `RestoreMedia`, `PermanentlyDeleteMedia` (with transaction + face cleanup)

#### **PostgresAlbumRepository**
- CRUD with ownership check on every mutation (`WHERE user_id = $2`)
- Media management via `album_photos` junction table
- Paginated media listing within albums
- Position field for ordering photos in albums

#### **PostgresFaceRepository**
- Create faces + retrieve by media ID
- Delete all faces when media is permanently deleted

#### **PostgresJobRepository**
- Create, GetByID, UpdateStatus
- `GetPending(limit)`: fetches pending jobs ordered by created_at ASC (FIFO processing)

---

### **8. Storage Service (`internal/storage`)**

```go
type StorageService struct {
    baseDir string // e.g., "/app/storage"
}
```

**Key behaviors:**
- User-scoped paths: `{userID}/{YYYY/MM/DD/filename}` for physical isolation
- Path traversal protection via `filepath.Clean` + stripping leading `/`
- Thumbnail path derivation: `{original_path}_thumb.webp` (or `.webp` for videos)
- Upload temp directory: `{baseDir}/.upload-temp`
---

### **9. Processor Package (`internal/processor`)**

#### **ImageEngine Interface**
```go
type ImageEngine interface {
    Resize(ctx context.Context, inputPath, outputPath string, width int) error
}
```
- `StandardImageEngine`: uses Go's standard library with quality 85
- Swappable for libvips in production

#### **ThumbnailProcessor**
- Generates 800px-wide WebP thumbnails from photos
- Skips videos (frame extraction not yet implemented)
- Output: `{thumbRoot}/{YYYY/MM/DD/filename_thumb.webp}`

#### **FaceDetectionProcessor**
- Orchestrates AI face detection → saves bounding boxes + embeddings to DB
- Uses `ai.FaceDetector` interface (currently Noop in production)

---

### **10. AI Package (`internal/ai`)**

```go
type FaceDetector interface {
    DetectFaces(ctx context.Context, imagePath string) ([]FaceDetectionResult, error)
}

type FaceDetectionResult struct {
    BoundingBox domain.BoundingBox
    Embedding   []float32  // pgvector-compatible
}
```

**Implementations:**
- `NoopFaceDetector`: returns empty results (current default — safe for production without AI)
- `MockEngine`: test implementation with configurable face count

---

### **11. Scanner Package (`internal/scanner`)**

Internal scanner used by `immich-migrate`. Walks directories, filters by extension, and uploads via API. Currently a stub (simulated uploads).

Supported extensions: `.jpg`, `.jpeg`, `.png`, `.gif`, `.webp`, `.mp4`, `.mov`, `.avi`

---

### **12. Database Migrations (`migrations/`) — 12 migrations**

| # | Migration | Key Changes |
| 0001 | init_schema | Creates `photos`, `faces`, `albums`, `album_photos`. Enables pgvector + uuid-ossp |
| 0002 | add_jobs_table | Creates `jobs` table (type, status, media_id) |
| 0002 | unique_album_name_per_user | Adds UNIQUE constraint on (name, user_id) for albums |
| 0003 | rename_photos_to_media | Renames table + FK references. Adds `media_type`, `video_metadata` columns |
| 0004 | add_tags_to_media | Adds `tags TEXT` column to media |
| 0005 | add_users_and_sessions | Creates `users` (email, password_hash) and `user_sessions` tables |
| 0006 | add_ownership_to_assets | Adds `user_id UUID NOT NULL DEFAULT uuid_generate_v4()` to media |
| 0007 | seed_admin | Seeds admin user (`admin@steadyphoto.com`, password: `password`) |
| 0008 | add_user_status | Adds `status TEXT NOT NULL DEFAULT 'active'` and index on users |
| 0009 | add_position_to_album_photos | Adds `position INT NOT NULL DEFAULT 0` to album_photos; adds unique_album_name_per_user constraint |
| 0010 | add_deleted_at_to_media | Soft delete: adds `deleted_at TIMESTAMPTZ` + index on media |
| 0011 | add_user_role | Adds `role TEXT NOT NULL DEFAULT 'user'` to users; sets admin role for seeded user |

---

### **13. Docker Build (Multi-stage)**

```
Stage 1: node:24-alpine → Angular production build (dist/)
Stage 2: golang:1.26.3-alpine → Go binaries (server + migrate) — static linking, CGO_ENABLED=0
Stage 3: alpine:3.19 → Production image with both binaries + UI assets

Health check: wget http://localhost:8080/ui/
User: appuser (non-root)
Port: 8080 (serves /ui for Angular SPA + /api/v1 for REST API)
```

**Entrypoint:** Waits for PostgreSQL → runs migrations → starts server.

---

### **14. Dependencies (`go.mod`)**

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/go-chi/chi/v5` | v5.2.5 | HTTP router with middleware support |
| `github.com/google/uuid` | v1.6.0 | UUID generation and parsing |
| `github.com/jmoiron/sqlx` | v1.4.0 | SQL helper with named queries, struct mapping |
| `github.com/lib/pq` | v1.12.3 | PostgreSQL driver for database/sql |
| `golang.org/x/crypto` | v0.51.0 | Argon2id password hashing + JWT signing (HMAC-SHA256) |
| `golang.org/x/image` | v0.40.0 | Image processing (resize, WebP encoding for thumbnails) |
| `github.com/stretchr/testify` | v1.11.1 | Testing framework |

---

### **15. Key Design Patterns Observed**

1. **Repository Pattern**: All DB access goes through typed repository structs with interfaces in `domain`
2. **Interface Segregation**: `ImageEngine`, `FaceDetector` — swappable implementations for testing/production
3. **Soft Delete**: Media uses `deleted_at` column; Users use `status = 'disabled'`
4. **Ownership Enforcement**: Every DB query includes `user_id` check (multi-tenancy)
5. **Background Jobs**: Polling-based job queue in PostgreSQL — worker polls for pending jobs every 10s
6. **Path Traversal Protection**: Storage service cleans and validates all paths against base directory
7. **Hash-Based Deduplication**: Media files are keyed by SHA256 hash

This is a well-structured Go project with clear separation of concerns, proper authentication/authorization, and production-ready patterns for multi-tenant file management.
