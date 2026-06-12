// Package api provides HTTP handlers and routing for the Svenskt Vin core API.
package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/svensktvin/core-api/internal/config"
	"github.com/svensktvin/core-api/internal/db"
)

// Router configures all API routes.
type Router struct {
	store     *db.Store
	cfg       *config.Config
	authLimit *rateLimiter
}

// NewRouter creates a new route handler.
func NewRouter(store *db.Store, cfg *config.Config) http.Handler {
	r := &Router{
		store:     store,
		cfg:       cfg,
		authLimit: newRateLimiter(cfg.RateLimit.AuthRequests, cfg.RateLimit.AuthWindow),
	}

	mux := http.NewServeMux()

	// Health check (public, no auth)
	mux.HandleFunc("GET /health", r.health)

	// Auth routes (public, rate-limited)
	mux.HandleFunc("POST /api/auth/send-link", r.rateLimit(r.sendMagicLink))
	mux.HandleFunc("POST /api/auth/verify", r.rateLimit(r.verifyMagicLink))

	// Variety routes
	mux.HandleFunc("GET /api/varieties", r.requireAuth(r.listVarieties))
	mux.HandleFunc("POST /api/varieties/submit", r.requireAuth(r.submitVariety))

	// Block search (public-ish, but requires auth for variety resolution)
	mux.HandleFunc("POST /api/blocks/search-varieties", r.requireAuth(r.searchVarieties))

	// Vineyard routes (require auth, all dispatched by path structure)
	mux.HandleFunc("/api/vineyards", r.vineyardRouter)

	return mux
}

// vineyardRouter dispatches vineyard-scoped requests based on sub-path.
func (r *Router) vineyardRouter(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path

	// GET /api/vineyards — list for current user
	if req.Method == http.MethodGet && path == "/api/vineyards" {
		r.listVineyards(w, req)
		return
	}

	// POST /api/vineyards — create vineyard
	if req.Method == http.MethodPost && path == "/api/vineyards" {
		r.createVineyard(w, req)
		return
	}

	// All other paths must be /api/vineyards/{id}/...
	if !strings.HasPrefix(path, "/api/vineyards/") {
		writeNotFound(w, "not found")
		return
	}

	vineyardID, rest, err := extractID(path)
	if err != nil {
		writeNotFound(w, "vineyard not found")
		return
	}

	// Attach vineyard ID to context
	ctx := req.Context()
	ctx = context.WithValue(ctx, vineyardKey, vineyardID)
	req = req.WithContext(ctx)

	switch {
	case req.Method == http.MethodGet && rest == "":
		// GET /api/vineyards/:id — get vineyard details
		r.getVineyard(w, req)
	case req.Method == http.MethodPut && rest == "":
		// PUT /api/vineyards/:id — update vineyard
		r.updateVineyard(w, req)
	case req.Method == http.MethodDelete && rest == "":
		// DELETE /api/vineyards/:id — soft-delete vineyard
		r.deleteVineyard(w, req)

	case strings.HasPrefix(rest, "/blocks"):
		r.handleBlocks(w, req, vineyardID)

	case strings.HasPrefix(rest, "/harvests"):
		r.handleHarvests(w, req, vineyardID)

	case strings.HasPrefix(rest, "/benchmarks"):
		r.getBenchmarks(w, req)

	case strings.HasPrefix(rest, "/members"):
		r.handleMembers(w, req, vineyardID)

	default:
		writeNotFound(w, "not found")
	}
}

// handleBlocks dispatches block-related requests.
func (r *Router) handleBlocks(w http.ResponseWriter, req *http.Request, vineyardID int64) {
	rest := strings.TrimPrefix(req.URL.Path, fmt.Sprintf("/api/vineyards/%d", vineyardID))

	// GET /api/vineyards/:id/blocks — list blocks
	if req.Method == http.MethodGet && rest == "/blocks" {
		r.listBlocks(w, req)
		return
	}

	// POST /api/vineyards/:id/blocks — create block
	if req.Method == http.MethodPost && rest == "/blocks" {
		r.createBlock(w, req)
		return
	}

	// /api/vineyards/:id/blocks/{blockId}/...
	if !strings.HasPrefix(rest, "/blocks/") {
		writeNotFound(w, "block not found")
		return
	}

	blockID, _, err := extractID(rest)
	if err != nil {
		writeNotFound(w, "block not found")
		return
	}

	subPath := strings.TrimPrefix(rest, fmt.Sprintf("/blocks/%d", blockID))
	switch {
	case req.Method == http.MethodGet && subPath == "":
		r.getBlock(w, req)
	case req.Method == http.MethodPut && subPath == "":
		r.updateBlock(w, req)
	case req.Method == http.MethodDelete && subPath == "":
		r.deleteBlock(w, req)
	default:
		writeNotFound(w, "not found")
	}
}

// handleHarvests dispatches harvest-related requests.
func (r *Router) handleHarvests(w http.ResponseWriter, req *http.Request, vineyardID int64) {
	rest := strings.TrimPrefix(req.URL.Path, fmt.Sprintf("/api/vineyards/%d", vineyardID))

	// GET /api/vineyards/:id/harvests — list harvests
	if req.Method == http.MethodGet && rest == "/harvests" {
		r.listHarvests(w, req)
		return
	}

	// POST /api/vineyards/:id/harvests — create harvest
	if req.Method == http.MethodPost && rest == "/harvests" {
		r.createHarvest(w, req)
		return
	}

	// /api/vineyards/:id/harvests/{recordId}/...
	if !strings.HasPrefix(rest, "/harvests/") {
		writeNotFound(w, "harvest record not found")
		return
	}

	recordID, _, err := extractID(rest)
	if err != nil {
		writeNotFound(w, "harvest record not found")
		return
	}

	subPath := strings.TrimPrefix(rest, fmt.Sprintf("/harvests/%d", recordID))
	switch {
	case req.Method == http.MethodGet && subPath == "":
		r.getHarvest(w, req)
	case req.Method == http.MethodPut && subPath == "":
		r.updateHarvest(w, req)
	case req.Method == http.MethodDelete && subPath == "":
		r.deleteHarvest(w, req)
	default:
		writeNotFound(w, "not found")
	}
}

// handleMembers dispatches member-related requests.
func (r *Router) handleMembers(w http.ResponseWriter, req *http.Request, vineyardID int64) {
	rest := strings.TrimPrefix(req.URL.Path, fmt.Sprintf("/api/vineyards/%d", vineyardID))

	// GET /api/vineyards/:id/members — list members
	if req.Method == http.MethodGet && rest == "/members" {
		r.listMembers(w, req)
		return
	}

	// POST /api/vineyards/:id/members — add member
	if req.Method == http.MethodPost && rest == "/members" {
		r.addMember(w, req)
		return
	}

	// /api/vineyards/:id/members/{userId}/...
	if !strings.HasPrefix(rest, "/members/") {
		writeNotFound(w, "member not found")
		return
	}

	userId, _, err := extractID(rest)
	if err != nil {
		writeNotFound(w, "member not found")
		return
	}

	subPath := strings.TrimPrefix(rest, fmt.Sprintf("/members/%d", userId))
	switch {
	case req.Method == http.MethodPut && subPath == "":
		r.updateMember(w, req)
	case req.Method == http.MethodDelete && subPath == "":
		r.removeMember(w, req)
	default:
		writeNotFound(w, "not found")
	}
}

// contextKey is a custom type for context values to avoid collisions.
type contextKey string

const (
	userKey    contextKey = "user"
	vineyardKey contextKey = "vineyard_id"
)

type userInfo struct {
	ID      int64
	Email   string
	Name    string
	IsAdmin bool
}

func contextWithUser(ctx context.Context, id int64, email, name string, isAdmin bool) context.Context {
	return context.WithValue(ctx, userKey, userInfo{ID: id, Email: email, Name: name, IsAdmin: isAdmin})
}

func getUserFromContext(ctx context.Context) (userInfo, bool) {
	u, ok := ctx.Value(userKey).(userInfo)
	return u, ok
}

// requireAuth wraps a handler to verify session authentication.
func (r *Router) requireAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		authHeader := req.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, APIError{
				Code:    "unauthorized",
				Message: "missing authorization header",
			})
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			token = authHeader
		}

		// Verify session token against DB
		var userID int64
		var email, name string
		var isAdmin bool
		err := r.store.Pool.QueryRow(req.Context(), `
			SELECT u.id, u.email, u.name, u.is_admin
			FROM sessions s
			JOIN users u ON u.id = s.user_id
			WHERE s.id = $1 AND s.expires_at > now() AND u.active = true
		`, token).Scan(&userID, &email, &name, &isAdmin)

		if err != nil {
			if err == pgx.ErrNoRows {
				writeError(w, http.StatusUnauthorized, APIError{
					Code:    "unauthorized",
					Message: "invalid or expired session",
				})
			} else {
				slog.Error("auth: session verify error", "err", err)
				writeInternalError(w, "session verification failed")
			}
			return
		}

		ctx := contextWithUser(req.Context(), userID, email, name, isAdmin)
		h.ServeHTTP(w, req.WithContext(ctx))
	}
}

// rateLimit wraps a handler with rate limiting for auth endpoints.
func (r *Router) rateLimit(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		clientIP := req.RemoteAddr
		if idx := strings.LastIndex(clientIP, ":"); idx != -1 {
			clientIP = clientIP[:idx]
		}

		allowed, retryAfter := r.authLimit.allow(clientIP)
		if !allowed {
			w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
			writeError(w, http.StatusTooManyRequests, APIError{
				Code:    "rate_limited",
				Message: "too many login attempts. Please try again later.",
			})
			return
		}

		h.ServeHTTP(w, req)
	}
}

// health returns a simple health check.
func (r *Router) health(w http.ResponseWriter, req *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":    "ok",
		"version":   "1.0.0",
		"database":  "connected",
	})
}

// extractID parses an ID from a URL path like /api/vineyards/123/...
func extractID(path string) (int64, string, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		return 0, "", fmt.Errorf("invalid path")
	}
	id := 0
	fmt.Sscanf(parts[1], "%d", &id)
	if id == 0 {
		return 0, "", fmt.Errorf("invalid ID")
	}
	rest := ""
	if len(parts) > 2 {
		rest = strings.Join(parts[2:], "/")
	}
	return int64(id), rest, nil
}
