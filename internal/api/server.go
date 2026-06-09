package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
}

func NewServer(
	mediaRepo domain.MediaRepository,
	albumRepo domain.AlbumRepository,
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	storageService *storage.StorageService,
	thumbRoot string,
) *Server {
	s := &Server{
		router:         chi.NewRouter(),
		mediaRepo:      mediaRepo,
		albumRepo:      albumRepo,
		userRepo:       userRepo,
		sessionRepo:    sessionRepo,
		storageService: storageService,
		thumbRoot:      thumbRoot,
		sessionManager: NewUploadSessionManager(storageService),
	}
	s.routes()
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
		// Authentication endpoints
		r.Route("/auth", func(r chi.Router) {
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

		// PROTECTED ROUTES group (for non-auth resources like media)
		r.Group(func(protected chi.Router) {
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
			uploadHandler := NewMediaUploadHandler(s.mediaRepo, s.storageService)
			protected.Route("/media/upload", func(r chi.Router) {
				// LimitBodySizeMiddleware removed - chunked uploads use small chunks (5MB),
				// and ParseMultipartForm handles per-part limits. The middleware's Content-Length check
				// was blocking large file uploads that Android sends as multipart forms with proper boundaries.

				r.Post("/", uploadHandler.Handle)

				// Single file upload endpoint (mobile client) - increased memory limit to 1GB
				sessionManager := s.sessionManager
				singleUploadHandler := NewMediaUploadHandlerSingle(s.mediaRepo, s.storageService, sessionManager)
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
					log.Printf("[ERROR] handleDeleteMedia - decode body: %v", err)
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
						log.Printf("[ERROR] handleDeleteMedia - parse UUID (%s): %v", idStr, err)
						continue // Skip invalid IDs but continue processing others
					}

					// First get the media to know which files to delete from storage
					media, err := s.mediaRepo.GetByID(ctx, id, &userID)
					if err != nil {
						log.Printf("[ERROR] handleDeleteMedia - GetByID (%s): %v", id, err)
						continue // Skip if media not found or doesn't belong to user
					}

					// Delete files from storage (main file and thumbnail if exists)
					if err := s.storageService.DeleteFile(media.Path); err != nil {
						log.Printf("[ERROR] handleDeleteMedia - delete main file (%s): %v", media.Path, err)
						// Continue with DB deletion even if file delete fails to avoid orphaned records
					}

					// Delete thumbnail separately (it's optional and may not exist)
					if thumbRelPath := s.storageService.GetThumbnailRelativePath(string(media.MediaType), media.Filename, filepath.Ext(media.Path)); thumbRelPath != "" {
						s.storageService.DeleteFileSilently(thumbRelPath) // Ignore error for thumbnails
					}

					// Permanently delete from database
					if err := s.mediaRepo.PermanentlyDeleteMedia(ctx, id, userID); err != nil {
						log.Printf("[ERROR] handleDeleteMedia - permanent delete (%s): %v", id, err)
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
		log.Printf("[ERROR] handleListPhotos: %v", err)
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
		log.Printf("[ERROR] handleGetPhoto - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	media, err := s.mediaRepo.GetByID(ctx, id, &userID)
	if err != nil {
		log.Printf("[ERROR] handleGetPhoto - getByID (%s): %v", id, err)
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
		log.Printf("[ERROR] handleGetPhotoFile - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	media, err := s.mediaRepo.GetByID(ctx, id, &userID)
	if err != nil {
		log.Printf("[ERROR] handleGetPhotoFile - getByID (%s): %v", id, err)
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
		log.Printf("[ERROR] handleGetPhotoFile - resolvePath (%s): %v", media.Path, err)
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

// handleSearchMedia searches for media by tags and returns matching items.
func (s *Server) handleSearchMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	tagQuery := r.URL.Query().Get("tag")
	if tagQuery == "" {
		http.Error(w, "Missing 'tag' query parameter", http.StatusBadRequest)
		return
	}

	matchingMedia, err := s.mediaRepo.SearchByTags(ctx, tagQuery, &userID)
	if err != nil {
		log.Printf("[ERROR] handleSearchMedia - search by tags: %v", err)
		http.Error(w, "Failed to search media: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matchingMedia)
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
		log.Printf("[ERROR] handleListMedia: %v", err)
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
		log.Printf("[ERROR] handleGetMedia - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	media, err := s.mediaRepo.GetByID(ctx, id, &userID)
	if err != nil {
		log.Printf("[ERROR] handleGetMedia - getByID (%s): %v", id, err)
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
		log.Printf("[ERROR] handleUpdateMedia - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	var input domain.Media
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Printf("[ERROR] handleUpdateMedia - decode body: %v", err)
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
		log.Printf("[ERROR] handleUpdateMedia - repo update (%s): %v", id, err)
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
		log.Printf("[ERROR] handlePatchMedia - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	var input map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Printf("[ERROR] handlePatchMedia - decode body: %v", err)
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get existing media first (checking ownership)
	existingMedia, err := s.mediaRepo.GetByID(ctx, id, &userID)
	if err != nil {
		log.Printf("[ERROR] handlePatchMedia - getByID (%s): %v", id, err)
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
		log.Printf("[ERROR] handlePatchMedia - repo update (%s): %v", id, err)
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
		log.Printf("[ERROR] handleUpdateTags - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	var input struct {
		Tags string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Printf("[ERROR] handleUpdateTags - decode body: %v", err)
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get existing media first (checking ownership)
	existingMedia, err := s.mediaRepo.GetByID(ctx, id, &userID)
	if err != nil {
		log.Printf("[ERROR] handleUpdateTags - getByID (%s): %v", id, err)
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}

	// Update tags
	existingMedia.Tags = input.Tags

	// Update in repository
	err = s.mediaRepo.Update(ctx, existingMedia)
	if err != nil {
		log.Printf("[ERROR] handleUpdateTags - repo update (%s): %v", id, err)
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
		log.Printf("[ERROR] handleDeleteMedia - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	err = s.mediaRepo.Delete(ctx, id, &userID)
	if err != nil {
		log.Printf("[ERROR] handleDeleteMedia - delete (%s): %v", id, err)
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
		log.Printf("[ERROR] handleGetOriginal - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	media, err := s.mediaRepo.GetByID(ctx, id, &userID)
	if err != nil {
		log.Printf("[ERROR] handleGetOriginal - getByID (%s): %v", id, err)
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
		log.Printf("[ERROR] handleGetOriginal - resolvePath (%s): %v", targetPath, err)
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
		log.Printf("[ERROR] handleGetThumbnail - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	media, err := s.mediaRepo.GetByID(ctx, id, &userID)
	if err != nil {
		log.Printf("[ERROR] handleGetThumbnail - getByID (%s): %v", id, err)
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}

	// Calculate thumbnail path
	relPath := filepath.Clean(media.Path)
	ext := filepath.Ext(relPath)

	var thumbRelPath string
	if media.MediaType == domain.MediaTypeVideo {
		cleanPath := strings.TrimPrefix(media.Path, "storage/")
		basePart := strings.TrimSuffix(cleanPath, ext)
		basePart = strings.Replace(basePart, "/.videos/", "/", 1)
		thumbRelPath = basePart + ".webp"
	} else {
		cleanPath := strings.TrimPrefix(media.Path, "storage/")
		basePart := strings.TrimSuffix(cleanPath, ext)
		thumbRelPath = basePart + "_thumb.webp"
	}

	fullThumbPath := filepath.Join(s.thumbRoot, ".thumbnails", thumbRelPath)

	// Check if file exists before serving
	if _, err := os.Stat(fullThumbPath); os.IsNotExist(err) {
		log.Printf("[WARN] handleGetThumbnail: Thumbnail NOT FOUND at %s. Attempting fallback to original.", fullThumbPath)
		targetPath := media.Path
		absPath, resolveErr := s.storageService.ResolvePath(targetPath)
		if resolveErr != nil {
			targetPath = strings.TrimLeft(targetPath, "/\\")
			absPath, resolveErr = s.storageService.ResolvePath(targetPath)
		}

		if resolveErr != nil {
			log.Printf("[ERROR] handleGetThumbnail - fallback failed for (%s): %v", targetPath, resolveErr)
			http.Error(w, "Thumbnail not found and fallback failed", http.StatusNotFound)
			return
		}

		contentType := s.getContentType(media.MediaType, media.Filename)
		w.Header().Set("Content-Type", contentType)
		log.Printf("[INFO] handleGetThumbnail: Serving original file as fallback from %s", absPath)
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
		log.Printf("[ERROR] handleListTrash: %v", err)
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
		log.Printf("[ERROR] handleRestoreMedia - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	err = s.mediaRepo.RestoreMedia(ctx, id, userID)
	if err != nil {
		log.Printf("[ERROR] handleRestoreMedia - restore (%s): %v", id, err)
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
		log.Printf("[ERROR] handlePermanentDeleteMedia - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	// First get the trashed media to know which files to delete from storage
	media, err := s.mediaRepo.GetTrashedMedia(ctx, id, userID)
	if err != nil {
		log.Printf("[ERROR] handlePermanentDeleteMedia - GetTrashedMedia (%s): %v", id, err)
		http.Error(w, "Failed to get media: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete files from storage (main file and thumbnail if exists)
	if err := s.storageService.DeleteFile(media.Path); err != nil {
		log.Printf("[ERROR] handlePermanentDeleteMedia - delete main file (%s): %v", media.Path, err)
		// Continue with DB deletion even if file delete fails to avoid orphaned records
	}

	// Delete thumbnail separately (it's optional and may not exist)
	if thumbRelPath := s.getThumbnailRelativePath(media); thumbRelPath != "" {
		s.storageService.DeleteFileSilently(thumbRelPath) // Ignore error for thumbnails
	}

	// Permanently delete from database
	err = s.mediaRepo.PermanentlyDeleteMedia(ctx, id, userID)
	if err != nil {
		log.Printf("[ERROR] handlePermanentDeleteMedia - permanent delete (%s): %v", id, err)
		http.Error(w, "Failed to permanently delete media: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "permanently_deleted"})
}

// getThumbnailRelativePath returns the relative path for a thumbnail based on media type.
func (s *Server) getThumbnailRelativePath(media *domain.Media) string {
	relPath := filepath.Clean(media.Path)
	ext := filepath.Ext(relPath)

	if media.MediaType == domain.MediaTypeVideo {
		cleanPath := strings.TrimPrefix(media.Path, "storage/")
		basePart := strings.TrimSuffix(cleanPath, ext)
		basePart = strings.Replace(basePart, "/.videos/", "/", 1)
		return basePart + ".webp"
	}

	cleanPath := strings.TrimPrefix(media.Path, "storage/")
	basePart := strings.TrimSuffix(cleanPath, ext)
	return basePart + "_thumb.webp"
}

// ServeHTTP makes our Server struct implement the http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
