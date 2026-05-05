package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"steadyphoto/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	photoRepo   domain.PhotoRepository
	faceRepo    domain.FaceRepository
	storageRoot string
	thumbRoot   string
}

type ListPhotosResponse struct {
	Photos      []*domain.Photo
	TotalCount  int
	CurrentPage int
	TotalPages  int
}

func (h *Handler) ListPhotos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Simple pagination parsing
	limit := 20
	offset := 0

	// In a real app, we'd parse query params properly
	// For now, just a hardcoded example of the logic

	photos, total, err := h.photoRepo.List(ctx, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	totalPages := (total + limit - 1) / limit

	resp := ListPhotosResponse{
		Photos:      photos,
		TotalCount:  total,
		CurrentPage: (offset / limit) + 1,
		TotalPages:  totalPages,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) GetPhoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid uuid", http.StatusBadRequest)
		return
	}

	photo, err := h.photoRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "photo not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(photo)
}

func (h *Handler) ServePhotoFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid uuid", http.StatusBadRequest)
		return
	}

	photo, err := h.photoRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "photo not found", http.StatusNotFound)
		return
	}

	h.serveFile(w, r, h.storageRoot, photo.Path)
}

func (h *Handler) ServeThumbnailFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid uuid", http.StatusBadRequest)
		return
	}

	photo, err := h.photoRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "photo not found", http.StatusNotFound)
		return
	}

	// Calculate thumbnail path
	relPath := filepath.Clean(photo.Path)
	ext := filepath.Ext(relPath)
	base := strings.TrimSuffix(relPath, ext)
	thumbRelPath := filepath.Join(base + "_thumb.webp")

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
