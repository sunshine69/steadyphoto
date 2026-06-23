package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"steadyphoto/internal/domain"
	"steadyphoto/internal/security"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ShareHandler struct {
	shareRepo         domain.ShareRepository
	mediaShareRepo    domain.MediaShareRepository
	albumShareRepo    domain.AlbumShareRepository
	publicShareRepo   domain.PublicShareRepository
	publicAccessRepo  domain.PublicShareAccessRepository
	userRepo          domain.UserRepository
	albumRepo         domain.AlbumRepository // For ownership validation of albums being shared
	mediaRepo         domain.MediaRepository // For ownership validation of media being shared
}

func NewShareHandler(shareRepo domain.ShareRepository, mediaShareRepo domain.MediaShareRepository, albumShareRepo domain.AlbumShareRepository, publicShareRepo domain.PublicShareRepository, publicAccessRepo domain.PublicShareAccessRepository, userRepo domain.UserRepository, albumRepo domain.AlbumRepository, mediaRepo domain.MediaRepository) *ShareHandler {
	return &ShareHandler{
		shareRepo:        shareRepo,
		mediaShareRepo:   mediaShareRepo,
		albumShareRepo:   albumShareRepo,
		publicShareRepo:  publicShareRepo,
		publicAccessRepo: publicAccessRepo,
		userRepo:         userRepo,
		albumRepo:        albumRepo,
		mediaRepo:        mediaRepo,
	}
}

// CreateShareRequest represents the request body for creating a share.
type CreateShareRequest struct {
	SharerUserID    uuid.UUID   `json:"-"` // Set from context (the authenticated user)
	ShareeUserIDs   []uuid.UUID `json:"sharee_user_ids"`
	MediaIDs        []uuid.UUID `json:"media_ids,omitempty"`
	AlbumIDs        []uuid.UUID `json:"album_ids,omitempty"`
}

// ShareResponse is a single share in API responses.
type ShareResponse struct {
	ID           uuid.UUID `json:"id"`
	SharerUserID uuid.UUID `json:"sharerUserId"`
	SharedAt     time.Time `json:"sharedAt"`
}

// SharedMediaResponse represents a shared media item for the API response.
type SharedMediaResponse struct {
	ID           uuid.UUID   `json:"id"`
	Filename     string      `json:"filename"`
	Path         string      `json:"path"`
	MediaType    string      `json:"mediaType"`
	SharerUserID uuid.UUID   `json:"sharerUserId"`
}

// SharedAlbumResponse represents a shared album for the API response.
type SharedAlbumResponse struct {
	ID           uuid.UUID   `json:"id"`
	Name         string      `json:"name"`
	Description  *string     `json:"description,omitempty"`
	SharerUserID uuid.UUID   `json:"sharerUserId"`
	Thumbnail    *string     `json:"thumbnail,omitempty"` // First photo thumbnail path
}

// CreatePublicShareRequest represents the request body for creating a public share link.
type CreatePublicShareRequest struct {
	ResourceType string  `json:"resource_type"` // "media" or "album"
	ResourceID   uuid.UUID `json:"resource_id"`
	Password     *string `json:"password,omitempty"` // Optional password for protection
	ExpiresAt    *time.Time `json:"expires_at,omitempty"` // Optional expiration date/time
}

// PublicShareLinkResponse represents the response after creating a public share link.
type PublicShareLinkResponse struct {
	ID                uuid.UUID  `json:"id"`
	Token             string     `json:"token"`
	SharerUserID      uuid.UUID  `json:"sharerUserId"`
	ResourceType      string     `json:"resourceType"`
	PasswordProtected bool       `json:"password_protected"`
	ExpiresAt         *time.Time `json:"expires_at"`
	CreatedAt         time.Time  `json:"created_at"`
}

// PublicShareListResponse represents the list of public share links.
type PublicShareListResponse struct {
	ID                uuid.UUID     `json:"id"`
	Token             string        `json:"token"`
	SharerUserID      uuid.UUID     `json:"sharerUserId"`
	ResourceType      string        `json:"resourceType"`
	ResourceID        uuid.UUID     `json:"-"` // Not exposed in response
	PasswordProtected bool          `json:"password_protected"`
	ExpiresAt         *time.Time    `json:"expires_at"`
	CreatedAt         time.Time     `json:"created_at"`
	AccessCount       int           `json:"access_count"`
}

// SharedAlbumWithMedia represents an album with its media items for public share responses.
type SharedAlbumWithMedia struct {
	ID           uuid.UUID        `json:"id"`
	Name         string           `json:"name"`
	Description  *string          `json:"description,omitempty"`
	CreatedAt    time.Time        `json:"createdAt"`
	UpdatedAt    time.Time        `json:"updatedAt"`
	MediaItems   []MediaShareItem `json:"media_items"`
}

// MediaShareItem represents a media item in a shared album.
type MediaShareItem struct {
	ID        uuid.UUID `json:"id"`
	Filename  string    `json:"filename"`
	Path      string    `json:"path"`
	MediaType string    `json:"mediaType"`
}

// handleCreateShare handles POST /api/v1/shares — Create a share.
func (h *ShareHandler) handleCreateShare(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[ERROR] handleCreateShare - decode body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.SharerUserID = userID

	// Validate that at least one media or album is being shared
	if len(req.MediaIDs) == 0 && len(req.AlbumIDs) == 0 {
		http.Error(w, "At least one media item or album must be shared", http.StatusBadRequest)
		return
	}

	// Validate sharee users and check they exist + are active
	for _, shareeID := range req.ShareeUserIDs {
		if shareeID == userID {
			http.Error(w, "Cannot share with yourself", http.StatusBadRequest)
			return
		}
		user, err := h.userRepo.GetByID(ctx, shareeID)
		if err != nil {
			log.Printf("[ERROR] handleCreateShare - user not found: %v", err)
			http.Error(w, "User not found", http.StatusBadRequest)
			return
		}
		if user.Status == domain.UserStatusDisabled || user.Status == domain.UserStatusRejected {
			log.Printf("[WARN] handleCreateShare - cannot share with disabled/rejected user: %s", shareeID)
			http.Error(w, "Cannot share with a disabled or rejected user", http.StatusBadRequest)
			return
		}
		if user.ID == userID {
			http.Error(w, "Cannot share with yourself", http.StatusBadRequest)
			return
		}
	}

	// Validate that media items belong to the sharer
	for _, mediaID := range req.MediaIDs {
		log.Printf("[DEBUG] handleCreateShare - validating ownership for mediaID: %s, userID: %s", mediaID, userID)
		media, err := h.mediaRepo.GetByID(ctx, mediaID, &userID)
		if err != nil || media == nil {
			log.Printf("[ERROR] handleCreateShare - media not found or access denied for mediaID=%s, userID=%s: %v", mediaID, userID, err)
			http.Error(w, "One or more media items do not belong to you", http.StatusBadRequest)
			return
		}
		log.Printf("[DEBUG] handleCreateShare - ownership validated for mediaID: %s (filename: %s)", media.ID, media.Filename)
		_ = media // validated
	}

	// Validate that albums belong to the sharer (sharer must own the album)
	for _, albumID := range req.AlbumIDs {
		_, err := h.albumRepo.GetByID(ctx, albumID, userID)
		if err != nil {
			log.Printf("[ERROR] handleCreateShare - album not found or access denied: %v", err)
			http.Error(w, "One or more albums do not belong to you", http.StatusBadRequest)
			return
		}
	}

	// Create shares (one per sharee) and media/album associations
	sharesCreated := make([]CreateShareResponse, 0, len(req.ShareeUserIDs))
	totalMediaShared := 0
	totalAlbumsShared := 0

	for _, shareeID := range req.ShareeUserIDs {
		// Create the share group
		share, err := h.shareRepo.CreateShare(ctx, userID, shareeID)
		if err != nil {
			log.Printf("[ERROR] handleCreateShare - create share: %v", err)
			http.Error(w, "Failed to create share", http.StatusInternalServerError)
			return
		}

		// Share media items (if any)
		for _, mediaID := range req.MediaIDs {
			if err := h.mediaShareRepo.CreateMediaShare(ctx, share.ID, mediaID); err != nil {
				log.Printf("[ERROR] handleCreateShare - create media share: %v", err)
				http.Error(w, "Failed to create media share", http.StatusInternalServerError)
				return
			}
			totalMediaShared++
		}

		// Share albums (if any)
		for _, albumID := range req.AlbumIDs {
			if err := h.albumShareRepo.CreateAlbumShare(ctx, share.ID, albumID); err != nil {
				log.Printf("[ERROR] handleCreateShare - create album share: %v", err)
				http.Error(w, "Failed to create album share", http.StatusInternalServerError)
				return
			}
			totalAlbumsShared++
		}

		sharesCreated = append(sharesCreated, CreateShareResponse{
			ID:           share.ID,
			SharerUserID: userID,
			SharedAt:     share.SharedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CreateShareResponseFull{
		SharesCreated:       sharesCreated,
		MediaSharedCount:    totalMediaShared,
		AlbumsSharedCount:   totalAlbumsShared,
	})
}

// CreateShareResponse is a single share created in the response.
type CreateShareResponse struct {
	ID           uuid.UUID `json:"id"`
	SharerUserID uuid.UUID `json:"sharerUserId"`
	SharedAt     time.Time `json:"sharedAt"`
}

// CreateShareResponseFull is the full response for creating a share.
type CreateShareResponseFull struct {
	SharesCreated   []CreateShareResponse `json:"shares_created"`
	MediaSharedCount int                  `json:"media_shared_count"`
	AlbumsSharedCount int                 `json:"albums_shared_count"`
}

// handleListSharedMedia handles GET /api/v1/media/shared — Get shared media for current user.
func (h *ShareHandler) handleListSharedMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	limit := 20
	offset := 0
	query := r.URL.Query()

	if lStr := query.Get("limit"); lStr != "" {
		fmt.Sscanf(lStr, "%d", &limit)
	}
	if oStr := query.Get("offset"); oStr != "" {
		fmt.Sscanf(oStr, "%d", &offset)
	}

	items, totalItems, err := h.mediaShareRepo.ListSharedMedia(ctx, userID, limit, offset)
	if err != nil {
		log.Printf("[ERROR] handleListSharedMedia: %v", err)
		http.Error(w, "Failed to list shared media", http.StatusInternalServerError)
		return
	}

	response := struct {
		Items []SharedMediaResponse `json:"items"`
		Total int                   `json:"total"`
		Limit int                   `json:"limit"`
		Offset int                  `json:"offset"`
	}{
		Items: make([]SharedMediaResponse, len(items)),
		Total: totalItems,
		Limit: limit,
		Offset: offset,
	}

	for i, item := range items {
		response.Items[i] = SharedMediaResponse{
			ID:           item.Media.ID,
			Filename:     item.Media.Filename,
			Path:         item.Media.Path,
			MediaType:    string(item.Media.MediaType),
			SharerUserID: item.SharerUserID,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleListSharedAlbums handles GET /api/v1/albums/shared — Get shared albums for current user.
func (h *ShareHandler) handleListSharedAlbums(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	limit := 20
	offset := 0
	query := r.URL.Query()

	if lStr := query.Get("limit"); lStr != "" {
		fmt.Sscanf(lStr, "%d", &limit)
	}
	if oStr := query.Get("offset"); oStr != "" {
		fmt.Sscanf(oStr, "%d", &offset)
	}

	items, totalItems, err := h.mediaShareRepo.ListSharedAlbums(ctx, userID, limit, offset)
	if err != nil {
		log.Printf("[ERROR] handleListSharedAlbums: %v", err)
		http.Error(w, "Failed to list shared albums", http.StatusInternalServerError)
		return
	}

	response := struct {
		Items []SharedAlbumResponse `json:"items"`
		Total int                   `json:"total"`
		Limit int                   `json:"limit"`
		Offset int                  `json:"offset"`
	}{
		Items: make([]SharedAlbumResponse, len(items)),
		Total: totalItems,
		Limit: limit,
		Offset: offset,
	}

	for i, item := range items {
		var thumbnail *string
		if item.FirstMediaID != uuid.Nil {
			thumbURL := "/api/v1/media/shared/" + item.FirstMediaID.String() + "/thumb"
			thumbnail = &thumbURL
		}
		response.Items[i] = SharedAlbumResponse{
			ID:           item.Album.ID,
			Name:         item.Album.Name,
			Description:  item.Album.Description,
			SharerUserID: item.SharerUserID,
			Thumbnail:    thumbnail,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCreatePublicShare handles POST /api/v1/public-shares — Create a public share link.
func (h *ShareHandler) handleCreatePublicShare(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreatePublicShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[ERROR] handleCreatePublicShare - decode body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ResourceType != "media" && req.ResourceType != "album" {
		http.Error(w, "resource_type must be 'media' or 'album'", http.StatusBadRequest)
		return
	}

	// Validate resource ownership based on type
	if req.ResourceType == "media" {
		_, err := h.mediaRepo.GetByID(ctx, req.ResourceID, &userID)
		if err != nil || req.ResourceID == uuid.Nil {
			log.Printf("[ERROR] handleCreatePublicShare - media not found or access denied: %v", err)
			http.Error(w, "Media item not found or does not belong to you", http.StatusBadRequest)
			return
		}
	} else if req.ResourceType == "album" {
		_, err := h.albumRepo.GetByID(ctx, req.ResourceID, userID)
		if err != nil || req.ResourceID == uuid.Nil {
			log.Printf("[ERROR] handleCreatePublicShare - album not found or access denied: %v", err)
			http.Error(w, "Album not found or does not belong to you", http.StatusBadRequest)
			return
		}
	}

	// Hash password if provided (bcrypt with default cost 12 from security package)
	var passwordHash *string
	if req.Password != nil && *req.Password != "" {
		hashed, err := security.HashPassword(*req.Password)
		if err != nil {
			log.Printf("[ERROR] handleCreatePublicShare - hash password: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		passwordHash = &hashed
	}

	publicShare, err := h.publicShareRepo.CreatePublicShare(ctx, userID, req.ResourceType, req.ResourceID, passwordHash, req.ExpiresAt)
	if err != nil {
		log.Printf("[ERROR] handleCreatePublicShare - create public share: %v", err)
		http.Error(w, "Failed to create public share link", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PublicShareLinkResponse{
		ID:                publicShare.ID,
		Token:             publicShare.Token,
		SharerUserID:      userID,
		ResourceType:      req.ResourceType,
		PasswordProtected: passwordHash != nil && *passwordHash != "",
		ExpiresAt:         publicShare.ExpiresAt,
		CreatedAt:         publicShare.CreatedAt,
	})
}

// handleDeletePublicShare handles DELETE /api/v1/public-shares/{id} — Revoke a public share link.
func (h *ShareHandler) handleDeletePublicShare(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	publicShareID, err := uuid.Parse(idStr)
	if err != nil {
		log.Printf("[ERROR] handleDeletePublicShare - invalid UUID: %v", err)
		http.Error(w, "Invalid share link ID", http.StatusBadRequest)
		return
	}

	// Get the public share to verify ownership (use ID for lookup)
	publicShare, err := h.publicShareRepo.GetByID(ctx, publicShareID)
	if err != nil {
		log.Printf("[ERROR] handleDeletePublicShare - get public share: %v", err)
		http.Error(w, "Share link not found", http.StatusNotFound)
		return
	}

	if publicShare.SharerUserID != userID {
		http.Error(w, "Forbidden: You do not own this share link", http.StatusForbidden)
		return
	}

	// Delete by token (need to get the token from the retrieved share)
	err = h.publicShareRepo.DeleteByToken(ctx, publicShare.Token)
	if err != nil {
		log.Printf("[ERROR] handleDeletePublicShare - delete public share: %v", err)
		http.Error(w, "Failed to revoke share link", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"deleted": true})
}

// handleListPublicShares handles GET /api/v1/public-shares — List public shares for current user.
func (h *ShareHandler) handleListPublicShares(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	list, err := h.publicShareRepo.ListBySharer(ctx, userID)
	if err != nil {
		log.Printf("[ERROR] handleListPublicShares: %v", err)
		http.Error(w, "Failed to list public shares", http.StatusInternalServerError)
		return
	}

	response := make([]PublicShareListResponse, len(list))
	for i, ps := range list {
		response[i] = PublicShareListResponse{
			ID:                ps.ID,
			Token:             ps.Token,
			SharerUserID:      ps.SharerUserID,
			ResourceType:      ps.ResourceType,
			ResourceID:        ps.ResourceID,
			PasswordProtected: ps.PasswordProtected,
			ExpiresAt:         ps.ExpiresAt,
			CreatedAt:         ps.CreatedAt,
			AccessCount:       ps.AccessCount,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetPublicShareMedia handles GET /public/shares/media/{token} — View shared media via public link.
func (h *ShareHandler) handleGetPublicShareMedia(w http.ResponseWriter, r *http.Request) {
log.Printf("[DEBUG] handleGetPublicShareMedia called - token: %s", chi.URLParam(r, "token"))
	ctx := r.Context()

	tokenStr := chi.URLParam(r, "token")

	// Get the public share by token
	publicShare, err := h.publicShareRepo.GetByToken(ctx, tokenStr)
	if err != nil {
		log.Printf("[ERROR] handleGetPublicShareMedia - get public share: %v", err)
		http.Error(w, "Share link not found or expired", http.StatusNotFound)
		return
	}

	// Check if the share has expired
	if publicShare.ExpiresAt != nil && time.Now().After(*publicShare.ExpiresAt) {
		log.Printf("[WARN] handleGetPublicShareMedia - expired public share accessed: %s", tokenStr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusGone)
		json.NewEncoder(w).Encode(map[string]string{"error": "This share link has expired"})
		return
	}

	// Check if password is required and verify it
	if publicShare.PasswordHash != nil && *publicShare.PasswordHash != "" {
		password := r.URL.Query().Get("password")
		if password == "" || !security.CheckPasswordHash(password, *publicShare.PasswordHash) {
			log.Printf("[WARN] handleGetPublicShareMedia - wrong/missing password for share: %s", tokenStr)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "Incorrect or missing password"})
			return
		}
	}

	// Get the media item by token (this does its own ownership check internally since it's public)
	mediaItem, err := h.publicShareRepo.GetSharedMediaByToken(ctx, tokenStr)
	if err != nil {
		log.Printf("[ERROR] handleGetPublicShareMedia - get shared media: %v", err)
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}

	// Log the access for auditing (best-effort — don't fail the request if logging fails)
	if ip := r.RemoteAddr; ip != "" {
		h.publicAccessRepo.CreateAccessLog(ctx, publicShare.ID, ip)
	}

	// Increment access count (best-effort — don't fail the request if incrementing fails)
	h.publicShareRepo.IncrementAccessCount(ctx, publicShare.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mediaItem.Media)
}

// handleGetPublicShareAlbum handles GET /public/shares/album/{token} — View shared album via public link.
func (h *ShareHandler) handleGetPublicShareAlbum(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tokenStr := chi.URLParam(r, "token")

	// Get the public share by token
	publicShare, err := h.publicShareRepo.GetByToken(ctx, tokenStr)
	if err != nil {
		log.Printf("[ERROR] handleGetPublicShareAlbum - get public share: %v", err)
		http.Error(w, "Share link not found or expired", http.StatusNotFound)
		return
	}

	// Check if the share has expired
	if publicShare.ExpiresAt != nil && time.Now().After(*publicShare.ExpiresAt) {
		log.Printf("[WARN] handleGetPublicShareAlbum - expired public share accessed: %s", tokenStr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusGone)
		json.NewEncoder(w).Encode(map[string]string{"error": "This share link has expired"})
		return
	}

	// Check if password is required and verify it
	if publicShare.PasswordHash != nil && *publicShare.PasswordHash != "" {
		password := r.URL.Query().Get("password")
		if password == "" || !security.CheckPasswordHash(password, *publicShare.PasswordHash) {
			log.Printf("[WARN] handleGetPublicShareAlbum - wrong/missing password for share: %s", tokenStr)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "Incorrect or missing password"})
			return
		}
	}

	// Get the album by token (this does its own ownership check internally since it's public)
	albumItem, err := h.publicShareRepo.GetSharedAlbumByToken(ctx, tokenStr)
	if err != nil {
		log.Printf("[ERROR] handleGetPublicShareAlbum - get shared album: %v", err)
		http.Error(w, "Album not found", http.StatusNotFound)
		return
	}

	// Log the access for auditing (best-effort — don't fail the request if logging fails)
	if ip := r.RemoteAddr; ip != "" {
		h.publicAccessRepo.CreateAccessLog(ctx, publicShare.ID, ip)
	}

	// Increment access count (best-effort — don't fail the request if incrementing fails)
	h.publicShareRepo.IncrementAccessCount(ctx, publicShare.ID)

	// Include media items in the response so Angular doesn't need to make additional requests
	mediaResponses := make([]MediaShareItem, len(albumItem.MediaItems))
	for i, m := range albumItem.MediaItems {
		mediaResponses[i] = MediaShareItem{
			ID:        m.ID,
			Filename:  m.Filename,
			Path:      m.Path,
			MediaType: string(m.MediaType),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SharedAlbumWithMedia{
		ID:           albumItem.Album.ID,
		Name:         albumItem.Album.Name,
		Description:  albumItem.Album.Description,
		CreatedAt:    albumItem.Album.CreatedAt,
		UpdatedAt:    albumItem.Album.UpdatedAt,
		MediaItems:   mediaResponses,
	})
}

// handleGetPublicShareAlbumMedia handles GET /public/shares/album/{token}/media — Paginated media for shared album via public link.
func (h *ShareHandler) handleGetPublicShareAlbumMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tokenStr := chi.URLParam(r, "token")

	// Get the public share by token
	publicShare, err := h.publicShareRepo.GetByToken(ctx, tokenStr)
	if err != nil {
		log.Printf("[ERROR] handleGetPublicShareAlbumMedia - get public share: %v", err)
		http.Error(w, "Share link not found or expired", http.StatusNotFound)
		return
	}

	// Check if the share has expired
	if publicShare.ExpiresAt != nil && time.Now().After(*publicShare.ExpiresAt) {
		log.Printf("[WARN] handleGetPublicShareAlbumMedia - expired public share accessed: %s", tokenStr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusGone)
		json.NewEncoder(w).Encode(map[string]string{"error": "This share link has expired"})
		return
	}

	// Check if password is required and verify it
	if publicShare.PasswordHash != nil && *publicShare.PasswordHash != "" {
		password := r.URL.Query().Get("password")
		if password == "" || !security.CheckPasswordHash(password, *publicShare.PasswordHash) {
			log.Printf("[WARN] handleGetPublicShareAlbumMedia - wrong/missing password for share: %s", tokenStr)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "Incorrect or missing password"})
			return
		}
	}

	// Get the album by token (this does its own ownership check internally since it's public)
	albumItem, err := h.publicShareRepo.GetSharedAlbumByToken(ctx, tokenStr)
	if err != nil {
		log.Printf("[ERROR] handleGetPublicShareAlbumMedia - get shared album: %v", err)
		http.Error(w, "Album not found", http.StatusNotFound)
		return
	}

	// Log the access for auditing (best-effort — don't fail the request if logging fails)
	if ip := r.RemoteAddr; ip != "" {
		h.publicAccessRepo.CreateAccessLog(ctx, publicShare.ID, ip)
	}

	// Increment access count (best-effort — don't fail the request if incrementing fails)
	h.publicShareRepo.IncrementAccessCount(ctx, publicShare.ID)

	// Apply pagination
	mediaItems := albumItem.MediaItems
	totalItems := len(mediaItems)

	limit := 20
	offset := 0
	query := r.URL.Query()

	if lStr := query.Get("limit"); lStr != "" {
		fmt.Sscanf(lStr, "%d", &limit)
	}
	if oStr := query.Get("offset"); oStr != "" {
		fmt.Sscanf(oStr, "%d", &offset)
	}

	// Ensure offset is not negative or beyond total
	if offset < 0 {
		offset = 0
	}
	if offset >= totalItems {
		offset = totalItems
	}

	// Calculate end index with bounds
	end := offset + limit
	if end > totalItems {
		end = totalItems
	}

	// Slice the media items for the current page
	pageItems := mediaItems[offset:end]

	// Convert to response format
	mediaResponses := make([]MediaShareItem, len(pageItems))
	for i, m := range pageItems {
		mediaResponses[i] = MediaShareItem{
			ID:        m.ID,
			Filename:  m.Filename,
			Path:      m.Path,
			MediaType: string(m.MediaType),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Media      []MediaShareItem `json:"media"`
		TotalItems int             `json:"totalItems"`
	}{
		Media:      mediaResponses,
		TotalItems: totalItems,
	})
}

// handleGetPublicShareMediaOriginal handles GET /public/shares/media/{token}/original — Stream original file via public link.
// Also serves album media when ?path=<media_path> query parameter is provided.
func (s *Server) handleGetPublicShareMediaOriginal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tokenStr := chi.URLParam(r, "token")

	// If path query parameter is provided, this is an album media request
	if mediaPath := r.URL.Query().Get("path"); mediaPath != "" {
		s.serveAlbumMediaOriginal(w, r, tokenStr, mediaPath)
		return
	}

	// Original file by token for individual media share
	media, err := s.publicShareRepo.GetOriginalFileByToken(ctx, tokenStr)
	if err != nil {
		log.Printf("[ERROR] handleGetPublicShareMediaOriginal - get shared media: %v", err)
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}

	// Resolve the absolute path using our Storage Service
	targetPath := media.Path
	absPath, resolveErr := s.storageService.ResolvePath(targetPath)
	if resolveErr != nil {
		targetPath = strings.TrimLeft(targetPath, "/\\")
		absPath, resolveErr = s.storageService.ResolvePath(targetPath)
	}

	if resolveErr != nil {
		log.Printf("[ERROR] handleGetPublicShareMediaOriginal - resolvePath (%s): %v", targetPath, resolveErr)
		http.Error(w, "Could not locate file: "+resolveErr.Error(), http.StatusInternalServerError)
		return
	}

	contentType := s.getContentType(media.MediaType, media.Filename)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type, Accept-Ranges, Content-Range")

	http.ServeFile(w, r, absPath)
}

// handleGetPublicShareMediaThumb handles GET /public/shares/media/{token}/thumb — Stream thumbnail via public link.
// Also serves album media when ?path=<media_path> query parameter is provided.
func (s *Server) handleGetPublicShareMediaThumb(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tokenStr := chi.URLParam(r, "token")

	// If path query parameter is provided, this is an album media request
	if mediaPath := r.URL.Query().Get("path"); mediaPath != "" {
		s.serveAlbumMediaThumb(w, r, tokenStr, mediaPath)
		return
	}

	// Thumbnail by token for individual media share
	media, err := s.publicShareRepo.GetThumbnailFileByToken(ctx, tokenStr)
	if err != nil {
		log.Printf("[ERROR] handleGetPublicShareMediaThumb - get shared media: %v", err)
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}

	// Calculate thumbnail path - strip user ID from path since thumbnails are stored without it
	cleanPath := strings.TrimPrefix(media.Path, "storage/")
	parts := strings.SplitN(cleanPath, string(filepath.Separator), 2)
	if len(parts) >= 2 {
		cleanPath = parts[1]
	}
	ext := filepath.Ext(cleanPath)

	thumbRelPath := s.storageService.GetThumbnailRelativePath(
		string(media.MediaType),
		cleanPath,
		ext,
	)

	fullThumbPath := filepath.Join(s.thumbRoot, thumbRelPath)

	if _, err := os.Stat(fullThumbPath); os.IsNotExist(err) {
		log.Printf("[WARN] handleGetPublicShareMediaThumb: Thumbnail NOT FOUND at %s. Attempting fallback to original.", fullThumbPath)
		targetPath := media.Path
		absPath, resolveErr := s.storageService.ResolvePath(targetPath)
		if resolveErr != nil {
			targetPath = strings.TrimLeft(targetPath, "/\\")
			absPath, resolveErr = s.storageService.ResolvePath(targetPath)
		}

		if resolveErr != nil {
			log.Printf("[ERROR] handleGetPublicShareMediaThumb - fallback failed for (%s): %v", targetPath, resolveErr)
			http.Error(w, "Thumbnail not found and fallback failed", http.StatusNotFound)
			return
		}

		contentType := s.getContentType(media.MediaType, media.Filename)
		w.Header().Set("Content-Type", contentType)
		log.Printf("[INFO] handleGetPublicShareMediaThumb: Serving original file as fallback from %s", absPath)
		http.ServeFile(w, r, absPath)
		return
	}

	contentType := s.getContentType(media.MediaType, media.Filename)
	w.Header().Set("Content-Type", contentType)
	http.ServeFile(w, r, fullThumbPath)
}

// serveAlbumMediaOriginal serves album media original file by path via public album link.
func (s *Server) serveAlbumMediaOriginal(w http.ResponseWriter, r *http.Request, tokenStr string, mediaPath string) {
	ctx := r.Context()

	// Get the media item by token for public serving
	publicShare, err := s.publicShareRepo.GetByToken(ctx, tokenStr)
	if err != nil {
		log.Printf("[ERROR] serveAlbumMediaOriginal - get public share: %v", err)
		http.Error(w, "Share link not found or expired", http.StatusNotFound)
		return
	}

	// Check if the share has expired
	if publicShare.ExpiresAt != nil && time.Now().After(*publicShare.ExpiresAt) {
		log.Printf("[WARN] serveAlbumMediaOriginal - expired public share accessed: %s", tokenStr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusGone)
		json.NewEncoder(w).Encode(map[string]string{"error": "This share link has expired"})
		return
	}

	// Check if password is required and verify it
	if publicShare.PasswordHash != nil && *publicShare.PasswordHash != "" {
		password := r.URL.Query().Get("password")
		if password == "" || !security.CheckPasswordHash(password, *publicShare.PasswordHash) {
			log.Printf("[WARN] serveAlbumMediaOriginal - wrong/missing password for share: %s", tokenStr)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "Incorrect or missing password"})
			return
		}
	}

	// Get the album with media items by token
	albumItem, err := s.publicShareRepo.GetSharedAlbumByToken(ctx, tokenStr)
	if err != nil {
		log.Printf("[ERROR] serveAlbumMediaOriginal - get shared album: %v", err)
		http.Error(w, "Album not found", http.StatusNotFound)
		return
	}

	// Find the media item in the album by path (case-insensitive comparison)
	var media *domain.Media
	for _, m := range albumItem.MediaItems {
		if strings.EqualFold(m.Path, mediaPath) {
			media = m
			break
		}
	}

	if media == nil {
		log.Printf("[ERROR] serveAlbumMediaOriginal - media not found for path: %s", mediaPath)
		http.Error(w, "Media not found in album", http.StatusNotFound)
		return
	}

	// Resolve the absolute path using our Storage Service
	targetPath := media.Path
	absPath, resolveErr := s.storageService.ResolvePath(targetPath)
	if resolveErr != nil {
		targetPath = strings.TrimLeft(targetPath, "/\\")
		absPath, resolveErr = s.storageService.ResolvePath(targetPath)
	}

	if resolveErr != nil {
		log.Printf("[ERROR] serveAlbumMediaOriginal - resolvePath (%s): %v", targetPath, resolveErr)
		http.Error(w, "Could not locate file: "+resolveErr.Error(), http.StatusInternalServerError)
		return
	}

	contentType := s.getContentType(media.MediaType, media.Filename)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type, Accept-Ranges, Content-Range")

	http.ServeFile(w, r, absPath)
}

// serveAlbumMediaThumb serves album media thumbnail by path via public album link.
func (s *Server) serveAlbumMediaThumb(w http.ResponseWriter, r *http.Request, tokenStr string, mediaPath string) {
	ctx := r.Context()

	// Get the media item by token for public serving
	publicShare, err := s.publicShareRepo.GetByToken(ctx, tokenStr)
	if err != nil {
		log.Printf("[ERROR] serveAlbumMediaThumb - get public share: %v", err)
		http.Error(w, "Share link not found or expired", http.StatusNotFound)
		return
	}

	// Check if the share has expired
	if publicShare.ExpiresAt != nil && time.Now().After(*publicShare.ExpiresAt) {
		log.Printf("[WARN] serveAlbumMediaThumb - expired public share accessed: %s", tokenStr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusGone)
		json.NewEncoder(w).Encode(map[string]string{"error": "This share link has expired"})
		return
	}

	// Check if password is required and verify it
	if publicShare.PasswordHash != nil && *publicShare.PasswordHash != "" {
		password := r.URL.Query().Get("password")
		if password == "" || !security.CheckPasswordHash(password, *publicShare.PasswordHash) {
			log.Printf("[WARN] serveAlbumMediaThumb - wrong/missing password for share: %s", tokenStr)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "Incorrect or missing password"})
			return
		}
	}

	// Get the album with media items by token
	albumItem, err := s.publicShareRepo.GetSharedAlbumByToken(ctx, tokenStr)
	if err != nil {
		log.Printf("[ERROR] serveAlbumMediaThumb - get shared album: %v", err)
		http.Error(w, "Album not found", http.StatusNotFound)
		return
	}

	// Find the media item in the album by path (case-insensitive comparison)
	var media *domain.Media
	for _, m := range albumItem.MediaItems {
		if strings.EqualFold(m.Path, mediaPath) {
			media = m
			break
		}
	}

	if media == nil {
		log.Printf("[ERROR] serveAlbumMediaThumb - media not found for path: %s", mediaPath)
		http.Error(w, "Media not found in album", http.StatusNotFound)
		return
	}

	// Calculate thumbnail path - strip user ID from path since thumbnails are stored without it
	cleanPath := strings.TrimPrefix(media.Path, "storage/")
	parts := strings.SplitN(cleanPath, string(filepath.Separator), 2)
	if len(parts) >= 2 {
		cleanPath = parts[1]
	}
	ext := filepath.Ext(cleanPath)

	thumbRelPath := s.storageService.GetThumbnailRelativePath(
		string(media.MediaType),
		cleanPath,
		ext,
	)

	fullThumbPath := filepath.Join(s.thumbRoot, thumbRelPath)

	if _, err := os.Stat(fullThumbPath); os.IsNotExist(err) {
		log.Printf("[WARN] serveAlbumMediaThumb: Thumbnail NOT FOUND at %s. Attempting fallback to original.", fullThumbPath)
		targetPath := media.Path
		absPath, resolveErr := s.storageService.ResolvePath(targetPath)
		if resolveErr != nil {
			targetPath = strings.TrimLeft(targetPath, "/\\")
			absPath, resolveErr = s.storageService.ResolvePath(targetPath)
		}

		if resolveErr != nil {
			log.Printf("[ERROR] serveAlbumMediaThumb - fallback failed for (%s): %v", targetPath, resolveErr)
			http.Error(w, "Thumbnail not found and fallback failed", http.StatusNotFound)
			return
		}

		contentType := s.getContentType(media.MediaType, media.Filename)
		w.Header().Set("Content-Type", contentType)
		log.Printf("[INFO] serveAlbumMediaThumb: Serving original file as fallback from %s", absPath)
		http.ServeFile(w, r, absPath)
		return
	}

	contentType := s.getContentType(media.MediaType, media.Filename)
	w.Header().Set("Content-Type", contentType)
	http.ServeFile(w, r, fullThumbPath)
}



// ShareGroupListResponse represents the response for listing outgoing share groups.
type ShareGroupListResponse struct {
	ShareID           uuid.UUID   `json:"id"`
	SharerName        string      `json:"sharerName,omitempty"`
	ShareeName        string      `json:"shareeName,omitempty"`
	MediaCount        int         `json:"mediaCount"`
	AlbumsCount       int         `json:"albumsCount"`
	SharedAt          time.Time   `json:"sharedAt"`
}

// handleListOutgoingShareGroups handles GET /api/v1/shares — List outgoing share groups for current user.
func (h *ShareHandler) handleListOutgoingShareGroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	groups, err := h.shareRepo.ListOutgoingShareGroups(ctx, userID)
	if err != nil {
		log.Printf("[ERROR] handleListOutgoingShareGroups: %v", err)
		http.Error(w, "Failed to list outgoing shares", http.StatusInternalServerError)
		return
	}

	// Fetch all details in a single query (O(n)) instead of O(n*m)
	details, _ := h.shareRepo.ListOutgoingShares(ctx, userID)
	detailMap := make(map[uuid.UUID]*domain.ShareWithDetails)
	if details != nil {
		for _, d := range details {
			detailMap[d.ID] = d
		}
	}

	response := make([]ShareGroupListResponse, len(groups))
	for i, g := range groups {
		detail, hasDetail := detailMap[g.ID]
		mediaCount := 0
		albumCount := 0
		if hasDetail && detail != nil {
			mediaCount = len(detail.Media)
			albumCount = len(detail.Albums)
		}

		response[i] = ShareGroupListResponse{
			ShareID:     g.ID,
			SharerName:  g.SharerName,
			ShareeName:  g.ShareeName,
			MediaCount:  mediaCount,
			AlbumsCount: albumCount,
			SharedAt:    g.SharedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRevokeOutgoingShare handles DELETE /api/v1/shares/{id} — Revoke an outgoing share group.
func (h *ShareHandler) handleRevokeOutgoingShare(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	shareGroupID, err := uuid.Parse(idStr)
	if err != nil {
		log.Printf("[ERROR] handleRevokeOutgoingShare - invalid UUID: %v", err)
		http.Error(w, "Invalid share group ID", http.StatusBadRequest)
		return
	}

	// Verify ownership by checking if this share is in the user's outgoing shares
	details, err := h.shareRepo.ListOutgoingShares(ctx, userID)
	if err != nil {
		log.Printf("[ERROR] handleRevokeOutgoingShare - list outgoing shares: %v", err)
		http.Error(w, "Failed to verify ownership", http.StatusInternalServerError)
		return
	}

	found := false
	for _, d := range details {
		if d.ID == shareGroupID {
			found = true
			break
		}
	}

	if !found {
		log.Printf("[WARN] handleRevokeOutgoingShare - user %s tried to revoke non-owned share: %s", userID, shareGroupID)
		http.Error(w, "Forbidden: You do not own this share group", http.StatusForbidden)
		return
	}

	err = h.shareRepo.DeleteShareGroup(ctx, shareGroupID)
	if err != nil {
		log.Printf("[ERROR] handleRevokeOutgoingShare - delete share: %v", err)
		http.Error(w, "Failed to revoke share", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Share group revoked successfully"})
}
