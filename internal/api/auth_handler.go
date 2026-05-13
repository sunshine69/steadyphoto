package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"steadyphoto/internal/domain"
	"steadyphoto/internal/security"

	"github.com/google/uuid"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UpdateProfileRequest struct {
	Email    *string `json:"email,omitempty"`
	Password *string `json:"password,omitempty"`
}

type AuthResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	UserID       uuid.UUID `json:"user_id"`
}

// HandleRegister handles POST /api/v1/auth/register
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	// 1. Check if user already exists
	existingUser, err := s.userRepo.GetByEmail(r.Context(), req.Email)
	if err == nil && existingUser != nil {
		http.Error(w, "User already exists", http.StatusConflict)
		return
	}

	// 2. Hash password
	hashedPassword, err := security.HashPassword(req.Password)
	if err != nil {
		log.Printf("[ERROR] handleRegister: failed to hash password: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 3. Create user
	newUser := &domain.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Status:       domain.UserStatusActive, // Set initial status as active
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(r.Context(), newUser); err != nil {
		log.Printf("[ERROR] handleRegister: failed to create user: %v", err)
		http.Error(w, "Failed to register user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// HandleLogin handles POST /api/v1/auth/login
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 1. Get user by email
	user, err := s.userRepo.GetByEmail(r.Context(), req.Email)
	if err != nil {
		// Use a generic error for security to prevent username enumeration
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Check if user is disabled (Soft-Delete check)
	if user.Status == domain.UserStatusDisabled {
		http.Error(w, "Account is disabled", http.StatusForbidden)
		return
	}

	// 2. Verify password
	if !security.CheckPasswordHash(req.Password, user.PasswordHash) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// 3. Create session (Opaque Token Pattern)
	refreshToken, err := security.GenerateRandomToken(32)
	if err != nil {
		log.Printf("[ERROR] handleLogin: failed to generate refresh token: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	session := &domain.UserSession{
		ID:               uuid.New(),
		UserID:           user.ID,
		RefreshTokenHash: security.HashToken(refreshToken),
		ExpiresAt:        time.Now().Add(7 * 24 * time.Hour), // 7 days
		IsRevoked:        false,
		CreatedAt:        time.Now(),
	}

	if err := s.sessionRepo.CreateSession(r.Context(), session); err != nil {
		log.Printf("[ERROR] handleLogin: failed to create session: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	resp := AuthResponse{
		AccessToken:  session.ID.String(), // Using Session ID as a placeholder Access Token
		RefreshToken: refreshToken,
		UserID:       user.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleUpdateProfile handles PATCH /api/v1/auth/profile
func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 1. Get current user profile
	user, err := s.userRepo.GetByID(r.Context(), userID)
	if err != nil || user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	updated := false

	// 2. Update Email if provided
	if req.Email != nil {
		newEmail := *req.Email
		if newEmail == "" || newEmail == user.Email {
			http.Error(w, "Invalid email address", http.StatusBadRequest)
			return
		}

		// Check for conflict with other users
		existingUser, err := s.userRepo.GetByEmail(r.Context(), newEmail)
		if err == nil && existingUser != nil && existingUser.ID != user.ID {
			http.Error(w, "Email already in use", http.StatusConflict)
			return
		}

		user.Email = newEmail
		updated = true
	}

	// 3. Update Password if provided
	if req.Password != nil {
		newPassword := *req.Password
		if len(newPassword) < 8 {
			http.Error(w, "Password must be at least 8 characters", http.StatusBadRequest)
			return
		}

		hashedPassword, err := security.HashPassword(newPassword)
		if err != nil {
			log.Printf("[ERROR] handleUpdateProfile: failed to hash password: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		user.PasswordHash = hashedPassword
		updated = true
	}

	if !updated {
		http.Error(w, "No changes provided", http.StatusBadRequest)
		return
	}

	// 4. Save updates
	user.UpdatedAt = time.Now()
	if err := s.userRepo.Update(r.Context(), user); err != nil {
		log.Printf("[ERROR] handleUpdateProfile: failed to update user: %v", err)
		http.Error(w, "Failed to update profile", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// handleDeleteProfile handles DELETE /api/v1/auth/profile (Soft-Delete)
func (s *Server) handleDeleteProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 1. Set user to disabled status (Soft-Delete)
	user, err := s.userRepo.GetByID(r.Context(), userID)
	if err != nil || user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	user.Status = domain.UserStatusDisabled
	user.UpdatedAt = time.Now()

	// 2. Update the user record in DB
	if err := s.userRepo.Update(r.Context(), user); err != nil {
		log.Printf("[ERROR] handleDeleteProfile: failed to update status: %v", err)
		http.Error(w, "Failed to delete account", http.StatusInternalServerError)
		return
	}

	// 3. Revoke all active sessions for this user immediately
	if err := s.sessionRepo.RevokeAllByUserID(r.Context(), userID); err != nil {
		log.Printf("[ERROR] handleDeleteProfile: failed to revoke sessions: %v", err)
		// We don't fail the whole request if session revocation fails, 
		// but we log it as a critical security concern for admin follow-up.
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content is standard successful deletion response
}
