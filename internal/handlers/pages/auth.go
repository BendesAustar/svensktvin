// Package pages provides HTTP handlers for page rendering (auth, vineyard, etc.).
package pages

import (
	"crypto/sha256"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/svensktvin/svensktvin/internal/auth"
	"github.com/svensktvin/svensktvin/internal/config"
	"github.com/svensktvin/svensktvin/internal/db"
	"github.com/svensktvin/svensktvin/internal/email"
)

// AuthHandler holds dependencies for auth handlers.
type AuthHandler struct {
	store        *db.Store
	sessionMgr   *auth.SessionManager
	magicLinkMgr *auth.MagicLinkManager
	rateLimiter  *auth.RateLimiter
	cfg          *config.Config
	emailSender  *email.Sender
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(store *db.Store, sessionMgr *auth.SessionManager, magicLinkMgr *auth.MagicLinkManager, rateLimiter *auth.RateLimiter, cfg *config.Config, emailSender *email.Sender) *AuthHandler {
	return &AuthHandler{
		store:        store,
		sessionMgr:   sessionMgr,
		magicLinkMgr: magicLinkMgr,
		rateLimiter:  rateLimiter,
		cfg:          cfg,
		emailSender:  emailSender,
	}
}

// generateCSRFToken creates a new CSRF token.
func generateCSRFToken() string {
	return auth.RandomHex(32)
}

// validateCSRFToken validates a CSRF token from form data against cookie.
func validateCSRFToken(r *http.Request) bool {
	formToken := r.FormValue("csrf_token")
	cookie, err := r.Cookie("csrf_token")
	if err != nil {
		return false
	}
	return formToken == cookie.Value
}

// setCSRFCookie sets the CSRF token cookie.
func setCSRFCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

// renderTemplate executes a Go template with the given data.
func renderTemplate(w http.ResponseWriter, tmpl *template.Template, name string, data map[string]any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := tmpl.ExecuteTemplate(w, name, data)
	if err != nil {
		slog.Error("template execute", "err", err)
		http.Error(w, "Ett internt fel uppstod.", http.StatusInternalServerError)
	}
}

// sanitizeInput removes any HTML tags from user input.
func sanitizeInput(s string) string {
	result := make([]byte, 0, len(s))
	inTag := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				result = append(result, s[i])
			}
		}
	}
	return string(result)
}

// joinStrings joins strings with a separator.
func joinStrings(ss []string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}

// HandleLoginGET renders the login page.
func (h *AuthHandler) HandleLoginGET(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if user is already authenticated
		user := h.sessionMgr.SessionFromRequest(r)
		if user != nil {
			http.Redirect(w, r, "/vineyard", http.StatusSeeOther)
			return
		}

		// Generate CSRF token
		csrfToken := generateCSRFToken()
		setCSRFCookie(w, csrfToken)

		// Parse invite token from query
		inviteToken := r.URL.Query().Get("invite")
		var vineyard *db.Vineyard
		if inviteToken != "" {
			// Load vineyard context if invite token present
			_ = inviteToken
			// TODO: Look up pending invite to get vineyard name
		}

		data := map[string]any{
			"CSRFToken":   csrfToken,
			"InviteToken": inviteToken,
			"Vineyard":    vineyard,
			"Title":       "Logga in — Svenskt Vin",
		}

		renderTemplate(w, tmpl, "auth/login.html", data)
	}
}

// HandleLoginPOST processes login form submissions.
func (h *AuthHandler) HandleLoginPOST(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Rate limit check
		allowed, _ := h.rateLimiter.Allow(auth.ExtractIP(r))
		if !allowed {
			http.Error(w, "För många inloggningsförsök. Försök igen senare.", http.StatusTooManyRequests)
			return
		}

		// CSRF validation
		if !validateCSRFToken(r) {
			http.Error(w, "Ogiltig begäran.", http.StatusBadRequest)
			return
		}

		action := r.FormValue("action")
		email := r.FormValue("email")
		password := r.FormValue("password")
		name := r.FormValue("name")

		switch action {
		case "login_password":
			h.doLogin(w, r, email, password, tmpl)

		case "request_membership":
			h.doRequestMembership(w, r, email, name, tmpl)

		default:
			http.Error(w, "Ogiltig åtgärd.", http.StatusBadRequest)
		}
	}
}

// doLogin handles password-based login.
func (h *AuthHandler) doLogin(w http.ResponseWriter, r *http.Request, email, password string, tmpl *template.Template) {
	// Get user by email
	user, err := h.store.GetUserByEmail(r.Context(), email)
	if err != nil {
		if err == pgx.ErrNoRows {
			// User not found - still show success message to prevent enumeration
			data := map[string]any{
				"Sent": true,
			}
			renderTemplate(w, tmpl, "auth/login.html", data)
			return
		}
		http.Error(w, "Ett internt fel uppstod.", http.StatusInternalServerError)
		return
	}

	// Verify password
	if user.PasswordHash == nil {
		// No password set - redirect to set-password
		http.Redirect(w, r, fmt.Sprintf("/auth/set-password?email=%s", sanitizeInput(email)), http.StatusSeeOther)
		return
	}

	match, err := auth.VerifyPassword(password, *user.PasswordHash)
	if err != nil || !match {
		data := map[string]any{
			"Error": "Fel e-postadress eller lösenord.",
		}
		renderTemplate(w, tmpl, "auth/login.html", data)
		return
	}

	// Create session
	sessionID, err := h.sessionMgr.CreateSession(r.Context(), user.ID)
	if err != nil {
		slog.Error("login: create session", "err", err)
		http.Error(w, "Ett internt fel uppstod.", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	h.sessionMgr.SetSessionCookie(w, sessionID)

	// Redirect to vineyard
	http.Redirect(w, r, "/vineyard", http.StatusSeeOther)
}

// doRequestMembership handles membership request form.
func (h *AuthHandler) doRequestMembership(w http.ResponseWriter, r *http.Request, email, name string, tmpl *template.Template) {
	// Validate input
	email = sanitizeInput(email)
	name = sanitizeInput(name)
	if email == "" || name == "" {
		data := map[string]any{
			"Error": "E-postadress och namn krävs.",
		}
		renderTemplate(w, tmpl, "auth/login.html", data)
		return
	}

	// TODO: In a real implementation, this would:
	// 1. Insert a pending membership request
	// 2. Send a confirmation email to the admin
	// 3. Show success message to user

	// For now, just show success message
	data := map[string]any{
		"MembershipSent": true,
	}
	renderTemplate(w, tmpl, "auth/login.html", data)
}

// HandleLogoutPOST processes logout.
func (h *AuthHandler) HandleLogoutPOST(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get session
		cookie, err := r.Cookie("session_id")
		if err == nil {
			// Invalidate session in database
			_ = h.sessionMgr.DeleteSession(r.Context(), cookie.Value)
		}

		// Clear session cookie
		h.sessionMgr.ClearSessionCookie(w)

		// Redirect to login with flash message
		http.Redirect(w, r, "/login?logged_out=true", http.StatusSeeOther)
	}
}

// HandleRegisterGET renders the registration page.
func (h *AuthHandler) HandleRegisterGET(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get invite token
		inviteToken := r.URL.Query().Get("invite")
		if inviteToken == "" {
			http.Error(w, "Ingen inbjudan tillgänglig.", http.StatusBadRequest)
			return
		}

		// Validate invite token
		invite, err := h.store.GetPendingInvite(r.Context(), inviteToken)
		if err != nil {
			data := map[string]any{
				"Error": "Inbjudan har gått ut eller är ogiltig.",
			}
			renderTemplate(w, tmpl, "auth/register.html", data)
			return
		}

		// Check if user has existing account
		email := r.URL.Query().Get("email")
		hasAccount := false
		if email != "" {
			_, err := h.store.GetUserByEmail(r.Context(), email)
			if err == nil {
				// User has existing account
				hasAccount = true
			}
		}

		// Get vineyard name
		vineyardName, _ := h.store.GetVineyardName(r.Context(), invite.VineyardID)

		csrfToken := generateCSRFToken()
		setCSRFCookie(w, csrfToken)

		data := map[string]any{
			"CSRFToken":    csrfToken,
			"InviteToken":  inviteToken,
			"InviteEmail":  invite.Email,
			"VineyardName": vineyardName,
			"Role":         invite.Role,
			"HasAccount":   hasAccount,
		}

		renderTemplate(w, tmpl, "auth/register.html", data)
	}
}

// HandleRegisterPOST processes registration.
func (h *AuthHandler) HandleRegisterPOST(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Rate limit check
		allowed, _ := h.rateLimiter.Allow(auth.ExtractIP(r))
		if !allowed {
			http.Error(w, "För många inloggningsförsök. Försök igen senare.", http.StatusTooManyRequests)
			return
		}

		// CSRF validation
		if !validateCSRFToken(r) {
			http.Error(w, "Ogiltig begäran.", http.StatusBadRequest)
			return
		}

		inviteToken := r.FormValue("invite_token")
		if inviteToken == "" {
			data := map[string]any{"Error": "Ogiltig inbjudan."}
			renderTemplate(w, tmpl, "auth/register.html", data)
			return
		}

		// Validate invite
		invite, err := h.store.GetPendingInvite(r.Context(), inviteToken)
		if err != nil {
			data := map[string]any{"Error": "Inbjudan har gått ut eller är ogiltig."}
			renderTemplate(w, tmpl, "auth/register.html", data)
			return
		}

		// Get vineyard name for error rendering
		vineyardName, _ := h.store.GetVineyardName(r.Context(), invite.VineyardID)

		email := r.FormValue("email")
		name := r.FormValue("name")
		password := r.FormValue("password")

		// Validate password strength
		valid, errors := auth.PasswordStrength(password)
		if !valid {
			data := map[string]any{
				"Error":        "Lösenord: " + joinStrings(errors),
				"InviteToken":  inviteToken,
				"InviteEmail":  invite.Email,
				"VineyardName": vineyardName,
				"Role":         invite.Role,
			}
			renderTemplate(w, tmpl, "auth/register.html", data)
			return
		}

		_ = email // used in CreateUser below
		_ = name  // used in CreateUser below

		// Hash password
		passHash, err := auth.HashPassword(password, 12)
		if err != nil {
			slog.Error("register: hash password", "err", err)
			http.Error(w, "Ett internt fel uppstod.", http.StatusInternalServerError)
			return
		}

		// Create user
		userID, err := h.store.CreateUser(r.Context(), email)
		if err != nil {
			slog.Error("register: create user", "err", err)
			http.Error(w, "Ett internt fel uppstod.", http.StatusInternalServerError)
			return
		}

		// Set password hash
		err = h.store.CreateUserPasswordHash(r.Context(), userID, passHash)
		if err != nil {
			slog.Error("register: set password hash", "err", err)
			http.Error(w, "Ett internt fel uppstod.", http.StatusInternalServerError)
			return
		}

		// Add to vineyard
		_, err = h.store.Pool.Exec(r.Context(), `
			INSERT INTO vineyard_members (vineyard_id, user_id, role)
			VALUES ($1, $2, $3)
			ON CONFLICT (vineyard_id, user_id) DO NOTHING
		`, invite.VineyardID, userID, invite.Role)
		if err != nil {
			slog.Error("register: add to vineyard", "err", err)
			http.Error(w, "Ett internt fel uppstod.", http.StatusInternalServerError)
			return
		}

		// Mark invite as used
		_ = h.store.UpdatePendingInviteUsed(r.Context(), invite.ID)

		// Create session
		sessionID, err := h.sessionMgr.CreateSession(r.Context(), userID)
		if err != nil {
			slog.Error("register: create session", "err", err)
			http.Error(w, "Ett internt fel uppstod.", http.StatusInternalServerError)
			return
		}

		// Set session cookie
		h.sessionMgr.SetSessionCookie(w, sessionID)

		// Redirect to vineyard
		http.Redirect(w, r, "/vineyard", http.StatusSeeOther)
	}
}

// HandleForgotPasswordGET renders the forgot password page.
func (h *AuthHandler) HandleForgotPasswordGET(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		csrfToken := generateCSRFToken()
		setCSRFCookie(w, csrfToken)

		data := map[string]any{
			"CSRFToken": csrfToken,
		}
		renderTemplate(w, tmpl, "auth/forgot-password.html", data)
	}
}

// HandleForgotPasswordPOST processes forgot password.
func (h *AuthHandler) HandleForgotPasswordPOST(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Rate limit check
		allowed, _ := h.rateLimiter.Allow(auth.ExtractIP(r))
		if !allowed {
			http.Error(w, "För många inloggningsförsök. Försök igen senare.", http.StatusTooManyRequests)
			return
		}

		// CSRF validation
		if !validateCSRFToken(r) {
			http.Error(w, "Ogiltig begäran.", http.StatusBadRequest)
			return
		}

		email := sanitizeInput(r.FormValue("email"))
		if email == "" {
			data := map[string]any{"Error": "E-postadress krävs."}
			renderTemplate(w, tmpl, "auth/forgot-password.html", data)
			return
		}

		// Upsert user (enumeration-safe: creates user if doesn't exist)
		userID, err := h.store.UpsertUser(r.Context(), email)
		if err != nil {
			slog.Error("forgot-password: upsert user", "err", err)
			data := map[string]any{"Error": "Ett internt fel uppstod."}
			renderTemplate(w, tmpl, "auth/forgot-password.html", data)
			return
		}

		// Generate magic link token
		rawToken := auth.RandomHex(32)
		hash := sha256.Sum256([]byte(rawToken))

		// Insert magic link token
		_, err = h.store.Pool.Exec(r.Context(), `
			INSERT INTO magic_link_tokens (user_id, token_hash, expires_at)
			VALUES ($1, $2, $3)
		`, userID, hash[:], time.Now().Add(15*time.Minute))
		if err != nil {
			slog.Error("forgot-password: insert token", "err", err)
		}

		// Send magic link email
		_ = h.emailSender.SendMagicLink(email, rawToken)

		// Always show success message (even if email fails) to prevent enumeration
		data := map[string]any{
			"Sent": true,
		}
		renderTemplate(w, tmpl, "auth/forgot-password.html", data)
	}
}

// HandleSetPasswordGET renders the set password page.
func (h *AuthHandler) HandleSetPasswordGET(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		_ = r // suppress unused variable

		if token == "" {
			http.Error(w, "Ingen token tillgänglig.", http.StatusBadRequest)
			return
		}

		// Validate magic link token
		userID, err := h.magicLinkMgr.VerifyToken(r.Context(), token)
		if err != nil {
			data := map[string]any{"Error": "Ogiltig eller utgången länk."}
			renderTemplate(w, tmpl, "auth/set-password.html", data)
			return
		}

		// Get user email
		user, err := h.store.GetUserInfo(r.Context(), userID)
		if err != nil {
			data := map[string]any{"Error": "Användaren hittades inte."}
			renderTemplate(w, tmpl, "auth/set-password.html", data)
			return
		}

		csrfToken := generateCSRFToken()
		setCSRFCookie(w, csrfToken)

		data := map[string]any{
			"CSRFToken": csrfToken,
			"Token":     token,
			"Email":     user.Email,
		}

		renderTemplate(w, tmpl, "auth/set-password.html", data)
	}
}

// HandleSetPasswordPOST processes password setup.
func (h *AuthHandler) HandleSetPasswordPOST(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// CSRF validation
		if !validateCSRFToken(r) {
			http.Error(w, "Ogiltig begäran.", http.StatusBadRequest)
			return
		}

		token := r.FormValue("token")
		password := r.FormValue("password")
		confirmPassword := r.FormValue("confirmPassword")

		if token == "" {
			data := map[string]any{"Error": "Ingen token tillgänglig."}
			renderTemplate(w, tmpl, "auth/set-password.html", data)
			return
		}

		// Validate password match
		if password != confirmPassword {
			data := map[string]any{"Error": "Lösenorden matchar inte."}
			renderTemplate(w, tmpl, "auth/set-password.html", data)
			return
		}

		// Validate password strength
		valid, errors := auth.PasswordStrength(password)
		if !valid {
			data := map[string]any{
				"Error":          "Lösenord: " + joinStrings(errors),
				"PasswordErrors": errors,
			}
			renderTemplate(w, tmpl, "auth/set-password.html", data)
			return
		}

		// Verify magic link token
		userID, err := h.magicLinkMgr.VerifyToken(r.Context(), token)
		if err != nil {
			data := map[string]any{"Error": "Ogiltig eller utgången länk."}
			renderTemplate(w, tmpl, "auth/set-password.html", data)
			return
		}

		// Hash password
		passHash, err := auth.HashPassword(password, 12)
		if err != nil {
			slog.Error("set-password: hash password", "err", err)
			http.Error(w, "Ett internt fel uppstod.", http.StatusInternalServerError)
			return
		}

		// Update password hash
		err = h.store.CreateUserPasswordHash(r.Context(), userID, passHash)
		if err != nil {
			slog.Error("set-password: update password", "err", err)
			http.Error(w, "Ett internt fel uppstod.", http.StatusInternalServerError)
			return
		}

		// Invalidate all sessions
		_ = h.store.DeleteSessionsByUser(r.Context(), userID)

		// Create new session
		sessionID, err := h.sessionMgr.CreateSession(r.Context(), userID)
		if err != nil {
			slog.Error("set-password: create session", "err", err)
			http.Error(w, "Ett internt fel uppstod.", http.StatusInternalServerError)
			return
		}

		// Set session cookie
		h.sessionMgr.SetSessionCookie(w, sessionID)

		// Redirect to vineyard
		http.Redirect(w, r, "/vineyard", http.StatusSeeOther)
	}
}

// HandleInviteConfirmGET renders the invite confirmation page.
func (h *AuthHandler) HandleInviteConfirmGET(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "Ogiltig inbjudan.", http.StatusBadRequest)
			return
		}

		// Validate invite
		invite, err := h.store.GetPendingInvite(r.Context(), token)
		if err != nil {
			data := map[string]any{"Error": "Inbjudan har gått ut eller är ogiltig."}
			renderTemplate(w, tmpl, "invite/confirm.html", data)
			return
		}

		// Get vineyard name
		vineyardName, _ := h.store.GetVineyardName(r.Context(), invite.VineyardID)

		// Get current user
		user := h.sessionMgr.SessionFromRequest(r)

		csrfToken := generateCSRFToken()
		setCSRFCookie(w, csrfToken)

		data := map[string]any{
			"CSRFToken":    csrfToken,
			"InviteToken":  token,
			"VineyardName": vineyardName,
			"Role":         invite.Role,
			"Email":        "",
		}
		if user != nil {
			data["Email"] = user.Email
		}

		renderTemplate(w, tmpl, "invite/confirm.html", data)
	}
}

// HandleInviteConfirmPOST processes invite acceptance.
func (h *AuthHandler) HandleInviteConfirmPOST(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// CSRF validation
		if !validateCSRFToken(r) {
			http.Error(w, "Ogiltig begäran.", http.StatusBadRequest)
			return
		}

		token := r.FormValue("invite_token")
		if token == "" {
			http.Error(w, "Ogiltig inbjudan.", http.StatusBadRequest)
			return
		}

		// Validate invite
		invite, err := h.store.GetPendingInvite(r.Context(), token)
		if err != nil {
			data := map[string]any{"Error": "Inbjudan har gått ut eller är ogiltig."}
			renderTemplate(w, tmpl, "invite/confirm.html", data)
			return
		}

		// Get current user
		user := h.sessionMgr.SessionFromRequest(r)
		if user == nil {
			http.Error(w, "Du måste vara inloggad.", http.StatusUnauthorized)
			return
		}

		// Check email match
		if user.Email != invite.Email {
			data := map[string]any{"Error": "E-postadressen matchar inte inbjudan."}
			renderTemplate(w, tmpl, "invite/confirm.html", data)
			return
		}

		// Add to vineyard (idempotent)
		_, err = h.store.Pool.Exec(r.Context(), `
			INSERT INTO vineyard_members (vineyard_id, user_id, role)
			VALUES ($1, $2, $3)
			ON CONFLICT (vineyard_id, user_id) DO UPDATE SET role = EXCLUDED.role
		`, invite.VineyardID, user.ID, invite.Role)
		if err != nil {
			slog.Error("invite-confirm: add to vineyard", "err", err)
			data := map[string]any{"Error": "Kunde inte gå med i vingården. Försök igen."}
			renderTemplate(w, tmpl, "invite/confirm.html", data)
			return
		}

		// Mark invite as used
		_ = h.store.UpdatePendingInviteUsed(r.Context(), invite.ID)

		// Redirect to vineyard
		vineyardName, _ := h.store.GetVineyardName(r.Context(), invite.VineyardID)
		data := map[string]any{
			"VineyardName": vineyardName,
			"VineyardID":   invite.VineyardID,
		}
		renderTemplate(w, tmpl, "invite/success.html", data)
	}
}