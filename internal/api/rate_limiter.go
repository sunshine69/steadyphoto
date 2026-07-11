package api

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/httprate"
	"github.com/jbrodriguez/mlog"
)

// RateLimitAuth is a rate limiter for authentication endpoints to prevent brute-force attacks.
// Limits to 5 requests per minute by IP address + URL path (configurable via RATE_LIMIT_AUTH_REQUESTS env var).
var RateLimitAuth func(http.Handler) http.Handler

// RateLimitGeneral is a general rate limiter for protected routes as a safety net.
// Limits to 100 requests per minute by IP address only (configurable via RATE_LIMIT_GENERAL_REQUESTS env var).
var RateLimitGeneral func(http.Handler) http.Handler

func init() {
	// Parse auth rate limit from environment variable, default to 5 req/min
	authRequests := 5
	if val := os.Getenv("RATE_LIMIT_AUTH_REQUESTS"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			authRequests = n
		} else {
			mlog.Warning("Invalid RATE_LIMIT_AUTH_REQUESTS value: %q — using default of 5\n", val)
		}
	}

	// Strict rate limiting for auth endpoints to prevent brute-force attacks.
	RateLimitAuth = httprate.Limit(
		int(authRequests),
		time.Minute,
		httprate.WithKeyFuncs(httprate.KeyByIP, httprate.KeyByEndpoint),
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprintf(w, `{"error": "Rate limited: too many authentication attempts. Please try again later."}`)
		}),
	)

	// Parse general rate limit from environment variable, default to 100 req/min
	generalRequests := 100
	if val := os.Getenv("RATE_LIMIT_GENERAL_REQUESTS"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			generalRequests = n
		} else {
			mlog.Warning("Invalid RATE_LIMIT_GENERAL_REQUESTS value: %q — using default of 100\n", val)
		}
	}

	// General rate limiting for protected routes as a safety net.
	RateLimitGeneral = httprate.Limit(
		generalRequests,
		time.Minute,
		httprate.WithKeyFuncs(httprate.KeyByIP),
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprintf(w, `{"error": "Rate limited: too many requests. Please try again later."}`)
		}),
	)

	fmt.Printf("[CONFIG] Rate limiting initialized: Auth=%d/min by IP+path, General=%d/min by IP\n", authRequests, generalRequests)
}
