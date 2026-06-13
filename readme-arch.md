# SteadyPhoto — Architecture Overview

## Project Overview

SteadyPhoto is a comprehensive photo and video management system with AI-powered capabilities. The platform allows users to organize, search, and manage their media collections efficiently through a secure, multi-tenant architecture.

**Last Updated:** June 13, 2026  
**Project Started:** May 2, 2026 (~6 weeks)  
**Version:** Pre-release (v1.0 preparation)

---

## Media Capabilities — Summary

### Photo & Video Support
- **Detection**: Automatic detection of supported formats during scanning ✅
- **Deduplication**: Content-based SHA256 hashing to prevent redundant storage ✅
- **Metadata Extraction**: ⚠️ Stubbed — defaults to upload timestamp; EXIF/video parsing planned for future phase

### Seamless Streaming & Playback
- ✅ HTTP Range requests for seekable video playback without full downloads
- ✅ Unified endpoint architecture serving media via consistent streaming interface

### Storage Organization
```
storage/
  {user_id}/         # Logical isolation per user
    YYYY/
      MM/
        DD/          # Actual file assets (Photos & Videos)
```
*Ensures physical data isolation while keeping directory tree predictable.*

---

## System Architecture

### Authentication & Security (IAM)
- ✅ **Multi-Tenancy**: Strict ownership enforcement; every repository method accepts a `*userID` parameter enforced at SQL level (`WHERE user_id = $2`)
- ✅ **Identity Management**: JWT/Session-based authentication with secure password hashing (Argon2id via `golang.org/x/crypto/argon2`)
  - Access tokens: Session IDs stored as opaque UUIDs in DB, returned as HttpOnly cookies AND for Bearer token usage
  - Refresh tokens: Opaque tokens hashed and stored in `user_sessions` table, used one-time (rotation after refresh)
  - Sessions expire after 7 days with revocation support
- ✅ **Security Layers**: CORS middleware properly configured — only specific origins listed in `AllowedCORSOrigins` are allowed. When no Origin header is present, CORS headers omitted entirely. Non-whitelisted origins get 403 Forbidden. ⚠️ CSP headers not set anywhere

### Frontend Integration
The Angular application provides a responsive dashboard featuring:
- ✅ Unified Media View — single seamless feed of photos and videos
- ✅ Advanced Playback — native video players with seekable support in media detail view
- ⚠️ Search & Discovery — client-side search/filter across 3 scopes + URL-based tag filtering; server-side search exists at `/api/v1/media/search`

### Android App (Kotlin/Go)
- ✅ Authentication — token refresh flow via OkHttp interceptor (`ApiClient.kt`) with automatic retry on 401
- ✅ Media Scanning — three-tier scan: type-specific → Files provider fallback; SHA256 dedup via Go gomobile bindings
- ✅ Upload — chunked streaming uploads with retry logic (OOM fix applied); UploadManager uses `UploadSingleFile` for multipart upload
- ⚠️ Permissions — uses deprecated `READ_EXTERNAL_STORAGE`; needs version-conditional requests using `READ_MEDIA_IMAGES`/`READ_MEDIA_VIDEO`
- ✅ Foreground Service — proper notification with channel creation, FileObserver + ContentObserver dual detection, periodic fallback sync every 5 min

---

## API Endpoints Reference *(Complete — verified against server.go)*

### Authentication
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/auth/register` | Register new user (pending admin approval) |
| `POST` | `/api/v1/auth/login` | Login — returns access_token, refresh_token, user_id, role, status. Sets HttpOnly cookie for access token. |
| `POST` | `/api/v1/auth/logout` | Logout — clears session on server side |
| `POST` | `/api/v1/auth/refresh` | Refresh access token using refresh token from body or cookie |

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

## Current Status — Summary

| **Feature** | **Status** | **Notes** |
|-------------|------------|-----------|
| Authentication & IAM | ✅ Completed | JWT/Session-based auth with HttpOnly cookies + Bearer token. Argon2id password hashing. CORS properly configured using chi/cors middleware. Secure cookie mode configurable via `SECURE_COOKIES=true`. |
| Data Isolation | ✅ Completed | Physical filesystem isolation (`{user_id}/YYYY/MM/DD/`) + API-level checks enforced at DB level via `WHERE user_id = $2` in every repository method |
| Streaming & Playback | ✅ Completed | HTTP Range requests for seekable video playback without full downloads, verified in handleGetOriginal and handleGetPhotoFile |
| Album Feature | ✅ Completed | Full-stack: DB schema (with position field), CRUD APIs with ownership validation, Angular UI (sidebar, detail view, bulk actions) |
| Search & Discovery | ✅ Completed | Client-side search across 3 scopes (`all`, `name`, `tags`) + URL-based tag filtering. Server-side tag search at `/api/v1/media/search` |
| Bulk Actions (Multi-Select) | ✅ Completed | Delete, Add to Album, Remove from Album, Bulk Tag Assignment via toolbar in photo list view |
| Media Deletion | ✅ Enhanced — Trash/Restore Flow | Three-stage deletion: soft delete → trash → permanent delete. Permanent delete removes both DB records AND physical files (including thumbnails and face detection data) |
| Admin User Management | ✅ Backend API Completed | Full CRUD + bulk operations for admin users with status validation, session revocation on delete/disable |
| Media Upload (Backend) | ✅ Completed with Multiple Modes | Multi-file upload, single file upload, chunked resumable uploads for mobile clients. SHA256 deduplication verified. MIME-type detection via `http.DetectContentType()` + extension whitelist enforced at upload time |
| Metadata Extraction | 🚧 Stubbed | Defaults to `time.Now()` for capture timestamp. EXIF/video parsing planned for future phase |
| Upload Frontend UI | 🚧 Remaining | Drag & drop zone with overlay feedback, progress bars, preview grid — minor visual polish needed on mobile devices |
| Android App Authentication | ✅ Completed | Token refresh flow fully implemented in OkHttp interceptor (`ApiClient.kt`) |
| Android App Media Scanning | ✅ Completed | Three-tier scan: type-specific → Files provider fallback. Hash dedup via SHA-256 using Go gomobile bindings |
| Android App Upload | ✅ Completed — OOM fix applied | Chunked streaming uploads with retry logic (fixes OOM crashes on large videos). ⚠️ Okio/OkHttp deprecation fixes applied: replaced deprecated static method calls with extension function patterns |
| Android App Permissions | ⚠️ Partially implemented | Uses deprecated `READ_EXTERNAL_STORAGE`; needs version-conditional permission requests using `READ_MEDIA_IMAGES` / `READ_MEDIA_VIDEO` |
| Android App Foreground Service | ✅ Completed | Proper foreground notification with channel creation, FileObserver + ContentObserver dual detection, periodic fallback sync every 5 minutes |
| Android App WorkManager Scheduling | ⚠️ Partially implemented | Missing network constraints on workers; WiFi-only toggle not wired to WorkManager constraints or UploadWorker |
| Android App Go Sync Engine | ⚠️ Partially implemented | `MediaProcessor.MediaHasher` (SHA256 hashing) and `UploadSingleFile` complete. Chunked upload endpoints exist in Go but never called from Kotlin — Android uses OkHttp instead. EXIF parsing missing |
| Android App Settings Persistence | ⚠️ Partially implemented | API URL config works, but auto-sync/WiFi-only toggles not persisted; logout button is a stub; storage info is placeholder text only |

### Outstanding / Future Work
- [ ] **EXIF/Video Metadata Extraction**: Parse actual capture times, camera info, video duration/resolution during upload — missing from both Go (`GetFileMetadata` returns basic MIME type only) and Kotlin layers.
- [ ] **Search by datetime range** and other advanced search filters.
- [ ] **Upload Frontend Polish (Web)**: Drag & drop zone with overlay feedback, progress bars, batch queue concurrency control.
- [ ] **Auto-add uploaded media to album** — feature mentioned in original design but not yet implemented in upload handler.
- [ ] **CSP (Content Security Policy) headers** — not set anywhere in the application.

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
9. File type validation enforced at upload time: MIME-type detection via `http.DetectContentType()` + extension whitelist in `upload_handler.go`. Rejects uploads where detected content type doesn't match expected media format for the given file extension.
10. **HTTPS/TLS support**: Server can start with TLS if `-tls-cert`/`-tls-key` CLI flags or `$TLS_CERT`/$TLS_KEY$ env vars are set (via `http.ListenAndServeTLS`). Falls back to plain HTTP otherwise — designed for reverse proxy setups where TLS termination happens at nginx/caddy level.
11. **Rate limiting**: Auth endpoints limited to 5 requests/minute by IP + path (brute-force protection). All protected routes have a safety net of 100 requests/minute by IP only. Custom JSON error responses on rate limit exceeded (HTTP 429). Implemented via `github.com/go-chi/httprate`.
12. **Secure cookie mode**: HttpOnly cookies can be set as Secure (HTTPS-only) when `$SECURE_COOKIES=true` env var is set, preventing token leakage over unencrypted connections.
13. **Token storage security fix**: Access token is now stored in `sessionStorage` instead of `localStorage`, reducing XSS risk (token cleared on tab close). Refresh token is NOT stored on the client — managed by HttpOnly cookies and server-side session rotation only.

### ⚠️ Security Concerns Requiring Attention
1. **No Content Security Policy (CSP) headers** — CSP not set anywhere in the application, leaving it vulnerable to XSS attacks if a vulnerability exists.
2. **No HTTPS enforcement on server when running behind a reverse proxy**: While TLS termination at nginx/caddy is the intended pattern, if a reverse proxy isn't configured and no `-tls-cert`/`-tls-key` flags or `$TLS_CERT`/$TLS_KEY$ env vars are provided, the server will serve over plain HTTP without any warning or redirect to HTTPS.
3. **No HSTS (HTTP Strict Transport Security) headers** — clients connecting over plain HTTP won't be redirected to HTTPS even if a reverse proxy terminates TLS.

---

## Document Index

For feature-specific details, see:
- [Album Feature](readme-album.md) — Data model, API specs, frontend implementation status
- [Media Upload Feature](readme-upload.md) — Backend design, upload modes, deduplication logic, remaining work
- [User Management Feature](readme-user-mgmt.md) — Admin endpoints, user profile, security fixes
- [Bulk Actions & Deletion Feature](readme-bulk-actions.md) — Multi-select operations, trash/restore flow
- [Search Feature](readme-search.md) — Client-side and server-side search capabilities
- [Sharing Feature](readme-sharing.md) — User-to-user sharing + public share links (In Progress)

---

*This document reflects the actual state of the codebase as verified against source files. Items marked with ⚠️ indicate discrepancies between design documentation and implementation, or security concerns that need addressing.*
