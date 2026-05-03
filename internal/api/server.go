package api

import (
	"encoding/json"
	"net/http"

	"steadyphoto/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server holds the dependencies for the API
type Server struct {
	router *chi.Mux
	repo   domain.PhotoRepository
}

// NewServer creates a new API server with the provided repository
func NewServer(repo domain.PhotoRepository) *Server {
	s := &Server{
		router: chi.NewRouter(),
		repo:   repo,
	}

	s.routes()
	return s
}

// routes configures the middleware and API endpoints
func (s *Server) routes() {
	// Standard middlewares for production-ready APIs
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.RealIP)

	// Health check (outside versioned API)
	s.router.Get("/health", s.handleHealth)

	// API Versioning
	s.router.Route("/api/v1", func(r chi.Router) {
		r.Route("/photos", func(r chi.Router) {
			// Future handlers:
			// r.Get("/", s.handleListPhotos)
			// r.Get("/{id}", s.handleGetPhoto)
		})
	})
}

// handleHealth provides a simple endpoint to verify the server is running
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Handler returns the underlying http.Handler
func (s *Server) Handler() http.Handler {
	return s.router
}
