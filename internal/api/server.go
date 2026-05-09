package api

import (
	"encoding/json"
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
		// CORS Middleware
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
				w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")

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
		r.Route("/media/{id}", func(r chi.Router) {
			r.Get("/", s.handleGetMedia)
			r.Get("/original", s.handleGetOriginal)
			r.Get("/thumb", s.handleGetThumbnail)
		})
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
	mediaList, total, err := s.mediaRepo.List(ctx, limit, offset)

	_ = total
	if err != nil {
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
		Total: len(photoList),
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
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	media, err := s.mediaRepo.GetByID(ctx, id)
	if err != nil {
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

// handleGetPhotoFile streams the photo file
func (s *Server) handleGetPhotoFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	media, err := s.mediaRepo.GetByID(ctx, id)
	if err != nil {
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
		http.Error(w, "Could not locate file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// http.ServeFile handles Range requests automatically
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

	_ = total
	if err != nil {
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
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	media, err := s.mediaRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(media)
}

// handleGetOriginal streams the actual file
func (s *Server) handleGetOriginal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	media, err := s.mediaRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}

	// Resolve the absolute path using our Storage Service
	absPath, err := s.storageService.ResolvePath(media.Path)
	if err != nil {
		http.Error(w, "Could not locate file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// http.ServeFile handles Range requests automatically (crucial for video/seeking)
	http.ServeFile(w, r, absPath)
}

// handleGetThumbnail streams the generated thumbnail
func (s *Server) handleGetThumbnail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	media, err := s.mediaRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "Media not found", http.StatusNotFound)
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

	// The thumbnail is stored relative to the thumbRoot
	fullThumbPath := filepath.Join(s.thumbRoot, thumbRelPath)

	// Check if file exists before serving
	if _, err := os.Stat(fullThumbPath); os.IsNotExist(err) {
		http.Error(w, "Thumbnail not found", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, fullThumbPath)
}

// ServeHTTP makes our Server struct implement the http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
