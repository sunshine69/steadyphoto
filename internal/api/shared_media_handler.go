package api

import (
	"encoding/json"
	"fmt"
	"github.com/jbrodriguez/mlog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// SharedMediaFullResponse includes path info so the frontend can generate thumbnails/serve files
type SharedMediaFullResponse struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"userId"`
	Filename     string    `json:"filename"`
	Path         string    `json:"path"`
	MediaType    string    `json:"mediaType"`
	CapturedAt   time.Time `json:"capturedAt"`
	SharerUserID uuid.UUID `json:"sharerUserId"`
	ThumbnailURL *string   `json:"thumbnailUrl,omitempty"` // Thumbnail URL for shared media
}

type SharedAlbumFullResponse struct {
	ID           uuid.UUID  `json:"id"`
	Name         string     `json:"name"`
	Description  *string    `json:"description,omitempty"`
	UserID       uuid.UUID  `json:"userId"`
	CreatedAt    time.Time  `json:"createdAt"`
	ThumbnailURL *string    `json:"thumbnailUrl,omitempty"`
	SharerUserID uuid.UUID  `json:"sharerUserId"`
}

type SharedAlbumMediaResponse struct {
	Items  []SharedAlbumMediaItem `json:"items"`
	Total  int                    `json:"total"`
	Limit  int                    `json:"limit"`
	Offset int                    `json:"offset"`
}

type SharedAlbumMediaItem struct {
	ID           uuid.UUID `json:"id"`
	Filename     string    `json:"filename"`
	Path         string    `json:"path"`
	MediaType    string    `json:"mediaType"`
	CapturedAt   time.Time `json:"capturedAt"`
	ThumbnailURL *string   `json:"thumbnailUrl,omitempty"`
}

// SharedAlbumsListResponse is the response for listing shared albums.
type SharedAlbumsListResponse struct {
	Items  []SharedAlbumFullResponse `json:"items"`
	Total  int                       `json:"total"`
	Limit  int                       `json:"limit"`
	Offset int                       `json:"offset"`
}

// logRequest logs a request with its path and user agent, returning the current time for duration tracking.
func logRequest(r *http.Request, label string) time.Time {
	start := time.Now()
	mlog.Info("[INFO] [%s] %s %s from %s", label, r.Method, r.URL.Path, r.RemoteAddr)
	return start
}

// logResponse logs the response status and duration.
func logResponse(label string, startTime time.Time) {
	duration := time.Since(startTime).String()
	fmt.Printf("[INFO] [%s] Response sent in %s\n", label, duration)
}

// logUnauthorized logs an unauthorized access attempt with IP address.
func logUnauthorized(r *http.Request, label string) {
	mlog.Info("[WARN] [%s] Unauthorized request - no user in context. IP: %s", label, r.RemoteAddr)
}

// logError logs an error with its label and duration.
func logError(label string, err error, startTime time.Time) {
	duration := time.Since(startTime).String()
	mlog.Info("[ERROR] [%s] %v — took %v", label, err, duration)
}

// logInfo logs an info message with its label and duration.
func logInfo(label string, msg string, startTime time.Time) {
	duration := time.Since(startTime).String()
	mlog.Info("[INFO] [%s] %s — took %v", label, msg, duration)
}

// handleGetSharedMedia returns metadata for a single media item shared with the current user.
// It checks both media_shares and album_shares to find the media item.
func (s *Server) handleGetSharedMedia(w http.ResponseWriter, r *http.Request) {
	startTime := logRequest(r, "handleGetSharedMedia")

	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		logUnauthorized(r, "handleGetSharedMedia")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		logError("handleGetSharedMedia - parse UUID", err, startTime)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	item, err := s.mediaShareRepo.GetSharedMediaByID(ctx, id, userID)
	if err != nil {
		// Try checking album shares as fallback
		albumItem, err := s.mediaShareRepo.GetSharedMediaFromAlbum(ctx, id, userID)
		if err != nil {
			logError("handleGetSharedMedia - GetSharedMediaByID", err, startTime)
			http.Error(w, "Shared media not found or access denied", http.StatusNotFound)
			return
		}
		item = albumItem
		logInfo("handleGetSharedMedia - Found via album share", fmt.Sprintf("ID=%s, filename=%s, userID=%s", idStr, item.Media.Filename, item.Media.UserID), startTime)
	} else {
		logInfo("handleGetSharedMedia - Found shared media", fmt.Sprintf("ID=%s, filename=%s, userID=%s", idStr, item.Media.Filename, item.Media.UserID), startTime)
	}

	w.Header().Set("Content-Type", "application/json")

	var thumbnailURL *string
	if item.Media.ID != uuid.Nil {
		thumbURL := "/media/shared/" + item.Media.ID.String() + "/thumb"
		thumbnailURL = &thumbURL
	}

	json.NewEncoder(w).Encode(SharedMediaFullResponse{
		ID:           item.Media.ID,
		UserID:       item.Media.UserID,
		Filename:     item.Media.Filename,
		Path:         item.Media.Path,
		MediaType:    string(item.Media.MediaType),
		CapturedAt:   item.Media.CapturedAt,
		SharerUserID: item.SharerUserID,
		ThumbnailURL: thumbnailURL,
	})

	logResponse("handleGetSharedMedia", startTime)
}

// handleGetSharedMediaOriginal streams the actual file for a shared media item.
// It checks both media_shares and album_shares to find the media item.
func (s *Server) handleGetSharedMediaOriginal(w http.ResponseWriter, r *http.Request) {
	startTime := logRequest(r, "handleGetSharedMediaOriginal")

	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		logUnauthorized(r, "handleGetSharedMediaOriginal")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		logError("handleGetSharedMediaOriginal - parse UUID", err, startTime)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	item, err := s.mediaShareRepo.GetSharedMediaByID(ctx, id, userID)
	if err != nil {
		// Try checking album shares as fallback
		albumItem, err := s.mediaShareRepo.GetSharedMediaFromAlbum(ctx, id, userID)
		if err != nil {
			logError("handleGetSharedMediaOriginal - GetSharedMediaByID", err, startTime)
			http.Error(w, "Shared media not found or access denied", http.StatusNotFound)
			return
		}
		item = albumItem
	}

	targetPath := item.Media.Path
	absPath, err := s.storageService.ResolvePath(targetPath)
	if err != nil {
		targetPath = strings.TrimLeft(targetPath, "/\\")
		absPath, err = s.storageService.ResolvePath(targetPath)
	}

	if err != nil {
		logError("handleGetSharedMediaOriginal - resolvePath", err, startTime)
		http.Error(w, "Could not locate file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	contentType := s.getContentType(item.Media.MediaType, item.Media.Filename)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type, Accept-Ranges, Content-Range")

	logInfo("handleGetSharedMediaOriginal - Serving original file", fmt.Sprintf("ID=%s, path=%s", idStr, absPath), startTime)
	http.ServeFile(w, r, absPath)
}

// handleGetSharedMediaThumb streams the thumbnail for a shared media item.
// It checks both media_shares and album_shares to find the media item.
func (s *Server) handleGetSharedMediaThumb(w http.ResponseWriter, r *http.Request) {
	startTime := logRequest(r, "handleGetSharedMediaThumb")

	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		logUnauthorized(r, "handleGetSharedMediaThumb")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		logError("handleGetSharedMediaThumb - parse UUID", err, startTime)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	item, err := s.mediaShareRepo.GetSharedMediaByID(ctx, id, userID)
	if err != nil {
		// Try checking album shares as fallback
		albumItem, err := s.mediaShareRepo.GetSharedMediaFromAlbum(ctx, id, userID)
		if err != nil {
			logError("handleGetSharedMediaThumb - GetSharedMediaByID", err, startTime)
			http.Error(w, "Shared media not found or access denied", http.StatusNotFound)
			return
		}
		item = albumItem
	}

	relPath := filepath.Clean(item.Media.Path)

	// Calculate thumbnail path - strip user ID from path since thumbnails are stored without it
	cleanPath := strings.TrimPrefix(relPath, "storage/")
	parts := strings.SplitN(cleanPath, string(filepath.Separator), 2)
	if len(parts) >= 2 {
		cleanPath = parts[1]
	}
	ext := filepath.Ext(cleanPath)

	thumbRelPath := s.storageService.GetThumbnailRelativePath(
		string(item.Media.MediaType),
		cleanPath,
		ext,
	)

	fullThumbPath := filepath.Join(s.thumbRoot, thumbRelPath)

	if _, err := os.Stat(fullThumbPath); os.IsNotExist(err) {
		logInfo("handleGetSharedMediaThumb - Thumbnail NOT FOUND", fmt.Sprintf("Attempting fallback to original from %s", fullThumbPath), startTime)
		targetPath := item.Media.Path
		absPath, resolveErr := s.storageService.ResolvePath(targetPath)
		if resolveErr != nil {
			targetPath = strings.TrimLeft(targetPath, "/\\")
			absPath, resolveErr = s.storageService.ResolvePath(targetPath)
		}

		if resolveErr != nil {
			logError("handleGetSharedMediaThumb - fallback failed", resolveErr, startTime)
			http.Error(w, "Thumbnail not found and fallback failed", http.StatusNotFound)
			return
		}

		contentType := s.getContentType(item.Media.MediaType, item.Media.Filename)
		w.Header().Set("Content-Type", contentType)
		logInfo("handleGetSharedMediaThumb - Serving original as fallback", fmt.Sprintf("from %s", absPath), startTime)
		http.ServeFile(w, r, absPath)
		return
	}

	logInfo("handleGetSharedMediaThumb - Serving thumbnail", fmt.Sprintf("ID=%s, path=%s", idStr, fullThumbPath), startTime)
	http.ServeFile(w, r, fullThumbPath)
}

// handleGetSharedAlbum returns metadata for a single album shared with the current user.
func (s *Server) handleGetSharedAlbum(w http.ResponseWriter, r *http.Request) {
	startTime := logRequest(r, "handleGetSharedAlbum")

	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		logUnauthorized(r, "handleGetSharedAlbum")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		logError("handleGetSharedAlbum - parse UUID", err, startTime)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	item, err := s.mediaShareRepo.GetSharedAlbumByID(ctx, id, userID)
	if err != nil {
		logError("handleGetSharedAlbum - GetSharedAlbumByID", err, startTime)
		http.Error(w, "Shared album not found or access denied", http.StatusNotFound)
		return
	}

	logInfo("handleGetSharedAlbum - Found shared album", fmt.Sprintf("ID=%s, name=%s", idStr, item.Album.Name), startTime)

	w.Header().Set("Content-Type", "application/json")

	var thumbnailURL *string
	if item.FirstMediaID != uuid.Nil {
		thumbURL := "/media/shared/" + item.FirstMediaID.String() + "/thumb"
		thumbnailURL = &thumbURL
	}

	json.NewEncoder(w).Encode(SharedAlbumFullResponse{
		ID:           item.Album.ID,
		Name:         item.Album.Name,
		Description:  item.Album.Description,
		UserID:       item.Album.UserID,
		CreatedAt:    item.Album.CreatedAt,
		ThumbnailURL: thumbnailURL,
		SharerUserID: item.SharerUserID,
	})

	logResponse("handleGetSharedAlbum", startTime)
}

// handleListSharedAlbumMedia returns media items in a shared album.
func (s *Server) handleListSharedAlbumMedia(w http.ResponseWriter, r *http.Request) {
	startTime := logRequest(r, "handleListSharedAlbumMedia")

	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		logUnauthorized(r, "handleListSharedAlbumMedia")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		logError("handleListSharedAlbumMedia - parse UUID", err, startTime)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	limit := 50
	offset := 0
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, e := strconv.Atoi(lStr); e == nil && l > 0 {
			limit = l
		}
	}
	if oStr := r.URL.Query().Get("offset"); oStr != "" {
		if o, e := strconv.Atoi(oStr); e == nil && o >= 0 {
			offset = o
		}
	}

	logInfo("handleListSharedAlbumMedia - Query params", fmt.Sprintf("limit=%d, offset=%d", limit, offset), startTime)

	items, totalItems, err := s.mediaShareRepo.ListMediaInSharedAlbum(ctx, id, userID, limit, offset)
	if err != nil {
		logError("handleListSharedAlbumMedia - ListMediaInSharedAlbum", err, startTime)
		http.Error(w, "Failed to list shared album media: "+err.Error(), http.StatusInternalServerError)
		return
	}

	logInfo("handleListSharedAlbumMedia - Found items", fmt.Sprintf("%d items (total=%d)", len(items), totalItems), startTime)

	respItems := make([]SharedAlbumMediaItem, 0, len(items))
	for _, item := range items {
		var thumbnailURL *string
		if item.Media.ID != uuid.Nil {
			thumbURL := "/media/shared/" + item.Media.ID.String() + "/thumb"
			thumbnailURL = &thumbURL
		}
		respItems = append(respItems, SharedAlbumMediaItem{
			ID:           item.Media.ID,
			Filename:     item.Media.Filename,
			Path:         item.Media.Path,
			MediaType:    string(item.Media.MediaType),
			CapturedAt:   item.Media.CapturedAt,
			ThumbnailURL: thumbnailURL,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SharedAlbumMediaResponse{
		Items:  respItems,
		Total:  totalItems,
		Limit:  limit,
		Offset: offset,
	})

	logResponse("handleListSharedAlbumMedia", startTime)
}

// handleListSharedAlbums returns a paginated list of albums shared with the current user.
func (s *Server) handleListSharedAlbums(w http.ResponseWriter, r *http.Request) {
	startTime := logRequest(r, "handleListSharedAlbums")

	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		logUnauthorized(r, "handleListSharedAlbums")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	limit := 20
	offset := 0
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, e := strconv.Atoi(lStr); e == nil && l > 0 {
			limit = l
		}
	}
	if oStr := r.URL.Query().Get("offset"); oStr != "" {
		if o, e := strconv.Atoi(oStr); e == nil && o >= 0 {
			offset = o
		}
	}

	logInfo("handleListSharedAlbums - Query params", fmt.Sprintf("limit=%d, offset=%d", limit, offset), startTime)

	items, totalItems, err := s.mediaShareRepo.ListSharedAlbums(ctx, userID, limit, offset)
	if err != nil {
		logError("handleListSharedAlbums - ListSharedAlbums", err, startTime)
		http.Error(w, "Failed to list shared albums: "+err.Error(), http.StatusInternalServerError)
		return
	}

	logInfo("handleListSharedAlbums - Found items", fmt.Sprintf("%d items (total=%d)", len(items), totalItems), startTime)

	respItems := make([]SharedAlbumFullResponse, 0, len(items))
	for _, item := range items {
		var thumbnailURL *string
		if item.FirstMediaID != uuid.Nil {
			thumbURL := "/media/shared/" + item.FirstMediaID.String() + "/thumb"
			thumbnailURL = &thumbURL
		}
		respItems = append(respItems, SharedAlbumFullResponse{
			ID:           item.Album.ID,
			Name:         item.Album.Name,
			Description:  item.Album.Description,
			UserID:       item.Album.UserID,
			CreatedAt:    item.Album.CreatedAt,
			ThumbnailURL: thumbnailURL,
			SharerUserID: item.SharerUserID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SharedAlbumsListResponse{
		Items:  respItems,
		Total:  totalItems,
		Limit:  limit,
		Offset: offset,
	})

	logResponse("handleListSharedAlbums", startTime)
}
