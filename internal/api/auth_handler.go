package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"steadyphoto/internal/domain"
	"steadyphoto/internal/security"

	"github.com/google/uuid"
)

// LoginRequest represents the request body for login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse represents the response after successful authentication.
type AuthResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	UserID       uuid.UUID `json:"user_id"`
	Role         string    `json:"role"`
	Status       string    `json:"status"`
}

// RegisterRequest represents the request body for user registration.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
}

// RefreshRequest represents the request body for token refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// CookieSecureMode determines whether to set HttpOnly cookies as Secure (HTTPS-only).
// Set via SECURE_COOKIES environment variable: "true" or "false". Defaults to false for local dev.
var CookieSecureMode bool = false

func init() {
	if val := os.Getenv("SECURE_COOKIES"); val != "" && (val == "true" || val == "1") {
		CookieSecureMode = true
	}
	log.Printf("[CONFIG] Secure cookie mode: %v", CookieSecureMode)
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

	// Check if user is pending approval, rejected, or disabled
	if user.Status == domain.UserStatusPending {
		http.Error(w, "Account pending admin approval", http.StatusForbidden)
		return
	}

	if user.Status == domain.UserStatusRejected {
		http.Error(w, "Registration rejected by admin", http.StatusForbidden)
		return
	}

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

	// 1. Set the access token cookie for resource requests (images/videos)
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    session.ID.String(),
		Path:     "/",
		HttpOnly: true,  // Prevent JS from accessing the token
		Secure:   CookieSecureMode, // Set to true in production with HTTPS (or behind reverse proxy)
		SameSite: http.SameSiteLaxMode,
	})

	resp := AuthResponse{
		AccessToken:  session.ID.String(), // Return access token for API authentication (e.g., Android scanner)
		RefreshToken: refreshToken,
		UserID:       user.ID,
		Role:         user.Role,
		Status:       user.Status,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleRefresh handles POST /api/v1/auth/refresh. It validates the refresh token,
// revokes the old session, creates a new one, and returns a fresh access token.
func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	// 1. Parse request body
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.RefreshToken == "" {
		http.Error(w, "refresh_token is required", http.StatusBadRequest)
		return
	}

	// 2. Hash the refresh token to look up in DB (never store plain-text tokens)
	tokenHash := security.HashToken(req.RefreshToken)

	// 3. Look up session by hash
	session, err := s.sessionRepo.GetByRefreshTokenHash(r.Context(), tokenHash)
	if err != nil {
		log.Printf("[ERROR] handleRefresh: session lookup failed for refresh token: %v", err)
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	if session == nil {
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	// 4. Check if the session is revoked or expired (using same logic as AuthMiddleware)
	if session.IsRevoked || time.Now().After(session.ExpiresAt) {
		log.Printf("[WARN] handleRefresh: Session %s rejected (revoked=%v, expires_at=%v)",
			session.ID, session.IsRevoked, session.ExpiresAt)
		http.Error(w, "Refresh token has expired or been revoked", http.StatusUnauthorized)
		return
	}

	// 5. Revoke the old session (one-time-use refresh tokens)
	if err := s.sessionRepo.RevokeSession(r.Context(), session.ID); err != nil {
		log.Printf("[ERROR] handleRefresh: failed to revoke old session %s: %v", session.ID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 6. Create a new session with a fresh access token for this user
	newSession := &domain.UserSession{
		ID:               uuid.New(),
		UserID:           session.UserID,
		RefreshTokenHash: security.HashToken(req.RefreshToken), // Same refresh token hash (token is reused)
		ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),   // 7 days from now
		IsRevoked:        false,
		CreatedAt:        time.Now(),
	}

	if err := s.sessionRepo.CreateSession(r.Context(), newSession); err != nil {
		log.Printf("[ERROR] handleRefresh: failed to create new session for user %s: %v", session.UserID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 7. Return the new access token (the client already knows who they are)
	resp := map[string]string{
		"access_token": newSession.ID.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleRegister handles POST /api/v1/auth/register - creates a new pending user account.
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

	// Check if user already exists
	existingUser, err := s.userRepo.GetByEmail(r.Context(), req.Email)
	// If the error is NOT "no rows found", it's a real database error
	if !errors.Is(err, sql.ErrNoRows) { // ErrNoRows means user doesn't exist (not an actual DB error)
		log.Printf("[ERROR] handleRegister: failed to check existing email (%s): %v", req.Email, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if existingUser != nil {
		http.Error(w, "Email already registered", http.StatusConflict)
		return
	}

	// Hash the password
	passwordHash, err := security.HashPassword(req.Password)
	if err != nil {
		log.Printf("[ERROR] handleRegister: failed to hash password: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create new user with pending status (requires admin approval)
	newUser := &domain.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: passwordHash,
		Status:       domain.UserStatusPending, // Pending admin approval by default
		Role:         domain.UserRoleUser,      // Default role is user
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(r.Context(), newUser); err != nil {
		log.Printf("[ERROR] handleRegister: failed to create user (%s): %v", req.Email, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "pending_approval",
		"user_id":  newUser.ID.String(),
		"message":  "Registration successful. Account pending admin approval.",
	})
}

// handleUpdateProfile handles PATCH /api/v1/auth/profile - updates user profile (currently only status).
func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get the user
	user, err := s.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		log.Printf("[ERROR] handleUpdateProfile: failed to get user (%s): %v", userID, err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Update status if provided
	if req.Status != "" && (req.Status == domain.UserStatusActive || req.Status == domain.UserStatusDisabled || req.Status == domain.UserStatusRejected) {
		user.Status = req.Status
	}

	if err := s.userRepo.Update(r.Context(), user); err != nil {
		log.Printf("[ERROR] handleUpdateProfile: failed to update user (%s): %v", userID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// handleDeleteProfile handles DELETE /api/v1/auth/profile - deletes user account and revokes all sessions.
func (s *Server) handleDeleteProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Revoke all sessions for this user first
	if err := s.sessionRepo.RevokeAllByUserID(r.Context(), userID); err != nil {
		log.Printf("[ERROR] handleDeleteProfile: failed to revoke sessions for user (%s): %v", userID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// TODO: Delete all media belonging to this user (would need a method in MediaRepository)
	// For now, we just delete the user from the database
	
	// Since UserRepo doesn't have a Delete method, we'll update status to disabled as a soft-delete
	user, err := s.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		log.Printf("[ERROR] handleDeleteProfile: failed to get user (%s): %v", userID, err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	user.Status = domain.UserStatusDisabled
	if err := s.userRepo.Update(r.Context(), user); err != nil {
		log.Printf("[ERROR] handleDeleteProfile: failed to disable user (%s): %v", userID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
