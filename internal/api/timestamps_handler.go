package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jbrodriguez/mlog"
)

// minValidDate is the minimum acceptable date (Jan 1, 1980)
var minValidDate = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

// handleUpdateMediaTimestamps handles PATCH /api/v1/media/{id}/timestamps to update capturedAt and fileCreatedAt
func (s *Server) handleUpdateMediaTimestamps(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid uuid", http.StatusBadRequest)
		return
	}

	var request struct {
		CapturedAt    *string `json:"capturedAt"`
		FileCreatedAt *string `json:"fileCreatedAt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Get existing media first (checking ownership)
	existingMedia, err := s.mediaRepo.GetByID(ctx, id, &userID)
	if err != nil {
		http.Error(w, "media not found", http.StatusNotFound)
		return
	}

	// Parse capturedAt if provided
	if request.CapturedAt != nil && *request.CapturedAt != "" {
		capturedAt, err := parseTimestamp(*request.CapturedAt)
		if err != nil {
			mlog.Info("[ERROR] handleUpdateMediaTimestamps - parse capturedAt (%s): %v", *request.CapturedAt, err)
			http.Error(w, "invalid capturedAt format: "+err.Error(), http.StatusBadRequest)
			return
		}
		if !isValidDate(capturedAt) {
			http.Error(w, "capturedAt date is not valid (must be between 1980 and today)", http.StatusBadRequest)
			return
		}
		existingMedia.CapturedAt = capturedAt
	}

	// Parse fileCreatedAt if provided
	if request.FileCreatedAt != nil && *request.FileCreatedAt != "" {
		fileCreatedAt, err := parseTimestamp(*request.FileCreatedAt)
		if err != nil {
			mlog.Info("[ERROR] handleUpdateMediaTimestamps - parse fileCreatedAt (%s): %v", *request.FileCreatedAt, err)
			http.Error(w, "invalid fileCreatedAt format: "+err.Error(), http.StatusBadRequest)
			return
		}
		if !isValidDate(fileCreatedAt) {
			http.Error(w, "fileCreatedAt date is not valid (must be between 1980 and today)", http.StatusBadRequest)
			return
		}
		existingMedia.FileCreatedAt = &fileCreatedAt
	}

	// Update in repository
	if err := s.mediaRepo.Update(ctx, existingMedia); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// parseTimestamp parses a timestamp string in the Go time format (yyyy/mm/dd hh:mm:ss or yyyy/mm/dd).
// These are the only formats the Android client sends.
func parseTimestamp(s string) (time.Time, error) {
	// Try Go time format first (yyyy/mm/dd hh:mm:ss)
	if t, err := time.Parse("2006/01/02 15:04:05", s); err == nil {
		return t, nil
	}
	// Try date only (yyyy/mm/dd)
	if t, err := time.Parse("2006/01/02", s); err == nil {
		return t, nil
	}
	// Unix timestamp (seconds since epoch) - only accept if the string is purely numeric
	if t, err := time.Parse("02", s); err == nil && t.Unix() > 0 && t.Unix() < 1<<32 {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("unable to parse timestamp: %q", s)
}

// isValidDate checks if a date is between 1980-01-01 and today.
// Rejects future dates and dates before 1980.
func isValidDate(t time.Time) bool {
	// Reject dates in the future (within a 1-minute tolerance for clock skew)
	now := time.Now().UTC().Add(time.Minute)
	if t.After(now) {
		return false
	}
	// Reject dates before 1980-01-01
	if t.Before(minValidDate) {
		return false
	}
	return true
}
