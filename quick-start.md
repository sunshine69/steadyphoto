# SteadyPhoto AI Development Guide

Quick reference for AI assistants to understand the codebase and start working on tasks efficiently.

---

## 📁 Project Structure

```
.
├── cmd/                          # Entry points
│   ├── server/main.go           # API server (HTTP)
│   ├── worker/main.go           # EXIF metadata updater (CLI)
│   ├── scanner/main.go          # Directory scanner & uploader
│   ├── migrate/main.go          # Database migrations
│   ├── immich-migrate/main.go   # Immich data migration
│   ├── cleanup/main.go          # Cleanup utility
│   ├── exif/main.go             # EXIF reader demo
│   └── exif-demo/main.go        # EXIF demo
├── internal/                     # Core application code
│   ├── api/                     # REST API handlers
│   ├── database/                # PostgreSQL repositories
│   ├── domain/                  # Domain models & interfaces
│   ├── processor/               # Image/EXIF processing
│   ├── scanner/                 # Media file scanner
│   ├── storage/                 # File storage service
│   ├── ai/                      # AI engine (stub)
│   ├── migration/               # Migration engine
│   └── utils/                   # Utilities
├── Documentation/               # Architecture docs
├── android/                     # Android Go Mobile bindings
└── angular-app/                 # Frontend (Angular)
```

---

## 🔧 Key Technologies

| Component | Technology |
|-----------|-----------|
| Language | Go 1.22+ |
| Database | PostgreSQL (sqlx) |
| API | net/http (custom routing) |
| EXIF | bep/imagemeta |
| Scheduling | robfig/cron |
| Auth | bcrypt + sessions |
| Storage | Local filesystem |

---

## 🚀 Build & Run

```bash
# Build server
go build -ldflags "-X main.version=$(git describe --tags)" -o bin/server ./cmd/server

# Build worker
go build -o bin/worker ./cmd/worker

# Run migrations
go run ./cmd/migrate

# Start server
DATABASE_URL=postgres://user:pass@localhost:5432/steadyphoto \
STORAGE_ROOT=./storage \
API_PORT=8081 \
./bin/server

# Run worker (EXIF updater)
./bin/worker -user email@example.com
```

---

## 🏗️ Architecture Patterns

### Dependency Injection

All components receive dependencies via constructors:

```go
// Example: API server
server := api.NewServer(
    mediaRepo,      // domain.MediaRepository
    albumRepo,      // domain.AlbumRepository
    userRepo,       // domain.UserRepository
    sessionRepo,    // domain.SessionRepository
    storageService, // *storage.StorageService
    thumbRoot,
    shareRepo,
    mediaShareRepo,
    albumShareRepo,
    publicShareRepo,
    publicAccessRepo,
    jobRepo,        // domain.JobRepository
)
```

### Repository Pattern

```go
// Interface definition (internal/domain/media.go)
type MediaRepository interface {
    Create(ctx context.Context, media *Media) error
    GetByID(ctx context.Context, id uuid.UUID, userID *uuid.UUID) (*Media, error)
    Update(ctx context.Context, media *Media) error
    Delete(ctx context.Context, id uuid.UUID, userID *uuid.UUID) error
    List(ctx context.Context, limit, offset int, userID *uuid.UUID) ([]*Media, int, error)
    // ... more methods
}

// Implementation (internal/database/media_repository.go)
type PostgresMediaRepository struct {
    db *sqlx.DB
}

func NewPostgresMediaRepository(db *sqlx.DB) *PostgresMediaRepository {
    return &PostgresMediaRepository{db: db}
}
```

### Storage Service

```go
// File organization: {baseDir}/{userID}/{date}/{filename}
storage := storage.NewStorageService("/mnt/data/storage", "/mnt/data/storage/.thumbnails")

// Get absolute path
absPath := storage.ResolveUserFile(userID, "2024/05/13/photo.jpg")
// Returns: /mnt/data/storage/{userID}/2024/05/13/photo.jpg

// Create thumbnail path
thumbRel := storage.GetThumbnailRelativePath("photo", "2024/05/13/photo.jpg", ".jpg")
// Returns: 2024/05/13/photo_thumb.webp
```

---

## 📦 Key Domain Models

### Media (`internal/domain/media.go`)

```go
type Media struct {
    ID            uuid.UUID
    Path          string
    Filename      string
    Hash          string
    SizeBytes     int64
    Width         int
    Height        int
    CapturedAt    time.Time
    MediaType     MediaType  // "photo" | "video"
    Metadata      Metadata   // JSONB: EXIF data
    VideoMetadata VideoMetadata // JSONB: video properties
    CreatedAt     time.Time
    UpdatedAt     time.Time
    Tags          string
    UserID        uuid.UUID
    ClientSource  ClientSource
    DeletedAt     *time.Time  // Soft delete
}

type VideoMetadata struct {
    Duration   float64
    Width      int
    Height     int
    Bitrate    int64
    VideoCodec string
    AudioCodec string
    FrameRate  float64
}
```

### Job (`internal/domain/job.go`)

```go
type Job struct {
    ID        uuid.UUID
    UserID    uuid.UUID
    Type      JobType   // "face_detection" | "thumbnail_generation"
    Status    JobStatus // "pending" | "processing" | "completed" | "failed"
    MediaID   uuid.UUID
    CreatedAt time.Time
    UpdatedAt time.Time
    Error     *string
}

type JobRepository interface {
    Create(ctx context.Context, job *Job) error
    GetByID(ctx context.Context, id uuid.UUID) (*Job, error)
    UpdateStatus(ctx context.Context, id uuid.UUID, status JobStatus, errStr string) error
    GetPending(ctx context.Context, limit int) ([]*Job, error)
}
```

---

## 🔌 Key Interfaces

### MediaRepository (`internal/domain/media.go`)

```go
type MediaRepository interface {
    Create(ctx context.Context, media *Media) error
    GetByID(ctx context.Context, id uuid.UUID, userID *uuid.UUID) (*Media, error)
    GetByHash(ctx context.Context, hash string) (*Media, error)
    Update(ctx context.Context, media *Media) error
    Delete(ctx context.Context, id uuid.UUID, userID *uuid.UUID) error
    List(ctx context.Context, limit, offset int, userID *uuid.UUID) ([]*Media, int, error)
    ListByType(ctx context.Context, mediaType MediaType, limit, offset int, userID *uuid.UUID) ([]*Media, int, error)
    SearchByTags(ctx context.Context, tags string, userID *uuid.UUID) ([]*Media, error)
    Search(ctx context.Context, query string, scope string, limit, offset int, userID *uuid.UUID) ([]*Media, int, error)
    ListTrashed(ctx context.Context, limit, offset int, userID uuid.UUID) ([]*Media, int, error)
    GetTrashedMedia(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*Media, error)
    RestoreMedia(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
    PermanentlyDeleteMedia(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}
```

### StorageService (`internal/storage/service.go`)

```go
type StorageService struct { ... }

func NewStorageService(baseDir string, thumbDir string) *StorageService
func (s *StorageService) GetUserRelativePath(userID uuid.UUID, relTimePath string) string
func (s *StorageService) ResolveUserFile(userID uuid.UUID, dbRelPath string) (string, error)
func (s *StorageService) GetAbsolutePath(relativePath string) string
func (s *StorageService) GetThumbnailAbsolutePath(thumbRelPath string) string
func (s *StorageService) ResolvePath(relativePath string) (string, error)
func (s *StorageService) EnsureDir(userID uuid.UUID, relTimePath string) error
func (s *StorageService) DeleteFile(path string) error
func (s *StorageService) DeleteThumbnailSilently(thumbRelPath string)
func (s *StorageService) DeleteMediaFiles(mediaPath string, thumbRelPath string) error
func (s *StorageService) GetUploadTempDir() string
```

### ExifReader (`internal/processor/exif_reader.go`)

```go
type ExifReader struct{}

func NewExifReader() *ExifReader
func (r *ExifReader) ReadOrientation(file *os.File) (Orientation, error)
func (r *ExifReader) ReadExif(file *os.File) (*ExifInfo, error)

type ExifInfo struct {
    Orientation Orientation
    Tags        []ExifTag
}

type ExifTag struct {
    Source    string
    Tag       string
    Namespace string
    Value     string
}
```

---

## 🌐 API Endpoints

### Authentication (`internal/api/auth_handler.go`)

```
POST   /api/auth/login
POST   /api/auth/logout
GET    /api/auth/session
```

### Media (`internal/api/upload_handler.go`)

```
POST   /api/media/upload           # Upload single file
POST   /api/media/upload-chunk     # Upload chunk (multipart)
POST   /api/media/complete         # Complete multipart upload
DELETE /api/media/:id              # Delete media
GET    /api/media                  # List media
GET    /api/media/:id              # Get media details
```

### Albums (`internal/api/album_handler.go`)

```
POST   /api/albums                 # Create album
GET    /api/albums                 # List albums
GET    /api/albums/:id             # Get album
PUT    /api/albums/:id             # Update album
DELETE /api/albums/:id             # Delete album
POST   /api/albums/:id/media       # Add media to album
DELETE /api/albums/:id/media/:id   # Remove media from album
```

### Search (`internal/api/search_handler.go`)

```
GET    /api/search?q=query&scope=name|tags|all
```

### Shares (`internal/api/shares_handler.go`)

```
POST   /api/shares                   # Create share
GET    /api/shares                   # List shares
GET    /api/shares/:id               # Get share details
DELETE /api/shares/:id               # Delete share
```

---

## 🗄️ Database Schema

### Core Tables

```sql
-- Media assets
media (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    path TEXT NOT NULL,
    filename TEXT NOT NULL,
    hash TEXT UNIQUE NOT NULL,
    size_bytes BIGINT,
    width INT,
    height INT,
    captured_at TIMESTAMP,
    media_type TEXT,
    metadata JSONB,           -- EXIF data
    video_metadata JSONB,     -- Video properties
    tags TEXT,
    client_source TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP      -- Soft delete
)

-- Albums
albums (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
)

-- Album-Media many-to-many
album_media (
    album_id UUID REFERENCES albums(id),
    media_id UUID REFERENCES media(id),
    position INT,
    PRIMARY KEY (album_id, media_id)
)

-- Users
users (
    id UUID PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    name TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
)

-- Sessions
sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    token TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMP,
    created_at TIMESTAMP
)

-- Jobs (background processing)
jobs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    job_type TEXT NOT NULL,
    status TEXT NOT NULL,
    media_id UUID,
    error_message TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
)

-- Faces
faces (
    id UUID PRIMARY KEY,
    media_id UUID NOT NULL,
    bounding_box JSONB,
    embedding FLOAT[],
    created_at TIMESTAMP
)

-- Shares
shares (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    media_id UUID,
    album_id UUID,
    password TEXT,
    expires_at TIMESTAMP,
    created_at TIMESTAMP
)
```

---

## 🎯 Common Tasks

### 1. Add a New API Endpoint

1. Create handler in `internal/api/` (e.g., `new_handler.go`)
2. Add routes in `internal/api/server.go`
3. Implement business logic (call repositories)
4. Add tests in `*_test.go`

### 2. Add a New Database Operation

1. Define interface in `internal/domain/` (e.g., `MediaRepository`)
2. Implement in `internal/database/` (e.g., `media_repository.go`)
3. Add SQL query with proper error handling
4. Update repository constructor in `cmd/server/main.go`

### 3. Process EXIF Metadata

```go
reader := processor.NewExifReader()
file, _ := os.Open("/path/to/photo.jpg")
defer file.Close()

// Read orientation
orientation, err := reader.ReadOrientation(file)

// Read all EXIF data
exifInfo, err := reader.ReadExif(file)
fmt.Printf("Orientation: %d\n", exifInfo.Orientation)
for _, tag := range exifInfo.Tags {
    fmt.Printf("%s: %s = %s\n", tag.Source, tag.Tag, tag.Value)
}
```

### 4. Upload a File

```go
// In upload handler
storageService := storage.NewStorageService(storageRoot, thumbRoot)

// Generate unique filename
filename := fmt.Sprintf("%s_%s%s", 
    uuid.New().String(), 
    filepath.Base(originalName),
    filepath.Ext(originalName))

// Create user-scoped path
relPath := storageService.GetUserRelativePath(userID, filename)

// Ensure directory exists
storageService.EnsureDir(userID, relPath)

// Save file to absolute path
absPath := storageService.ResolveUserFile(userID, relPath)
os.WriteFile(absPath, fileData, 0644)

// Create thumbnail
thumbRel := storageService.GetThumbnailRelativePath("photo", relPath, ext)
generateThumbnail(absPath, storageService.GetThumbnailAbsolutePath(thumbRel))
```

### 5. Run a Background Job

```go
// Create job
job := &domain.Job{
    ID:     uuid.New(),
    UserID: userID,
    Type:   domain.JobTypeThumbnail,
    Status: domain.JobStatusPending,
    MediaID: mediaID,
}
jobRepo.Create(ctx, job)

// In worker (currently cron-triggered)
pendingJobs, _ := jobRepo.GetPending(ctx, 10)
for _, job := range pendingJobs {
    jobRepo.UpdateStatus(ctx, job.ID, domain.JobStatusProcessing, "")
    
    // Process job
    err := processJob(job)
    
    if err != nil {
        jobRepo.UpdateStatus(ctx, job.ID, domain.JobStatusFailed, err.Error())
    } else {
        jobRepo.UpdateStatus(ctx, job.ID, domain.JobStatusCompleted, "")
    }
}
```

---

## ⚠️ Known Issues & Architecture Decisions

### 🔴 Critical Issues

2. **Face detection is a stub**
   - Code exists but not functional
   - **Fix needed**: Implement actual face detection

3. **EXIF processing incomplete**
   - Only reads orientation, doesn't apply corrections
   - **Fix needed**: Add orientation correction + video metadata

### 🟡 Medium Priority

### 🟢 Low Priority

6. **Documentation gaps**
   - Some components lack README
   - **Fix needed**: Add docs for each `cmd/` binary

---

## 🔍 Quick Reference: File Locations

| Task | File to Edit |
|------|--------------|
| Add API endpoint | `internal/api/*_handler.go` + `internal/api/server.go` |
| Add DB operation | `internal/database/*_repository.go` |
| Update domain model | `internal/domain/*.go` |
| Process images | `internal/processor/*.go` |
| Storage logic | `internal/storage/service.go` |
| Scanner logic | `internal/scanner/media_scanner.go` |
| Server config | `cmd/server/main.go` |
| Worker logic | `cmd/worker/main.go` |
| Migrations | `internal/migration/*.go` |

---

## 📝 Conventions

### Error Handling

```go
// Return errors, don't panic
if err != nil {
    return fmt.Errorf("failed to process: %w", err)
}

// Use context for cancellation
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

### Database Queries

```go
// Use sqlx for typed queries
var media Media
err := db.GetContext(ctx, &media, "SELECT * FROM media WHERE id = $1", id)
if err != nil {
    return nil, fmt.Errorf("media not found: %w", err)
}
```

### JSON Tags

```go
// Use camelCase for JSON, snake_case for DB
type Media struct {
    ID        uuid.UUID `db:"id" json:"id"`
    UserID    uuid.UUID `db:"user_id" json:"userId"`
    CreatedAt time.Time `db:"created_at" json:"createdAt"`
}
```

### Context Usage

```go
// Always pass context to DB operations
func (r *PostgresMediaRepository) GetByID(ctx context.Context, id uuid.UUID, userID *uuid.UUID) (*Media, error) {
    var media Media
    err := r.db.GetContext(ctx, &media, "...")
    return &media, err
}
```

---

## 🧪 Testing Patterns

### Unit Test Example

```go
func TestExifReader_ReadOrientation(t *testing.T) {
    reader := processor.NewExifReader()
    
    // Create test file
    file, _ := os.CreateTemp("", "test*.jpg")
    defer os.Remove(file.Name())
    
    // Write test data
    file.Write(testJPEGData)
    file.Seek(0, 0)
    
    // Test
    orientation, err := reader.ReadOrientation(file)
    assert.NoError(t, err)
    assert.Equal(t, processor.OrientationNormal, orientation)
}
```

### Integration Test Example

```go
func TestMediaRepository_CreateAndGet(t *testing.T) {
    db := setupTestDB(t)
    repo := database.NewPostgresMediaRepository(db)
    
    media := &domain.Media{
        ID:       uuid.New(),
        UserID:   uuid.New(),
        Filename: "test.jpg",
        Hash:     "abc123",
    }
    
    err := repo.Create(context.Background(), media)
    assert.NoError(t, err)
    
    got, err := repo.GetByID(context.Background(), media.ID, &media.UserID)
    assert.NoError(t, err)
    assert.Equal(t, media.Filename, got.Filename)
}
```

---

## 🚦 Decision Framework

When making changes, consider:

1. **Does it affect the domain model?** → Update `internal/domain/` first
2. **Does it need database changes?** → Update repository interface + implementation
3. **Does it expose a new API?** → Add handler + routes
4. **Does it process files?** → Use `internal/processor/` or `internal/storage/`
5. **Does it run in background?** → Use job queue (currently broken, needs fix)

---

## 📚 Additional Resources

- **Architecture doc**: `Documentation/readme-arch.md`
- **API docs**: Check `internal/api/server.go` for route definitions
- **DB schema**: Run migrations via `go run ./cmd/migrate`
- **Worker status**: Currently cron-triggered EXIF updater only

---

*Last updated: July 2026*>

