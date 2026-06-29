# Bulk Actions & Media Deletion Feature

**Status:** ✅ Completed — Enhanced with Trash/Restore flow  
**Last Updated:** June 13, 2026

---

## Overview

The photo list view supports multi-select mode with a toolbar for bulk operations on selected media items. **Significant enhancement: the deletion flow now includes trash, restore, and permanent delete capabilities.**

---

## Features Implemented

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

---

## Deletion Behavior Note ✅ **Clarified based on codebase**

- **Soft Delete**: Move media to trash (sets `deleted_at` timestamp). Physical files remain on disk. Media can be restored from trash later.
- **Permanent Delete**: Two paths:
  1. From active library via bulk delete endpoint (`POST /api/v1/media/delete`) — immediately deletes files and DB records for all selected items.
  2. From trash view via `DELETE /api/v1/trash/{id}` — permanently removes the media record AND its physical file(s) including thumbnail, plus face detection data (handled in transaction).

---

## Related Documents

- [Architecture Overview](readme-arch.md) — High-level system architecture
- [Album Feature](readme-album.md) — Bulk actions include add/remove from album
