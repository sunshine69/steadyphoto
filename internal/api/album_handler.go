package api

import (
	"encoding/json"
	"fmt"
	"github.com/jbrodriguez/mlog"
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
	mlog.Info("[DEBUG] AlbumHandler:CreateAlbum starting")
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		mlog.Info("[DEBUG] AlbumHandler:CreateAlbum - unauthorized (no userID in context)")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	mlog.Info("[DEBUG] AlbumHandler:CreateAlbum - authenticated user: %s", userID)

	var req CreateAlbumRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		mlog.Info("[ERROR] AlbumHandler:CreateAlbum - decode body error: %v", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	album, err := h.albumRepo.Create(ctx, req.Name, userID)
	if err != nil {
		mlog.Info("[ERROR] AlbumHandler:CreateAlbum - repository create error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mlog.Info("[DEBUG] AlbumHandler:CreateAlbum - success for album ID: %s", album.ID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(album)
}

// ListAlbums handles GET /api/v1/albums
func (h *AlbumHandler) ListAlbums(w http.ResponseWriter, r *http.Request) {
	mlog.Info("[DEBUG] AlbumHandler:ListAlbums starting")
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		mlog.Info("[DEBUG] AlbumHandler:ListAlbums - unauthorized (no userID in context)")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	mlog.Info("[DEBUG] AlbumHandler:ListAlbums - authenticated user: %s", userID)

	albums, err := h.albumRepo.List(ctx, userID)
	if err != nil {
		mlog.Info("[ERROR] AlbumHandler:ListAlbums - repository list error for user %s: %v", userID, err)
		http.Error(w, "failed to list albums: "+err.Error(), http.StatusInternalServerError)
		return
	}

	mlog.Info("[DEBUG] AlbumHandler:ListAlbums - success. Found %d albums", len(albums))
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(albums); err != nil {
		mlog.Info("[ERROR] AlbumHandler:ListAlbums - encode response error: %v", err)
	}
}

// GetAlbum handles GET /api/v1/albums/{id}
func (h *AlbumHandler) GetAlbum(w http.ResponseWriter, r *http.Request) {
	mlog.Info("[DEBUG] AlbumHandler:GetAlbum starting")
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		mlog.Info("[DEBUG] AlbumHandler:GetAlbum - unauthorized (no userID in context)")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	albumID, err := uuid.Parse(idStr)
	if err != nil {
		mlog.Info("[ERROR] AlbumHandler:GetAlbum - invalid album UUID %s: %v", idStr, err)
		http.Error(w, "invalid album id", http.StatusBadRequest)
		return
	}

	album, err := h.albumRepo.GetByID(ctx, albumID, userID)
	if err != nil {
		mlog.Info("[ERROR] AlbumHandler:GetAlbum - repository get error for ID %s (user %s): %v", albumID, userID, err)
		http.Error(w, "album not found", http.StatusNotFound)
		return
	}

	mlog.Info("[DEBUG] AlbumHandler:GetAlbum - success for album %s", albumID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(album)
}

// UpdateAlbum handles PUT /api/v1/albums/{id}
func (h *AlbumHandler) UpdateAlbum(w http.ResponseWriter, r *http.Request) {
	mlog.Info("[DEBUG] AlbumHandler:UpdateAlbum starting")
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		mlog.Info("[DEBUG] AlbumHandler:UpdateAlbum - unauthorized (no userID in context)")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	albumID, err := uuid.Parse(idStr)
	if err != nil {
		mlog.Info("[ERROR] AlbumHandler:UpdateAlbum - invalid album UUID %s: %v", idStr, err)
		http.Error(w, "invalid album id", http.StatusBadRequest)
		return
	}

	var req UpdateAlbumRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		mlog.Info("[ERROR] AlbumHandler:UpdateAlbum - decode body error for ID %s: %v", albumID, err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	album, err := h.albumRepo.GetByID(ctx, albumID, userID)
	if err != nil {
		mlog.Info("[ERROR] AlbumHandler:UpdateAlbum - repository get error for ID %s (user %s): %v", albumID, userID, err)
		http.Error(w, "album not found", http.StatusNotFound)
		return
	}

	album.Name = req.Name
	desc := req.Description
	album.Description = &desc

	if err := h.albumRepo.Update(ctx, album); err != nil {
		mlog.Info("[ERROR] AlbumHandler:UpdateAlbum - repository update error for ID %s: %v", albumID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mlog.Info("[DEBUG] AlbumHandler:UpdateAlbum - success for album %s", albumID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(album)
}

// DeleteAlbum handles DELETE /api/v1/albums/{id}
func (h *AlbumHandler) DeleteAlbum(w http.ResponseWriter, r *http.Request) {
	mlog.Info("[DEBUG] AlbumHandler:DeleteAlbum starting")
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		mlog.Info("[DEBUG] AlbumHandler:DeleteAlbum - unauthorized (no userID in context)")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	albumID, err := uuid.Parse(idStr)
	if err != nil {
		mlog.Info("[ERROR] AlbumHandler:DeleteAlbum - invalid album UUID %s: %v", idStr, err)
		http.Error(w, "invalid album id", http.StatusBadRequest)
		return
	}

	if err := h.albumRepo.Delete(ctx, albumID, userID); err != nil {
		mlog.Info("[ERROR] AlbumHandler:DeleteAlbum - repository delete error for ID %s (user %s): %v", albumID, userID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mlog.Info("[DEBUG] AlbumHandler:DeleteAlbum - success for album %s", albumID)
	w.WriteHeader(http.StatusNoContent)
}

// AddMediaToAlbum handles POST /api/v1/albums/{id}/media
func (h *AlbumHandler) AddMediaToAlbum(w http.ResponseWriter, r *http.Request) {
	mlog.Info("[DEBUG] AlbumHandler:AddMediaToAlbum starting")
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		mlog.Info("[DEBUG] AlbumHandler:AddMediaToAlbum - unauthorized (no userID in context)")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	albumID, err := uuid.Parse(idStr)
	if err != nil {
		mlog.Info("[ERROR] AlbumHandler:AddMediaToAlbum - invalid album UUID %s: %v", idStr, err)
		http.Error(w, "invalid album id", http.StatusBadRequest)
		return
	}

	// Verify ownership of album first
	_, err = h.albumRepo.GetByID(ctx, albumID, userID)
	if err != nil {
		mlog.Info("[ERROR] AlbumHandler:AddMediaToAlbum - unauthorized/not found for ID %s (user %s): %v", albumID, userID, err)
		http.Error(w, "unauthorized or album not found", http.StatusForbidden)
		return
	}

	var req AddMediaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		mlog.Info("[ERROR] AlbumHandler:AddMediaToAlbum - decode body error for ID %s: %v", albumID, err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// For security: Validate that all media IDs belong to the same user before adding
	for _, mID := range req.MediaIDs {
		media, err := h.mediaRepo.GetByID(ctx, mID, &userID)
		if err != nil {
			mlog.Info("[ERROR] AlbumHandler:AddMediaToAlbum - media item %s not found or access denied (user %s): %v", mID, userID, err)
			http.Error(w, fmt.Sprintf("media item %s not found or access denied", mID), http.StatusBadRequest)
			return
		}
		_ = media // validated
	}

	if err := h.albumRepo.AddMedia(ctx, albumID, req.MediaIDs); err != nil {
		mlog.Info("[ERROR] AlbumHandler:AddMediaToAlbum - repository add error for ID %s: %v", albumID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mlog.Info("[DEBUG] AlbumHandler:AddMediaToAlbum - success for album %s with %d media items", albumID, len(req.MediaIDs))
	w.WriteHeader(http.StatusNoContent)
}

// RemoveMediaFromAlbum handles DELETE /api/v1/albums/{id}/media/{media_id}
func (h *AlbumHandler) RemoveMediaFromAlbum(w http.ResponseWriter, r *http.Request) {
	mlog.Info("[DEBUG] AlbumHandler:RemoveMediaFromAlbum starting")
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		mlog.Info("[ERROR] AlbumHandler:RemoveMediaFromAlbum - unauthorized (no userID in context)")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	albumID, err := uuid.Parse(idStr)
	if err != nil {
		mlog.Info("[ERROR] AlbumHandler:RemoveMediaFromAlbum - invalid album UUID %s: %v", idStr, err)
		http.Error(w, "invalid album id", http.StatusBadRequest)
		return
	}

	mediaIDStr := chi.URLParam(r, "media_id")
	mediaID, err := uuid.Parse(mediaIDStr)
	if err != nil {
		mlog.Info("[ERROR] AlbumHandler:RemoveMediaFromAlbum - invalid media UUID %s: %v", mediaIDStr, err)
		http.Error(w, "invalid media id", http.StatusBadRequest)
		return
	}

	// Verify album ownership
	_, err = h.albumRepo.GetByID(ctx, albumID, userID)
	if err != nil {
		mlog.Info("[ERROR] AlbumHandler:RemoveMediaFromAlbum - unauthorized/not found for ID %s (user %s): %v", albumID, userID, err)
		http.Error(w, "unauthorized or album not found", http.StatusForbidden)
		return
	}

	if err := h.albumRepo.RemoveMedia(ctx, albumID, mediaID); err != nil {
		mlog.Info("[ERROR] AlbumHandler:RemoveMediaFromAlbum - repository remove error for ID %s (media %s): %v", albumID, mediaID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mlog.Info("[DEBUG] AlbumHandler:RemoveMediaFromAlbum - success for album %s (removed media %s)", albumID, mediaID)
	w.WriteHeader(http.StatusNoContent)
}

// BulkRemoveMediaFromAlbum handles DELETE /api/v1/albums/{id}/media
func (h *AlbumHandler) BulkRemoveMediaFromAlbum(w http.ResponseWriter, r *http.Request) {
	mlog.Info("[DEBUG] AlbumHandler:BulkRemoveMediaFromAlbum starting")
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		mlog.Info("[ERROR] AlbumHandler:BulkRemoveMediaFromAlbum - unauthorized (no userID in context)")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	albumID, err := uuid.Parse(idStr)
	if err != nil {
		mlog.Info("[ERROR] AlbumHandler:BulkRemoveMediaFromAlbum - invalid album UUID %s: %v", idStr, err)
		http.Error(w, "invalid album id", http.StatusBadRequest)
		return
	}

	// Verify ownership of album first
	_, err = h.albumRepo.GetByID(ctx, albumID, userID)
	if err != nil {
		mlog.Info("[ERROR] AlbumHandler:BulkRemoveMediaFromAlbum - unauthorized/not found for ID %s (user %s): %v", albumID, userID, err)
		http.Error(w, "unauthorized or album not found", http.StatusForbidden)
		return
	}

	var req BulkRemoveMediaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		mlog.Info("[ERROR] AlbumHandler:BulkRemoveMediaFromAlbum - decode body error: %v", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.albumRepo.BulkRemoveMedia(ctx, albumID, req.MediaIDs); err != nil {
		mlog.Info("[ERROR] AlbumHandler:BulkRemoveMediaFromAlbum - repository remove error for ID %s: %v", albumID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mlog.Info("[DEBUG] AlbumHandler:BulkRemoveMediaFromAlbum - success for album %s (removed %d items)", albumID, len(req.MediaIDs))
	w.WriteHeader(http.StatusNoContent)
}

// GetAlbumMedia handles GET /api/v1/albums/{id}/media with optional pagination (limit, offset, before, after)
// The endpoint supports two pagination modes:
// - Offset-based: limit + offset (default, returns items oldest-first by captured_at)
// - Cursor-based: limit + before (returns items OLDER than the given timestamp) or limit + after (returns items NEWER than the given timestamp)
func (h *AlbumHandler) GetAlbumMedia(w http.ResponseWriter, r *http.Request) {
	mlog.Info("[DEBUG] AlbumHandler:GetAlbumMedia starting")
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		mlog.Info("[DEBUG] AlbumHandler:GetAlbumMedia - unauthorized (no userID in context)")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	albumID, err := uuid.Parse(idStr)
	if err != nil {
		mlog.Info("[ERROR] AlbumHandler:GetAlbumMedia - invalid album UUID %s: %v", idStr, err)
		http.Error(w, "invalid album id", http.StatusBadRequest)
		return
	}

	// Check ownership of the album to ensure user can see its media via this endpoint
	_, err = h.albumRepo.GetByID(ctx, albumID, userID)
	if err != nil {
		mlog.Info("[ERROR] AlbumHandler:GetAlbumMedia - unauthorized/not found for ID %s (user %s): %v", albumID, userID, err)
		http.Error(w, "unauthorized or album not found", http.StatusForbidden)
		return
	}

	// Parse pagination parameters
	query := r.URL.Query()
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")
	beforeStr := query.Get("before")
	afterStr := query.Get("after")

	var limit int = 20 // Default page size
	var offset int = 0

	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}
	if offsetStr != "" {
		fmt.Sscanf(offsetStr, "%d", &offset)
	}

	// Ensure reasonable bounds
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// If "before" parameter is provided, use timestamp-based pagination for items OLDER than the given timestamp
	// If "after" parameter is provided, use timestamp-based pagination for items NEWER than the given timestamp
	// Otherwise, use offset-based pagination (default)
	if beforeStr != "" && afterStr == "" {
		// Load items OLDER than the given timestamp
		mlog.Info("[DEBUG] AlbumHandler:GetAlbumMedia - pagination: limit=%d, before=%s", limit, beforeStr)
		mediaList, totalItems, err := h.albumRepo.GetMediaPaginatedBefore(ctx, albumID, userID, limit, beforeStr)
		if err != nil {
			mlog.Info("[ERROR] AlbumHandler:GetAlbumMedia - repository get error for ID %s: %v", albumID, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"media":      mediaList,
			"totalItems": totalItems,
			"limit":      limit,
			"before":     beforeStr,
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			mlog.Info("[ERROR] AlbumHandler:GetAlbumMedia - encode response error: %v", err)
		}
		return
	} else if afterStr != "" && beforeStr == "" {
		// Load items NEWER than the given timestamp
		mlog.Info("[DEBUG] AlbumHandler:GetAlbumMedia - pagination: limit=%d, after=%s", limit, afterStr)
		mediaList, totalItems, err := h.albumRepo.GetMediaPaginatedAfter(ctx, albumID, userID, limit, afterStr)
		if err != nil {
			mlog.Info("[ERROR] AlbumHandler:GetAlbumMedia - repository get error for ID %s: %v", albumID, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"media":      mediaList,
			"totalItems": totalItems,
			"limit":      limit,
			"after":      afterStr,
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			mlog.Info("[ERROR] AlbumHandler:GetAlbumMedia - encode response error: %v", err)
		}
		return
	} else {
		// Offset-based pagination (default)
		mlog.Info("[DEBUG] AlbumHandler:GetAlbumMedia - pagination: limit=%d, offset=%d", limit, offset)
		mediaList, totalItems, err := h.albumRepo.GetMediaPaginated(ctx, albumID, userID, limit, offset)
		if err != nil {
			mlog.Info("[ERROR] AlbumHandler:GetAlbumMedia - repository get error for ID %s: %v", albumID, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Construct response with pagination metadata
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"media":      mediaList,
			"totalItems": totalItems,
			"limit":      limit,
			"offset":     offset,
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			mlog.Info("[ERROR] AlbumHandler:GetAlbumMedia - encode response error: %v", err)
		}
	}
}
