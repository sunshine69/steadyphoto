# Bug Analysis: Shared Thumbnails Redirect to Login Dialog

## Problem Statement

When User B views a share from User A in the "Shared with Me" section, the thumbnail doesn't display properly (shows fallback SVG), and clicking on it redirects to the login dialog instead of displaying the photo detail page. Even when logged in as User B, they can't view shared content from User A.

## Root Cause

The issue is multi-layered:

### Layer 1: Broken Thumbnails in "Shared with Me"

**Endpoint:** `GET /api/v1/media/shared` → `handleListSharedMedia()` (server.go line ~253)

This endpoint returns **only limited metadata**:
```go
response.Items[i] = SharedMediaResponse{
    ID:           item.Media.ID,
    Filename:     item.Media.Filename,
    MediaType:    string(item.Media.MediaType),
    SharerUserID: item.SharerUserID,
}
```

**What's MISSING:** 
- `path` — needed to serve thumbnails/origins
- `thumbnailUrl` — not computed at all
- No endpoint exists for serving shared thumbnails

The frontend (`shared-with-me.component.ts`) tries `[src]="item.thumbnailUrl"` but since it's null, images immediately fail and fall back to a placeholder SVG.

### Layer 2: Login Redirect When Viewing Shared Items

**Flow:** Click View → navigate to `photo-detail/:id` → `PhotoService.getMedia(id)` → `GET /api/v1/media/{id}`

**Endpoint:** `GET /api/v1/media/{id}` → `handleGetMedia()` (server.go line ~390)

```go
func (s *Server) handleGetMedia(w http.ResponseWriter, r *http.Request) {
    // ...
    media, err := s.mediaRepo.GetByID(ctx, id, &userID)  // checks ownership!
}
```

The `mediaRepo.GetByID()` does:
```sql
SELECT * FROM media WHERE deleted_at IS NULL AND id = $1 AND user_id = $2
```

Since User B doesn't own the media (User A does), this query returns nothing, causing a 403 → frontend redirects to login.

### Layer 3: Similarly Broken for Thumbnails/Original Files

**Endpoints:**
- `GET /api/v1/media/{id}/thumb` — uses same ownership check ❌
- `GET /api/v1/media/{id}/original` — uses same ownership check ❌  
- `GET /api/v1/photos/{id}` — uses same ownership check ❌

## The Fix Requires:

### Backend Changes (Go):

#### 1. New endpoint for serving shared media metadata
```go
// GET /api/v1/media/shared/{mediaId} — Get shared media detail
func (s *Server) handleGetSharedMedia(w http.ResponseWriter, r *http.Request) {
    userID := ... // current authenticated user
    mediaID := chi.URLParam(r, "id")
    
    // Check if this media is shared WITH the current user
    shareItem, err := s.mediaShareRepo.GetSharedMediaByID(ctx, mediaID, userID)
    if err != nil {
        http.NotFound(w, r)  // Not shared with this user
        return
    }
    
    // Return full metadata (including path)
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(shareItem.Media)
}

// GET /api/v1/media/shared/{mediaId}/thumb — Serve shared thumbnail
func (s *Server) handleGetSharedMediaThumb(w http.ResponseWriter, r *http.Request) {
    // Same ownership check replaced with sharee access check
    // Then serve the same way as regular thumb endpoint
}

// GET /api/v1/media/shared/{mediaId}/original — Serve shared original file
func (s *Server) handleGetSharedMediaOriginal(w http.ResponseWriter, r *http.Request) {
    // Same ownership check replaced with sharee access check
    // Then serve the same way as regular original endpoint
}

// GET /api/v1/albums/shared/{albumId} — Get shared album detail  
func (s *Server) handleGetSharedAlbum(w http.ResponseWriter, r *http.Request) {
    // Check if this album is shared WITH the current user
    shareItem, err := s.mediaShareRepo.GetSharedAlbumByID(ctx, albumID, userID)
    if err != nil {
        http.NotFound(w, r)  // Not shared with this user
        return
    }
    json.NewEncoder(w).Encode(shareItem.Album)
}

// GET /api/v1/albums/shared/{albumId}/media — List media in shared album
func (s *Server) handleListSharedAlbumMedia(w http.ResponseWriter, r *http.Request) {
    // Use mediaShareRepo.ListMediaInSharedAlbum() instead of ownership check
}
```

#### 2. Add `path` to the SharedMediaResponse returned by list endpoint

Modify `handleListSharedMedia()` to include file paths:
```go
type SharedMediaFullResponse struct {
    ID           uuid.UUID   `json:"id"`
    Filename     string      `json:"filename"`
    MediaType    string      `json:"mediaType"`
    Path         string      `json:"path"`          // NEW
    SharerUserID uuid.UUID   `json:"sharerUserId"`
}

// Then modify handleListSharedMedia to use this type instead of SharedMediaResponse
```

### Frontend Changes (Angular):

#### 1. Update `photo.service.ts` — add shared media methods

```typescript
/** Fetch metadata for a shared media item */
getSharedMedia(id: string): Observable<Photo> {
    return this.http.get<any>(`${this.API_BASE_URL}/media/shared/${id}`)
        .pipe(
            map(response => {
                const mediaData = response?.Media || response;
                return this.normalizePhoto(mediaData);
            }),
            catchError(this.handleError)
        );
}

/** Fetch a shared album detail */
getSharedAlbum(id: string): Observable<any> {
    return this.http.get<any>(`${this.API_BASE_URL}/albums/shared/${id}`)
        .pipe(
            map(response => response?.Album || response),
            catchError(this.handleError)
        );
}

/** List media in a shared album */
listSharedAlbumMedia(albumId: string, limit = 50, offset = 0): Observable<any> {
    return this.http.get<any>(`${this.API_BASE_URL}/albums/shared/${albumId}/media?limit=${limit}&offset=${offset}`)
        .pipe(
            map(response => ({ items: response?.items || [], total: response?.total || 0 })),
            catchError(this.handleError)
        );
}
```

#### 2. Update `shared-with-me.component.ts` — use correct paths for thumbnails

Since shared media now includes the `path`, we can construct thumbnail URLs:
```typescript
// Use path from backend instead of null thumbnailUrl
const thumbPath = item.path ? `${this.API_BASE_URL}/media/shared/${item.id}/thumb` : '';
// or if path is available in response, use it directly with original endpoint structure
```

#### 3. Update `photo-detail.component.ts` — route shared items to shared endpoints

In ngOnInit:
```typescript
// Check if we're viewing a shared item (query param ?shared=true)
const id = this.route.snapshot.paramMap.get('id');
if (id) {
    const isShared = this.route.snapshot.queryParamMap.has('shared');
    if (isShared) {
        // Use shared media endpoint instead of ownership-checking endpoint
        this.subscription = this.photoService.getSharedMedia(id).subscribe({
            next: (photo) => { this.photo = photo; },
            error: (err) => { console.error('Error fetching shared photo', err); }
        });
    } else {
        // Normal ownership-checking endpoint
        this.subscription = this.photoService.getMedia(id).subscribe({
            next: (photo) => { this.photo = photo; },
            error: (err) => { console.error('Error fetching photo', err); this.router.navigate(['/']); }
        });
    }
}
```

#### 4. Update `shared-with-me.component.ts` viewItem method — navigate with shared flag

```typescript
viewItem(item: SharedMediaItem & { sharerName?: string }): void {
    window.dispatchEvent(new CustomEvent('navigate-to-photo', { 
        detail: { id: item.id, shared: true }  // NEW: pass 'shared' flag
    }));
}
```

## Database Schema Verification

The existing `media_share_repository.go` already has the necessary repository methods:
- `GetSharedMediaByID(ctx, mediaID, shareeUserID)` — checks if user is a valid sharee ✓
- `ListMediaInSharedAlbum(ctx, albumID, shareeUserID, limit, offset)` — lists album media for shared access ✓

No database migration needed!

## Priority Fix Order

1. **High:** Add `/api/v1/media/shared/{id}` endpoint (metadata) + fix frontend routing → fixes login redirect
2. **Medium:** Add `/api/v1/media/shared/{id}/thumb` and `/original` endpoints → fixes image display  
3. **Low:** Update list endpoint to include paths, add shared album detail/media endpoints

## Summary

The core issue is that the share system creates "read access" for the sharee but the photo viewing endpoints only check ownership (user_id), not share permissions. Every photo/album/media serving endpoint needs a corresponding "shared access" variant that checks `media_shares` / `album_shares` tables instead of user_id.
