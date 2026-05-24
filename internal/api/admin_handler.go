package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"steadyphoto/internal/domain"

	"github.com/google/uuid"
)

type AdminUpdateUserRequest struct {
	Status string `json:"status"` // pending, active, disabled, rejected
	Role   string `json:"role"`   // admin, user
}

// handleAdminListUsers handles GET /api/v1/admin/users
func (s *Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")

	users, err := s.userRepo.ListUsers(r.Context(), status)
	if err != nil {
		log.Printf("[ERROR] handleAdminListUsers: failed to list users: %v", err)
		http.Error(w, "Failed to list users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// handleAdminGetUser handles GET /api/v1/admin/users/{id}
func (s *Server) handleAdminGetUser(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from URL path: /api/v1/admin/users/{id}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 6 {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(parts[len(parts)-1])
	if err != nil {
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	user, err := s.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		log.Printf("[ERROR] handleAdminGetUser: failed to get user: %v", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// handleAdminUpdateUser handles PATCH /api/v1/admin/users/{id}
func (s *Server) handleAdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from URL path: /api/v1/admin/users/{id}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 6 {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(parts[len(parts)-1])
	if err != nil {
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	var req AdminUpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate status if provided
	if req.Status != "" && !isValidStatus(req.Status) {
		http.Error(w, "Invalid status value", http.StatusBadRequest)
		return
	}

	// Validate role if provided
	if req.Role != "" && !isValidRole(req.Role) {
		http.Error(w, "Invalid role value", http.StatusBadRequest)
		return
	}

	user, err := s.userRepo.GetByID(r.Context(), userID)
	if err != nil || user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if req.Status != "" {
		user.Status = req.Status
	}
	if req.Role != "" {
		user.Role = req.Role
	}

	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(r.Context(), user); err != nil {
		log.Printf("[ERROR] handleAdminUpdateUser: failed to update user: %v", err)
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// handleAdminDeleteUser handles DELETE /api/v1/admin/users/{id}
func (s *Server) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from URL path: /api/v1/admin/users/{id}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 6 {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(parts[len(parts)-1])
	if err != nil {
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	user, err := s.userRepo.GetByID(r.Context(), userID)
	if err != nil || user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Soft-delete: set status to disabled
	user.Status = domain.UserStatusDisabled
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(r.Context(), user); err != nil {
		log.Printf("[ERROR] handleAdminDeleteUser: failed to delete user: %v", err)
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	// Revoke all sessions for this user
	if err := s.sessionRepo.RevokeAllByUserID(r.Context(), userID); err != nil {
		log.Printf("[ERROR] handleAdminDeleteUser: failed to revoke sessions: %v", err)
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper functions
func isValidStatus(status string) bool {
	switch status {
	case domain.UserStatusPending, domain.UserStatusActive, domain.UserStatusDisabled, domain.UserStatusRejected:
		return true
	default:
		return false
	}
}

func isValidRole(role string) bool {
	switch role {
	case domain.UserRoleAdmin, domain.UserRoleUser:
		return true
	default:
		return false
	}
}
