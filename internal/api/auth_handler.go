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

// RefreshRequest represents the request body for token refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
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
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	})

	// 2. Also set it as a Secure/None cookie if we were on HTTPS (for strict cross-origin)
	// For local development over HTTP, Lax is our best bet for most browsers.

	resp := AuthResponse{
		AccessToken:  session.ID.String(), // Still return it for AJAX/Bearer usage
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
