// Package pages provides HTTP handlers for page rendering (auth, vineyard, etc.).
package pages

import (
	"crypto/sha256"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/svensktvin/svensktvin/internal/auth"
	"github.com/svensktvin/svensktvin/internal/config"
	"github.com/svensktvin/svensktvin/internal/db"
	"github.com/svensktvin/svensktvin/internal/email"
)

// AdminHandler holds dependencies for admin page handlers.
type AdminHandler struct {
	store       *db.Store
	sessionMgr  *auth.SessionManager
	cookieCfg   config.CookieConfig
	emailSender *email.Sender
	appHost     string
	tmpl        *template.Template
}

// NewAdminHandler creates a new admin handler.
func NewAdminHandler(store *db.Store, sessionMgr *auth.SessionManager,
	cookieCfg config.CookieConfig, emailSender *email.Sender,
	appHost string, tmpl *template.Template) *AdminHandler {
	return &AdminHandler{
		store:       store,
		sessionMgr:  sessionMgr,
		cookieCfg:   cookieCfg,
		emailSender: emailSender,
		appHost:     appHost,
		tmpl:        tmpl,
	}
}

// HandleDashboardGET renders the admin dashboard overview.
func (h *AdminHandler) HandleDashboardGET() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := h.sessionMgr.SessionFromRequest(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Count users
		var userCount, adminCount, activeCount int
		err := h.store.Pool.QueryRow(r.Context(), `
			SELECT COUNT(*),
			       COUNT(*) FILTER (WHERE is_admin),
			       COUNT(*) FILTER (WHERE active)
			FROM users
		`).Scan(&userCount, &adminCount, &activeCount)
		if err != nil {
			slog.Error("admin: count users", "err", err)
		}

		// Recent logins
		var recentLogins []struct {
			Email     string
			Name      string
			LastLogin *time.Time
			IsAdmin   bool
		}
		rows, err := h.store.Pool.Query(r.Context(), `
			SELECT email, name, last_login, is_admin
			FROM users
			ORDER BY last_login DESC NULLS LAST
			LIMIT 5
		`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var item struct {
					Email     string
					Name      string
					LastLogin *time.Time
					IsAdmin   bool
				}
				if err := rows.Scan(&item.Email, &item.Name, &item.LastLogin, &item.IsAdmin); err != nil {
					continue
				}
				recentLogins = append(recentLogins, item)
			}
		}

		// Recent admin actions
		var recentActions []struct {
			Action string
			Email  string
			Time   time.Time
		}
		rows2, err := h.store.Pool.Query(r.Context(), `
			SELECT aa.action, u.email, aa.created_at
			FROM admin_actions aa
			JOIN users u ON u.id = aa.admin_user_id
			ORDER BY aa.created_at DESC
			LIMIT 10
		`)
		if err == nil {
			defer rows2.Close()
			for rows2.Next() {
				var a struct {
					Action string
					Email  string
					Time   time.Time
				}
				if err := rows2.Scan(&a.Action, &a.Email, &a.Time); err != nil {
					continue
				}
				recentActions = append(recentActions, a)
			}
		}

		csrfToken := generateCSRFToken()
		setCSRFCookie(w, csrfToken, h.cookieCfg)

		data := map[string]any{
			"User":             user,
			"Title":            "Adminpanel — Svenskt Vin",
			"CSRFToken":        csrfToken,
			"UserCount":        userCount,
			"AdminCount":       adminCount,
			"ActiveCount":      activeCount,
			"RecentLogins":     recentLogins,
			"RecentActions":    recentActions,
			"IsAdminDashboard": true,
		}
		renderTemplate(w, h.tmpl, "admin/dashboard.html", data)
	}
}

// HandleUsersGET renders the user list page.
func (h *AdminHandler) HandleUsersGET() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := h.sessionMgr.SessionFromRequest(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		users, err := h.store.ListAllUsers(r.Context())
		if err != nil {
			slog.Error("admin: list users", "err", err)
		}

		csrfToken := generateCSRFToken()
		setCSRFCookie(w, csrfToken, h.cookieCfg)

		data := map[string]any{
			"User":         user,
			"Title":        "Användare — Svenskt Vin",
			"CSRFToken":    csrfToken,
			"Users":        users,
			"IsAdminUsers": true,
		}
		renderTemplate(w, h.tmpl, "admin/users.html", data)
	}
}

// HandleUserDetailGET renders the user edit page.
func (h *AdminHandler) HandleUserDetailGET() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := h.sessionMgr.SessionFromRequest(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Extract userID from path: /admin/users/{id}
		path := r.URL.Path
		idStr := path[len("/admin/users/"):]
		userID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || userID == 0 {
			http.NotFound(w, r)
			return
		}

		// Get user
		targetUser, err := h.store.GetUserByID(r.Context(), userID)
		if err != nil {
			slog.Error("admin: get user detail", "err", err)
			http.NotFound(w, r)
			return
		}

		csrfToken := generateCSRFToken()
		setCSRFCookie(w, csrfToken, h.cookieCfg)

		data := map[string]any{
			"User":         user,
			"TargetUser":   *targetUser,
			"Title":        fmt.Sprintf("%s — Användarinställningar", targetUser.Name),
			"CSRFToken":    csrfToken,
			"IsAdminUsers": true,
		}
		renderTemplate(w, h.tmpl, "admin/user_detail.html", data)
	}
}

// HandleUserDetailPOST handles user edit actions (deactivate, reset password, toggle admin).
func (h *AdminHandler) HandleUserDetailPOST() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract userID from path: /admin/users/{id}
		path := r.URL.Path
		idStr := path[len("/admin/users/"):]
		userID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || userID == 0 {
			http.NotFound(w, r)
			return
		}

		// CSRF validation
		if !validateCSRFToken(r) {
			http.Error(w, "Ogiltig begäran.", http.StatusBadRequest)
			return
		}

		action := r.FormValue("action")
		user := h.sessionMgr.SessionFromRequest(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		switch action {
		case "deactivate":
			err = h.store.UpdateUserActive(r.Context(), userID, false)
			if err != nil {
				slog.Error("admin: deactivate user", "err", err)
			}
			http.Redirect(w, r, "/admin/users", http.StatusSeeOther)

		case "activate":
			err = h.store.UpdateUserActive(r.Context(), userID, true)
			if err != nil {
				slog.Error("admin: activate user", "err", err)
			}
			http.Redirect(w, r, "/admin/users", http.StatusSeeOther)

		case "reset_password":
			// Generate a magic link token for password reset
			rawToken := auth.RandomHex(32)
			hash := sha256.Sum256([]byte(rawToken))
			_, err = h.store.Pool.Exec(r.Context(), `
				INSERT INTO magic_link_tokens (user_id, token_hash, expires_at)
				VALUES ($1, $2, $3)
			`, userID, hash[:], time.Now().Add(1*time.Hour))
			if err != nil {
				slog.Error("admin: reset password", "err", err)
			}
			// Show magic link in flash
			inviteURL := fmt.Sprintf("%s/auth/set-password?token=%s", h.appHost, rawToken)
			data := map[string]any{
				"User":         user,
				"Flash":        fmt.Sprintf("Återställningslänk: %s", inviteURL),
				"CSRFToken":    generateCSRFToken(),
				"TargetUser":   user,
				"Title":        "Återställ lösenord — Svenskt Vin",
				"IsAdminUsers": true,
			}
			setCSRFCookie(w, data["CSRFToken"].(string), h.cookieCfg)
			renderTemplate(w, h.tmpl, "admin/user_detail.html", data)

		case "toggle_admin":
			var currentAdmin bool
			err := h.store.Pool.QueryRow(r.Context(), `
				SELECT is_admin FROM users WHERE id = $1
			`, userID).Scan(&currentAdmin)
			if err == nil {
				_, err = h.store.Pool.Exec(r.Context(), `
					UPDATE users SET is_admin = $1 WHERE id = $2
				`, !currentAdmin, userID)
			}
			if err != nil {
				slog.Error("admin: toggle admin", "err", err)
			}
			http.Redirect(w, r, "/admin/users", http.StatusSeeOther)

		default:
			http.NotFound(w, r)
		}
	}
}

// HandleInviteGeneratePOST generates an invite for a new user.
func (h *AdminHandler) HandleInviteGeneratePOST() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := h.sessionMgr.SessionFromRequest(r)
		if user == nil {
			http.Error(w, "Obehörig", http.StatusUnauthorized)
			return
		}

		// CSRF validation
		if !validateCSRFToken(r) {
			http.Error(w, "Ogiltig begäran.", http.StatusBadRequest)
			return
		}

		emailAddr := sanitizeInput(r.FormValue("email"))
		role := r.FormValue("role") // "owner" or "editor"

		// Validate role
		if role != "owner" && role != "editor" {
			w.Header().Set("HX-Trigger", `{"showInviteError":"Ogiltig roll. Använd 'owner' eller 'editor'."}`)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if emailAddr == "" {
			w.Header().Set("HX-Trigger", `{"showInviteError":"E-postadress krävs."}`)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Create pending invite
		token := auth.RandomHex(32)
		expiresAt := time.Now().Add(7 * 24 * time.Hour)
		_, err := h.store.Pool.Exec(r.Context(), `
			INSERT INTO pending_invites (email, vineyard_id, role, token, expires_at)
			SELECT $1, id, $2, $3, $4
			FROM vineyards WHERE deleted_at IS NULL
			ORDER BY id LIMIT 1
		`, emailAddr, role, token, expiresAt)
		if err != nil {
			slog.Error("admin: create invite", "err", err)
			w.Header().Set("HX-Trigger", `{"showInviteError":"Kunde inte skapa inbjudan."}`)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Send invite email (or fail gracefully)
		vineyardName, _ := h.store.GetVineyardName(r.Context(), 1)
		emailErr := h.emailSender.SendInviteWithEmail(emailAddr, h.appHost, vineyardName, token)

		// Build response
		var inviteURL string
		if emailErr != nil {
			// SMTP not configured or failed — show the manual link
			inviteURL = fmt.Sprintf("%s/invite/confirm?token=%s", h.appHost, token)
		}

		// Return HTMX partial
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		renderTemplate(w, h.tmpl, "admin/invite_result.html", map[string]any{
			"InviteEmail":  emailAddr,
			"InviteURL":    inviteURL,
			"EmailSent":    emailErr == nil,
			"VineyardName": vineyardName,
		})
	}
}
