package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
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
	Media       []*domain.Media
	TotalCount  int
	CurrentPage int
	TotalPages  int
}

func (h *Handler) ListMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Simple pagination parsing
	limit := 20
	offset := 0

	mediaList, total, err := h.mediaRepo.List(ctx, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	totalPages := (total + limit - 1) / limit

	resp := ListMediaResponse{
		Media:       mediaList,
		TotalCount:  total,
		CurrentPage: (offset / limit) + 1,
		TotalPages:  totalPages,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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
