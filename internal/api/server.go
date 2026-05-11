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

	"steadyphoto/internal/domain"
	"steadyphoto/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type Server struct {
	router         *chi.Mux
	mediaRepo      domain.MediaRepository
	storageService *storage.StorageService
	thumbRoot      string
}

func NewServer(mediaRepo domain.MediaRepository, storageService *storage.StorageService, thumbRoot string) *Server {
	s := &Server{
		router:         chi.NewRouter(),
		mediaRepo:      mediaRepo,
		storageService: storageService,
		thumbRoot:      thumbRoot,
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	// Standard middleware
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)

	// API Versioning
	s.router.Route("/api/v1", func(r chi.Router) {
		// CORS Middleware - exposed headers must include Accept-Ranges and Content-Range for video seeking to work cross-origin
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS, PUT, PATCH, DELETE")
				w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Range")
				w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type, Accept-Ranges, Content-Range")

				if r.Method == "OPTIONS" {
					w.WriteHeader(http.StatusOK)
					return
				}

				next.ServeHTTP(w, r)
			})
		})

		// Photo-specific endpoints (backward compatible)
		r.Get("/photos", s.handleListPhotos)
		r.Get("/photos/{id}", s.handleGetPhoto)
		r.Get("/photos/{id}/file", s.handleGetPhotoFile)
		r.Get("/photos/{id}/thumb", s.handleGetThumbnail)

		// Unified media endpoints (for video support)
		r.Get("/media", s.handleListMedia)
		r.Get("/media/search", s.handleSearchMedia)
		r.Route("/media/{id}", func(r chi.Router) {
			r.Get("/", s.handleGetMedia)
			r.Put("/", s.handleUpdateMedia)
			r.Patch("/", s.handlePatchMedia)
			r.Delete("/", s.handleDeleteMedia)
			r.Get("/original", s.handleGetOriginal)
			r.Get("/thumb", s.handleGetThumbnail)
			r.Patch("/tags", s.handleUpdateTags)
		})
	})

	// Serve static frontend files (if needed for SPA fallback)
	s.router.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		http.ServeFile(w, r, "angular-app/dist/angular-app/index.html")
	})
}

// handleListPhotos returns a paginated list of photos only
func (s *Server) handleListPhotos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := r.URL.Query()

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

	// List returns (mediaList, total, error)
	mediaList, _, err := s.mediaRepo.List(ctx, limit, offset)

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

// handleGetPhoto returns metadata for a single photo
func (s *Server) handleGetPhoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Printf("[ERROR] handleGetPhoto - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	media, err := s.mediaRepo.GetByID(ctx, id)
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

// handleGetPhotoFile streams the photo file with Range request support for backward compatibility
func (s *Server) handleGetPhotoFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Printf("[ERROR] handleGetPhotoFile - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	media, err := s.mediaRepo.GetByID(ctx, id)
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

// handleListMedia returns a paginated list of media items (photos + videos)
func (s *Server) handleListMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := r.URL.Query()

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

	// List returns (mediaList, total, error)
	mediaList, total, err := s.mediaRepo.List(ctx, limit, offset)

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

// handleGetMedia returns metadata for a single media item
func (s *Server) handleGetMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Printf("[ERROR] handleGetMedia - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	media, err := s.mediaRepo.GetByID(ctx, id)
	if err != nil {
		log.Printf("[ERROR] handleGetMedia - getByID (%s): %v", id, err)
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(media)
}

// handleUpdateMedia handles PUT requests to update media metadata (including tags)
func (s *Server) handleUpdateMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

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

	// Ensure ID matches
	input.ID = id

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

// handlePatchMedia handles PATCH requests for partial updates (e.g., just tags)
func (s *Server) handlePatchMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

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

	// Get existing media first
	existingMedia, err := s.mediaRepo.GetByID(ctx, id)
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

// handleUpdateTags handles PATCH /media/{id}/tags for updating tags specifically
func (s *Server) handleUpdateTags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

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

	// Get existing media first
	existingMedia, err := s.mediaRepo.GetByID(ctx, id)
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

// handleDeleteMedia handles DELETE requests for media items
func (s *Server) handleDeleteMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Printf("[ERROR] handleDeleteMedia - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	err = s.mediaRepo.Delete(ctx, id)
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

// handleGetOriginal streams the actual file with Range request support for video seeking
func (s *Server) handleGetOriginal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Printf("[ERROR] handleGetOriginal - parse UUID (%s): %v", idStr, err)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	media, err := s.mediaRepo.GetByID(ctx, id)
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

	fmt.Fprintf(os.Stderr, "[DEBUG] Entering handleGetThumbnail for ID: %s\n", idStr)

	id, err := uuid.Parse(idStr)
	if err != nil {
		errMsg := fmt.Sprintf("[ERROR] handleGetThumbnail - parse UUID (%s): %v\n", idStr, err)
		fmt.Fprint(os.Stderr, errMsg)
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	media, err := s.mediaRepo.GetByID(ctx, id)
	if err != nil {
		errMsg := fmt.Sprintf("[ERROR] handleGetThumbnail - getByID (%s): %v\n", id, err)
		fmt.Fprint(os.Stderr, errMsg)
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}
	fmt.Fprintf(os.Stderr, "[DEBUG] Found media item: Path=%s, Type=%s\n", media.Path, media.MediaType)

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
	fmt.Fprintf(os.Stderr, "[DEBUG] Calculated fullThumbPath: %s\n", fullThumbPath)

	// Check if file exists before serving
	if _, err := os.Stat(fullThumbPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "[DEBUG] Thumbnail NOT FOUND at %s. Attempting fallback to original.\n", fullThumbPath)
		targetPath := media.Path
		absPath, resolveErr := s.storageService.ResolvePath(targetPath)
		if resolveErr != nil {
			targetPath = strings.TrimLeft(targetPath, "/\\")
			absPath, resolveErr = s.storageService.ResolvePath(targetPath)
		}

		if resolveErr != nil {
			errMsg := fmt.Sprintf("[ERROR] handleGetThumbnail - fallback failed for (%s): %v\n", targetPath, resolveErr)
			fmt.Fprint(os.Stderr, errMsg)
			http.Error(w, "Thumbnail not found and fallback failed", http.StatusNotFound)
			return
		}

		fmt.Fprintf(os.Stderr, "[DEBUG] Fallback successful. Serving original file from: %s\n", absPath)
		contentType := s.getContentType(media.MediaType, media.Filename)
		w.Header().Set("Content-Type", contentType)
		http.ServeFile(w, r, absPath)
		return
	}

	fmt.Fprintf(os.Stderr, "[DEBUG] Thumbnail found! Serving: %s\n", fullThumbPath)
	http.ServeFile(w, r, fullThumbPath)
}

// handleSearchMedia handles GET /api/v1/media/search?tags=...
func (s *Server) handleSearchMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tags := r.URL.Query().Get("tags")
	if tags == "" {
		http.Error(w, "tags parameter is required", http.StatusBadRequest)
		return
	}

	mediaList, err := s.mediaRepo.SearchByTags(ctx, tags)
	if err != nil {
		log.Printf("[ERROR] handleSearchMedia: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := struct {
		Media []*domain.Media `json:"media"`
		Total int             `json:"total"`
	}{
		Media: mediaList,
		Total: len(mediaList),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ServeHTTP makes our Server struct implement the http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
