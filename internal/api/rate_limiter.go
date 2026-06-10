package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/httprate"
)

// RateLimitAuth is a rate limiter for authentication endpoints to prevent brute-force attacks.
// Limits to 5 requests per minute by IP address + URL path.
var RateLimitAuth func(http.Handler) http.Handler

// RateLimitGeneral is a general rate limiter for protected routes as a safety net.
// Limits to 100 requests per minute by IP address only.
var RateLimitGeneral func(http.Handler) http.Handler

func init() {
	// Strict rate limiting for auth endpoints: 5 requests per minute by IP + endpoint path.
	RateLimitAuth = httprate.Limit(
		5,
		time.Minute,
		httprate.WithKeyFuncs(httprate.KeyByIP, httprate.KeyByEndpoint),
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprintf(w, `{"error": "Rate limited: too many authentication attempts. Please try again later."}`)
		}),
	)

	// General rate limiting for protected routes: 100 requests per minute by IP only.
	RateLimitGeneral = httprate.Limit(
		100,
		time.Minute,
		httprate.WithKeyFuncs(httprate.KeyByIP),
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprintf(w, `{"error": "Rate limited: too many requests. Please try again later."}`)
		}),
	)

	fmt.Println("[CONFIG] Rate limiting initialized: Auth=5/min, General=100/min by IP")
}
