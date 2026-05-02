package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"steadyphoto/internal/domain"
)

type PhotoHandler struct {
	repo       domain.PhotoRepository
	storageRoot string
}

func NewPhotoHandler(repo domain.PhotoRepository, storageRoot string) *PhotoHandler {
	return &PhotoHandler{
		repo:        repo,
		storageRoot: storageRoot,
	}
}

// ListPhotos returns a paginated list of photos
func (h *PhotoHandler) ListPhotos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Parse pagination params
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	photos, total, err := h.repo.List(ctx, limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list photos: %v", err), http.StatusInternalServerError)
		return
	}

	response := struct {
		Data  []*domain.Photo `json:"data"`
		Total int             `json:"total"`
		Page  int             `json:"page"`
		Limit int             `json:"limit"`
	}{
		Data:  photos,
		Total: total,
		Page:  page,
		Limit: limit,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetPhoto returns metadata for a single photo
func (h *PhotoHandler) GetPhoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid photo id", http.StatusBadRequest)
		return
	}

	photo, err := h.repo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "photo not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(photo)
}

// ServePhotoFile streams the actual image file
func (h *PhotoHandler) ServePhotoFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid photo id", http.StatusBadRequest)
		return
	}

	photo, err := h.repo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "photo not found", http.StatusNotFound)
		return
	}

	// Safety check: Ensure the file path is within the storage root
	// to prevent path traversal attacks.
	absStorage, _ := filepath.Abs(h.storageRoot)
	absFile, err := filepath.Abs(photo.Path)
	if err != nil || !filepath.HasPrefix(absFile, absStorage) {
		http.Error(w, "forbidden: invalid file path", http.StatusForbidden)
		return
	}

	// Check if file exists
	if _, err := os.Stat(absFile); os.IsNotExist(err) {
		http.Error(w, "file not found on disk", http.StatusNotFound)
		return
	}

	// http.ServeFile handles Range requests, Content-Type, and ETag automatically
	http.ServeFile(w, r, absFile)
}
