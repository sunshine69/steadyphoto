# Album Feature

**Status:** ✅ Completed — Full-stack implementation verified against codebase  
**Last Updated:** June 13, 2026

---

## Data Model

Albums are logical groupings of existing media assets via a many-to-many relationship to avoid file duplication.

### `albums` Table
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Primary Key |
| `user_id` | UUID | Foreign Key (owner) — enforced at DB level with UNIQUE constraint on `(name, user_id)` |
| `name` | VARCHAR | Album title |
| `description` | TEXT | Optional description |
| `created_at` | TIMESTAMP | Creation time |

### `album_photos` Table *(Note: renamed from `album_media`)*
| Column | Type | Description |
| :--- | :--- | :--- |
| `album_id` | UUID | Foreign Key (`albums.id`) |
| `media_id` | UUID/INT | Foreign Key to media asset |
| `position` | INT | Ordering position (default: 0) — added in migration 0009 |

---

## API Specifications *(Updated paths from `/api/` to `/api/v1/`)*

### Album Management
- ✅ **`POST /api/v1/albums`** — Create album. Returns created album with ID and metadata.
- ✅ **`GET /api/v1/albums`** — List user's albums (authenticated).
- ✅ **`PUT /api/v1/albums/{id}`** — Update metadata (name, description). Ownership verified before update.
- ✅ **`DELETE /api/v1/albums/{id}`** — Remove album (does not delete media). Ownership verified before deletion.

### Media Association
- ✅ **`POST /api/v1/albums/{id}/media`** — Bulk add assets to an album. Requires ownership verification of all provided IDs — each media item is validated against the requesting user's ID before adding.
- ✅ **`DELETE /api/v1/albums/{id}/media/{media_id}`** — Unlink asset from album (single).
- ✅ **`DELETE /api/v1/albums/{id}/media`** — Bulk remove assets from an album.

---

## Frontend Implementation (Angular)

- ✅ **Sidebar**: Navigation link for "Albums".
- ✅ **Bulk Actions**: Selection mode in media grid with batch operations: add to album, remove from album, delete media, and bulk tag assignment.
- ✅ **Album View**: Dedicated route `/albums/:id` displaying a filtered view of the unified media feed based on album membership.

---

## Implementation Notes

- The junction table was originally named `album_media` but has been renamed to `album_photos` in migration 0009. Ensure any references use the new name.
- Position field (`position` column) allows ordering within albums — added after initial implementation.
- Ownership is verified at every operation level: album creation requires user_id match, media association validates each media item's ownership against the requesting user, and deletion/update operations verify album ownership before proceeding.

---

## Related Documents

- [Architecture Overview](readme-arch.md) — High-level system architecture
- [Sharing Feature](readme-sharing.md) — Album sharing with other users (In Progress)
