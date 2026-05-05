package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"steadyphoto/internal/domain"
	"steadyphoto/internal/storage"
)

type Server struct {
	router         *mux.Router
	photoRepo      domain.PhotoRepository
	storageService *storage.StorageService
}

func NewServer(photoRepo domain.PhotoRepository, storageService *storage.StorageService) *Server {
	s := &Server{
		router:         mux.NewRouter(),
		photoRepo:      photoRepo,
		storageService: storageService,
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

// ServeHTTP makes our Server struct implement the http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
