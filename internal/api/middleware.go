package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

// contextKey is a private type to avoid collisions with other packages in the context
type contextKey string

const (
	// UserIDContextKey is the key used to store and retrieve UserID from the request context
	UserIDContextKey contextKey = "user_id"
)

// AuthMiddleware validates authentication for protected routes.
// It expects a Bearer token in the Authorization header, which we currently treat as the Session ID.
func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tokenStr string

		// 1. Try to get token from Authorization header (for AJAX/API calls)
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				tokenStr = parts[1]
				fmt.Fprintf(os.Stderr, "[DEBUG] Middleware: Found token in Authorization header\n")
			}
		}

		// 2. If no header token found, try to get token from Cookie (for images/videos)
		if tokenStr == "" {
			cookie, err := r.Cookie("access_token")
			if err != nil {
				fmt.Fprintf(os.Stderr, "[DEBUG] Middleware: No Authorization header AND no access_token cookie found (%v)\n", err)
			} else if cookie != nil {
				tokenStr = cookie.Value
				fmt.Fprintf(os.Stderr, "[DEBUG] Middleware: Found token in Cookie\n")
			}
		}

		// 3. If still no token after both attempts, unauthorized
		if tokenStr == "" {
			http.Error(w, "Unauthorized: No valid authentication method found", http.StatusUnauthorized)
			return
		}

		sessionID, err := uuid.Parse(tokenStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[DEBUG] Middleware: Token parsing error for %s: %v\n", tokenStr, err)
			http.Error(w, "Unauthorized: Invalid token format", http.StatusUnauthorized)
			return
		}

		// Look up the session in our repository
		session, err := s.sessionRepo.GetByID(r.Context(), sessionID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[DEBUG] Middleware: Session lookup error for %s: %v\n", sessionID, err)
			http.Error(w, "Unauthorized: Invalid or expired session", http.StatusUnauthorized)
			return
		}

		if session == nil {
			fmt.Fprintf(os.Stderr, "[DEBUG] Middleware: Session not found in DB for %s\n", sessionID)
			http.Error(w, "Unauthorized: Invalid or expired session", http.StatusUnauthorized)
			return
		}

		// Check if the session has been revoked or expired
		if session.IsRevoked || time.Now().After(session.ExpiresAt) {
			fmt.Fprintf(os.Stderr, "[DEBUG] Middleware: Session %s is invalid (Revoked=%v, ExpiresAt=%v, Now=%v)\n",
				sessionID, session.IsRevoked, session.ExpiresAt, time.Now())
			http.Error(w, "Unauthorized: Session is invalid or expired", http.StatusUnauthorized)
			return
		}

		fmt.Fprintf(os.Stderr, "[DEBUG] Middleware: Authentication successful for User %s\n", session.UserID)

		// Inject UserID into the request context
		ctx := context.WithValue(r.Context(), UserIDContextKey, session.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserIDFromContext is a helper function to retrieve the authenticated UserID from a context
func GetUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(UserIDContextKey).(uuid.UUID)
	return userID, ok
}
