# README-design.md

## Project Overview

SteadyPhoto is a comprehensive photo and video management system with AI-powered capabilities. The platform allows users to organize, search, and manage their media collections efficiently through a secure, multi-tenant architecture.

## Media Capabilities

### 1. Photo & Video Support
The system treats photos and videos as first-class citizens within the same unified pipeline:
- **Detection**: Automatic detection of supported formats (Images: JPG, PNG; Videos: MP4, MOV, AVI, etc.) during scanning.
- **Deduplication**: Content-based deduplication using SHA256 hashing to prevent redundant storage.
- **Metadata Extraction**: 
  - Images: EXIF data (camera model, timestamps, location).
  - Videos: Basic properties and duration via existing metadata parsing logic.

### 2. Seamless Streaming & Playback
The API is optimized for high-performance media consumption:
- **Range Request Support**: The backend supports HTTP Range requests out of the box, enabling seamless video seeking (scrubbing) in modern web players without downloading entire files first.
- **Unified Endpoint Architecture**: All media assets are served via a consistent streaming interface that respects user ownership and authentication.

### 3. Storage Organization
To maintain simplicity and performance, all media is stored using a clean, date-based hierarchy:
```
storage/
  {user_id}/         # Logical isolation per user
    YYYY/
      MM/
        DD/          # Actual file assets (Photos & Videos)
```
*Note: This structure ensures physical data isolation while keeping the directory tree predictable and easy to manage.*

## System Architecture

### Authentication & Security (IAM)
- **Multi-Tenancy**: Strict ownership enforcement; users can only access, view, or delete media linked to their unique `user_id`.
- **Identity Management**: JWT/Session-based authentication with secure password hashing.
- **Security Layers**: Integrated middleware for token validation and cross-origin resource sharing (CORS) protection.

### Frontend Integration
The Angular application provides a responsive dashboard featuring:
- **Unified Media View**: A single, seamless feed of both photos and videos.
- **Advanced Playback**: Native video players with seekable support integrated directly into the media detail view.
- **Search & Discovery**: Fast name-based search and tag-based filtering for rapid asset retrieval.

## Album Feature (Planned)

### Data Model
Albums are logical groupings of existing media assets via a many-to-many relationship to avoid file duplication.

#### `albums` Table
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Primary Key |
| `user_id` | UUID | Foreign Key (owner) |
| `name` | VARCHAR | Album title |
| `description` | TEXT | Optional description |
| `created_at` | TIMESTAMP | Creation time |

#### `album_media` Table
| Column | Type | Description |
| :--- | :--- | :--- |
| `album_id` | UUID | Foreign Key (`albums.id`) |
| `media_id` | UUID/INT | Foreign Key to media asset |

### API Specifications
- **Album Management**: 
    - `POST /api/albums`: Create album.
    - `GET /api/albums`: List user's albums.
    - `PUT /api/albums/{id}`: Update metadata.
    - `DELETE /api/albums/{id}`: Remove album (does not delete media).
- **Media Association**:
    - `POST /api/albums/{id}/media`: Bulk add assets to an album. Requires ownership verification of all provided IDs.
    - `DELETE /api/albums/{id}/media/{media_id}`: Unlink asset from album.

### Frontend Requirements (Angular)
- **Sidebar**: Navigation link for "Albums".
- **Bulk Actions**: Selection mode in media grid to allow adding multiple items to an album at once.
- **Album View**: Dedicated route `/albums/:id` displaying a filtered view of the unified media feed based on album membership.

## Media Upload Feature (Planned)

### 1. Overview & Goals
- Allow users to upload photos (JPG, PNG) and videos (MP4, MOV, AVI) via the web interface.
- Enforce strict user ownership (`user_id` from JWT context).
- Automatically organize files into `storage/{user_id}/YYYY/MM/DD/`.
- Deduplicate uploads using SHA256 hashing to prevent redundant storage.
- Extract EXIF (images) and basic properties/duration (videos) for metadata enrichment.
- Provide real-time upload progress, batch handling, and preview capabilities.

### 2. Configurable Upload Limits
- **Environment Variable**: `MAX_UPLOAD_SIZE` (in bytes)
- **Default Value**: `1073741824` (1 GB)
- **Rationale**: Supports large 4K video clips while preventing accidental abuse or network timeouts. Files exceeding this limit will be rejected with a `413 Request Entity Too Large` error before consuming server resources.

### 3. API Specifications
| Method | Path | Description | Auth Required |
|--------|------|-------------|---------------|
| `POST` | `/api/v1/media/upload` | Upload one or more media files (multipart/form-data) for Web & Mobile clients | ✅ JWT |

**Request Format**: `multipart/form-data`
- `files[]`: Array of file objects.
- `albumId` (optional): UUID to auto-add uploaded media to an existing album.

**Response Format**: `application/json`
```json
{
  "uploaded": [
    {
      "id": "uuid-...",
      "filename": "IMG_1234.jpg",
      "mediaType": "photo",
      "path": "/api/v1/media/uuid-.../original",
      "size": 2048576,
      "captured_at": "2024-05-17T10:30:00Z"
    }
  ],
  "skipped_duplicates": ["uuid-dup1"],
  "errors": []
}
```

### 4. Backend Design (Go)
- **Handler**: `upload_handler.go` registered in `server.go`. Designed to be client-agnostic (Web, Android, iOS).
- **Validation**: Strict MIME/extension whitelist (`image/jpeg`, `image/png`, `video/mp4`, etc.). Enforces max file size via middleware check on `Content-Length`.
- **Storage Path Generation**: 
  ```go
  dir := fmt.Sprintf("storage/%s/%s", userID, time.Now().Format("2006/01/02"))
  destPath := filepath.Join(dir, filename)
  ```
- **Deduplication Logic**: 
  1. Compute SHA256 hash of the uploaded file stream.
  2. Query DB: `SELECT id FROM media WHERE sha256 = ? LIMIT 1`
  3. If exists → skip copy, return existing ID in `skipped_duplicates`.
  4. If new → save file to disk, proceed with metadata extraction.
- **Metadata Extraction**: 
  - **Images**: Parse EXIF data (camera model, ISO, aperture, GPS, capture time).
  - **Videos**: Extract duration, resolution, and codecs via `ffprobe` or `ffmpeg-go`.

### 5. Frontend Design (Angular)
- **Components**: 
  - `UploadModalComponent`: Triggered by nav button/drag zone. Contains file picker, preview grid, progress bars.
  - `PreviewService`: Generates local object URLs (`URL.createObjectURL`) for instant thumbnails/video previews before upload.
- **Service Layer**: `upload.service.ts` handles HTTP POST with `FormData`, manages progress events via `HttpClient.reportProgress`.
- **UI/UX Features**: 
  - Drag & Drop zone with overlay feedback.
  - Pre-upload preview grid (images show thumbnails, videos show first frame or native player).
  - Per-file and overall progress tracking.
  - Batch handling queue (max 3 concurrent uploads to prevent browser limits).
  - Toast notifications for success, duplicates skipped, or failures with retry buttons.

### 6. Implementation Roadmap
- [ ] **Phase 1: Backend Foundation**
  - Create `upload_handler.go` and register route in `server.go`.
  - Implement file validation (MIME, extension, size) and middleware for max upload limit.
  - Build storage path generator & disk save logic.
  - Integrate SHA256 hashing + DB deduplication check.
  - Add EXIF extraction for images and video metadata parsing.
- [ ] **Phase 2: Frontend Core**
  - Create `UploadService` with progress reporting.
  - Build `PreviewService` for local object URLs.
  - Implement `UploadModalComponent` with drag-and-drop zone and file picker fallback.
  - Wire up progress bars using `HttpEventType.UploadProgress`.
- [ ] **Phase 3: Polish & Integration**
  - Add batch upload queue (concurrency limit).
  - Implement error handling & retry logic per file.
  - Connect "Upload" button to existing nav/sidebar.
  - Auto-link uploads to active album if `albumId` query param exists.
  - Update main gallery grid to refresh after successful upload.

## Current Status

- ✅ **Core Features**: Photo/Video scanning, deduplication via SHA256 hashing.
- ✅ **Authentication**: Full Multi-user support (IAM) with secure login/registration.
- ✅ **Data Isolation**: Physical storage isolation and API ownership enforcement completed.
- ✅ **Streaming**: High-performance playback for both images and videos with seekable support.
- ✅ **Album Feature**: Database, API, and UI implementation completed.
- ✅ **Presentation Mode**: Completed.
- ✅ **Search**: Tag-based, name-based, and combined search in photo view page completed.
- 🚧 **Media Upload**: done - some minor visual UI on mobile device but not high priority


