package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"steadyphoto/internal/security"
)

// handleGetProfile returns the current user's profile information.
func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := s.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		log.Printf("[ERROR] handleGetProfile: failed to get user (%s): %v", userID, err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     user.ID.String(),
		"email":  user.Email,
		"status": user.Status,
		"role":   user.Role,
	})
}

// handleUpdateProfileEmail handles PATCH /api/v1/auth/profile/email - updates the user's email address.
func (s *Server) handleUpdateProfileEmail(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		NewEmail string `json:"new_email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NewEmail == "" {
		http.Error(w, "Invalid request body: new_email is required", http.StatusBadRequest)
		return
	}

	// Get the user
	user, err := s.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		log.Printf("[ERROR] handleUpdateProfileEmail: failed to get user (%s): %v", userID, err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Check if the new email is already in use by another user
	existingUser, err := s.userRepo.GetByEmail(r.Context(), req.NewEmail)
	if !errors.Is(err, sql.ErrNoRows) {
		// If error other than no rows found, it's a DB issue
		log.Printf("[ERROR] handleUpdateProfileEmail: failed to check existing email (%s): %v", req.NewEmail, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if existingUser != nil && existingUser.ID.String() != userID.String() {
		http.Error(w, "Email already in use by another account", http.StatusConflict)
		return
	}

	user.Email = req.NewEmail
	if err := s.userRepo.Update(r.Context(), user); err != nil {
		log.Printf("[ERROR] handleUpdateProfileEmail: failed to update email for user (%s): %v", userID, err)
		http.Error(w, "Failed to update email", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// handleChangePassword handles PATCH /api/v1/auth/profile/password - changes the user's password.
func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CurrentPassword == "" || req.NewPassword == "" {
		http.Error(w, "Invalid request body: current_password and new_password are required", http.StatusBadRequest)
		return
	}

	// Get the user
	user, err := s.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		log.Printf("[ERROR] handleChangePassword: failed to get user (%s): %v", userID, err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Verify current password
	if !security.CheckPasswordHash(req.CurrentPassword, user.PasswordHash) {
		http.Error(w, "Current password is incorrect", http.StatusUnauthorized)
		return
	}

	// Hash the new password
	newPasswordHash, err := security.HashPassword(req.NewPassword)
	if err != nil {
		log.Printf("[ERROR] handleChangePassword: failed to hash new password: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	user.PasswordHash = newPasswordHash
	if err := s.userRepo.Update(r.Context(), user); err != nil {
		log.Printf("[ERROR] handleChangePassword: failed to update password for user (%s): %v", userID, err)
		http.Error(w, "Failed to change password", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
// handleSearchUsers handles GET /api/v1/users/search?query=... - searches for users by email/username
func (s *Server) handleSearchUsers(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	queryParam := r.URL.Query().Get("query")
	if queryParam == "" || len(queryParam) < 2 {
		http.Error(w, "Query parameter 'query' is required and must be at least 2 characters", http.StatusBadRequest)
		return
	}

	users, err := s.userRepo.SearchUsers(r.Context(), queryParam)
	if err != nil {
		log.Printf("[ERROR] handleSearchUsers: failed to search users for user (%s): %v", userID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}
