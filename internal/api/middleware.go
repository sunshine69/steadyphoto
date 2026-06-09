package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"steadyphoto/internal/domain"

	"github.com/google/uuid"
)

// contextKey is a private type to avoid collisions with other packages in the context
type contextKey string

const (
	// UserIDContextKey is the key used to store and retrieve UserID from the request context
	UserIDContextKey contextKey = "user_id"
)

var MaxUploadSizeBytes int64 = 512 << 20 // Default: 512MB (supports large single-file uploads)

// AllowedCORSOrigins defines which origins are permitted to make cross-origin requests.
// This can be overridden via the CORS_ALLOWED_ORIGINS environment variable (comma-separated list).
var AllowedCORSOrigins []string = []string{
	"http://localhost:4200", // Angular dev server
	"http://192.168.20.23:4200", // Your specific IP for Angular dev server
}

func init() {
	if val := os.Getenv("MAX_UPLOAD_SIZE"); val != "" {
		if mb, err := strconv.ParseInt(val, 10, 64); err == nil && mb > 0 {
			MaxUploadSizeBytes = mb
			fmt.Printf("[CONFIG] Max upload size set to %d bytes\n", mb)
		} else {
			fmt.Printf("[WARN] Invalid MAX_UPLOAD_SIZE value '%s', using default (10MB)\n", val)
		}
	}

	// Allow overriding CORS origins via environment variable
	if corsOrigins := os.Getenv("CORS_ALLOWED_ORIGINS"); corsOrigins != "" {
		AllowedCORSOrigins = strings.Split(corsOrigins, ",")
		for i, origin := range AllowedCORSOrigins {
			AllowedCORSOrigins[i] = strings.TrimSpace(origin)
		}
		fmt.Printf("[CONFIG] CORS allowed origins: %v\n", AllowedCORSOrigins)
	}
}

// isValidCORSOrigin checks if the given origin is in the list of allowed CORS origins.
func isValidCORSOrigin(origin string) bool {
	for _, allowed := range AllowedCORSOrigins {
		if strings.EqualFold(origin, allowed) {
			return true
		}
	}
	return false
}

// LimitBodySizeMiddleware checks the Content-Length header before processing.
func LimitBodySizeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if request body exceeds limit (works for multipart forms sent with Content-Length)
		if r.ContentLength > MaxUploadSizeBytes {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			w.Write([]byte(`{"error": "File too large. Maximum upload size is configured."}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}



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

// AdminMiddleware checks if the authenticated user has admin role
func (s *Server) AdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserIDFromContext(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Fetch user to check role
		user, err := s.userRepo.GetByID(r.Context(), userID)
		if err != nil || user == nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		if user.Role != domain.UserRoleAdmin {
			http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
