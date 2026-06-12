package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// APIError represents a structured API error response.
type APIError struct {
	Code    string            `json:"error"`
	Message string            `json:"message"`
	Details []ValidationError `json:"details,omitempty"`
}

type ValidationError struct {
	Field string `json:"field"`
	Issue string `json:"issue"`
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError writes a structured API error response.
func writeError(w http.ResponseWriter, status int, err APIError) {
	writeJSON(w, status, err)
}

// writeValidationError writes a 400 validation error.
func writeValidationError(w http.ResponseWriter, message string, details ...ValidationError) {
	writeError(w, http.StatusBadRequest, APIError{
		Code:    "validation_error",
		Message: message,
		Details: details,
	})
}

// writeConflict writes a 409 conflict error.
func writeConflict(w http.ResponseWriter, message string) {
	writeError(w, http.StatusConflict, APIError{
		Code:    "conflict",
		Message: message,
	})
}

// writeNotFound writes a 404 not found error.
func writeNotFound(w http.ResponseWriter, message string) {
	writeError(w, http.StatusNotFound, APIError{
		Code:    "not_found",
		Message: message,
	})
}

// writeForbidden writes a 403 forbidden error.
func writeForbidden(w http.ResponseWriter, message string) {
	writeError(w, http.StatusForbidden, APIError{
		Code:    "forbidden",
		Message: message,
	})
}

// writeInternalError writes a 500 internal error.
func writeInternalError(w http.ResponseWriter, msg string) {
	writeError(w, http.StatusInternalServerError, APIError{
		Code:    "internal_error",
		Message: msg,
	})
}

// rateLimiter is a simple in-memory sliding window rate limiter.
type rateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	hard     int
	window   time.Duration
}

// newRateLimiter creates a new rate limiter.
// hard is the max requests per window.
func newRateLimiter(hard int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		requests: make(map[string][]time.Time),
		hard:     hard,
		window:   window,
	}
}

// allow checks if a request from key is allowed. Returns (allowed, retryAfter).
func (rl *rateLimiter) allow(key string) (bool, time.Duration) {
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
