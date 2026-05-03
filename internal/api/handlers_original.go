package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"steadyphoto/internal/storage"
)

// handleGetOriginal streams the original file from disk
func (s *Server) handleGetOriginal(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid photo ID format", http.StatusBadRequest)
		return
	}

	// 1. Look up the photo in the DB to get its path
	photo, err := s.repo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Photo not found", http.StatusNotFound)
		return
	}

	// 2. Resolve the absolute path using the storage service
	// We assume s.storage is added to the Server struct in the next step
	// For now, I'll use a placeholder logic or I should update the Server struct first.
	// Let's update the Server struct first to include storage.
}
