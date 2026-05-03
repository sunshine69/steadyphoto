package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// handleListPhotos returns a paginated list of photos
func (s *Server) handleListPhotos(w http.ResponseWriter, r *http.Request) {
	// Parse pagination parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 20 // default
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	offset := 0 // default
	if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
		offset = o
	}

	photos, total, err := s.repo.List(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, "Failed to list photos", http.StatusInternalServerError)
		return
	}

	response := struct {
		Data  interface{} `json:"data"`
		Total int         `json:"total"`
		Limit int         `json:"limit"`
		Offset int        `json:"offset"`
	}{
		Data:   photos,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetPhoto returns detailed information for a single photo
func (s *Server) handleGetPhoto(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid photo ID format", http.StatusBadRequest)
		return
	}

	photo, err := s.repo.GetByID(r.Context(), id)
	if err != nil {
		// In a production app, we would differentiate between 
		// "not found" and "database error". 
		// For now, we'll treat both as 404/500 generically.
		http.Error(w, "Photo not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(photo)
}
