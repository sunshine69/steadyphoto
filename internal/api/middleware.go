package api

import (
	"context"
	"net/http"
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
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized: Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Expected format: Bearer <token>
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, "Unauthorized: Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		tokenStr := parts[1]
		sessionID, err := uuid.Parse(tokenStr)
		if err != nil {
			http.Error(w, "Unauthorized: Invalid token format", http.StatusUnauthorized)
			return
		}

		// Look up the session in our repository
		session, err := s.sessionRepo.GetByID(r.Context(), sessionID)
		if err != nil || session == nil {
			http.Error(w, "Unauthorized: Invalid or expired session", http.StatusUnauthorized)
			return
		}

		// Check if the session has been revoked or expired
		if session.IsRevoked || time.Now().After(session.ExpiresAt) {
			http.Error(w, "Unauthorized: Session is invalid or expired", http.StatusUnauthorized)
			return
		}

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
