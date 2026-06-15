// Package api provides HTTP handlers for JSON API endpoints (varieties, geo, account, etc.).
package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/svensktvin/svensktvin/internal/auth"
	"github.com/svensktvin/svensktvin/internal/db"
)

// AccountHandler holds dependencies for account API handlers.
type AccountHandler struct {
	store      *db.Store
	sessionMgr *auth.SessionManager
}

// NewAccountHandler creates a new account handler.
func NewAccountHandler(store *db.Store, sessionMgr *auth.SessionManager) *AccountHandler {
	return &AccountHandler{
		store:      store,
		sessionMgr: sessionMgr,
	}
}

// HandleAccountExportGET exports user data as JSON download.
func (h *AccountHandler) HandleAccountExportGET(w http.ResponseWriter, r *http.Request) {
	// Auth check
	user := h.sessionMgr.SessionFromRequest(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Get user data
	userData, err := h.store.GetUserWithDetails(r.Context(), user.ID)
	if err != nil {
		slog.Error("account: export", "err", err, "user_id", user.ID)
		http.Error(w, "Ett internt fel uppstod.", http.StatusInternalServerError)
		return
	}

	// Set JSON content type and attachment header
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"svenskt-vin-export-%s.json\"", time.Now().Format("2006-01-02")))

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(userData); err != nil {
		slog.Error("account: export encode", "err", err)
	}
}

// HandleAccountDeletePOST soft-deletes a user account.
func (h *AccountHandler) HandleAccountDeletePOST(w http.ResponseWriter, r *http.Request) {
	// Auth check
	user := h.sessionMgr.SessionFromRequest(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Validate confirm flag
	if r.FormValue("confirm") != "true" {
		http.Error(w, "Bekräfta radering med checkrutan.", http.StatusBadRequest)
		return
	}

	// Soft delete user
	err := h.store.SoftDeleteUser(r.Context(), user.ID)
	if err != nil {
		slog.Error("account: delete", "err", err, "user_id", user.ID)
		http.Error(w, "Ett internt fel uppstod.", http.StatusInternalServerError)
		return
	}

	// Redirect to login with flash (using cookie-based flash)
	http.SetCookie(w, &http.Cookie{
		Name:     "flash_message",
		Value:    "Ditt konto har raderats.",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
