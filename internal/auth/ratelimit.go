package auth

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RateLimiter is a simple in-memory sliding window rate limiter.
type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	hard     int
	window   time.Duration
}

// NewRateLimiter creates a new rate limiter.
// hard is the max requests per window.
func NewRateLimiter(hard int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		hard:     hard,
		window:   window,
	}
}

// Allow checks if a request from key is allowed. Returns (allowed, retryAfter).
func (rl *RateLimiter) Allow(key string) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// Remove expired entries
	timestamps := rl.requests[key]
	var valid []time.Time
	for _, t := range timestamps {
		if t.After(windowStart) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.hard {
		rl.requests[key] = valid
		// Retry after the oldest request in the window expires
		retryAfter := rl.window - now.Sub(valid[0])
		if retryAfter < 0 {
			retryAfter = 1 * time.Second
		}
		return false, retryAfter
	}

	rl.requests[key] = append(valid, now)
	return true, 0
}

// ExtractIP extracts the client IP from the request.
func ExtractIP(r *http.Request) string {
	clientIP := r.RemoteAddr
	if idx := strings.LastIndex(clientIP, ":"); idx != -1 {
		clientIP = clientIP[:idx]
	}
	return clientIP
}

// RateLimitMiddleware wraps a handler with rate limiting.
func RateLimitMiddleware(limiter *RateLimiter, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientIP := ExtractIP(r)

		allowed, retryAfter := limiter.Allow(clientIP)
		if !allowed {
			w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
			http.Error(w, `{"error":"rate_limited","message":"för många inloggningsförsök. Försök igen senare."}`, http.StatusTooManyRequests)
			return
		}

		h.ServeHTTP(w, r)
	}
}
