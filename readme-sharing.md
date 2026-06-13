# Sharing Feature

**Status:** 🚧 In Progress — Design complete, implementation pending  
**Last Updated:** June 13, 2026

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

## API Specifications

### User-to-User Sharing Endpoints (Authenticated)
| Method | Path | Description | Auth Required? |
|--------|------|-------------|----------------|
| `POST` | `/api/v1/shares` | Create a share — select user(s) + media/albums to share | Yes |
| `GET` | `/api/v1/media/shared` | Get shared media for current user (paginated: `?limit=20&offset=0`) | Yes |
| `GET` | `/api/v1/albums/shared` | Get shared albums for current user (paginated: `?limit=20&offset=0`) | Yes |

### Public Sharing Endpoints
| Method | Path | Description | Auth Required? |
|--------|------|-------------|----------------|
| `POST` | `/api/v1/public-shares` | Create a public share link (optionally with password) | Yes |
| `DELETE` | `/api/v1/public-shares/{id}` | Revoke a public share link | Yes |
| GET | `/public/shares/media/{token}` | View shared media via public link (`?password=optional_password`) | No |
| GET | `/public/shares/album/{token}` | View shared album via public link (`?password=optional_password`) | No |

---

## Detailed Request/Response Formats

### `POST /api/v1/shares` — Create a share
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

### `GET /api/v1/media/shared` — Get shared media for current user
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

### `GET /api/v1/albums/shared` — Get shared albums for current user
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

### `POST /api/v1/public-shares` — Create a public share link
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

### `DELETE /api/v1/public-shares/{id}` — Revoke a public share link
```json
Response: {
  "deleted": true                                          // success confirmation
}
```

### `GET /public/shares/media/{token}` — View shared media via public link (no auth)
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

### `GET /public/shares/album/{token}` — View shared album via public link (no auth)
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

---

## Frontend Implementation Plan

### User-to-User Sharing Flow

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

### Public Share Link Flow

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

---

## Related Documents

- [Architecture Overview](readme-arch.md) — High-level system architecture, API endpoints reference
- [Album Feature](readme-album.md) — Album data model (used by album sharing)
