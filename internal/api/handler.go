package api

import (
	"encoding/json"
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
	Total  int              `json:"total"`
}

// ListMedia handles GET /api/v1/media
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

	mediaList, total, err := h.mediaRepo.List(ctx, limit, offset)
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

// ListPhotos handles GET /api/v1/photos
// Backward compatible endpoint for frontend photo listing
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

	mediaList, total, err := h.mediaRepo.ListByType(ctx, domain.MediaTypePhoto, limit, offset)
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
// Backward compatible endpoint for frontend photo retrieval
func (h *Handler) GetPhoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid uuid", http.StatusBadRequest)
		return
	}

	media, err := h.mediaRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "media not found", http.StatusNotFound)
		return
	}
	
	// We allow videos too as they are part of the same media collection
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(media)
}

// ServePhotoFile handles GET /api/v1/photos/{id}/file
// Backward compatible endpoint for serving photo files
func (h *Handler) ServePhotoFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid uuid", http.StatusBadRequest)
		return
	}

	media, err := h.mediaRepo.GetByID(ctx, id)
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

	media, err := h.mediaRepo.GetByID(ctx, id)
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

	media, err := h.mediaRepo.GetByID(ctx, id)
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

	media, err := h.mediaRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "media not found", http.StatusNotFound)
		return
	}

	// Calculate thumbnail path
	relPath := filepath.Clean(media.Path)
	ext := filepath.Ext(relPath)
	base := strings.TrimSuffix(relPath, ext)
	
	var thumbRelPath string
	if media.MediaType == domain.MediaTypeVideo {
		// For videos, thumbnails are in .thumbnails/ directory
		thumbRelPath = filepath.Join(".thumbnails", base+".webp")
	} else {
		thumbRelPath = filepath.Join(base+"_thumb.webp")
	}

	fullThumbPath := filepath.Join(h.thumbRoot, thumbRelPath)
	if _, err := os.Stat(fullThumbPath); os.IsNotExist(err) {
		// Fallback: serve the original file if thumbnail doesn't exist yet
		h.serveFile(w, r, h.storageRoot, media.Path)
		return
	}

	h.serveFile(w, r, h.thumbRoot, thumbRelPath)
}

func (h *Handler) serveFile(w http.ResponseWriter, r *http.Request, root string, relPath string) {
	absRoot, _ := filepath.Abs(root)
	absFile, err := filepath.Abs(filepath.Join(absRoot, relPath))
	if err != nil {
		http.Error(w, "invalid path", http.StatusInternalServerError)
		return
	}

	if !filepath.HasPrefix(absFile, absRoot) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	http.ServeFile(w, r, absFile)
}
