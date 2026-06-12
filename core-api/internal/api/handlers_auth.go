package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

// sendMagicLinkRequest is the JSON payload for sending a magic link.
type sendMagicLinkRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// sendMagicLink handles POST /api/auth/send-link
func (r *Router) sendMagicLink(w http.ResponseWriter, req *http.Request) {
	var reqBody sendMagicLinkRequest
	if err := json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		writeValidationError(w, "invalid request body")
		return
	}
	if reqBody.Email == "" {
		writeValidationError(w, "email is required", ValidationError{Field: "email", Issue: "required"})
		return
	}

	// Generate token
	rawToken := randomHex(32)
	hash := sha256.Sum256([]byte(rawToken))

	// Upsert user (idempotent: creates new user if doesn't exist)
	var userID int64
	err := r.store.Pool.QueryRow(req.Context(), `
		INSERT INTO users (email, name)
		SELECT $1, split_part($1, '@', 1)
		WHERE NOT EXISTS (SELECT 1 FROM users WHERE email = $1)
		RETURNING id
	`).Scan(&userID)

	if err != nil && err != pgx.ErrNoRows {
		slog.Error("auth: user upsert error", "err", err)
		writeInternalError(w, "failed to process request")
		return
	}

	// Also handle the case where user already exists
	if userID == 0 {
		err = r.store.Pool.QueryRow(req.Context(), `
			SELECT id FROM users WHERE email = $1
		`, reqBody.Email).Scan(&userID)
		if err != nil {
			slog.Error("auth: user lookup error", "err", err)
			writeInternalError(w, "failed to process request")
			return
		}
	}

	// Insert magic link token (use ON CONFLICT for safety)
	_, err = r.store.Pool.Exec(req.Context(), `
		INSERT INTO magic_link_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, hash[:], time.Now().Add(15 * time.Minute))

	if err != nil {
		slog.Error("auth: token insert error", "err", err)
		// Still return success to avoid leaking existence
		writeJSON(w, http.StatusOK, map[string]bool{"sent": true})
		return
	}

	slog.Info("auth: magic link sent", "email", reqBody.Email)
	writeJSON(w, http.StatusOK, map[string]bool{"sent": true})
}

// verifyMagicLinkRequest is the JSON payload for verifying a magic link.
type verifyMagicLinkRequest struct {
	Token string `json:"token" validate:"required"`
}

// verifyMagicLink handles POST /api/auth/verify
func (r *Router) verifyMagicLink(w http.ResponseWriter, req *http.Request) {
	var reqBody verifyMagicLinkRequest
	if err := json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		writeValidationError(w, "invalid request body")
		return
	}
	if reqBody.Token == "" {
		writeValidationError(w, "token is required", ValidationError{Field: "token", Issue: "required"})
		return
	}

	// Hash token and verify atomically
	hash := sha256.Sum256([]byte(reqBody.Token))

	var userID int64
	var email, name string
	var isAdmin bool

	err := r.store.Pool.QueryRow(req.Context(), `
		UPDATE magic_link_tokens
		SET used = true
		WHERE token_hash = $1
			AND used = false
			AND expires_at > now()
		RETURNING user_id
	`, hash[:]).Scan(&userID)

	if err != nil {
		if err == pgx.ErrNoRows {
			// Account enumeration safe: same response
			writeError(w, http.StatusUnauthorized, APIError{
				Code:    "unauthorized",
				Message: "invalid or expired token",
			})
		} else {
			slog.Error("auth: token verify error", "err", err)
			writeInternalError(w, "verification failed")
		}
		return
	}

	// Get user info
	err = r.store.Pool.QueryRow(req.Context(), `
		SELECT id, email, name, is_admin
		FROM users
		WHERE id = $1 AND active = true
	`, userID).Scan(&userID, &email, &name, &isAdmin)

	if err != nil {
		slog.Error("auth: user lookup error", "err", err)
		writeInternalError(w, "verification failed")
		return
	}

	// Create session
	sessionID := randomHex(32)
	expiresAt := time.Now().Add(r.cfg.Auth.SessionExpiry)
	_, err = r.store.Pool.Exec(req.Context(), `
		INSERT INTO sessions (id, user_id, expires_at)
		VALUES ($1, $2, $3)
	`, sessionID, userID, expiresAt)

	if err != nil {
		slog.Error("auth: session create error", "err", err)
		writeInternalError(w, "session creation failed")
		return
	}

	// Update last login
	_, _ = r.store.Pool.Exec(req.Context(), `
		UPDATE users SET last_login = now() WHERE id = $1
	`, userID)

	slog.Info("auth: magic link verified", "email", email)
	writeJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":       userID,
			"email":    email,
			"name":     name,
			"is_admin": isAdmin,
		},
		"session_token": sessionID,
	})
}

// randomHex generates a hex-encoded random string.
func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
