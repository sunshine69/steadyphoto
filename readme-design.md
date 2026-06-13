# README-design.md

## Project Overview

SteadyPhoto is a comprehensive photo and video management system with AI-powered capabilities. The platform allows users to organize, search, and manage their media collections efficiently through a secure, multi-tenant architecture.

---

**Last Updated:** July 2025  
**Version:** Pre-release (v1.0 preparation)

---

---

## Media Capabilities

### 1. Photo & Video Support
The system treats photos and videos as first-class citizens within the same unified pipeline:
- **Detection**: Automatic detection of supported formats (Images: JPG, PNG; Videos: MP4, MOV, AVI, etc.) during scanning. ✅ Implemented
- **Deduplication**: Content-based deduplication using SHA256 hashing to prevent redundant storage. ✅ Implemented — duplicate files are detected and skipped during upload with `skipped_duplicates` response field
- **Metadata Extraction**: 
  - ⚠️ **Stubbed** — Currently defaults to upload timestamp (`time.Now()`). Actual EXIF/video property extraction (camera model, ISO, GPS, duration, resolution) is planned for a future phase.

### 2. Seamless Streaming & Playback
The API is optimized for high-performance media consumption:
- ✅ **Range Request Support**: The backend supports HTTP Range requests out of the box, enabling seamless video seeking (scrubbing) in modern web players without downloading entire files first. Implemented via `Accept-Ranges` header and `http.ServeFile()`.
- ✅ **Unified Endpoint Architecture**: All media assets are served via a consistent streaming interface that respects user ownership and authentication.

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

---

## System Architecture

### Authentication & Security (IAM)
- ✅ **Multi-Tenancy**: Strict ownership enforcement; users can only access, view, or delete media linked to their unique `user_id`. Every repository method accepts a `*userID` parameter that is enforced at the SQL level (`WHERE user_id = $2`).
- ✅ **Identity Management**: JWT/Session-based authentication with secure password hashing (Argon2id via `golang.org/x/crypto/argon2`). 
  - Access tokens: Session IDs stored as opaque UUIDs in DB, returned as HttpOnly cookies AND for Bearer token usage
  - Refresh tokens: Opaque tokens hashed and stored in `user_sessions` table, used one-time (rotation after refresh)
  - Sessions expire after 7 days with revocation support
- ✅ **Security Layers**: CORS middleware is properly implemented — only specific origins listed in `AllowedCORSOrigins` (configurable via `CORS_ALLOWED_ORIGINS` env var) are allowed to make cross-origin requests. When no Origin header is present, CORS headers are omitted entirely (same-origin or mobile app requests). Additionally, when an origin is not in the allowed list, a 403 Forbidden error is returned. **⚠️ Content Security Policy (CSP) headers are not set anywhere in the application.**

### Frontend Integration
The Angular application provides a responsive dashboard featuring:
- ✅ **Unified Media View**: A single, seamless feed of both photos and videos.
- ✅ **Advanced Playback**: Native video players with seekable support integrated directly into the media detail view.
- ⚠️ **Search & Discovery**: Client-side search and filtering across the media grid (all/name/tags scopes) + URL-based tag filtering (`?tag=...`). Server-side search by tags also exists at `/api/v1/media/search`.

---

## Album Feature ✅ Completed

### Data Model
Albums are logical groupings of existing media assets via a many-to-many relationship to avoid file duplication.

#### `albums` Table
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Primary Key |
| `user_id` | UUID | Foreign Key (owner) — enforced at DB level with UNIQUE constraint on `(name, user_id)` |
| `name` | VARCHAR | Album title |
| `description` | TEXT | Optional description |
| `created_at` | TIMESTAMP | Creation time |

#### `album_photos` Table *(Note: renamed from `album_media`)*
| Column | Type | Description |
| :--- | :--- | :--- |
| `album_id` | UUID | Foreign Key (`albums.id`) |
| `media_id` | UUID/INT | Foreign Key to media asset |
| `position` | INT | Ordering position (default: 0) — added in migration 0009 |

### API Specifications *(Updated paths from `/api/` to `/api/v1/`)*
- **Album Management**: 
    - ✅ `POST /api/v1/albums`: Create album. Returns created album with ID and metadata.
    - ✅ `GET /api/v1/albums`: List user's albums (authenticated).
    - ✅ `PUT /api/v1/albums/{id}`: Update metadata (name, description). Ownership verified before update.
    - ✅ `DELETE /api/v1/albums/{id}`: Remove album (does not delete media). Ownership verified before deletion.
- **Media Association**:
    - ✅ `POST /api/v1/albums/{id}/media`: Bulk add assets to an album. Requires ownership verification of all provided IDs — each media item is validated against the requesting user's ID before adding.
    - ✅ `DELETE /api/v1/albums/{id}/media/{media_id}`: Unlink asset from album (single).
    - ✅ `DELETE /api/v1/albums/{id}/media`: Bulk remove assets from an album.

### Frontend Implementation (Angular)
- ✅ **Sidebar**: Navigation link for "Albums".
- ✅ **Bulk Actions**: Selection mode in media grid with batch operations: add to album, remove from album, delete media, and bulk tag assignment.
- ✅ **Album View**: Dedicated route `/albums/:id` displaying a filtered view of the unified media feed based on album membership.

---

## Media Upload Feature ✅ Backend Completed (Enhanced)

### Status: Backend implementation complete with multiple upload modes. Frontend UI polish remaining for mobile devices.

### Overview & Goals
- Allow users to upload photos (JPG, PNG) and videos (MP4, MOV, AVI) via the web interface.
- Enforce strict user ownership (`user_id` from JWT context). ✅ Verified — `GetUserIDFromContext()` is called at entry point of all handlers.
- Automatically organize files into `storage/{user_id}/YYYY/MM/DD/`. ✅ Verified in upload_handler.go: ```go dir := fmt.Sprintf("storage/%s/%s", userID, time.Now().Format("2006/01/02"))```
- Deduplicate uploads using SHA256 hashing to prevent redundant storage. ✅ Implemented — hash computed during upload stream copy, checked against DB before saving duplicate file data.

### Configurable Upload Limits ⚠️ **Updated from documentation**
- **Environment Variable**: `MAX_UPLOAD_SIZE` (in bytes, configurable)
- ⚠️ **Actual Default Value**: `512 << 20 = 536870912` (**~512 MB**) — *not* 10MB as previously documented in readme-design.md. This is set in `internal/api/middleware.go`:
```go
var MaxUploadSizeBytes int64 = 512 << 20 // Default: 512MB (supports large single-file uploads)
```
- **Rationale**: Supports large 4K video clips while preventing accidental abuse or network timeouts. Files exceeding this limit will be rejected with a `413 Request Entity Too Large` error before consuming server resources.

### API Specifications *(Updated paths and methods)*
| Method | Path | Description | Auth Required |
|--------|------|-------------|---------------|
| `POST` | `/api/v1/media/upload` | Upload one or more media files (multipart/form-data) for Web & Mobile clients | ✅ JWT/Session |
| `POST` | `/api/v1/media/upload/single` | Single file upload endpoint for mobile clients — increased memory limit to 1GB | ✅ JWT/Session |
| `POST` | `/api/v1/media/upload/chunk` | Chunked upload endpoint for resumable uploads (mobile) — 128MB per chunk | ✅ JWT/Session |
| `GET` | `/api/v1/media/upload/status` | Upload status endpoint (mobile client) | ✅ JWT/Session |
| `POST` | `/api/v1/media/upload/abort` | Abort upload endpoint (mobile client) — small limit is fine for abort requests | ✅ JWT/Session |
| `POST` | `/api/v1/media/upload/complete` | Complete resumable upload — assemble chunks into final file | ✅ JWT/Session |

**Request Format**: `multipart/form-data`
- ⚠️ **Note**: The original design specified a single endpoint with an optional `albumId` parameter. However, the actual implementation does NOT include this feature yet — uploaded files are not automatically added to albums. This is a gap between design and implementation that should be addressed.

**Response Format**: `application/json`
```json
{
  "uploaded": [
    {
      "id": "uuid-...",
      "filename": "IMG_1234.jpg",
      "mediaType": "photo",
      "path": "/storage/{user_id}/YYYY/MM/DD/filename_ext", // Actual filesystem path, not API URL
      "size": 5248096,
      "captured_at": "2024-12-17T10:30:00Z"
    }
  ],
  "skipped_duplicates": [
    {
      "filename": "IMG_1235.jpg",
      "id": "uuid-dup1"
    }
  ]
}
```

### Backend Design (Go) ✅ Implemented
- **Handler**: `upload_handler.go` registered in `server.go`. Designed to be client-agnostic (Web, Android, iOS).
- ✅ **Validation**: File type validation now includes both extension checking AND MIME-type detection via `http.DetectContentType()` which reads the first 512 bytes of uploaded file content. The `isValidMediaType()` function validates that detected MIME types (`image/jpeg`, `image/png`, `video/mp4`, etc.) match expected formats for given extensions. Rejects uploads where MIME type doesn't match (e.g., `.jpg` extension with executable content).
- **Storage Path Generation**: ✅ Verified in upload_handler.go:
```go
dateDir := time.Now().Format("2006/01/02")
relTimePath := filepath.Join(dateDir, newFilename)
relPathFromRoot := filepath.Join(userID.String(), relTimePath)
// Result: "storage/{user_id}/YYYY/MM/DD/{uuid}.{ext}"
```
- **Deduplication Logic** ✅ Implemented:
  1. Compute SHA256 hash of the uploaded file stream (during `io.Copy(io.MultiWriter(tempFile, hasher), file)`).
  2. Query DB: `SELECT id FROM media WHERE sha256 = ? LIMIT 1`
  3. If exists → skip copy to disk, return existing ID in `skipped_duplicates`. File is deleted from temp storage.
  4. If new → save file to disk (via temp file), proceed with metadata extraction.

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

## User Management ✅ Backend API Completed — Frontend UI Status Unknown

### Overview
Admin users can manage other user accounts through a modal interface accessible from the top-right avatar icon. This feature provides full lifecycle management of user accounts including status changes, role assignments, and account deletion.

### Data Model
#### `users` Table
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Primary Key |
| `email` | VARCHAR | User email address (unique) — enforced at DB level via UNIQUE constraint |
| `password_hash` | TEXT | Securely hashed password (Argon2id) |
| `status` | VARCHAR | Account status: pending, active, disabled, rejected |
| `role` | VARCHAR | User role: admin, user |
| `created_at` | TIMESTAMP | Account creation time |
| `updated_at` | TIMESTAMP | Last update time |

### API Specifications (Admin Endpoints) *(Updated paths from `/api/` to `/api/v1/`)*
| Method | Path | Description | Auth Required | Admin Only? |
|--------|------|-------------|---------------|-------------|
| `GET` | `/api/v1/admin/users` | List all users with optional status filter (`?status=active`) | ✅ Admin JWT | Yes |
| `GET` | `/api/v1/admin/users/{id}` | Get specific user details | ✅ Admin JWT | Yes |
| `PATCH` | `/api/v1/admin/users/{id}` | Update user status or role (validated against allowed values) | ✅ Admin JWT | Yes |
| `DELETE` | `/api/v1/admin/users/{id}` | Soft-delete: set status to disabled, revoke all sessions | ✅ Admin JWT | Yes |
| `POST` | `/api/v1/admin/users/bulk-approve` | Bulk approve pending users (set status = active) | ✅ Admin JWT | Yes |
| `POST` | `/api/v1/admin/users/bulk-disable` | Bulk disable users (set status = disabled, revoke sessions) | ✅ Admin JWT | Yes |
| `DELETE` | `/api/v1/admin/users/bulk-delete` | Bulk delete: revoke all sessions for selected users before deletion | ✅ Admin JWT | Yes |

### User Profile Endpoints (Authenticated Users)
| Method | Path | Description | Auth Required? |
|--------|------|-------------|----------------|
| `GET` | `/api/v1/auth/profile` | Get current user profile | Yes — JWT/Session cookie or Bearer token |
| `PATCH` | `/api/v1/auth/profile/email` | Update email address | Yes |
| `PATCH` | `/api/v1/auth/profile/password` | Change password (requires current + new password) | Yes |
| `DELETE` | `/api/v1/auth/profile` | Delete user account — revokes all sessions, sets status to disabled | Yes |

### Frontend Implementation (Angular) ✅ Updated — Security Fix Applied
- The Angular auth service (`auth.service.ts`) confirms the following features exist:
  - ✅ Login with email/password → returns `access_token`, `refresh_token`, `user_id`, `role`, `status`
  - ✅ Registration creates pending user account (requires admin approval)
  - ✅ Logout clears local storage and calls backend logout endpoint
  - ✅ Token refresh via `/auth/refresh` using HttpOnly cookie
  - ✅ **Security Fix**: access_token is now stored in `sessionStorage` instead of `localStorage`. This means the token is cleared when the browser tab is closed, reducing XSS risk. The HttpOnly cookie remains the primary authentication mechanism for normal browser requests (sent automatically via `withCredentials`).
  - ✅ **Security Fix**: refresh_token is NOT stored on the client at all — it's managed by server-side session rotation and HttpOnly cookies only.

---

## Bulk Actions & Media Deletion ✅ Completed (Enhanced with Trash/Restore)

### Overview
The photo list view supports multi-select mode with a toolbar for bulk operations on selected media items. **Significant enhancement: the deletion flow now includes trash, restore, and permanent delete capabilities.**

### Features Implemented
| Action | Description | Implementation Details |
|--------|-------------|----------------------|
| **Multi-Select** | Click checkboxes to select multiple photos/videos in the grid | Selection state tracked via `Set<string>` of media IDs; visual checkbox indicators at top-left corner of each card |
| **Delete Media (Soft Delete)** | Move selected items to trash | Calls `DELETE /api/v1/media/{id}` — sets `deleted_at` timestamp. Physical files on disk are **NOT** deleted. |
| **List Trashed Items** | View soft-deleted media in trash | `GET /api/v1/trash` — returns paginated list of trashed items for the authenticated user |
| **Restore from Trash** | Restore a single item from trash to active library | `PATCH /api/v1/media/{id}/restore` — clears `deleted_at` timestamp |
| **Permanently Delete from Trash** | Completely remove media and its files from disk + DB | `DELETE /api/v1/trash/{id}` — deletes both database record AND physical file(s) including thumbnail. Also cleans up face detection data via transaction. |
| **Bulk Delete (Permanent)** | Permanently delete selected items in one operation | Calls `POST /api/v1/media/delete` with array of media IDs. Deletes files from storage + DB records for all selected items at once. Includes ownership verification per item before deletion. |
| **Add to Album** | Add multiple selected items to an existing album | Dropdown selector populated from user's albums; bulk `POST /api/v1/albums/{id}/media` call with array of IDs |
| **Remove from Album** | Remove selected items from a specific album | Bulk removal via `DELETE /api/v1/albums/{id}/media`, ownership verified for all media and album before processing |
| **Add Tags** | Assign new tags to multiple items at once | Tag input field accepts colon-separated values (e.g., `vacation:sunset:beach`); fetches each item's existing tags, merges with new ones, updates individually via `PATCH /api/v1/media/{id}/tags` |

### Deletion Behavior Note ✅ **Clarified based on codebase**
- **Soft Delete**: Move media to trash (sets `deleted_at` timestamp). Physical files remain on disk. Media can be restored from trash later.
- **Permanent Delete**: Two paths:
  1. From active library via bulk delete endpoint (`POST /api/v1/media/delete`) — immediately deletes files and DB records for all selected items.
  2. From trash view via `DELETE /api/v1/trash/{id}` — permanently removes the media record AND its physical file(s) including thumbnail, plus face detection data (handled in transaction).

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
- ✅ **Tag URL Filter**: `?tag=...` query parameter filters grid to items matching that specific tag (case-insensitive). Clear filter button resets view. Server-side implementation at `/api/v1/media/search`.

---

## API Endpoints Reference *(Complete — verified against server.go)*

### Authentication
| Method | Path | Auth Required? | Description |
|--------|------|----------------|-------------|
| `POST` | `/api/v1/auth/register` | No | Register new user (pending admin approval) |
| `POST` | `/api/v1/auth/login` | No | Login — returns access_token, refresh_token, user_id, role, status. Sets HttpOnly cookie for access token. |
| `POST` | `/api/v1/auth/logout` | Yes | Logout — clears session on server side |
| `POST` | `/api/v1/auth/refresh` | No (uses refresh token) | Refresh access token using refresh token from body or cookie |

### User Profile (Authenticated)
| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/auth/profile` | Get current user profile |
| `PATCH` | `/api/v1/auth/profile/email` | Update email address |
| `PATCH` | `/api/v1/auth/profile/password` | Change password (requires current + new) |
| `DELETE` | `/api/v1/auth/profile` | Delete account — revokes all sessions, sets status to disabled |

### Admin Endpoints
| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/admin/users` | List users with optional `?status=` filter |
| `GET` | `/api/v1/admin/users/{id}` | Get user by ID |
| `PATCH` | `/api/v1/admin/users/{id}` | Update user status or role (validated) |
| `DELETE` | `/api/v1/admin/users/{id}` | Soft-delete user (set disabled, revoke sessions) |
| `POST` | `/api/v1/admin/users/bulk-approve` | Bulk approve pending users |
| `POST` | `/api/v1/admin/users/bulk-disable` | Bulk disable users + revoke sessions |
| `DELETE` | `/api/v1/admin/users/bulk-delete` | Bulk delete users (revoke sessions first) |

### Media Endpoints
| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/media` | List media items for user (paginated: `?limit=20&offset=0`) |
| `GET` | `/api/v1/media/{id}` | Get single media item metadata |
| `PUT` | `/api/v1/media/{id}` | Update media metadata (full update, ownership verified) |
| `PATCH` | `/api/v1/media/{id}` | Partial update to media metadata |
| `DELETE` | `/api/v1/media/{id}` | Soft delete — move to trash |
| `GET` | `/api/v1/media/search?tag=...` | Server-side search by tag (returns all matching items for user) |

### Media File Endpoints *(Streaming)*
| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/media/{id}/original` | Stream original file with Range request support (video seeking) |
| `GET` | `/api/v1/media/{id}/thumb` | Stream thumbnail (falls back to original if thumb missing) |

### Trash Endpoints
| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/trash` | List trashed media items for user (paginated: `?limit=20&offset=0`) |
| `PATCH` | `/api/v1/media/{id}/restore` | Restore item from trash to active library |
| `DELETE` | `/api/v1/trash/{id}` | Permanently delete media + files from trash view |

### Bulk Delete Endpoint
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/media/delete` | Bulk permanent deletion — accepts JSON body with `{media_ids: [...]}`, deletes all specified items including physical files |

### Album Endpoints *(Updated paths)*
| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/albums` | List user's albums |
| `POST` | `/api/v1/albums` | Create album (name, description) |
| `GET` | `/api/v1/albums/{id}` | Get album by ID (ownership verified) |
| `PUT` | `/api/v1/albums/{id}` | Update album metadata |
| `DELETE` | `/api/v1/albums/{id}` | Delete album (does not delete media) |
| `GET` | `/api/v1/albums/{id}/media` | List media in album with pagination (`?limit=20&offset=0`) |
| `POST` | `/api/v1/albums/{id}/media` | Bulk add media to album (ownership verified for all items) |
| `DELETE` | `/api/v1/albums/{id}/media/{media_id}` | Remove single item from album |
| `DELETE` | `/api/v1/albums/{id}/media` | Bulk remove items from album |

### Media Upload Endpoints *(Enhanced with multiple upload modes)*
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/media/upload` | Multi-file upload (multipart/form-data, field name: `files[]`) — 32MB memory limit for form parsing |
| `POST` | `/api/v1/media/upload/single` | Single file upload endpoint for mobile clients — increased memory limit to 1GB |
| `POST` | `/api/v1/media/upload/chunk` | **✅ Confirmed in use by Android app** — Chunked upload for resumable uploads (mobile) — 5MB chunks on client side, 64MB per chunk on server side. UploadManager.kt uses this endpoint when file size >5MB and `enableChunkedUpload=true`. Session management with persistence to disk for crash recovery. |
| `GET` | `/api/v1/media/upload/status` | Upload status endpoint (mobile client) |
| `POST` | `/api/v1/media/upload/abort` | Abort upload (mobile client) |
| `POST` | `/api/v1/media/upload/complete` | Complete resumable upload — assemble chunks into final file |
### Sharing Endpoints *(User-to-User + Public Links)*
| Method | Path | Description | Auth Required? |
|--------|------|-------------|----------------|
| `POST` | `/api/v1/shares` | Create a share — select user(s) + media/albums to share | Yes |
| `GET` | `/api/v1/media/shared` | Get shared media for current user (paginated: `?limit=20&offset=0`) | Yes |
| `GET` | `/api/v1/albums/shared` | Get shared albums for current user (paginated: `?limit=20&offset=0`) | Yes |
| `POST` | `/api/v1/public-shares` | Create a public share link (optionally with password) | Yes |
| `DELETE` | `/api/v1/public-shares/{id}` | Revoke a public share link | Yes |
| GET | `/public/shares/media/{token}` | View shared media via public link (`?password=optional_password`) | No |
| GET | `/public/shares/album/{token}` | View shared album via public link (`?password=optional_password`) | No |

---
## Sharing Feature 🚧 In Progress

### Overview
The sharing feature enables users to share photos and albums with other app users, as well as generate public share links for anyone with the link. User A can select multiple photos or an album and share them with User B (view-only). User B can view shared content but cannot edit or delete it — however, User B CAN create their own albums containing the shared media. Additionally, User A can generate public share links (optionally password-protected) for individual photos or entire albums to share outside the app.

### Data Model

#### `shares` Table — Groups user-to-user shares together
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Primary Key |
| `sharer_user_id` | UUID | Foreign Key (owner) — who is sharing the content |
| `sharee_user_id` | UUID | Foreign Key — who receives the share |
| `shared_at` | TIMESTAMP | Creation time of this share group |

#### `media_shares` Table — Individual media items shared with a user
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Primary Key |
| `share_id` | UUID | Foreign Key (`shares.id`) — link to parent share group, cascade delete on share removal |
| `media_id` | UUID | Foreign Key to media asset being shared |
| UNIQUE constraint | `(share_id, media_id)` | Prevent duplicate sharing of same photo in the same share group |

#### `album_shares` Table — Albums shared with a user
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Primary Key |
| `share_id` | UUID | Foreign Key (`shares.id`) — link to parent share group, cascade delete on share removal |
| `album_id` | UUID | Foreign Key to album being shared |
| UNIQUE constraint | `(share_id, album_id)` | Prevent duplicate sharing of same album in the same share group |

#### `public_shares` Table — Public share links for media or albums
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Primary Key |
| `sharer_user_id` | UUID | Foreign Key (owner) — who created the public link |
| `resource_type` | VARCHAR(10) | Either 'media' or 'album' — what is being shared |
| `resource_id` | UUID | The media_id or album_id being shared |
| `password_hash` | TEXT | Optional bcrypt-hashed password for protection (cost factor 12) |
| `expires_at` | TIMESTAMP | Optional expiration date/time when the link becomes invalid |
| `created_at` | TIMESTAMP | Creation time of this public share |
| `access_count` | INT | Track how many times the link has been accessed (default: 0) |

#### `public_share_accesses` Table — Access log for security auditing
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Primary Key |
| `public_share_id` | UUID | Foreign Key (`public_shares.id`) |
| `ip_address` | INET | IP address of the person who accessed the link |
| `accessed_at` | TIMESTAMP | Time when the link was accessed |

### Security Considerations
- **Password Protection**: Public links with passwords use bcrypt hashing (cost factor 12) — never store plain-text passwords in `public_shares.password_hash`. Passwords are verified using `bcrypt.CompareHashAndPassword()` at access time.
- **Link Expiration**: Expired public shares return a 410 Gone response instead of 403/404 to indicate the resource existed but is no longer available.
- **Access Logging**: IP address and timestamp are logged for each access to public share links — useful for auditing if abuse occurs.
- **No Edit Permissions on Shared Content**: When User B views shared media/albums, only read-only data is returned. No edit/delete endpoints should be accessible from the frontend for shared content. The "Add to Album" button still works — User B can create their own albums containing the shared media (the new album belongs to User B).
- **User Deletion Cleanup**: When a user is deleted, their outgoing shares remain but incoming shares are removed — preventing orphaned share records.

### API Specifications

#### User-to-User Sharing Endpoints (Authenticated)
| Method | Path | Description | Auth Required? |
|--------|------|-------------|----------------|
| `POST` | `/api/v1/shares` | Create a share — select user(s) + media/albums to share | Yes |
| `GET` | `/api/v1/media/shared` | Get shared media for current user (paginated: `?limit=20&offset=0`) | Yes |
| `GET` | `/api/v1/albums/shared` | Get shared albums for current user (paginated: `?limit=20&offset=0`) | Yes |

#### Public Sharing Endpoints
| Method | Path | Description | Auth Required? |
|--------|------|-------------|----------------|
| `POST` | `/api/v1/public-shares` | Create a public share link (optionally with password) | Yes |
| `DELETE` | `/api/v1/public-shares/{id}` | Revoke a public share link | Yes |
| GET | `/public/shares/media/{token}` | View shared media via public link (`?password=optional_password`) | No |
| GET | `/public/shares/album/{token}` | View shared album via public link (`?password=optional_password`) | No |

#### Detailed Request/Response Formats

**`POST /api/v1/shares`** — Create a share
```json
Request: {
  "sharee_user_ids": ["uuid-of-user-b", "uuid-of-user-c"], // can share with multiple users at once
  "media_ids": ["uuid-1", "uuid-2"],                      // optional, list of media IDs to share
  "album_ids": ["uuid-album-1"]                            // optional, list of album IDs to share
}

Response: {
  "shares_created": [                                      // array of shares with their IDs
    {
      "id": "share-id",
      "sharer_user_id": "user-a-id",
      "shared_at": "2024-12-17T10:30:00Z"
    }
  ],
  "media_shared_count": 2,                                 // total media items shared
  "albums_shared_count": 1                                 // total albums shared
}
```

**`GET /api/v1/media/shared`** — Get shared media for current user
```json
Response: {
  "items": [                                               // array of shared media (same format as GET /media)
    {
      "id": "media-uuid",
      "filename": "IMG_001.jpg",
      "mediaType": "photo",
      "sharer_user_id": "user-a-id"                        // added field to show who shared it
    }
  ],
  "total": 42,                                             // total count for pagination
  "limit": 20,
  "offset": 0
}
```

**`GET /api/v1/albums/shared`** — Get shared albums for current user
```json
Response: {
  "items": [                                               // array of shared albums (same format as GET /albums)
    {
      "id": "album-uuid",
      "name": "Vacation Trip",
      "description": "Summer holiday photos",
      "sharer_user_id": "user-a-id"                        // added field to show who shared it
    }
  ],
  "total": 5,
  "limit": 20,
  "offset": 0
}
```

**`POST /api/v1/public-shares`** — Create a public share link
```json
Request: {
  "resource_type": "media",                                // or "album"
  "resource_id": "uuid-of-media-or-album",                 // the item being shared
  "password": null,                                        // optional password for protection (bcrypt hashed on server)
  "expires_at": "2024-12-31T23:59:59Z"                    // optional expiration date/time
}

Response: {
  "id": "public-share-id",
  "token": "abc123xyz",                                    // unique token for the share link
  "sharer_user_id": "user-a-id",
  "resource_type": "media",
  "password_protected": false,                             // whether a password was set
  "expires_at": null,                                      // when it expires (if any)
  "created_at": "2024-12-17T10:30:00Z"
}
```

**`DELETE /api/v1/public-shares/{id}`** — Revoke a public share link
```json
Response: {
  "deleted": true                                          // success confirmation
}
```

**`GET /public/shares/media/{token}`** — View shared media via public link (no auth)
```json
Query params: ?password=optional_password                 // required if password was set on the share

Response (200 OK): {
  "item": {                                                // same format as GET /media/{id}
    "id": "media-uuid",
    "filename": "IMG_001.jpg",
    "mediaType": "photo"
  }
}

Response (403 Forbidden): if password is wrong or missing
```

**`GET /public/shares/album/{token}`** — View shared album via public link (no auth)
```json
Query params: ?password=optional_password                 // required if password was set on the share

Response (200 OK): {
  "item": {                                                // same format as GET /albums/{id} but with media list
    "id": "album-uuid",
    "name": "Vacation Trip",
    "description": "Summer holiday photos",
    "media_items": [                                       // list of all media in the album (position sorted)
      {
        "id": "media-1",
        "filename": "IMG_001.jpg",
        "mediaType": "photo"
      }
    ]
  }
}

Response (403 Forbidden): if password is wrong or missing, or if the share link has expired
```

### Frontend Implementation

#### User-to-User Sharing Flow

1. **Share Button in Photo Grid** (multi-select mode):
   - Add a "Share" button to the toolbar when items are selected
   - Clicking opens a dialog with options:
     - Select users from existing app users (search by email)
     - Toggle between sharing photos vs albums — if User A has created an album containing the selected media, offer to share the whole album instead of individual photos

2. **"Shared With Me" View**:
   - New sidebar link "Shared With Me"
   - Two sections: Shared Photos and Shared Albums
   - Each item shows a "View" button that opens the photo/album in a read-only view (no edit/delete buttons shown for shared content)
   - User B can add shared photos to their own albums — this is already possible since there's no FK constraint on `media.user_id = albums.user_id`

3. **Read-Only View for Shared Content**:
   - When viewing a shared photo/album, the UI should not show any edit/delete buttons
   - The "Add to Album" button still works — User B can create their own album containing the shared media (the new album belongs to User B)

#### Public Share Link Flow

1. **Generate Public Link**:
   - Add a "Share via link" option in the share dialog or context menu for individual photos/albums
   - Options: set password, set expiration date/time
   - After creation, display the generated URL (e.g., `https://app.steadyphoto.com/public/shares/media/abc123xyz`) with a copy-to-clipboard button

2. **Viewing via Public Link**:
   - Landing page at `/public/shares/{type}/{token}` that displays the shared content
   - If password is required, show a password input field before displaying the content
   - Show sharer's username (e.g., "Shared by John Doe") and original album name if applicable

3. **Manage Public Links**:
   - User A can view their list of active public share links from a new page/tab
   - Each link shows: resource type, token (copyable), password status, expiration date, access count
   - "Revoke" button to delete the link and invalidate it for viewers

## Current Status

| **Authentication & IAM** | ✅ Completed | JWT/Session-based auth with HttpOnly cookies + Bearer token support. Argon2id password hashing. Session expiration and revocation. CORS properly configured using chi/cors middleware — only allows explicitly whitelisted origins (configurable via `CORS_ALLOWED_ORIGINS` env var). Secure cookie mode configurable via `SECURE_COOKIES=true`. |
| **Data Isolation** | ✅ Completed | Physical filesystem isolation (`{user_id}/YYYY/MM/DD/`) + API-level checks enforced at DB level via `WHERE user_id = $2` in every repository method |
| **Streaming & Playback** | ✅ Completed | HTTP Range requests for seekable video playback without full downloads, verified in handleGetOriginal and handleGetPhotoFile |
| **Album Feature** | ✅ Completed | Full-stack: DB schema (with position field), CRUD APIs with ownership validation, Angular UI (sidebar, detail view, bulk actions). Note: junction table renamed from `album_media` to `album_photos`. |
| **Search & Discovery** | ✅ Completed | Client-side search across 3 scopes (`all`, `name`, `tags`) + URL-based tag filtering. Server-side tag search at `/api/v1/media/search`. |
| **Bulk Actions (Multi-Select)** | ✅ Completed | Delete, Add to Album, Remove from Album, Bulk Tag Assignment via toolbar in photo list view |
| **Media Deletion** | ✅ Enhanced — Trash/Restore Flow | Three-stage deletion: soft delete → trash → permanent delete. Permanent delete removes both DB records AND physical files (including thumbnails and face detection data). |
| **Admin User Management** | ✅ Backend API Completed | Full CRUD + bulk operations for admin users with status validation, session revocation on delete/disable. Frontend UI needs verification against Angular codebase. |
| **Media Upload (Backend)** | ✅ Completed with Multiple Modes | Multi-file upload, single file upload, chunked resumable uploads for mobile clients. SHA256 deduplication verified. MIME-type detection via `http.DetectContentType()` + extension whitelist enforced at upload time. |
| **Metadata Extraction** | 🚧 Stubbed | Defaults to `time.Now()` for capture timestamp. EXIF/video parsing planned for future phase. |
| **Upload Frontend UI** | 🚧 Remaining | Drag & drop zone with overlay feedback, progress bars, preview grid — minor visual polish needed on mobile devices (not high priority). Note: album auto-association not yet implemented despite design document mentioning it. |
| **Android App Authentication** | ✅ Completed | Token refresh flow is fully implemented in OkHttp interceptor (`ApiClient.kt`). On 401 Unauthorized, the app automatically calls `/api/v1/auth/refresh`, stores the new access token via `storeAuthToken()`, and retries the original request transparently to the user. If refresh fails or no refresh token exists, it clears auth state and triggers `onAuthFailure` callback for UI navigation back to login screen. |
| **Android App Media Scanning** | ✅ Completed | Three-tier scan: type-specific (Images/Video) → broad Files provider fallback. Hash dedup via SHA-256 using Go gomobile bindings (`MediaProcessor.MediaHasher`). ContentObserver with debounced notifications on Images + Video URIs. FileObserver for real-time filesystem detection (though only watches 3 hardcoded directories currently). |
| **Android App Upload** | ✅ Completed — OOM fix applied. MainViewModel and SyncService both use `UploadManager.uploadMedia()` for chunked streaming uploads with retry logic (fixes OOM crashes on large videos). UploadWorker also uses the same path via AppContainer. The old inconsistent upload path in `SyncRepositoryImpl.uploadMedia()` (which loaded entire file into memory via `it.readBytes()`) is now dead code — MainViewModel no longer calls it. However, that method still exists as a risk: future developers adding new upload paths through SyncRepository could accidentally reintroduce the OOM issue. Recommendation: delete `SyncRepositoryImpl.uploadMedia()` entirely or mark it as deprecated with a clear warning comment. **⚠️ Okio deprecation fixes applied**: Replaced deprecated `Okio.buffer(Sink)` static method calls in ProgressRequestBody.kt and StreamingUploadHelper with extension function patterns (`sink.buffer()`, `countingSink.buffer()`). **⚠️ OkHttp deprecation fix applied**: Replaced deprecated `RequestBody.create(mediaType, content)` calls in UploadManager.kt with the extension function pattern `content.toRequestBody(mediaType)`. Both were compilation errors — the old APIs no longer compile. |
| **Android App Permissions** | ⚠️ Partially implemented — Uses deprecated `READ_EXTERNAL_STORAGE` which requires manifest permission on API 30 but is unnecessary on API 33+. Need version-conditional permission requests using `READ_MEDIA_IMAGES` / `READ_MEDIA_VIDEO`. Permission rationale UI shown when denied. |
| **Android App Foreground Service** | ✅ Completed | Proper foreground notification with channel creation, FileObserver + ContentObserver dual detection (real-time primary, content observer secondary), periodic fallback sync every 5 minutes as safety net. Missing: Doze mode / battery optimization handling. |
| **Android App WorkManager Scheduling** | ⚠️ Partially implemented — `UploadWorker` (one-time upload) and `MediaScannerWorker` (periodic scan, 5 min interval). Missing network constraints on workers (`setRequiredNetworkType(TRANSPORT_ANY)`), so they run even when offline. WiFi-only toggle exists in SettingsScreen but is not wired to WorkManager constraints or UploadWorker. |
| **Android App Go Sync Engine** | ⚠️ Partially implemented — `MediaProcessor.MediaHasher` (SHA256 hashing) and `UploadSingleFile` (multipart upload with retry logic) are complete. Chunked upload endpoints exist in Go but are never called from Kotlin — Android uses OkHttp for chunked uploads instead. `GetFileMetadata` only handles basic MIME type from extension; EXIF parsing is missing. |
| **Android App Settings Persistence** | ⚠️ Partially implemented — API URL config works, but auto-sync/WiFi-only toggles are not persisted (SharedPreferences/DataStore) and reset on every screen recreation. Logout button is a stub. Storage info is placeholder text only. |

### Outstanding / Future Work

- [ ] **EXIF/Video Metadata Extraction**: Parse actual capture times, camera info, video duration/resolution during upload — missing from both Go (`GetFileMetadata` returns basic MIME type only) and Kotlin layers.
- [ ] **Search by datetime range** and other advanced search filters.
- [ ] **Upload Frontend Polish (Web)**: Drag & drop zone with overlay feedback, progress bars, batch queue concurrency control.
- [ ] **Auto-add uploaded media to album** — feature mentioned in original design but not yet implemented in upload handler.
- [ ] **CSP (Content Security Policy) headers** — not set anywhere in the application.
- [ ] **Clean up dead code**: `SyncRepositoryImpl.uploadMedia()` is dead code (MainViewModel uses UploadManager directly), but it still exists and loads entire file into memory via `it.readBytes()`. Should be deleted or deprecated with a clear warning to prevent future developers from reintroducing OOM crashes on large files. |

---
## Security Summary *(Based on codebase review)*

### ✅ Implemented Safeguards
1. All database queries use parameterized statements (no SQL injection via string concatenation)
2. Passwords hashed with Argon2id (`golang.org/x/crypto/argon2`)
3. JWT session tokens stored as opaque UUIDs in DB, never plain-text
4. Refresh token rotation — old sessions revoked after each refresh
5. Ownership enforced at both API and database levels via `WHERE user_id = $2`
6. Path traversal protection: storage service uses `filepath.Clean()` + strips leading `/` before joining paths
7. File deduplication prevents storing duplicate content
8. CORS properly configured using chi/cors middleware — only allows explicitly whitelisted origins (configurable via `CORS_ALLOWED_ORIGINS` env var), with credentials support for authenticated cross-origin requests
9. File type validation enforced at upload time: MIME-type detection via `http.DetectContentType()` (reads first 512 bytes) + extension whitelist in `upload_handler.go`. Rejects uploads where detected content type doesn't match expected media format for the given file extension.
10. **HTTPS/TLS support**: Server can start with TLS if `-tls-cert`/`-tls-key` CLI flags or `$TLS_CERT`/$TLS_KEY$ env vars are set (via `http.ListenAndServeTLS`). Falls back to plain HTTP otherwise — designed for reverse proxy setups where TLS termination happens at nginx/caddy level.
11. **Rate limiting**: Auth endpoints limited to 5 requests/minute by IP + path (brute-force protection). All protected routes have a safety net of 100 requests/minute by IP only. Custom JSON error responses on rate limit exceeded (HTTP 429). Implemented via `github.com/go-chi/httprate`.
12. **Secure cookie mode**: HttpOnly cookies can be set as Secure (HTTPS-only) when `$SECURE_COOKIES=true` env var is set, preventing token leakage over unencrypted connections.
13. **Token storage security fix**: Access token is now stored in `sessionStorage` instead of `localStorage`, reducing XSS risk (token cleared on tab close). Refresh token is NOT stored on the client — managed by HttpOnly cookies and server-side session rotation only.

### ⚠️ Security Concerns Requiring Attention
1. **No Content Security Policy (CSP) headers** — CSP not set anywhere in the application, leaving it vulnerable to XSS attacks if a vulnerability exists.
2. **No HTTPS enforcement on server when running behind a reverse proxy**: While TLS termination at nginx/caddy is the intended pattern, if a reverse proxy isn't configured and no `-tls-cert`/`-tls-key` flags or `$TLS_CERT`/$TLS_KEY$ env vars are provided, the server will serve over plain HTTP without any warning or redirect to HTTPS.
3. **No HSTS (HTTP Strict Transport Security) headers** — clients connecting over plain HTTP won't be redirected to HTTPS even if a reverse proxy terminates TLS.

---

*This document reflects the actual state of the codebase as verified against source files in December 2024. Items marked with ⚠️ indicate discrepancies between design documentation and implementation, or security concerns that need addressing.*
