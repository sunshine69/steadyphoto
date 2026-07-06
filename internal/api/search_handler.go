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
	dateRange := r.URL.Query().Get("dateRange")

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

	// Parse date range
	var startDate, endDate string
	if dateRange != "" {
		start, end, err := parseDateRange(dateRange)
		if err != nil {
			log.Printf("[ERROR] handleSearchMedia - dateRange parse: %v", err)
			http.Error(w, "Invalid date range format", http.StatusBadRequest)
			return
		}
		startDate = start.Format("2006/01/02")
		endDate = end.Format("2006/01/02")
	}

	matchingMedia, total, err := s.mediaRepo.Search(ctx, query, scope, limit, offset, &userID, startDate, endDate)
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
