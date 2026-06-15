// Package pages provides HTTP handlers for page rendering (auth, vineyard, etc.).
package pages

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/svensktvin/svensktvin/internal/auth"
	"github.com/svensktvin/svensktvin/internal/db"
)

// VineyardHandler holds dependencies for vineyard page handlers.
type VineyardHandler struct {
	store      *db.Store
	sessionMgr *auth.SessionManager
}

// NewVineyardHandler creates a new vineyard handler.
func NewVineyardHandler(store *db.Store, sessionMgr *auth.SessionManager) *VineyardHandler {
	return &VineyardHandler{
		store:      store,
		sessionMgr: sessionMgr,
	}
}

// HandleLandingGET renders the vineyard landing page.
func (h *VineyardHandler) HandleLandingGET(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if user is already authenticated
		user := h.sessionMgr.SessionFromRequest(r)
		if user == nil {
			// Unauthenticated: show login CTAs
			data := map[string]any{
				"Title": "Välkommen — Svenskt Vin",
			}
			renderTemplate(w, tmpl, "vineyard/index.html", data)
			return
		}

		// Get user's vineyards
		vineyards, err := h.store.ListVineyards(r.Context(), user.ID)
		if err != nil {
			slog.Error("landing: list vineyards", "err", err)
			data := map[string]any{
				"Error": "Kunde inte läsa dina vingårdar. Försök igen senare.",
				"User":  user,
				"Title": "Vingårdar — Svenskt Vin",
			}
			renderTemplate(w, tmpl, "vineyard/index.html", data)
			return
		}

		if len(vineyards) == 0 {
			// No vineyards yet — show message
			data := map[string]any{
				"User":      user,
				"NoVineyards": true,
				"Title":     "Vingårdar — Svenskt Vin",
			}
			renderTemplate(w, tmpl, "vineyard/index.html", data)
			return
		}

		if len(vineyards) == 1 {
			// Exactly one vineyard — redirect directly
			http.Redirect(w, r, fmt.Sprintf("/vineyard/%d", vineyards[0].ID), http.StatusSeeOther)
			return
		}

		// Multiple vineyards — show list
		data := map[string]any{
			"User":     user,
			"Vineyards": vineyards,
			"Title":    "Vingårdar — Svenskt Vin",
		}
		renderTemplate(w, tmpl, "vineyard/index.html", data)
	}
}

// HandleVineyardGET renders the vineyard dashboard (or delegates to block handlers).
func (h *VineyardHandler) HandleVineyardGET(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Check if this is a block route — delegate to BlockHandler
		if strings.HasPrefix(path, "/vineyard/") && strings.Contains(path, "/blocks/") {
			// Create a temporary BlockHandler and delegate
			blockHandler := NewBlockHandler(h.store, h.sessionMgr)
			blockHandler.routeBlockRequest(tmpl).ServeHTTP(w, r)
			return
		}

		// Auth check
		user := h.sessionMgr.SessionFromRequest(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Extract vineyard ID from path: /vineyard/{id}
		var vineyardID int64
		_, err := fmt.Sscanf(path, "/vineyard/%d", &vineyardID)
		if err != nil || vineyardID == 0 {
			http.Redirect(w, r, "/vineyard", http.StatusSeeOther)
			return
		}

		// Check membership
		role, err := h.store.GetVineyardRole(r.Context(), vineyardID, user.ID)
		if err != nil {
			slog.Error("vineyard: check membership", "err", err, "vineyard_id", vineyardID, "user_id", user.ID)
			http.Redirect(w, r, "/vineyard", http.StatusSeeOther)
			return
		}

		// Load vineyard details
		vineyard, err := h.store.GetVineyard(r.Context(), vineyardID)
		if err != nil {
			slog.Error("vineyard: get vineyard", "err", err, "vineyard_id", vineyardID)
			http.NotFound(w, r)
			return
		}

		// Load blocks with latest harvest
		blocks, err := h.store.ListBlocksWithHarvest(r.Context(), vineyardID)
		if err != nil {
			slog.Error("vineyard: list blocks", "err", err, "vineyard_id", vineyardID)
			blocks = nil
		}

		// Load benchmark teaser
		teaser, err := h.store.GetBenchmarkTeaser(r.Context(), vineyardID, user.ID)
		if err != nil {
			slog.Warn("vineyard: benchmark teaser", "err", err)
		}

		csrfToken := generateCSRFToken()
		setCSRFCookie(w, csrfToken)

		data := map[string]any{
			"User":            user,
			"Role":            role,
			"Vineyard":        *vineyard,
			"Blocks":          blocks,
			"BenchmarkTeaser": teaser,
			"CSRFToken":       csrfToken,
			"Title":           fmt.Sprintf("%s — Svenskt Vin", vineyard.Name),
			"IsHome":          true,
		}
		renderTemplate(w, tmpl, "vineyard/dashboard.html", data)
	}
}

// HandleBenchmarkGET renders the benchmark page.
func (h *VineyardHandler) HandleBenchmarkGET(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Auth check
		user := h.sessionMgr.SessionFromRequest(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Extract vineyard ID
		path := r.URL.Path
		var vineyardID int64
		_, err := fmt.Sscanf(path, "/vineyard/%d/benchmark", &vineyardID)
		if err != nil || vineyardID == 0 {
			http.Redirect(w, r, "/vineyard", http.StatusSeeOther)
			return
		}

		// Check membership
		_, err = h.store.GetVineyardRole(r.Context(), vineyardID, user.ID)
		if err != nil {
			http.Redirect(w, r, "/vineyard", http.StatusSeeOther)
			return
		}

		// Load vineyard details
		vineyard, err := h.store.GetVineyard(r.Context(), vineyardID)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		// Load benchmark data
		benchData := h.store.GetBenchmarkData(r.Context(), vineyardID)

		csrfToken := generateCSRFToken()
		setCSRFCookie(w, csrfToken)

		data := map[string]any{
			"User":            user,
			"Vineyard":        *vineyard,
			"UserYields":      benchData.UserYields,
			"RegionalBench":   benchData.RegionalBench,
			"Timeline":        benchData.Timeline,
			"CSRFToken":       csrfToken,
			"Title":           fmt.Sprintf("Jämförelse — %s — Svenskt Vin", vineyard.Name),
			"IsBenchmark":     true,
		}
		renderTemplate(w, tmpl, "vineyard/benchmark.html", data)
	}
}

// ParseInt64 safely parses a URL path segment as int64.
func ParseInt64(s string) (int64, bool) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, n > 0
}
