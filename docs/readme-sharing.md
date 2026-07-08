# Sharing Feature

**Status:** ✅ Complete — Both user-to-user sharing and public sharing are working  
**Last Updated:** June 22, 2026

---

## Overview

The sharing feature enables users to share photos and albums with other app users, as well as generate public share links for anyone with the link. User A can select multiple photos or an album and share them with User B (view-only). User B can view shared content but cannot edit or delete it — however, User B CAN create their own albums containing the shared media. Additionally, User A can generate public share links (optionally password-protected) for individual photos or entire albums to share outside the app.

---

## Data Model

### `shares` Table — Groups user-to-user shares together
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Primary Key |
| `sharer_user_id` | UUID | Foreign Key (owner) — who is sharing the content |
| `sharee_user_id` | UUID | Foreign Key — who receives the share |
| `shared_at` | TIMESTAMP | Creation time of this share group |

### `media_shares` Table — Individual media items shared with a user
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Primary Key |
| `share_id` | UUID | Foreign Key (`shares.id`) — link to parent share group, cascade delete on share removal |
| `media_id` | UUID | Foreign Key to media asset being shared |
| UNIQUE constraint | `(share_id, media_id)` | Prevent duplicate sharing of same photo in the same share group |

### `album_shares` Table — Albums shared with a user
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Primary Key |
| `share_id` | UUID | Foreign Key (`shares.id`) — link to parent share group, cascade delete on share removal |
| `album_id` | UUID | Foreign Key to album being shared |
| UNIQUE constraint | `(share_id, album_id)` | Prevent duplicate sharing of same album in the same share group |

### `public_shares` Table — Public share links for media or albums
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Primary Key |
| `sharer_user_id` | UUID | Foreign Key (owner) — who created the public link |
| `token` | VARCHAR(16) | Unique token for the share link |
| `resource_type` | VARCHAR(10) | Either 'media' or 'album' — what is being shared |
| `resource_id` | UUID | The media_id or album_id being shared |
| `password_hash` | TEXT | Optional bcrypt-hashed password for protection (cost factor 12) |
| `expires_at` | TIMESTAMP | Optional expiration date/time when the link becomes invalid |
| `created_at` | TIMESTAMP | Creation time of this public share |
| `access_count` | INT | Track how many times the link has been accessed (default: 0) |

### `public_share_accesses` Table — Access log for security auditing
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Primary Key |
| `public_share_id` | UUID | Foreign Key (`public_shares.id`) |
| `ip_address` | INET | IP address of the person who accessed the link |
| `accessed_at` | TIMESTAMP | Time when the link was accessed |

---

## Security Considerations

- **Password Protection**: Public links with passwords use bcrypt hashing (cost factor 12) — never store plain-text passwords in `public_shares.password_hash`. Passwords are verified using `bcrypt.CompareHashAndPassword()` at access time.
- **Link Expiration**: Expired public shares return a 410 Gone response instead of 403/404 to indicate the resource existed but is no longer available.
- **Access Logging**: IP address and timestamp are logged for each access to public share links — useful for auditing if abuse occurs.
- **No Edit Permissions on Shared Content**: When User B views shared media/albums, only read-only data is returned. No edit/delete endpoints should be accessible from the frontend for shared content. The "Add to Album" button still works — User B can create their own albums containing the shared media (the new album belongs to User B).
- **User Deletion Cleanup**: When a user is deleted, their outgoing shares remain but incoming shares are removed — preventing orphaned share records.

---

## API Endpoints

### User-to-User Sharing Endpoints (Authenticated)
| Method | Path | Description | Auth Required? |
|--------|------|-------------|----------------|
| `POST` | `/api/v1/shares` | Create a share — select user(s) + media/albums to share | Yes |
| `GET` | `/api/v1/shares` | List outgoing share groups created by current user | Yes |
| `DELETE` | `/api/v1/shares/{id}` | Revoke an outgoing share group | Yes |
| `GET` | `/api/v1/media/shared` | Get shared media for current user (paginated: `?limit=20&offset=0`) | Yes |
| `GET` | `/api/v1/media/shared/{id}` | Get full details for a specific shared media item | Yes |
| `GET` | `/api/v1/media/shared/{id}/thumb` | Get thumbnail for a shared media item | Yes |
| `GET` | `/api/v1/albums/shared` | Get shared albums for current user (paginated: `?limit=20&offset=0`) | Yes |
| `GET` | `/api/v1/albums/shared/{id}` | Get full details for a specific shared album | Yes |
| `GET` | `/api/v1/auth/users/search?query=` | Search users by email/username for sharing purposes | Yes |

### Public Sharing Endpoints
| Method | Path | Description | Auth Required? |
|--------|------|-------------|----------------|
| `POST` | `/api/v1/public-shares` | Create a public share link (optionally with password) | Yes |
| `GET` | `/api/v1/public-shares` | List all public share links for current user | Yes |
| `DELETE` | `/api/v1/public-shares/{id}` | Revoke a public share link | Yes |
| `GET` | `/public/shares/media/{token}` | View shared media via public link (`?password=optional_password`) | No |
| `GET` | `/public/shares/album/{token}` | View shared album via public link (`?password=optional_password`) | No |

---

## Frontend Components

### `SharingDashboardComponent` (`/sharing-dashboard`)
The main sharing dashboard accessible from the sidebar. Contains two tabs:
- **"Shared with Me"** — Shows photos and albums shared with the current user
  - Pagination support (20 items per page)
  - Thumbnail display for photos and albums
  - Clicking opens the photo/album in the standard viewer
- **"My Shares"** — Shows links the current user has created
  - **Public Share Links** — Links created for anyone to view (with copy/revoke actions)
  - **User-to-User Shares** — Share groups sent to other users (with revoke actions)
  - Expiration badge for expired links
  - Copy-to-clipboard and revoke actions for each link

### `SharingComponent` (`/sharing`)
The multi-select share dialog triggered from the photo grid toolbar when items are selected.
- Opens a modal for sharing selected items
- Search users by email/username
- Toggle between sharing individual photos vs the containing album
- Create both user-to-user shares and public share links

### `public-share.component.ts`
Standalone component for viewing public share links without authentication.
- Displays shared photo or album
- Password prompt if the share is password-protected
- Shows "Shared by" attribution

---

## API Response Formats

### `POST /api/v1/shares` — Create a share
```json
Request: {
  "sharee_user_ids": ["uuid-of-user-b", "uuid-of-user-c"],
  "media_ids": ["uuid-1", "uuid-2"],
  "album_ids": ["uuid-album-1"]
}

Response: {
  "shares_created": [
    {
      "id": "share-id",
      "sharer_user_id": "user-a-id",
      "shared_at": "2024-12-17T10:30:00Z"
    }
  ],
  "media_shared_count": 2,
  "albums_shared_count": 1
}
```

### `GET /api/v1/media/shared` — Get shared media for current user
```json
Response: {
  "items": [
    {
      "id": "media-uuid",
      "filename": "IMG_001.jpg",
      "mediaType": "photo",
      "sharer_user_id": "user-a-id"
    }
  ],
  "total": 42,
  "limit": 20,
  "offset": 0
}
```

### `GET /api/v1/albums/shared` — Get shared albums for current user
```json
Response: {
  "items": [
    {
      "id": "album-uuid",
      "name": "Vacation Trip",
      "description": "Summer holiday photos",
      "sharer_user_id": "user-a-id"
    }
  ],
  "total": 5,
  "limit": 20,
  "offset": 0
}
```

### `POST /api/v1/public-shares` — Create a public share link
```json
Request: {
  "resource_type": "media",
  "resource_id": "uuid-of-media-or-album",
  "password": null,
  "expires_at": null
}

Response: {
  "id": "public-share-id",
  "token": "abc123xyz",
  "sharer_user_id": "user-a-id",
  "resource_type": "media",
  "password_protected": false,
  "expires_at": null,
  "created_at": "2024-12-17T10:30:00Z"
}
```

### `GET /public/shares/media/{token}` — View shared media via public link (no auth)
```json
Query params: ?password=optional_password

Response (200 OK): {
  "item": {
    "id": "media-uuid",
    "filename": "IMG_001.jpg",
    "mediaType": "photo"
  }
}

Response (403 Forbidden): if password is wrong or missing
```

---

## Service Layer

### `ShareService` (`share.service.ts`)
Full-featured service with its own auth interceptor pattern (duplicate of AuthService pattern). Key methods:
- `createShare()` — User-to-user sharing
- `searchUsers()` — Find users to share with
- `createPublicShareLink()` — Create public link
- `revokePublicShareLink()` — Revoke public link
- `listMyPublicShares()` — List own public links
- `listSharedMedia()` — List incoming shared media
- `listSharedAlbums()` — List incoming shared albums
- `getSharedMediaDetail()` — Get full detail of shared media
- `getSharedAlbumDetail()` — Get full detail of shared album
- `listMyOutgoingShares()` — List outgoing user-to-user share groups
- `revokeOutgoingShare()` — Revoke an outgoing share group
- Auth management methods (same pattern as AuthService)

### `ShareModel` (`share.model.ts`)
Shared interfaces for all share-related data types:
- `ShareRequest`, `PublicShareRequest`
- `SharedMediaItem`, `SharedAlbumItem`
- `CreateShareResponseFull`, `PublicShareLinkResponse`
- `PublicShareListItem`, `ShareGroupListItem`
- `SearchUser`

---

## Related Documents

- [Authentication & Multi-User Architecture](README-auth.md) — Auth system including admin status via `isAdmin$` observable
- [User Management](readme-user-mgmt.md) — Admin user account management
- [Architecture Overview](readme-arch.md) — High-level system architecture
