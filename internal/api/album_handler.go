package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"steadyphoto/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type AlbumHandler struct {
	albumRepo domain.AlbumRepository
	mediaRepo domain.MediaRepository
}

func NewAlbumHandler(albumRepo domain.AlbumRepository, mediaRepo domain.MediaRepository) *AlbumHandler {
	return &AlbumHandler{
		albumRepo: albumRepo,
		mediaRepo: mediaRepo,
	}
}

type CreateAlbumRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateAlbumRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type AddMediaRequest struct {
	MediaIDs []uuid.UUID `json:"mediaIds"`
}

type BulkRemoveMediaRequest struct {
	MediaIDs []uuid.UUID `json:"mediaIds"`
}

// CreateAlbum handles POST /api/v1/albums
func (h *AlbumHandler) CreateAlbum(w http.ResponseWriter, r *http.Request) {
	log.Printf("[DEBUG] AlbumHandler:CreateAlbum starting")
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		log.Printf("[DEBUG] AlbumHandler:CreateAlbum - unauthorized (no userID in context)")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	log.Printf("[DEBUG] AlbumHandler:CreateAlbum - authenticated user: %s", userID)

	var req CreateAlbumRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[ERROR] AlbumHandler:CreateAlbum - decode body error: %v", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	album, err := h.albumRepo.Create(ctx, req.Name, userID)
	if err != nil {
		log.Printf("[ERROR] AlbumHandler:CreateAlbum - repository create error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[DEBUG] AlbumHandler:CreateAlbum - success for album ID: %s", album.ID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(album)
}

// ListAlbums handles GET /api/v1/albums
func (h *AlbumHandler) ListAlbums(w http.ResponseWriter, r *http.Request) {
	log.Printf("[DEBUG] AlbumHandler:ListAlbums starting")
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		log.Printf("[DEBUG] AlbumHandler:ListAlbums - unauthorized (no userID in context)")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	log.Printf("[DEBUG] AlbumHandler:ListAlbums - authenticated user: %s", userID)

	albums, err := h.albumRepo.List(ctx, userID)
	if err != nil {
		log.Printf("[ERROR] AlbumHandler:ListAlbums - repository list error for user %s: %v", userID, err)
		http.Error(w, "failed to list albums: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[DEBUG] AlbumHandler:ListAlbums - success. Found %d albums", len(albums))
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(albums); err != nil {
		log.Printf("[ERROR] AlbumHandler:ListAlbums - encode response error: %v", err)
	}
}

// GetAlbum handles GET /api/v1/albums/{id}
func (h *AlbumHandler) GetAlbum(w http.ResponseWriter, r *http.Request) {
	log.Printf("[DEBUG] AlbumHandler:GetAlbum starting")
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		log.Printf("[DEBUG] AlbumHandler:GetAlbum - unauthorized (no userID in context)")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	albumID, err := uuid.Parse(idStr)
	if err != nil {
		log.Printf("[ERROR] AlbumHandler:GetAlbum - invalid album UUID %s: %v", idStr, err)
		http.Error(w, "invalid album id", http.StatusBadRequest)
		return
	}

	album, err := h.albumRepo.GetByID(ctx, albumID, userID)
	if err != nil {
		log.Printf("[ERROR] AlbumHandler:GetAlbum - repository get error for ID %s (user %s): %v", albumID, userID, err)
		http.Error(w, "album not found", http.StatusNotFound)
		return
	}

	log.Printf("[DEBUG] AlbumHandler:GetAlbum - success for album %s", albumID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(album)
}

// UpdateAlbum handles PUT /api/v1/albums/{id}
func (h *AlbumHandler) UpdateAlbum(w http.ResponseWriter, r *http.Request) {
	log.Printf("[DEBUG] AlbumHandler:UpdateAlbum starting")
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		log.Printf("[DEBUG] AlbumHandler:UpdateAlbum - unauthorized (no userID in context)")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	albumID, err := uuid.Parse(idStr)
	if err != nil {
		log.Printf("[ERROR] AlbumHandler:UpdateAlbum - invalid album UUID %s: %v", idStr, err)
		http.Error(w, "invalid album id", http.StatusBadRequest)
		return
	}

	var req UpdateAlbumRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[ERROR] AlbumHandler:UpdateAlbum - decode body error for ID %s: %v", albumID, err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	album, err := h.albumRepo.GetByID(ctx, albumID, userID)
	if err != nil {
		log.Printf("[ERROR] AlbumHandler:UpdateAlbum - repository get error for ID %s (user %s): %v", albumID, userID, err)
		http.Error(w, "album not found", http.StatusNotFound)
		return
	}

	album.Name = req.Name
	desc := req.Description
	album.Description = &desc

	if err := h.albumRepo.Update(ctx, album); err != nil {
		log.Printf("[ERROR] AlbumHandler:UpdateAlbum - repository update error for ID %s: %v", albumID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[DEBUG] AlbumHandler:UpdateAlbum - success for album %s", albumID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(album)
}

// DeleteAlbum handles DELETE /api/v1/albums/{id}
func (h *AlbumHandler) DeleteAlbum(w http.ResponseWriter, r *http.Request) {
	log.Printf("[DEBUG] AlbumHandler:DeleteAlbum starting")
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		log.Printf("[DEBUG] AlbumHandler:DeleteAlbum - unauthorized (no userID in context)")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	albumID, err := uuid.Parse(idStr)
	if err != nil {
		log.Printf("[ERROR] AlbumHandler:DeleteAlbum - invalid album UUID %s: %v", idStr, err)
		http.Error(w, "invalid album id", http.StatusBadRequest)
		return
	}

	if err := h.albumRepo.Delete(ctx, albumID, userID); err != nil {
		log.Printf("[ERROR] AlbumHandler:DeleteAlbum - repository delete error for ID %s (user %s): %v", albumID, userID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[DEBUG] AlbumHandler:DeleteAlbum - success for album %s", albumID)
	w.WriteHeader(http.StatusNoContent)
}

// AddMediaToAlbum handles POST /api/v1/albums/{id}/media
func (h *AlbumHandler) AddMediaToAlbum(w http.ResponseWriter, r *http.Request) {
	log.Printf("[DEBUG] AlbumHandler:AddMediaToAlbum starting")
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		log.Printf("[DEBUG] AlbumHandler:AddMediaToAlbum - unauthorized (no userID in context)")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	albumID, err := uuid.Parse(idStr)
	if err != nil {
		log.Printf("[ERROR] AlbumHandler:AddMediaToAlbum - invalid album UUID %s: %v", idStr, err)
		http.Error(w, "invalid album id", http.StatusBadRequest)
		return
	}

	// Verify ownership of album first
	_, err = h.albumRepo.GetByID(ctx, albumID, userID)
	if err != nil {
		log.Printf("[ERROR] AlbumHandler:AddMediaToAlbum - unauthorized/not found for ID %s (user %s): %v", albumID, userID, err)
		http.Error(w, "unauthorized or album not found", http.StatusForbidden)
		return
	}

	var req AddMediaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[ERROR] AlbumHandler:AddMediaToAlbum - decode body error for ID %s: %v", albumID, err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// For security: Validate that all media IDs belong to the same user before adding
	for _, mID := range req.MediaIDs {
		media, err := h.mediaRepo.GetByID(ctx, mID, &userID)
		if err != nil {
			log.Printf("[ERROR] AlbumHandler:AddMediaToAlbum - media item %s not found or access denied (user %s): %v", mID, userID, err)
			http.Error(w, fmt.Sprintf("media item %s not found or access denied", mID), http.StatusBadRequest)
			return
		}
		_ = media // validated
	}

	if err := h.albumRepo.AddMedia(ctx, albumID, req.MediaIDs); err != nil {
		log.Printf("[ERROR] AlbumHandler:AddMediaToAlbum - repository add error for ID %s: %v", albumID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[DEBUG] AlbumHandler:AddMediaToAlbum - success for album %s with %d media items", albumID, len(req.MediaIDs))
	w.WriteHeader(http.StatusNoContent)
}

// RemoveMediaFromAlbum handles DELETE /api/v1/albums/{id}/media/{media_id}
func (h *AlbumHandler) RemoveMediaFromAlbum(w http.ResponseWriter, r *http.Request) {
	log.Printf("[DEBUG] AlbumHandler:RemoveMediaFromAlbum starting")
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		log.Printf("[ERROR] AlbumHandler:RemoveMediaFromAlbum - unauthorized (no userID in context)")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	albumID, err := uuid.Parse(idStr)
	if err != nil {
		log.Printf("[ERROR] AlbumHandler:RemoveMediaFromAlbum - invalid album UUID %s: %v", idStr, err)
		http.Error(w, "invalid album id", http.StatusBadRequest)
		return
	}

	mediaIDStr := chi.URLParam(r, "media_id")
	mediaID, err := uuid.Parse(mediaIDStr)
	if err != nil {
		log.Printf("[ERROR] AlbumHandler:RemoveMediaFromAlbum - invalid media UUID %s: %v", mediaIDStr, err)
		http.Error(w, "invalid media id", http.StatusBadRequest)
		return
	}

	// Verify album ownership
	_, err = h.albumRepo.GetByID(ctx, albumID, userID)
	if err != nil {
		log.Printf("[ERROR] AlbumHandler:RemoveMediaFromAlbum - unauthorized/not found for ID %s (user %s): %v", albumID, userID, err)
		http.Error(w, "unauthorized or album not found", http.StatusForbidden)
		return
	}

	if err := h.albumRepo.RemoveMedia(ctx, albumID, mediaID); err != nil {
		log.Printf("[ERROR] AlbumHandler:RemoveMediaFromAlbum - repository remove error for ID %s (media %s): %v", albumID, mediaID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[DEBUG] AlbumHandler:RemoveMediaFromAlbum - success for album %s (removed media %s)", albumID, mediaID)
	w.WriteHeader(http.StatusNoContent)
}

// BulkRemoveMediaFromAlbum handles DELETE /api/v1/albums/{id}/media
func (h *AlbumHandler) BulkRemoveMediaFromAlbum(w http.ResponseWriter, r *http.Request) {
	log.Printf("[DEBUG] AlbumHandler:BulkRemoveMediaFromAlbum starting")
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		log.Printf("[ERROR] AlbumHandler:BulkRemoveMediaFromAlbum - unauthorized (no userID in context)")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	albumID, err := uuid.Parse(idStr)
	if err != nil {
		log.Printf("[ERROR] AlbumHandler:BulkRemoveMediaFromAlbum - invalid album UUID %s: %v", idStr, err)
		http.Error(w, "invalid album id", http.StatusBadRequest)
		return
	}

	// Verify ownership of album first
	_, err = h.albumRepo.GetByID(ctx, albumID, userID)
	if err != nil {
		log.Printf("[ERROR] AlbumHandler:BulkRemoveMediaFromAlbum - unauthorized/not found for ID %s (user %s): %v", albumID, userID, err)
		http.Error(w, "unauthorized or album not found", http.StatusForbidden)
		return
	}

	var req BulkRemoveMediaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[ERROR] AlbumHandler:BulkRemoveMediaFromAlbum - decode body error: %v", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.albumRepo.BulkRemoveMedia(ctx, albumID, req.MediaIDs); err != nil {
		log.Printf("[ERROR] AlbumHandler:BulkRemoveMediaFromAlbum - repository remove error for ID %s: %v", albumID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[DEBUG] AlbumHandler:BulkRemoveMediaFromAlbum - success for album %s (removed %d items)", albumID, len(req.MediaIDs))
	w.WriteHeader(http.StatusNoContent)
}

// GetAlbumMedia handles GET /api/v1/albums/{id}/media
func (h *AlbumHandler) GetAlbumMedia(w http.ResponseWriter, r *http.Request) {
	log.Printf("[DEBUG] AlbumHandler:GetAlbumMedia starting")
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		log.Printf("[DEBUG] AlbumHandler:GetAlbumMedia - unauthorized (no userID in context)")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	albumID, err := uuid.Parse(idStr)
	if err != nil {
		log.Printf("[ERROR] AlbumHandler:GetAlbumMedia - invalid album UUID %s: %v", idStr, err)
		http.Error(w, "invalid album id", http.StatusBadRequest)
		return
	}

	// Check ownership of the album to ensure user can see its media via this endpoint
	_, err = h.albumRepo.GetByID(ctx, albumID, userID)
	if err != nil {
		log.Printf("[ERROR] AlbumHandler:GetAlbumMedia - unauthorized/not found for ID %s (user %s): %v", albumID, userID, err)
		http.Error(w, "unauthorized or album not found", http.StatusForbidden)
		return
	}

	mediaList, err := h.albumRepo.GetMedia(ctx, albumID, userID)
	if err != nil {
		log.Printf("[ERROR] AlbumHandler:GetAlbumMedia - repository get error for ID %s: %v", albumID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[DEBUG] AlbumHandler:GetAlbumMedia - success for album %s (%d items)", albumID, len(mediaList))
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(mediaList); err != nil {
		log.Printf("[ERROR] AlbumHandler:GetAlbumMedia - encode response error: %v", err)
	}
}
