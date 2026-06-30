package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

// handleSearchMedia searches for media by text across filename, tags, and metadata.
func (s *Server) handleSearchMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	query := r.URL.Query().Get("query")
	scope := r.URL.Query().Get("scope")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	// Parse limit and offset with defaults
	limit := 20
	offset := 0
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	matchingMedia, total, err := s.mediaRepo.Search(ctx, query, scope, limit, offset, &userID)
	if err != nil {
		log.Printf("[ERROR] handleSearchMedia - search: %v", err)
		http.Error(w, "Failed to search media: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": matchingMedia,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}
