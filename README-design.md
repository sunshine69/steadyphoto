# README-design.md

## Project Overview

SteadyPhoto is a comprehensive photo and video management system with AI-powered capabilities. The platform allows users to organize, search, and manage their media collections efficiently through a secure, multi-tenant architecture.

## Media Capabilities

### 1. Photo & Video Support
The system treats photos and videos as first-class citizens within the same unified pipeline:
- **Detection**: Automatic detection of supported formats (Images: JPG, PNG; Videos: MP4, MOV, AVI, etc.) during scanning.
- **Deduplication**: Content-based deduplication using SHA256 hashing to prevent redundant storage.
- **Metadata Extraction**: 
  - *Note*: Currently defaults to upload timestamp (`time.Now()`). Actual EXIF/video property extraction (camera model, ISO, GPS, duration, resolution) is planned for a future phase.

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

## Album Feature ✅ Completed

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

### Frontend Implementation (Angular ✅ Completed)
- **Sidebar**: Navigation link for "Albums".
- **Bulk Actions**: Selection mode in media grid with batch operations: add to album, remove from album, delete media, and bulk tag assignment.
- **Album View**: Dedicated route `/albums/:id` displaying a filtered view of the unified media feed based on album membership.

## Media Upload Feature ✅ Backend Completed

### Status: Backend implementation complete (Phase 1). Frontend UI polish remaining for mobile devices.

### Overview & Goals
- Allow users to upload photos (JPG, PNG) and videos (MP4, MOV, AVI) via the web interface.
- Enforce strict user ownership (`user_id` from JWT context).
- Automatically organize files into `storage/{user_id}/YYYY/MM/DD/`.
- Deduplicate uploads using SHA256 hashing to prevent redundant storage.

### Configurable Upload Limits
- **Environment Variable**: `MAX_UPLOAD_SIZE` (in bytes)
- **Default Value**: `1073741824` (1 GB)
- **Rationale**: Supports large 4K video clips while preventing accidental abuse or network timeouts. Files exceeding this limit will be rejected with a `413 Request Entity Too Large` error before consuming server resources.

### API Specifications
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

### Backend Design (Go) ✅ Implemented
- **Handler**: `upload_handler.go` registered in `server.go`. Designed to be client-agnostic (Web, Android, iOS).
- **Validation**: Strict MIME/extension whitelist (`image/jpeg`, `image/png`, `video/mp4`, etc.). Enforces max file size via middleware check on `Content-Length`.
- **Storage Path Generation**: 
  ```go
  dir := fmt.Sprintf("storage/%s/%s", userID, time.Now().Format("2006/01/02"))
  destPath := filepath.Join(dir, filename)
  ```
- **Deduplication Logic** ✅ Implemented:
  1. Compute SHA256 hash of the uploaded file stream.
  2. Query DB: `SELECT id FROM media WHERE sha256 = ? LIMIT 1`
  3. If exists → skip copy, return existing ID in `skipped_duplicates`.
  4. If new → save file to disk, proceed with metadata extraction.

### ⚠️ Metadata Extraction - Stubbed (Planned for Future Phase)
- **Current Behavior**: Defaults capture timestamp (`captured_at`) to upload time (`time.Now()`). Actual EXIF/video property parsing is not yet implemented in `upload_handler.go`.
- **Images** (Planned): Parse EXIF data (camera model, ISO, aperture, GPS, capture time).
- **Videos** (Planned): Extract duration, resolution, and codecs via `ffprobe` or `ffmpeg-go`.

### Frontend Design - Remaining Work 🚧
- Drag & Drop zone with overlay feedback.
- Pre-upload preview grid (images show thumbnails, videos show first frame or native player).
- Per-file and overall progress tracking.
- Batch handling queue (max 3 concurrent uploads to prevent browser limits).

---

## Bulk Actions & Media Deletion ✅ Completed

### Overview
The photo list view supports multi-select mode with a toolbar for bulk operations on selected media items.

### Features Implemented
| Action | Description | Implementation Details |
|--------|-------------|----------------------|
| **Multi-Select** | Click checkboxes to select multiple photos/videos in the grid | Selection state tracked via `Set<string>` of photo IDs; visual checkbox indicators at top-left corner of each card |
| **Delete Media** | Permanently delete selected items (1 or more) | Confirmation dialog before deletion. Iterates through all selected media, calls individual delete API per item with progress tracking and error handling for partial failures |
| **Add to Album** | Add multiple selected items to an existing album | Dropdown selector populated from user's albums; bulk `POST /api/albums/{id}/media` call with array of IDs |
| **Remove from Album** | Remove selected items from a specific album | Bulk removal via API endpoint, ownership verified for all media and album |
| **Add Tags** | Assign new tags to multiple items at once | Tag input field accepts colon-separated values (e.g., `vacation:sunset:beach`); fetches each item's existing tags, merges with new ones, updates individually |

### Deletion Behavior Note ⚠️
- **Current Implementation**: Delete operation is a logical deletion — removes the database record and cascades to related tables (`faces`, `album_media`). Physical files on disk are **not** deleted.
- **Impact**: Orphaned files remain in `storage/{user_id}/YYYY/MM/DD/` until cleanup or trash implementation. This aligns with planned future Trash feature where soft-deleted items can be restored before permanent removal.

---

## Search & Discovery ✅ Completed

### Overview
Client-side search and filtering across the media grid with three scope options:

| Scope | Behavior | Implementation |
|-------|----------|----------------|
| **All** (default) | Searches both filename AND tags; returns items matching either criterion | Combines results using OR logic, deduplicates by ID |
| **Name Only** | Filters by `filename` containing the search term (case-insensitive substring match) | Simple string `.includes()` on lowercased filenames |
| **Tags Only** | Filters by tag values containing the search term | Splits tags field (`:`-delimited), checks each tag for substring match |

### Additional Features
- **Tag URL Filter**: `?tag=...` query parameter filters grid to items matching that specific tag (case-insensitive). Clear filter button resets view.

## Current Status

| Feature | Status | Notes |
|---------|--------|-------|
| **Core Features** (Scanning & Deduplication) | ✅ Completed | SHA256 hashing prevents redundant storage |
| **Authentication & IAM** | ✅ Completed | JWT/Session-based auth, multi-user support with strict ownership enforcement |
| **Data Isolation** | ✅ Completed | Physical filesystem isolation (`{user_id}/YYYY/MM/DD/`) + API-level checks |
| **Streaming & Playback** | ✅ Completed | HTTP Range requests for seekable video playback without full downloads |
| **Album Feature** | ✅ Completed | Full-stack: DB schema, CRUD APIs with ownership validation, Angular UI (sidebar, detail view, bulk actions) |
| **Presentation Mode** | ✅ Completed | Full-screen slideshow with keyboard navigation and thumbnail strip |
| **Search & Discovery** | ✅ Completed | Client-side search across 3 scopes (`all`, `name`, `tags`) + URL-based tag filtering |
| **Bulk Actions (Multi-Select)** | ✅ Completed | Delete, Add to Album, Remove from Album, Bulk Tag Assignment via toolbar in photo list view |
| **Media Deletion** | ⚠️ Logical Only | Removes DB records; physical files on disk are NOT deleted. Orphaned files remain until cleanup/trash implementation |
| **Media Upload (Backend)** | ✅ Completed | Handler registered, MIME validation, SHA256 deduplication, storage path generation all working |
| **Metadata Extraction** | 🚧 Stubbed | Defaults to `time.Now()` for capture timestamp. EXIF/video parsing planned for future phase |
| **Upload Frontend UI** | 🚧 Remaining | Drag & drop zone, progress bars, preview grid — minor visual polish needed on mobile devices (not high priority) |

### Outstanding / Future Work
- [ ] **EXIF/Video Metadata Extraction**: Parse actual capture times, camera info, video duration/resolution during upload.
- [ ] **Upload Frontend Polish**: Drag & drop zone with overlay feedback, progress bars, batch queue concurrency control.
- [ ] **Trash Feature**: Soft-delete mechanism to allow restoration before permanent removal; cleanup job for orphaned disk files.

