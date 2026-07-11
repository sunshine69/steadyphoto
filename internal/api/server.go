package api

import (
	"encoding/json"
	"fmt"
	"github.com/jbrodriguez/mlog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"steadyphoto/internal/database"
	"steadyphoto/internal/domain"
	"steadyphoto/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
)

// Ensure MediaUploadHandler is defined in upload_handler.go and implements the necessary logic.

type Server struct {
	router         *chi.Mux
	mediaRepo      domain.MediaRepository
	albumRepo      domain.AlbumRepository
	userRepo       domain.UserRepository
	sessionRepo    domain.SessionRepository
	storageService *storage.StorageService
	thumbRoot      string
	sessionManager *UploadSessionManager // For resumable uploads
	// Sharing repositories and handler
	shareRepo        domain.ShareRepository
	mediaShareRepo   domain.MediaShareRepository
	albumShareRepo   domain.AlbumShareRepository
	publicShareRepo  domain.PublicShareRepository
	publicAccessRepo domain.PublicShareAccessRepository
	jobRepo          domain.JobRepository
}

func NewServer(
	mediaRepo domain.MediaRepository,
	albumRepo domain.AlbumRepository,
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	storageService *storage.StorageService,
	thumbRoot string,
	shareRepo domain.ShareRepository,
	mediaShareRepo domain.MediaShareRepository,
	albumShareRepo domain.AlbumShareRepository,
	publicShareRepo domain.PublicShareRepository,
	publicAccessRepo domain.PublicShareAccessRepository,
	jobRepo domain.JobRepository,
) *Server {
	s := &Server{
		router:           chi.NewRouter(),
		mediaRepo:        mediaRepo,
		albumRepo:        albumRepo,
		userRepo:         userRepo,
		sessionRepo:      sessionRepo,
		storageService:   storageService,
		thumbRoot:        thumbRoot,
		sessionManager:   NewUploadSessionManager(storageService),
		shareRepo:        shareRepo,
		mediaShareRepo:   mediaShareRepo,
		albumShareRepo:   albumShareRepo,
		publicShareRepo:  publicShareRepo,
		publicAccessRepo: publicAccessRepo,
		jobRepo:          jobRepo,
	}
	s.routes()
	s.startCleanupGoroutine()
	return s
}

func (s *Server) routes() {
	// Standard middleware
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   AllowedCORSOrigins, // Dynamic from env var CORS_ALLOWED_ORIGINS or defaults in middleware.go
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "Range"},
		ExposedHeaders:   []string{"Content-Length", "Content-Type", "Accept-Ranges", "Content-Range"},
		AllowCredentials: true,
	}))
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)

	// Request timeout - prevents DB queries from hanging indefinitely and causing "unexpected end of stream" errors.
	// Returns 504 Gateway Timeout if any request exceeds this duration, giving OkHttp a proper HTTP response instead of a closed connection.
	s.router.Use(middleware.Timeout(30 * time.Second))

	// API Versioning
	s.router.Route("/api/v1", func(r chi.Router) {
		// Authentication endpoints - strict rate limiting to prevent brute-force attacks
		r.Route("/auth", func(r chi.Router) {
			r.Use(RateLimitAuth) // 5 requests per minute by IP + endpoint

			// Public sub-routes (No middleware applied here)
			r.Post("/register", s.handleRegister)
			r.Post("/login", s.handleLogin)
			r.Post("/refresh", s.handleRefresh)
			// Profile routes - MUST be authenticated (CSRF protection via SameSite cookie attribute)
			r.Group(func(profile chi.Router) {
				profile.Use(s.AuthMiddleware)
				profile.Patch("/profile", s.handleUpdateProfile)
				profile.Delete("/profile", s.handleDeleteProfile)
				profile.Get("/profile", s.handleGetProfile)
				profile.Patch("/profile/email", s.handleUpdateProfileEmail)
				profile.Patch("/profile/password", s.handleChangePassword)

				// User search endpoint for sharing - returns partial user info (email only)
				profile.Get("/users/search", s.handleSearchUsers)
			})
		})

		// Admin endpoints - protected by both Auth and Admin middleware
		r.Route("/admin", func(r chi.Router) {
			r.Group(func(adminRoutes chi.Router) {
				adminRoutes.Use(s.AuthMiddleware)
				adminRoutes.Use(s.AdminMiddleware)

				// User management
				adminRoutes.Get("/users", s.handleAdminListUsers)
				adminRoutes.Get("/users/{id}", s.handleAdminGetUser)
				adminRoutes.Patch("/users/{id}", s.handleAdminUpdateUser)
				adminRoutes.Delete("/users/{id}", s.handleAdminDeleteUser)

				// Bulk operations
				adminRoutes.Post("/users/bulk-approve", s.handleBulkApproveUsers)
				adminRoutes.Post("/users/bulk-disable", s.handleBulkDisableUsers)
				adminRoutes.Delete("/users/bulk-delete", s.handleBulkDeleteUsers)
			})
		})

		// Sharing handler instance — needed for both protected and public routes
		sharesHandler := NewShareHandler(
			s.shareRepo, s.mediaShareRepo, s.albumShareRepo,
			s.publicShareRepo, s.publicAccessRepo,
			s.userRepo, s.albumRepo, s.mediaRepo,
		)

		// PROTECTED ROUTES group (for non-auth resources like media) - general rate limiting as safety net
		r.Group(func(protected chi.Router) {
			protected.Use(RateLimitGeneral) // 100 requests per minute by IP only
			protected.Use(s.AuthMiddleware)

			// Photo-specific endpoints (backward compatible, but now authenticated)
			protected.Get("/photos", s.handleListPhotos)
			protected.Get("/photos/{id}", s.handleGetPhoto)
			protected.Get("/photos/{id}/file", s.handleGetPhotoFile)
			protected.Get("/photos/{id}/thumb", s.handleGetThumbnail)

			// Unified media endpoints (for video support, now authenticated)
			protected.Get("/media", s.handleListMedia)
			protected.Get("/media/search", s.handleSearchMedia)
			protected.Route("/media/{id}", func(r chi.Router) {
				r.Get("/", s.handleGetMedia)
				r.Put("/", s.handleUpdateMedia)
				r.Patch("/", s.handlePatchMedia)
				r.Delete("/", s.handleDeleteMedia)
				r.Get("/original", s.handleGetOriginal)
				r.Get("/thumb", s.handleGetThumbnail)
				r.Patch("/tags", s.handleUpdateTags)
			})
			albumH := NewAlbumHandler(s.albumRepo, s.mediaRepo)
			// Album endpoints
			protected.Route("/albums", func(r chi.Router) {
				r.Post("/", albumH.CreateAlbum)
				r.Get("/", albumH.ListAlbums)
				r.Get("/{id}", albumH.GetAlbum)
				r.Put("/{id}", albumH.UpdateAlbum)
				r.Delete("/{id}", albumH.DeleteAlbum)

				// Album media sub-routes
				r.Post("/{id}/media", albumH.AddMediaToAlbum)
				r.Get("/{id}/media", albumH.GetAlbumMedia)
				r.Delete("/{id}/media/{media_id}", albumH.RemoveMediaFromAlbum)
				r.Delete("/{id}/media", albumH.BulkRemoveMediaFromAlbum)
			})

			// Trash endpoints - list, restore and permanently delete soft-deleted media
			protected.Get("/trash", s.handleListTrash)
			protected.Delete("/trash/{id}", s.handlePermanentDeleteMedia)
			protected.Patch("/media/{id}/restore", s.handleRestoreMedia)

			// Media Upload endpoint (Web & Mobile clients)
			uploadHandler := NewMediaUploadHandler(s.mediaRepo, s.albumRepo, s.jobRepo.(*database.PostgresJobRepository), s.storageService)
			protected.Route("/media/upload", func(r chi.Router) {
				// LimitBodySizeMiddleware removed - chunked uploads use small chunks (5MB),
				// and ParseMultipartForm handles per-part limits. The middleware's Content-Length check
				// was blocking large file uploads that Android sends as multipart forms with proper boundaries.

				r.Post("/", uploadHandler.Handle)

				// Single file upload endpoint (mobile client) - increased memory limit to 1GB
				sessionManager := s.sessionManager
				singleUploadHandler := NewMediaUploadHandlerSingle(s.mediaRepo, s.jobRepo, s.storageService, sessionManager)
				r.Post("/single", singleUploadHandler.HandleSingleFileUpload)

				// Chunked/upload session endpoints for resumable uploads - increased memory limit to 128MB per chunk
				r.Post("/chunk", func(w http.ResponseWriter, r *http.Request) {
					singleUploadHandler.HandleChunkUpload(w, r, sessionManager)
				})

				// Upload status endpoint (mobile client)
				r.Get("/status", singleUploadHandler.HandleStatus)

				// Abort upload endpoint (mobile client) - small limit is fine for abort requests
				r.Post("/abort", singleUploadHandler.HandleAbort)

				// Complete resumable upload - assemble chunks into final file - small limit is fine since no new data
				r.Post("/complete", func(w http.ResponseWriter, r *http.Request) {
					singleUploadHandler.HandleComplete(w, r)
				})
			})

			// Media delete endpoint (mobile client - permanently deletes from storage and DB)
			protected.Delete("/media/delete", func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				userID, ok := GetUserIDFromContext(ctx)
				if !ok {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				var req struct {
					MediaIDs []string `json:"media_ids"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					mlog.Info("[ERROR] handleDeleteMedia - decode body: %v", err)
					http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
					return
				}

				if len(req.MediaIDs) == 0 {
					http.Error(w, "No media IDs provided", http.StatusBadRequest)
					return
				}

				var deletedCount int
				for _, idStr := range req.MediaIDs {
					id, err := uuid.Parse(idStr)
					if err != nil {
						mlog.Info("[ERROR] handleDeleteMedia - parse UUID (%s): %v", idStr, err)
						continue // Skip invalid IDs but continue processing others
					}

					// First get the media to know which files to delete from storage
					media, err := s.mediaRepo.GetByID(ctx, id, &userID)
					if err != nil {
						mlog.Info("[ERROR] handleDeleteMedia - GetByID (%s): %v", id, err)
						continue // Skip if media not found or doesn't belong to user
					}

					// Delete files from storage (main file and thumbnail if exists)
					if err := s.storageService.DeleteFile(media.Path); err != nil {
						mlog.Info("[ERROR] handleDeleteMedia - delete main file (%s): %v", media.Path, err)
						// Continue with DB deletion even if file delete fails to avoid orphaned records
					}

					// Delete thumbnail separately (it's optional and may not exist)
					// Thumbnails are stored WITH user ID in the path
					ext := filepath.Ext(media.Path)
					if thumbRelPath := s.storageService.GetThumbnailRelativePath(string(media.MediaType), media.Path, ext); thumbRelPath != "" {
						s.storageService.DeleteThumbnailSilently(thumbRelPath) // Ignore error for thumbnails
					}

					// Permanently delete from database
					if err := s.mediaRepo.PermanentlyDeleteMedia(ctx, id, userID); err != nil {
						mlog.Info("[ERROR] handleDeleteMedia - permanent delete (%s): %v", id, err)
						continue // Skip if DB deletion fails
					}

					deletedCount++
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":          "success",
					"deleted_count":   deletedCount,
					"total_requested": len(req.MediaIDs),
				})
			})

			// SHARING ROUTES — user-to-user + public share link management (auth required)
			protected.Route("/shares", func(r chi.Router) {
				r.Post("/", sharesHandler.handleCreateShare)                       // Create a share with specific users + media/albums to share
				r.Get("/", sharesHandler.handleListOutgoingShareGroups)           // List outgoing share groups for current user
				r.Delete("/{id}", sharesHandler.handleRevokeOutgoingShare)        // Revoke an outgoing share group by ID
			})
			protected.Get("/media/shared", sharesHandler.handleListSharedMedia)   // List media shared with current user
			protected.Get("/albums/shared", sharesHandler.handleListSharedAlbums) // List albums shared with current user
			// Serve shared media items (metadata + files) - checks sharee access instead of ownership
			protected.Route("/media/shared/{id}", func(r chi.Router) {
				r.Get("/", s.handleGetSharedMedia)                  // Get metadata for a single shared media item
				r.Get("/thumb", s.handleGetSharedMediaThumb)        // Serve thumbnail for a shared media item
				r.Get("/original", s.handleGetSharedMediaOriginal)  // Stream original file for a shared media item
			})

			// Shared album detail and media endpoints - checks sharee access instead of ownership
			protected.Route("/albums/shared/{id}", func(r chi.Router) {
				r.Get("/", s.handleGetSharedAlbum)        // Get metadata for a single shared album
				r.Get("/media", s.handleListSharedAlbumMedia)  // List media items in a shared album
			})

			protected.Route("/public-shares", func(r chi.Router) {
				r.Post("/", sharesHandler.handleCreatePublicShare)       // Create public share link (with optional password + expiration)
				r.Delete("/{id}", sharesHandler.handleDeletePublicShare) // Revoke public share link by ID
				r.Get("/", sharesHandler.handleListPublicShares)         // List all public shares for current user

	
			})

		})

		// Public share viewing endpoints (no authentication required - OUTSIDE protected group)
		r.Route("/public", func(public chi.Router) {
			public.Get("/shares/media/{token}", sharesHandler.handleGetPublicShareMedia)
			public.Get("/shares/album/{token}", sharesHandler.handleGetPublicShareAlbum)
			public.Get("/shares/media/{token}/original", s.handleGetPublicShareMediaOriginal)
			public.Get("/shares/media/{token}/thumb", s.handleGetPublicShareMediaThumb)
			// Album media serving endpoints - serve by token (each item in album has its own share record)
			public.Get("/shares/album/{token}/media/original", func(w http.ResponseWriter, r *http.Request) {
				tokenStr := chi.URLParam(r, "token")
				mediaPath := r.URL.Query().Get("path")
				s.serveAlbumMediaOriginal(w, r, tokenStr, mediaPath)
			})
			public.Get("/shares/album/{token}/media/thumb", func(w http.ResponseWriter, r *http.Request) {
				tokenStr := chi.URLParam(r, "token")
				mediaPath := r.URL.Query().Get("path")
				s.serveAlbumMediaThumb(w, r, tokenStr, mediaPath)
			})
			// Paginated album media endpoint
			public.Get("/shares/album/{token}/media", sharesHandler.handleGetPublicShareAlbumMedia)
		})
	})

	// Serve static frontend files from /ui path
	s.router.Handle("/ui/*", http.StripPrefix("/ui/", http.FileServer(http.Dir("./ui"))))

	// SPA fallback: serve index.html for any other non-API routes
	s.router.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		http.ServeFile(w, r, "./ui/index.html")
	})
}

// startCleanupGoroutine starts a background goroutine that periodically cleans up expired upload sessions and orphaned chunk files
func (s *Server) startCleanupGoroutine() {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		mlog.Info("[INFO] UploadSessionManager: Cleanup goroutine started")

		for range ticker.C {
			cleaned := s.sessionManager.CleanupExpiredSessions()
			if cleaned > 0 {
				mlog.Info("[INFO] UploadSessionManager: Cleaned up %d expired sessions", cleaned)
			}

			// Also clean orphaned chunks (sessions lost during crash)
			removed := s.storageService.CleanupOrphanedChunks()
			if removed > 0 {
				mlog.Info("[INFO] StorageService: Cleaned up %d orphaned chunk files", removed)
			}
		}
	}()
}


// handleListPhotos returns a paginated list of photos only (now user-scoped)
func (s *Server) handleListPhotos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := r.URL.Query()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse limit
	limit := 20
	if lStr := query.Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Parse offset
	offset := 0
	if oStr := query.Get("offset"); oStr != "" {
		if o, err := strconv.Atoi(oStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// List returns (mediaList, total, error) - now user scoped
	mediaList, _, err := s.mediaRepo.List(ctx, limit, offset, &userID)

	// Use filtered count for pagination (not DB total since we filter by type)
	if err != nil {
		mlog.Info("[ERROR] handleListPhotos: %v", err)
		http.Error(w, "Failed to list media: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Filter to only photos
	// Note: We treat empty MediaType as photo for backward compatibility
	photoList := make([]*domain.Media, 0, len(mediaList))
	for _, m := range mediaList {
		if m.MediaType == domain.MediaTypePhoto || m.MediaType == "" {
			photoList = append(photoList, m)
		}
	}

	// Return both the media and the total count for the frontend to manage pagination
	response := struct {
		Media []*domain.Media `json:"media"`
		Total int             `json:"total"`
	}{
		Media: photoList,
		Total: len(photoList), // Use filtered count, not DB total
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetPhoto returns metadata for a single photo (now user-scoped)
func (s *Server) handleGetPhoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		mlog.Info("[ERROR] handleGetPhoto - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	media, err := s.mediaRepo.GetByID(ctx, id, &userID)
	if err != nil {
		mlog.Info("[ERROR] handleGetPhoto - getByID (%s): %v", id, err)
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}

	if media.MediaType != domain.MediaTypePhoto && media.MediaType != "" {
		http.Error(w, "Not a photo", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(media)
}

// handleGetPhotoFile streams the photo file with Range request support for backward compatibility (now user-scoped)
func (s *Server) handleGetPhotoFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		mlog.Info("[ERROR] handleGetPhotoFile - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	media, err := s.mediaRepo.GetByID(ctx, id, &userID)
	if err != nil {
		mlog.Info("[ERROR] handleGetPhotoFile - getByID (%s): %v", id, err)
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}

	if media.MediaType != domain.MediaTypePhoto && media.MediaType != "" {
		http.Error(w, "Not a photo", http.StatusNotFound)
		return
	}

	// Resolve the absolute path using our Storage Service
	absPath, err := s.storageService.ResolvePath(media.Path)
	if err != nil {
		mlog.Info("[ERROR] handleGetPhotoFile - resolvePath (%s): %v", media.Path, err)
		http.Error(w, "Could not locate file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set Content-Type based on media type and filename
	contentType := s.getContentType(media.MediaType, media.Filename)
	w.Header().Set("Content-Type", contentType)

	// Enable Range requests for video seeking (browsers need Accept-Ranges header)
	w.Header().Set("Accept-Ranges", "bytes")

	// http.ServeFile handles Range requests automatically (crucial for video/seeking)
	http.ServeFile(w, r, absPath)
}


// handleListMedia returns a paginated list of media items (photos + videos) (now user-scoped)
func (s *Server) handleListMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := r.URL.Query()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse limit
	limit := 20
	if lStr := query.Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Parse offset
	offset := 0
	if oStr := query.Get("offset"); oStr != "" {
		if o, err := strconv.Atoi(oStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// List returns (mediaList, total, error) - now user scoped
	mediaList, total, err := s.mediaRepo.List(ctx, limit, offset, &userID)

	// Use total count from database for pagination
	if err != nil {
		mlog.Info("[ERROR] handleListMedia: %v", err)
		http.Error(w, "Failed to list media: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return both the media and the total count for the frontend to manage pagination
	response := struct {
		Media []*domain.Media `json:"media"`
		Total int             `json:"total"`
	}{
		Media: mediaList,
		Total: total,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetMedia returns metadata for a single media item (now user-scoped)
func (s *Server) handleGetMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		mlog.Info("[ERROR] handleGetMedia - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	media, err := s.mediaRepo.GetByID(ctx, id, &userID)
	if err != nil {
		mlog.Info("[ERROR] handleGetMedia - getByID (%s): %v", id, err)
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(media)
}

// handleUpdateMedia handles PUT requests to update media metadata (including tags) (now user-scoped)
func (s *Server) handleUpdateMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		mlog.Info("[ERROR] handleUpdateMedia - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	var input domain.Media
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		mlog.Info("[ERROR] handleUpdateMedia - decode body: %v", err)
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Ensure ID matches and belongs to user
	input.ID = id
	if input.UserID != userID {
		http.Error(w, "Forbidden: Media does not belong to you", http.StatusForbidden)
		return
	}

	// Update in repository
	err = s.mediaRepo.Update(ctx, &input)
	if err != nil {
		mlog.Info("[ERROR] handleUpdateMedia - repo update (%s): %v", id, err)
		http.Error(w, "Failed to update media: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// handlePatchMedia handles PATCH requests for partial updates (e.g., just tags) (now user-scoped)
func (s *Server) handlePatchMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		mlog.Info("[ERROR] handlePatchMedia - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	var input map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		mlog.Info("[ERROR] handlePatchMedia - decode body: %v", err)
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get existing media first (checking ownership)
	existingMedia, err := s.mediaRepo.GetByID(ctx, id, &userID)
	if err != nil {
		mlog.Info("[ERROR] handlePatchMedia - getByID (%s): %v", id, err)
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}

	// Apply patches to tags if provided
	if tagsVal, ok := input["tags"]; ok && tagsVal != nil {
		existingMedia.Tags = fmt.Sprintf("%v", tagsVal)
	}

	// Update in repository
	err = s.mediaRepo.Update(ctx, existingMedia)
	if err != nil {
		mlog.Info("[ERROR] handlePatchMedia - repo update (%s): %v", id, err)
		http.Error(w, "Failed to update media: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// handleUpdateTags handles PATCH /media/{id}/tags for updating tags specifically (now user-scoped)
func (s *Server) handleUpdateTags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		mlog.Info("[ERROR] handleUpdateTags - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	var input struct {
		Tags string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		mlog.Info("[ERROR] handleUpdateTags - decode body: %v", err)
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get existing media first (checking ownership)
	existingMedia, err := s.mediaRepo.GetByID(ctx, id, &userID)
	if err != nil {
		mlog.Info("[ERROR] handleUpdateTags - getByID (%s): %v", id, err)
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}

	// Update tags
	existingMedia.Tags = input.Tags

	// Update in repository
	err = s.mediaRepo.Update(ctx, existingMedia)
	if err != nil {
		mlog.Info("[ERROR] handleUpdateTags - repo update (%s): %v", id, err)
		http.Error(w, "Failed to update media: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// handleDeleteMedia handles DELETE requests for media items (now user-scoped)
func (s *Server) handleDeleteMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		mlog.Info("[ERROR] handleDeleteMedia - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	err = s.mediaRepo.Delete(ctx, id, &userID)
	if err != nil {
		mlog.Info("[ERROR] handleDeleteMedia - delete (%s): %v", id, err)
		http.Error(w, "Failed to delete media: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// getContentType returns the appropriate Content-Type based on media type or file extension
func (s *Server) getContentType(mediaType domain.MediaType, filename string) string {
	mediaTypeStr := strings.ToLower(string(mediaType))

	// If we have explicit media type from database, use it
	if mediaType == domain.MediaTypeVideo {
		ext := strings.ToLower(filepath.Ext(filename))
		switch ext {
		case ".mp4":
			return "video/mp4"
		case ".webm":
			return "video/webm"
		case ".ogg":
			return "video/ogg"
		case ".mov":
			return "video/quicktime"
		case ".avi":
			return "video/x-msvideo"
		case ".mkv":
			return "video/x-matroska"
		default:
			return "video/mp4"
		}
	}

	if mediaType == domain.MediaTypePhoto || mediaTypeStr == "" {
		ext := strings.ToLower(filepath.Ext(filename))
		switch ext {
		case ".jpg", ".jpeg":
			return "image/jpeg"
		case ".png":
			return "image/png"
		case ".gif":
			return "image/gif"
		case ".webp":
			return "image/webp"
		case ".heic", ".heif":
			return "image/heic"
		default:
			return "application/octet-stream"
		}
	}

	return "application/octet-stream"
}

// handleGetOriginal streams the actual file with Range request support for video seeking (now user-scoped)
func (s *Server) handleGetOriginal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		mlog.Info("[ERROR] handleGetOriginal - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	media, err := s.mediaRepo.GetByID(ctx, id, &userID)
	if err != nil {
		mlog.Info("[ERROR] handleGetOriginal - getByID (%s): %v", id, err)
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}

	// Resolve the absolute path using our Storage Service
	targetPath := media.Path
	absPath, err := s.storageService.ResolvePath(targetPath)
	if err != nil {
		// If resolution fails (e.g., due to incorrect absolute path), try treating it as relative
		targetPath = strings.TrimLeft(targetPath, "/\\")
		absPath, err = s.storageService.ResolvePath(targetPath)
	}

	if err != nil {
		mlog.Info("[ERROR] handleGetOriginal - resolvePath (%s): %v", targetPath, err)
		http.Error(w, "Could not locate file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set Content-Type based on media type and filename
	contentType := s.getContentType(media.MediaType, media.Filename)
	w.Header().Set("Content-Type", contentType)

	// Enable Range requests for video seeking (browsers need Accept-Ranges header)
	w.Header().Set("Accept-Ranges", "bytes")

	// Expose Content-Range header via CORS so browsers can use Range requests from different origins
	w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type, Accept-Ranges, Content-Range")

	// http.ServeFile handles Range requests automatically (crucial for video/seeking)
	http.ServeFile(w, r, absPath)
}

// handleGetThumbnail streams the generated thumbnail with fallback to original file
func (s *Server) handleGetThumbnail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		mlog.Info("[ERROR] handleGetThumbnail - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	media, err := s.mediaRepo.GetByID(ctx, id, &userID)
	if err != nil {
		mlog.Info("[ERROR] handleGetThumbnail - getByID (%s): %v", id, err)
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}

	// Calculate thumbnail path - thumbnails are stored WITH user ID in the path
	ext := filepath.Ext(media.Path)
	thumbRelPath := s.storageService.GetThumbnailRelativePath(
		string(media.MediaType),
		media.Path,
		ext,
	)

	fullThumbPath := filepath.Join(s.thumbRoot, thumbRelPath)

	// Check if file exists before serving
	if _, err := os.Stat(fullThumbPath); os.IsNotExist(err) {
		mlog.Info("[WARN] handleGetThumbnail: Thumbnail NOT FOUND at %s. Attempting fallback to original.", fullThumbPath)
		targetPath := media.Path
		absPath, resolveErr := s.storageService.ResolvePath(targetPath)
		if resolveErr != nil {
			targetPath = strings.TrimLeft(targetPath, "/\\")
			absPath, resolveErr = s.storageService.ResolvePath(targetPath)
		}

		if resolveErr != nil {
			mlog.Info("[ERROR] handleGetThumbnail - fallback failed for (%s): %v", targetPath, resolveErr)
			http.Error(w, "Thumbnail not found and fallback failed", http.StatusNotFound)
			return
		}

		contentType := s.getContentType(media.MediaType, media.Filename)
		w.Header().Set("Content-Type", contentType)
		mlog.Info("[INFO] handleGetThumbnail: Serving original file as fallback from %s", absPath)
		http.ServeFile(w, r, absPath)
		return
	}

	http.ServeFile(w, r, fullThumbPath)
}

// handleListTrash returns a paginated list of trashed (soft-deleted) media items for the authenticated user.
func (s *Server) handleListTrash(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := r.URL.Query()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse limit and offset for pagination
	limit := 20
	offset := 0
	if lStr := query.Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}
	if oStr := query.Get("offset"); oStr != "" {
		if o, err := strconv.Atoi(oStr); err == nil && o >= 0 {
			offset = o
		}
	}

	trashedList, totalItems, err := s.mediaRepo.ListTrashed(ctx, limit, offset, userID)
	if err != nil {
		mlog.Info("[ERROR] handleListTrash: %v", err)
		http.Error(w, "Failed to list trashed media: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := struct {
		Media []*domain.Media `json:"media"`
		Total int             `json:"total"`
	}{
		Media: trashedList,
		Total: totalItems,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRestoreMedia restores a soft-deleted media item back to the active library.
func (s *Server) handleRestoreMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		mlog.Info("[ERROR] handleRestoreMedia - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	err = s.mediaRepo.RestoreMedia(ctx, id, userID)
	if err != nil {
		mlog.Info("[ERROR] handleRestoreMedia - restore (%s): %v", id, err)
		http.Error(w, "Failed to restore media: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "restored"})
}

// handlePermanentDeleteMedia permanently deletes a media item from the database and storage.
func (s *Server) handlePermanentDeleteMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		mlog.Info("[ERROR] handlePermanentDeleteMedia - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	// First get the trashed media to know which files to delete from storage
	media, err := s.mediaRepo.GetTrashedMedia(ctx, id, userID)
	if err != nil {
		mlog.Info("[ERROR] handlePermanentDeleteMedia - GetTrashedMedia (%s): %v", id, err)
		http.Error(w, "Failed to get media: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete files from storage (main file and thumbnail if exists)
	if err := s.storageService.DeleteFile(media.Path); err != nil {
		mlog.Info("[ERROR] handlePermanentDeleteMedia - delete main file (%s): %v", media.Path, err)
		// Continue with DB deletion even if file delete fails to avoid orphaned records
	}

	// Delete thumbnail separately (it's optional and may not exist)
	if thumbRelPath := s.getThumbnailRelativePath(media); thumbRelPath != "" {
		s.storageService.DeleteThumbnailSilently(thumbRelPath) // Ignore error for thumbnails
	}

	// Permanently delete from database
	err = s.mediaRepo.PermanentlyDeleteMedia(ctx, id, userID)
	if err != nil {
		mlog.Info("[ERROR] handlePermanentDeleteMedia - permanent delete (%s): %v", id, err)
		http.Error(w, "Failed to permanently delete media: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "permanently_deleted"})
}

// getThumbnailRelativePath returns the relative path for a thumbnail based on media type.
// Thumbnails are stored WITH the user ID in the path.
func (s *Server) getThumbnailRelativePath(media *domain.Media) string {
	relPath := filepath.Clean(media.Path)
	ext := filepath.Ext(relPath)

	// Do NOT strip user ID from path - thumbnails are stored WITH user ID
	if relPath == "" {
		return ""
	}

	if media.MediaType == domain.MediaTypeVideo {
		basePart := strings.TrimSuffix(relPath, ext)
		basePart = strings.Replace(basePart, "/.videos/", "/", 1)
		return basePart + ".webp"
	}

	basePart := strings.TrimSuffix(relPath, ext)
	return basePart + "_thumb.webp"
}

// ServeHTTP makes our Server struct implement the http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
