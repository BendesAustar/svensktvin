// Package auth provides session management, magic link, password, and rate limiting.
package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/svensktvin/svensktvin/internal/config"
	"github.com/svensktvin/svensktvin/internal/db"
)

// UserContextKey is the context key for storing user info.
type UserContextKey struct{}

// UserInfo stores authenticated user information.
type UserInfo struct {
	ID      int64
	Email   string
	Name    string
	IsAdmin bool
}

// contextWithUser adds user info to context.
func contextWithUser(ctx context.Context, info UserInfo) context.Context {
	return context.WithValue(ctx, UserContextKey{}, info)
}

// getUserFromContext retrieves user info from context.
func getUserFromContext(ctx context.Context) (UserInfo, bool) {
	u, ok := ctx.Value(UserContextKey{}).(UserInfo)
	return u, ok
}

// SessionManager handles session creation and validation.
type SessionManager struct {
	store        *db.Store
	sessionExpiry time.Duration
	cookieCfg    config.CookieConfig
}

// NewSessionManager creates a new session manager.
func NewSessionManager(store *db.Store, sessionExpiry time.Duration, cookieCfg config.CookieConfig) *SessionManager {
	return &SessionManager{
		store:       store,
		sessionExpiry: sessionExpiry,
		cookieCfg:    cookieCfg,
	}
}

// CreateSession creates a session for a user.
func (sm *SessionManager) CreateSession(ctx context.Context, userID int64) (string, error) {
	// Create session record
	sessionID := RandomHex(32)
	expiresAt := time.Now().Add(sm.sessionExpiry)
	_, err := sm.store.Pool.Exec(ctx, `
		INSERT INTO sessions (id, user_id, expires_at)
		VALUES ($1, $2, $3)
	`, sessionID, userID, expiresAt)
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	// Update last login
	_, _ = sm.store.Pool.Exec(ctx, `
		UPDATE users SET last_login = now() WHERE id = $1
	`, userID)

	return sessionID, nil
}

// VerifySession verifies a session token and returns user info.
func (sm *SessionManager) VerifySession(ctx context.Context, token string) (*UserInfo, error) {
	var userID int64
	var email, name string
	var isAdmin bool
	err := sm.store.Pool.QueryRow(ctx, `
		SELECT u.id, u.email, u.name, u.is_admin
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.id = $1 AND s.expires_at > now() AND u.active = true
	`, token).Scan(&userID, &email, &name, &isAdmin)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("invalid or expired session")
		}
		return nil, fmt.Errorf("verify session: %w", err)
	}

	return &UserInfo{
		ID:      userID,
		Email:   email,
		Name:    name,
		IsAdmin: isAdmin,
	}, nil
}

// DeleteSession invalidates a session.
func (sm *SessionManager) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := sm.store.Pool.Exec(ctx, `
		DELETE FROM sessions WHERE id = $1
	`, sessionID)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// RequireAuth is a middleware that verifies session authentication.
func (sm *SessionManager) RequireAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"unauthorized","message":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		token := authHeader
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		userInfo, err := sm.VerifySession(r.Context(), token)
		if err != nil {
			slog.Warn("auth: session verification failed", "err", err)
			http.Error(w, `{"error":"unauthorized","message":"invalid or expired session"}`, http.StatusUnauthorized)
			return
		}

		ctx := contextWithUser(r.Context(), *userInfo)
		h.ServeHTTP(w, r.WithContext(ctx))
	}
}

// RequireAdmin is a middleware that verifies session authentication AND admin status.
// Wraps RequireAuth: first checks session validity, then checks is_admin flag.
func (sm *SessionManager) RequireAdmin(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userInfo := sm.SessionFromRequest(r)
		if userInfo == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}
		if !userInfo.IsAdmin {
			http.Error(w, "Åtkomst nekad", http.StatusForbidden)
			return
		}
		ctx := contextWithUser(r.Context(), *userInfo)
		h.ServeHTTP(w, r.WithContext(ctx))
	}
}

// SessionFromRequest extracts session from cookie (for template-based auth).
func (sm *SessionManager) SessionFromRequest(r *http.Request) *UserInfo {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return nil
	}

	userInfo, err := sm.VerifySession(r.Context(), cookie.Value)
	if err != nil {
		return nil
	}
	return userInfo
}

// SetSessionCookie sets the session cookie in the response.
func (sm *SessionManager) SetSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   sm.cookieCfg.Secure,
		SameSite: sameSite(sm.cookieCfg.SameSite),
		Expires:  time.Now().Add(sm.sessionExpiry),
	})
}

// ClearSessionCookie clears the session cookie.
func (sm *SessionManager) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   sm.cookieCfg.Secure,
		SameSite: sameSite(sm.cookieCfg.SameSite),
		Expires:  time.Now().Add(-24 * time.Hour), // Expired
	})
}

// sameSite converts a string to http.SameSite mode.
func sameSite(mode string) http.SameSite {
	switch mode {
	case "Strict":
		return http.SameSiteStrictMode
	case "Lax":
		return http.SameSiteLaxMode
	case "None":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
