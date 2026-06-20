package api

// SharedMediaResponse includes path info so the frontend can generate thumbnails/serve files
import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"steadyphoto/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type SharedMediaFullResponse struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"userId"`
	Filename     string    `json:"filename"`
	Path         string    `json:"path"`
	MediaType    string    `json:"mediaType"`
	CapturedAt   time.Time `json:"capturedAt"`
	SharerUserID uuid.UUID `json:"sharerUserId"`
}

type SharedAlbumFullResponse struct {
	ID           uuid.UUID  `json:"id"`
	Name         string     `json:"name"`
	Description  *string    `json:"description,omitempty"`
	UserID       uuid.UUID  `json:"userId"`
	CreatedAt    time.Time  `json:"createdAt"`
	SharerUserID uuid.UUID  `json:"sharerUserId"`
}

type SharedAlbumMediaResponse struct {
	Items  []SharedAlbumMediaItem `json:"items"`
	Total  int                    `json:"total"`
	Limit  int                    `json:"limit"`
	Offset int                    `json:"offset"`
}

type SharedAlbumMediaItem struct {
	ID         uuid.UUID `json:"id"`
	Filename   string    `json:"filename"`
	Path       string    `json:"path"`
	MediaType  string    `json:"mediaType"`
	CapturedAt time.Time `json:"capturedAt"`
}

// handleGetSharedMedia returns metadata for a single media item shared with the current user.
func (s *Server) handleGetSharedMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Printf("[ERROR] handleGetSharedMedia - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	item, err := s.mediaShareRepo.GetSharedMediaByID(ctx, id, userID)
	if err != nil {
		log.Printf("[ERROR] handleGetSharedMedia - GetSharedMediaByID (%s): %v", id, err)
		http.Error(w, "Shared media not found or access denied", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SharedMediaFullResponse{
		ID:           item.Media.ID,
		UserID:       item.Media.UserID,
		Filename:     item.Media.Filename,
		Path:         item.Media.Path,
		MediaType:    string(item.Media.MediaType),
		CapturedAt:   item.Media.CapturedAt,
		SharerUserID: item.SharerUserID,
	})
}

// handleGetSharedMediaOriginal streams the actual file for a shared media item.
func (s *Server) handleGetSharedMediaOriginal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Printf("[ERROR] handleGetSharedMediaOriginal - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	item, err := s.mediaShareRepo.GetSharedMediaByID(ctx, id, userID)
	if err != nil {
		log.Printf("[ERROR] handleGetSharedMediaOriginal - GetSharedMediaByID (%s): %v", id, err)
		http.Error(w, "Shared media not found or access denied", http.StatusNotFound)
		return
	}

	targetPath := item.Media.Path
	absPath, err := s.storageService.ResolvePath(targetPath)
	if err != nil {
		targetPath = strings.TrimLeft(targetPath, "/\\")
		absPath, err = s.storageService.ResolvePath(targetPath)
	}

	if err != nil {
		log.Printf("[ERROR] handleGetSharedMediaOriginal - resolvePath (%s): %v", targetPath, err)
		http.Error(w, "Could not locate file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	contentType := s.getContentType(item.Media.MediaType, item.Media.Filename)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type, Accept-Ranges, Content-Range")

	http.ServeFile(w, r, absPath)
}

// handleGetSharedMediaThumb streams the thumbnail for a shared media item.
func (s *Server) handleGetSharedMediaThumb(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Printf("[ERROR] handleGetSharedMediaThumb - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	item, err := s.mediaShareRepo.GetSharedMediaByID(ctx, id, userID)
	if err != nil {
		log.Printf("[ERROR] handleGetSharedMediaThumb - GetSharedMediaByID (%s): %v", id, err)
		http.Error(w, "Shared media not found or access denied", http.StatusNotFound)
		return
	}

	relPath := filepath.Clean(item.Media.Path)
	ext := filepath.Ext(relPath)

	var thumbRelPath string
	if item.Media.MediaType == domain.MediaTypeVideo {
		cleanPath := strings.TrimPrefix(item.Media.Path, "storage/")
		basePart := strings.TrimSuffix(cleanPath, ext)
		basePart = strings.Replace(basePart, "/.videos/", "/", 1)
		thumbRelPath = basePart + ".webp"
	} else {
		cleanPath := strings.TrimPrefix(item.Media.Path, "storage/")
		basePart := strings.TrimSuffix(cleanPath, ext)
		thumbRelPath = basePart + "_thumb.webp"
	}

	fullThumbPath := filepath.Join(s.thumbRoot, ".thumbnails", thumbRelPath)

	if _, err := os.Stat(fullThumbPath); os.IsNotExist(err) {
		log.Printf("[WARN] handleGetSharedMediaThumb: Thumbnail NOT FOUND at %s. Attempting fallback to original.", fullThumbPath)
		targetPath := item.Media.Path
		absPath, resolveErr := s.storageService.ResolvePath(targetPath)
		if resolveErr != nil {
			targetPath = strings.TrimLeft(targetPath, "/\\")
			absPath, resolveErr = s.storageService.ResolvePath(targetPath)
		}

		if resolveErr != nil {
			log.Printf("[ERROR] handleGetSharedMediaThumb - fallback failed for (%s): %v", targetPath, resolveErr)
			http.Error(w, "Thumbnail not found and fallback failed", http.StatusNotFound)
			return
		}

		contentType := s.getContentType(item.Media.MediaType, item.Media.Filename)
		w.Header().Set("Content-Type", contentType)
		log.Printf("[INFO] handleGetSharedMediaThumb: Serving original file as fallback from %s", absPath)
		http.ServeFile(w, r, absPath)
		return
	}

	http.ServeFile(w, r, fullThumbPath)
}

// handleGetSharedAlbum returns metadata for a single album shared with the current user.
func (s *Server) handleGetSharedAlbum(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Printf("[ERROR] handleGetSharedAlbum - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	item, err := s.mediaShareRepo.GetSharedAlbumByID(ctx, id, userID)
	if err != nil {
		log.Printf("[ERROR] handleGetSharedAlbum - GetSharedAlbumByID (%s): %v", id, err)
		http.Error(w, "Shared album not found or access denied", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SharedAlbumFullResponse{
		ID:           item.Album.ID,
		Name:         item.Album.Name,
		Description:  item.Album.Description,
		UserID:       item.Album.UserID,
		CreatedAt:    item.Album.CreatedAt,
		SharerUserID: item.SharerUserID,
	})
}

// handleListSharedAlbumMedia returns media items in a shared album.
func (s *Server) handleListSharedAlbumMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Printf("[ERROR] handleListSharedAlbumMedia - parse UUID (%s): %v", idStr, err)
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

	items, totalItems, err := s.mediaShareRepo.ListMediaInSharedAlbum(ctx, id, userID, limit, offset)
	if err != nil {
		log.Printf("[ERROR] handleListSharedAlbumMedia - ListMediaInSharedAlbum (%s): %v", id, err)
		http.Error(w, "Failed to list shared album media: "+err.Error(), http.StatusInternalServerError)
		return
	}

	respItems := make([]SharedAlbumMediaItem, 0, len(items))
	for _, item := range items {
		respItems = append(respItems, SharedAlbumMediaItem{
			ID:         item.Media.ID,
			Filename:   item.Media.Filename,
			Path:       item.Media.Path,
			MediaType:  string(item.Media.MediaType),
			CapturedAt: item.Media.CapturedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SharedAlbumMediaResponse{
		Items:  respItems,
		Total:  totalItems,
		Limit:  limit,
		Offset: offset,
	})
}
