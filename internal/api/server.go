package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"steadyphoto/internal/domain"
	"steadyphoto/internal/storage"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Server struct {
	router         *mux.Router
	photoRepo      domain.PhotoRepository
	storageService *storage.StorageService
	thumbRoot      string
}

func NewServer(photoRepo domain.PhotoRepository, storageService *storage.StorageService, thumbRoot string) *Server {
	s := &Server{
		router:         mux.NewRouter(),
		photoRepo:      photoRepo,
		storageService: storageService,
		thumbRoot:      thumbRoot,
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	// API Versioning
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// CORS Middleware - Simple implementation
	api.Use(func(next http.Handler) http.Handler {
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

	// Photo Routes
	api.HandleFunc("/photos", s.handleListPhotos).Methods(http.MethodGet)
	api.HandleFunc("/photos/{id}", s.handleGetPhoto).Methods(http.MethodGet)
	api.HandleFunc("/photos/{id}/original", s.handleGetOriginal).Methods(http.MethodGet)
	api.HandleFunc("/photos/{id}/thumb", s.handleGetThumbnail).Methods(http.MethodGet)
}

// handleListPhotos returns a paginated list of photos
func (s *Server) handleListPhotos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// List returns (photos, total, error)
	photos, _, err := s.photoRepo.List(ctx, 20, 0)
	if err != nil {
		http.Error(w, "Failed to list photos: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(photos)
}

// handleGetPhoto returns metadata for a single photo
func (s *Server) handleGetPhoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	photo, err := s.photoRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "Photo not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(photo)
}

// handleGetOriginal streams the actual file
func (s *Server) handleGetOriginal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	photo, err := s.photoRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "Photo not found", http.StatusNotFound)
		return
	}

	// Resolve the absolute path using our Storage Service
	absPath, err := s.storageService.ResolvePath(photo.Path)
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
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
		return
	}

	photo, err := s.photoRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "Photo not found", http.StatusNotFound)
		return
	}

	// Calculate thumbnail path
	relPath := filepath.Clean(photo.Path)
	ext := filepath.Ext(relPath)
	base := strings.TrimSuffix(relPath, ext)
	thumbRelPath := filepath.Join(base + "_thumb.webp")

	// The thumbnail is stored relative to the thumbRoot
	fullThumbPath := filepath.Join(s.thumbRoot, thumbRelPath)

	// Check if file exists before serving
	if _, err := os.Stat(fullThumbPath); os.IsNotExist(err) {
		// Fallback: Try to serve the original if thumbnail is missing (optional, but helpful for debugging)
		// For now, we stick to the design requirement: return 404 if thumb is missing
		http.Error(w, "Thumbnail not found", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, fullThumbPath)
}

// ServeHTTP makes our Server struct implement the http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
