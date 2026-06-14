package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
}

func NewShareHandler(shareRepo domain.ShareRepository, mediaShareRepo domain.MediaShareRepository, albumShareRepo domain.AlbumShareRepository, publicShareRepo domain.PublicShareRepository, publicAccessRepo domain.PublicShareAccessRepository, userRepo domain.UserRepository, albumRepo domain.AlbumRepository) *ShareHandler {
	return &ShareHandler{
		shareRepo:        shareRepo,
		mediaShareRepo:   mediaShareRepo,
		albumShareRepo:   albumShareRepo,
		publicShareRepo:  publicShareRepo,
		publicAccessRepo: publicAccessRepo,
		userRepo:         userRepo,
		albumRepo:        albumRepo,
	}
}

// CreateShareRequest represents the request body for creating a share.
type CreateShareRequest struct {
	SharerUserID    uuid.UUID   `json:"-"` // Set from context (the authenticated user)
	ShareeUserIDs   []uuid.UUID `json:"sharee_user_ids"`
	MediaIDs        []uuid.UUID `json:"media_ids,omitempty"`
	AlbumIDs        []uuid.UUID `json:"album_ids,omitempty"`
}

// SharedMediaResponse represents a shared media item for the API response.
type SharedMediaResponse struct {
	ID           uuid.UUID   `json:"id"`
	Filename     string      `json:"filename"`
	MediaType    string      `json:"mediaType"`
	SharerUserID uuid.UUID   `json:"sharerUserId"`
}

// SharedAlbumResponse represents a shared album for the API response.
type SharedAlbumResponse struct {
	ID           uuid.UUID   `json:"id"`
	Name         string      `json:"name"`
	Description  *string     `json:"description"`
	SharerUserID uuid.UUID   `json:"sharerUserId"`
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
	ID             uuid.UUID     `json:"id"`
	Token          string        `json:"token"`
	SharerUserID   uuid.UUID     `json:"sharerUserId"`
	ResourceType   string        `json:"resourceType"`
	ResourceID     uuid.UUID     `json:"-"` // Not exposed in response
	PasswordProtected bool       `json:"password_protected"`
	ExpiresAt      *time.Time    `json:"expires_at"`
	CreatedAt      time.Time     `json:"created_at"`
	AccessCount    int           `json:"access_count"`
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
		media, err := h.mediaShareRepo.GetSharedMediaByID(ctx, mediaID, userID)
		if err != nil || media == nil {
			log.Printf("[ERROR] handleCreateShare - media not found or access denied: %v", err)
			http.Error(w, "One or more media items do not belong to you", http.StatusBadRequest)
			return
		}
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
		response.Items[i] = SharedAlbumResponse{
			ID:           item.Album.ID,
			Name:         item.Album.Name,
			Description:  item.Album.Description,
			SharerUserID: item.SharerUserID,
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
		_, err := h.mediaShareRepo.GetSharedMediaByID(ctx, req.ResourceID, userID)
		if err != nil || req.ResourceID == uuid.Nil {
			log.Printf("[ERROR] handleCreatePublicShare - media not found or access denied: %v", err)
			http.Error(w, "Media item not found or does not belong to you", http.StatusBadRequest)
			return
		}
	} else if req.ResourceType == "album" {
		_, err := h.mediaShareRepo.GetSharedAlbumByID(ctx, req.ResourceID, userID)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(albumItem.Album)
}
