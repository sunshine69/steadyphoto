package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"steadyphoto/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	mediaRepo   domain.MediaRepository
	faceRepo    domain.FaceRepository
	storageRoot string
	thumbRoot   string
}

type ListMediaResponse struct {
	Photos []*domain.Media `json:"photos"`
	Total  int             `json:"total"`
}

type SearchMediaResponse struct {
	Photos []*domain.Media `json:"photos"`
	Total  int             `json:"total"`
}

// ListMedia handles GET /api/v1/media (now user-scoped via context if available)
func (h *Handler) ListMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse pagination parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	offset := 0
	if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
		offset = o
	}

	// Note: In the current Handler version (not Server), we don't have access to userID directly via context here yet because this is a legacy-style handler.
	// However, for consistency with our new Repository interface, we pass nil as user_id if it can't be determined from ctx or handled by middleware at Server level.
	mediaList, total, err := h.mediaRepo.List(ctx, limit, offset, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := ListMediaResponse{
		Photos: mediaList,
		Total:  total,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// SearchMedia handles GET /api/v1/media/search?tags=... (now user-scoped via context if available)
func (h *Handler) SearchMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tags := r.URL.Query().Get("tags")
	if tags == "" {
		http.Error(w, "tags parameter is required", http.StatusBadRequest)
		return
	}

	// Passing nil for user_id as the legacy Handler doesn't have direct access to it yet via context easily in this implementation pattern
	mediaList, err := h.mediaRepo.SearchByTags(ctx, tags, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := SearchMediaResponse{
		Photos: mediaList,
		Total:  len(mediaList),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ListPhotos handles GET /api/v1/photos
// Backward compatible endpoint for frontend photo listing (now user-scoped via context if available)
func (h *Handler) ListPhotos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse pagination parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	offset := 0
	if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
		offset = o
	}

	// Passing nil for user_id as the legacy Handler doesn't have direct access to it yet via context easily in this implementation pattern
	mediaList, total, err := h.mediaRepo.ListByType(ctx, domain.MediaTypePhoto, limit, offset, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := ListMediaResponse{
		Photos: mediaList,
		Total:  total,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetPhoto handles GET /api/v1/photos/{id}
// Backward compatible endpoint for frontend photo retrieval (now user-scoped via context if available)
func (h *Handler) GetPhoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid uuid", http.StatusBadRequest)
		return
	}

	// Passing nil for user_id as the legacy Handler doesn't have direct access to it yet via context easily in this implementation pattern
	media, err := h.mediaRepo.GetByID(ctx, id, nil)
	if err != nil {
		http.Error(w, "media not found", http.StatusNotFound)
		return
	}

	// We allow videos too as they are part of the same media collection
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(media)
}

// ServePhotoFile handles GET /api/v1/photos/{id}/file
// Backward compatible endpoint for serving photo files (now user-scoped via context if available)
func (h *Handler) ServePhotoFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid uuid", http.StatusBadRequest)
		return
	}

	// Passing nil for user_id as the legacy Handler doesn't have direct access to it yet via context easily in this implementation pattern
	media, err := h.mediaRepo.GetByID(ctx, id, nil)
	if err != nil {
		http.Error(w, "media not found", http.StatusNotFound)
		return
	}

	h.serveFile(w, r, h.storageRoot, media.Path)
}

func (h *Handler) GetMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid uuid", http.StatusBadRequest)
		return
	}

	// Passing nil for user_id as the legacy Handler doesn't have direct access to it yet via context easily in this implementation pattern
	media, err := h.mediaRepo.GetByID(ctx, id, nil)
	if err != nil {
		http.Error(w, "media not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(media)
}

func (h *Handler) ServeMediaFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid uuid", http.StatusBadRequest)
		return
	}

	// Passing nil for user_id as the legacy Handler doesn't have direct access to it yet via context easily in this implementation pattern
	media, err := h.mediaRepo.GetByID(ctx, id, nil)
	if err != nil {
		http.Error(w, "media not found", http.StatusNotFound)
		return
	}

	h.serveFile(w, r, h.storageRoot, media.Path)
}

func (h *Handler) ServeThumbnailFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid uuid", http.StatusBadRequest)
		return
	}

	// Passing nil for user_id as the legacy Handler doesn't have direct access to it yet via context easily in this implementation pattern
	media, err := h.mediaRepo.GetByID(ctx, id, nil)
	if err != nil {
		http.Error(w, "media not found", http.StatusNotFound)
		return
	}
	fmt.Printf("[DEBUG] h.mediaRepo.GetByID Output - %v\n", media)
	// Calculate thumbnail path
	cleanPath := strings.TrimPrefix(media.Path, "storage/")
	extClean := filepath.Ext(cleanPath)
	basePart := strings.TrimSuffix(cleanPath, extClean)

	var thumbRelPath string
	if media.MediaType == domain.MediaTypeVideo {
		// For videos, thumbnails are in storage/.thumbnails/YYYY/MM/DD/filename.webp
		// We strip '/.videos/' from the path to place it under '.thumbnails/'
		thumbRelPath = strings.Replace(basePart, "/.videos/", "/", 1) + ".webp"
	} else {
		// For photos, thumbnail is in storage/.thumbnails/YYYY/MM/DD/filename_thumb.webp
		thumbRelPath = basePart + "_thumb.webp"
	}

	// Construct the absolute-ish path within the storage directory structure
	// Since we already have a clean relative path (starting with YYYY/...),
	// joining it with storage/.thumbnails results in storage/.thumbnails/YYYY/...
	fullThumbPath := filepath.Join(h.thumbRoot, ".thumbnails", thumbRelPath)

	// Check if the calculated thumbnail exists
	if _, err := os.Stat(fullThumbPath); os.IsNotExist(err) {
		// Fallback: serve the original file if thumbnail doesn't exist yet
		h.serveFile(w, r, h.storageRoot, media.Path)
		return
	}

	// Serve the thumbnail using its path relative to the thumbRoot
	relToThumbRoot, _ := filepath.Rel(h.thumbRoot, fullThumbPath)
	h.serveFile(w, r, h.thumbRoot, relToThumbRoot)
}

// UpdateMediaTags handles PATCH /api/v1/media/{id}/tags to update tags for a media item (now user-scoped via context if available)
func (h *Handler) UpdateMediaTags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid uuid", http.StatusBadRequest)
		return
	}

	var request struct {
		Tags string `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Passing nil for user_id as the legacy Handler doesn't have direct access to it yet via context easily in this implementation pattern
	media, err := h.mediaRepo.GetByID(ctx, id, nil)
	if err != nil {
		http.Error(w, "media not found", http.StatusNotFound)
		return
	}

	// Update tags
	media.Tags = request.Tags

	if err := h.mediaRepo.Update(ctx, media); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (h *Handler) serveFile(w http.ResponseWriter, r *http.Request, root string, relPath string) {
	absRoot, _ := filepath.Abs(root)
	absFile, err := filepath.Abs(filepath.Join(absRoot, relPath))
	if err != nil {
		http.Error(w, "invalid path", http.StatusInternalServerError)
		return
	}

	if !strings.HasPrefix(absFile, absRoot) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	http.ServeFile(w, r, absFile)
}
