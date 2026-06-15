# Svenskt Vin Admin Bootstrap & Dashboard

**Status**: Design Complete
**Created**: 2026-06-15
**Author**: DesignBrain
**Review**: EngineeringBrain
**Dependencies**: None (standalone)

---

## Problem Statement

The Svenskt Vin application has no mechanism to create the **first admin account**. The login page requires authentication to access anything — including user management. The invite/magic-link flow assumes an admin already exists to send invitations. The seed data (`003_users.sql`) creates a hardcoded admin for test fixtures, but a production deployment has no way to bootstrap.

Additionally, there is no admin dashboard to manage users, vineyards, and invites after the first admin exists.

---

## Requirements

### Must Have
1. **Bootstrap CLI command** — one-time CLI to create the initial admin account with email + password
2. **Admin middleware** — `RequireAdmin` that extends `RequireAuth`, checks `is_admin` flag
3. **Admin dashboard** — authenticated-only page for the first admin to manage the system
4. **User management** — list users, invite users with roles, deactivate/reactivate, reset passwords
5. **Self-disabling bootstrap** — idempotent: re-running updates the password; never fails with a "already exists" error

### Nice to Have (MVP+)
1. Admin activity log (simple `admin_actions` table)
2. Bulk import users from CSV

### Not In Scope (MVP)
1. Role-based access control beyond admin/non-admin
2. Multi-vineyard admin (single vineyard for now)
3. API key management
4. Audit trail export

---

## Architecture Overview

```
┌──────────────────────────────────────────────────────────────────┐
│                        Svenskt Vin App                          │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  CLI Entry Point (cmd/web/main.go)                               │
│  ├── "admin bootstrap"  ← Creates first admin                   │
│  └── (default)         ← HTTP server mode                       │
│                                                                  │
│  HTTP Server                                                      │
│  ├── Public routes (no auth)                                     │
│  │   ├── GET /login                                              │
│  │   ├── POST /login                                             │
│  │   ├── GET /auth/forgot-password                               │
│  │   ├── POST /auth/forgot-password                              │
│  │   └── GET /health                                             │
│  │                                                               │
│  ├── Authenticated routes (RequireAuth)                          │
│  │   ├── GET /vineyard                                           │
│  │   ├── POST /vineyard/                                         │
│  │   └── ...                                                     │
│  │                                                               │
│  └── Admin routes (RequireAdmin → RequireAuth)                   │
│      ├── GET  /admin             ← Dashboard overview           │
│      ├── GET  /admin/users       ← User list                    │
│      ├── GET  /admin/users/:id   ← User detail                  │
│      ├── POST /admin/users/:id   ← User update (deactivate, etc.)│
│      ├── POST /admin/users/invite ← Generate invite             │
│      └── GET  /admin/vineyard    ← Vineyard settings            │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### Bootstrap Mechanism

Two complementary bootstrap mechanisms, both idempotent:

#### 1. Bootstrap CLI Command (primary)

```bash
go run ./cmd/web/ admin bootstrap \
  --email=admin@example.com \
  --password=Secure123!
```

**Behavior:**
- Checks if any user with `is_admin=true` exists
- If **no admin exists**: creates new user with `is_admin=true`, sets password hash (bcrypt, 12 rounds), prints confirmation
- If **an admin already exists**: updates that admin's password, prints "Password updated for <email>"
- Returns exit code 0 on success, non-zero on failure
- **Self-disabling**: after bootstrap, the admin dashboard becomes the primary management interface. The CLI command still works but is documented as "emergency access only."

**Why CLI?**
- No SMTP required
- No network dependency
- Safe for Docker entrypoints, CI/CD, and initial deployment
- Idempotent by design — can be re-run safely

#### 2. Self-Service Bootstrap Endpoint (fallback)

For headless deployments where CLI access is limited, a one-time self-service endpoint:

```
POST /admin/bootstrap
```

- **Only active when ZERO users exist** — checks `SELECT COUNT(*) FROM users`
- Accepts `email`, `password`, `name` in JSON or form body
- Creates first user with `is_admin=true`
- **Self-disables after first use**: sets an internal flag or simply checks count
- Returns the admin's email + a one-time setup link
- After first use, returns `410 Gone` with message "System already initialized"

**Decision**: The CLI command is the primary bootstrap mechanism. The self-service endpoint is deferred to MVP+ unless a specific headless deployment requirement emerges. The CLI command alone satisfies MVP.

---

## Component Design

### 1. Bootstrap CLI Command

**File**: `cmd/web/admin_bootstrap.go` (new)

```go
package main

import (
    "flag"
    "fmt"
    "os"

    "github.com/svensktvin/svensktvin/internal/auth"
    "github.com/svensktvin/svensktvin/internal/db"
)

type adminBootstrapCmd struct {
    email    string
    password string
}

func (c *adminBootstrapCmd) Run(store *db.Store) error {
    // 1. Check if any admin exists
    var adminCount int
    err := store.Pool.QueryRow(context.Background(), `
        SELECT COUNT(*) FROM users WHERE is_admin = true
    `).Scan(&adminCount)
    if err != nil {
        return fmt.Errorf("check admin: %w", err)
    }

    if adminCount > 0 {
        // Admin exists — update password
        adminUser, err := store.GetFirstAdmin(context.Background())
        if err != nil {
            return fmt.Errorf("find admin: %w", err)
        }
        hash, err := auth.HashPassword(c.password, 12)
        if err != nil {
            return fmt.Errorf("hash password: %w", err)
        }
        _, err = store.Pool.Exec(context.Background(), `
            UPDATE users SET password_hash = $1 WHERE id = $2
        `, hash, adminUser.ID)
        if err != nil {
            return fmt.Errorf("update password: %w", err)
        }
        fmt.Printf("Password updated for admin: %s\n", adminUser.Email)
        return nil
    }

    // 2. No admin exists — create one
    hash, err := auth.HashPassword(c.password, 12)
    if err != nil {
        return fmt.Errorf("hash password: %w", err)
    }

    var userID int64
    err = store.Pool.QueryRow(context.Background(), `
        INSERT INTO users (email, name, is_admin, password_hash, active)
        VALUES ($1, $2, true, $3, true)
        RETURNING id
    `, c.email, splitName(c.email), hash).Scan(&userID)
    if err != nil {
        return fmt.Errorf("create admin: %w", err)
    }

    fmt.Printf("✓ Admin account created\n")
    fmt.Printf("  Email:    %s\n", c.email)
    fmt.Printf("  Password: %s\n", c.password)
    fmt.Printf("\n  Log in at: %s/login\n", os.Getenv("APP_HOST") || "http://localhost:8080")
    fmt.Printf("\n  ⚠  This bootstrap command can be re-run to reset the admin password.\n")

    // 3. Create default vineyard if none exists
    var vineyardCount int
    err = store.Pool.QueryRow(context.Background(), `
        SELECT COUNT(*) FROM vineyards WHERE deleted_at IS NULL
    `).Scan(&vineyardCount)
    if err != nil {
        return fmt.Errorf("count vineyards: %w", err)
    }
    if vineyardCount == 0 {
        var vineyardID int64
        err = store.Pool.QueryRow(context.Background(), `
            INSERT INTO vineyards (name, organic, biodynamic)
            VALUES ('Min vingård', false, false)
            RETURNING id
        `).Scan(&vineyardID)
        if err == nil {
            _, _ = store.Pool.Exec(context.Background(), `
                INSERT INTO vineyard_members (vineyard_id, user_id, role)
                VALUES ($1, $2, 'owner')
            `, vineyardID, userID)
            fmt.Printf("\n✓ Default vineyard created: 'Min vingård'\n")
        }
    }

    return nil
}
```

**Subcommand Detection in `main.go`**:
```go
// In main(), before HTTP server setup:
if len(os.Args) > 1 && os.Args[1] == "admin" && len(os.Args) > 2 && os.Args[2] == "bootstrap" {
    // Parse bootstrap flags and run CLI
    runAdminBootstrap(os.Args[3:])
    return
}
```

**Password validation** (reuse existing `auth.PasswordStrength`):
- Must pass Swedish password strength requirements
- Returns error if weak password

---

### 2. Admin Middleware

**File**: `internal/auth/session.go` (extend existing)

```go
// RequireAdmin is a middleware that verifies session authentication AND admin status.
// Wraps RequireAuth: first checks session validity, then checks is_admin flag.
func (sm *SessionManager) RequireAdmin(h http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Reuse RequireAuth's cookie-based session lookup
        userInfo := sm.SessionFromRequest(r)
        if userInfo == nil {
            http.Redirect(w, r, "/login", http.StatusSeeOther)
            return
        }
        if !userInfo.IsAdmin {
            // Non-admin tries to access admin area
            data := map[string]any{
                "Title": "Åtkomst nekad",
                "Error": "Du behöver administratörsrättigheter för att komma åt denna sida.",
            }
            renderTemplate(w, getTemplates(), "admin/forbidden.html", data)
            // Note: renderTemplate is in pages package — we'll need a shared template loader
            // or move this logic to the handler
            return
        }
        // Admin authenticated — add to context and continue
        ctx := contextWithUser(r.Context(), *userInfo)
        h.ServeHTTP(w, r.WithContext(ctx))
    }
}
```

**Integration with existing `RequireAuth`**:
- Existing `RequireAuth` uses `Authorization` header for API routes
- New `RequireAdmin` uses `SessionFromRequest` (cookie) for page routes
- Both paths set `UserInfo` in context for downstream access
- For HTMX SPA-style navigation, admins see the admin link in nav; non-admins do not

---

### 3. Admin Handler

**File**: `internal/handlers/pages/admin.go` (new)

```go
package pages

type AdminHandler struct {
    store       *db.Store
    sessionMgr  *auth.SessionManager
    cookieCfg   config.CookieConfig
    emailSender *email.Sender
    appHost     string
    tmpl        *template.Template  // Pre-loaded template set
}

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
```

#### 3a. Dashboard Overview (`GET /admin`)

**Store method needed**: `ListAllUsers` — returns all users with `is_admin` flag.

```go
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

        // Recent activity — last logins
        var recentLogins []struct {
            Email   string
            Name    string
            LastLogin *time.Time
            IsAdmin bool
        }
        rows, err := h.store.Pool.Query(r.Context(), `
            SELECT email, name, last_login, is_admin
            FROM users
            ORDER BY last_login DESC NULLS LAST
            LIMIT 5
        `)
        // ... process rows

        csrfToken := generateCSRFToken()
        setCSRFCookie(w, csrfToken, h.cookieCfg)

        data := map[string]any{
            "User":         user,
            "Title":        "Adminpanel — Svenskt Vin",
            "CSRFToken":    csrfToken,
            "UserCount":    userCount,
            "AdminCount":   adminCount,
            "ActiveCount":  activeCount,
            "RecentLogins": recentLogins,
        }
        renderTemplate(w, h.tmpl, "admin/dashboard.html", data)
    }
}
```

#### 3b. User List (`GET /admin/users`)

```go
func (h *AdminHandler) HandleUsersGET() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        user := h.sessionMgr.SessionFromRequest(r)
        if user == nil {
            http.Redirect(w, r, "/login", http.StatusSeeOther)
            return
        }

        rows, err := h.store.Pool.Query(r.Context(), `
            SELECT id, email, name, is_admin, active, created_at, last_login
            FROM users
            ORDER BY created_at DESC
        `)
        // ... process rows into []UserSummary

        csrfToken := generateCSRFToken()
        setCSRFCookie(w, csrfToken, h.cookieCfg)

        data := map[string]any{
            "User":        user,
            "Title":       "Användare — Svenskt Vin",
            "CSRFToken":   csrfToken,
            "Users":       users,
        }
        renderTemplate(w, h.tmpl, "admin/users.html", data)
    }
}
```

#### 3c. User Detail (`GET/POST /admin/users/:id`)

**GET** — Show user edit form:
- Email, name, role (owner/editor/admin), active status
- Buttons: Deactivate, Reset Password, Make Admin, Demote from Admin

**POST** — Handle form submissions:

```go
func (h *AdminHandler) HandleUserDetailPOST() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Parse userID from path
        userID, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/admin/users/"), 10, 64)
        if err != nil {
            http.NotFound(w, r)
            return
        }

        // CSRF validation
        if !validateCSRFToken(r) {
            http.Error(w, "Ogiltig begäran.", http.StatusBadRequest)
            return
        }

        action := r.FormValue("action")
        switch action {
        case "deactivate":
            _, _ = h.store.Pool.Exec(r.Context(), `
                UPDATE users SET active = false WHERE id = $1
            `, userID)
            h.store.DeleteSessionsByUser(r.Context(), userID)
            redirectBack(w, r, "/admin/users")

        case "activate":
            _, _ = h.store.Pool.Exec(r.Context(), `
                UPDATE users SET active = true WHERE id = $1
            `, userID)
            redirectBack(w, r, "/admin/users")

        case "reset_password":
            // Generate a temporary password hash or magic link
            // For MVP: reset to a random temp password (email it?)
            // Simpler: generate magic link and show to admin
            rawToken := auth.RandomHex(32)
            hash := sha256.Sum256([]byte(rawToken))
            _, err := h.store.Pool.Exec(r.Context(), `
                INSERT INTO magic_link_tokens (user_id, token_hash, expires_at)
                VALUES ($1, $2, $3)
            `, userID, hash[:], time.Now().Add(1*time.Hour))
            if err != nil {
                slog.Error("admin: reset password", "err", err)
            }
            // Show the magic link URL to the admin (or email it)
            // For MVP, show in flash message
            redirectBack(w, r, fmt.Sprintf("/admin/users/%d", userID))

        case "toggle_admin":
            var currentAdmin bool
            err := h.store.Pool.QueryRow(r.Context(), `
                SELECT is_admin FROM users WHERE id = $1
            `, userID).Scan(&currentAdmin)
            if err == nil {
                _, _ = h.store.Pool.Exec(r.Context(), `
                    UPDATE users SET is_admin = $1 WHERE id = $2
                `, !currentAdmin, userID)
            }
            redirectBack(w, r, "/admin/users")

        default:
            http.NotFound(w, r)
        }
    }
}
```

#### 3d. Generate Invite (`POST /admin/users/invite`)

**HTMX POST** to `/admin/users/invite`:

```go
func (h *AdminHandler) HandleInviteGeneratePOST() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if !validateCSRFToken(r) {
            http.Error(w, "Ogiltig begäran.", http.StatusBadRequest)
            return
        }

        email := sanitizeInput(r.FormValue("email"))
        role := r.FormValue("role") // "owner" or "editor" (or "viewer" if we extend)

        // Validate role — current DB constraint only allows 'owner' or 'editor'
        // See migration 016: CHECK (role IN ('owner', 'editor'))
        if role != "owner" && role != "editor" {
            w.Header().Set("HX-Trigger", `{"showInviteError":"Ogiltig roll. Använd 'owner' eller 'editor'."}`)
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
        `, email, role, token, expiresAt)
        if err != nil {
            slog.Error("admin: create invite", "err", err)
            w.Header().Set("HX-Trigger", `{"showInviteError":"Kunde inte skapa inbjudan."}`)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }

        // Send invite email (or fail gracefully)
        appHost := h.appHost
        vineyardName, _ := h.store.GetVineyardName(r.Context(), 1) // first vineyard
        emailErr := h.emailSender.SendInviteWithEmail(email, appHost, vineyardName, token)

        // Build response
        var inviteURL string
        if emailErr != nil {
            // SMTP not configured or failed — show the manual link
            inviteURL = fmt.Sprintf("%s/invite/confirm?token=%s", appHost, token)
        } else {
            inviteURL = "" // Email sent successfully
        }

        // Return HTMX partial to show result
        data := map[string]any{
            "InviteEmail":  email,
            "InviteURL":    inviteURL,
            "EmailSent":    emailErr == nil,
            "VineyardName": vineyardName,
        }
        // Execute "admin/invite_result.html" template
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        tmpl := h.tmpl.Lookup("admin/invite_result.html")
        if tmpl != nil {
            tmpl.Execute(w, data)
        }
    }
}
```

#### 3e. Vineyard Settings (`GET/POST /admin/vineyard`)

For MVP, this is a simple CRUD form for vineyard name and basic settings.
In practice, since there's typically one vineyard, this manages the first/primary vineyard.

---

### 4. Store Methods (new)

**File**: `internal/db/store.go` (extend)

```go
// ListAllUsers returns all users ordered by creation date.
func (s *Store) ListAllUsers(ctx context.Context) ([]User, error) {
    rows, err := s.Pool.Query(ctx, `
        SELECT id, email, name, is_admin, active, created_at, last_login
        FROM users
        ORDER BY created_at DESC
    `)
    if err != nil {
        return nil, fmt.Errorf("list all users: %w", err)
    }
    defer rows.Close()

    var users []User
    for rows.Next() {
        var u User
        if err := rows.Scan(
            &u.ID, &u.Email, &u.Name, &u.IsAdmin, &u.Active,
            &u.CreatedAt, &u.LastLogin,
        ); err != nil {
            continue
        }
        users = append(users, u)
    }
    return users, nil
}

// GetFirstAdmin returns the first admin user (ordered by creation date).
func (s *Store) GetFirstAdmin(ctx context.Context) (*User, error) {
    var u User
    err := s.Pool.QueryRow(ctx, `
        SELECT id, email, name, is_admin, active, created_at, last_login
        FROM users
        WHERE is_admin = true
        ORDER BY created_at ASC
        LIMIT 1
    `).Scan(&u.ID, &u.Email, &u.Name, &u.IsAdmin, &u.Active, &u.CreatedAt, &u.LastLogin)
    if err != nil {
        return nil, fmt.Errorf("get first admin: %w", err)
    }
    return &u, nil
}

// UpdateUserActive sets a user's active status and invalidates their sessions.
func (s *Store) UpdateUserActive(ctx context.Context, userID int64, active bool) error {
    _, err := s.Pool.Exec(ctx, `
        UPDATE users SET active = $1 WHERE id = $2
    `, active, userID)
    if err != nil {
        return fmt.Errorf("update user active: %w", err)
    }
    if !active {
        _ = s.DeleteSessionsByUser(ctx, userID)
    }
    return nil
}

// UpdateUserPasswordHash updates a user's password hash.
func (s *Store) UpdateUserPasswordHash(ctx context.Context, userID int64, hash string) error {
    _, err := s.Pool.Exec(ctx, `
        UPDATE users SET password_hash = $1 WHERE id = $2
    `, hash, userID)
    if err != nil {
        return fmt.Errorf("update user password hash: %w", err)
    }
    return nil
}

// CountUsers returns total, admin, and active counts.
func (s *Store) CountUsers(ctx context.Context) (total, admins, active int) {
    err := s.Pool.QueryRow(context.Background(), `
        SELECT COUNT(*),
               COUNT(*) FILTER (WHERE is_admin),
               COUNT(*) FILTER (WHERE active)
        FROM users
    `).Scan(&total, &admins, &active)
    if err != nil {
        slog.Error("db: count users", "err", err)
    }
    return
}
```

**Note on `User` struct**: The `User` struct in `models.go` doesn't currently have `CreatedAt` and `LastLogin` exported fields in a way that supports `Scan` from all columns. The `LastLogin` is `*time.Time`, but `CreatedAt` is missing from the struct. **Migration needed**: Add `CreatedAt` to `User` struct.

---

### 5. Database Migration (new)

**File**: `db/migrations/020_admin_dashboard.sql` (new)

```sql
-- Migration 020: Admin Dashboard Support
BEGIN;

-- Add created_at to users if not present (for bootstrap ordering)
-- The migration 002 already creates users with created_at, but let's ensure
-- it's present for ListAllUsers ordering.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS created_at timestamptz NOT NULL DEFAULT now();

-- Add viewer role support to pending_invites
-- Current constraint: CHECK (role IN ('owner', 'editor'))
-- Extend to include 'viewer' for MVP+ (deferred: optional for MVP)
-- DO NOT alter the CHECK constraint in MVP — keep 'owner'/'editor' only.
-- The admin UI will only offer these two roles.

-- Create admin_actions table for audit trail (MVP+ placeholder)
CREATE TABLE IF NOT EXISTS admin_actions (
  id         serial PRIMARY KEY,
  admin_user_id integer NOT NULL REFERENCES users(id),
  target_user_id integer REFERENCES users(id),
  action     text NOT NULL,       -- 'create_user', 'deactivate', 'reset_password', 'toggle_admin', 'send_invite'
  details    jsonb DEFAULT '{}',  -- additional context
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_admin_actions_admin ON admin_actions (admin_user_id);
CREATE INDEX IF NOT EXISTS idx_admin_actions_target ON admin_actions (target_user_id);
CREATE INDEX IF NOT EXISTS idx_admin_actions_created ON admin_actions (created_at DESC);

COMMENT ON TABLE admin_actions IS 'Audit log for admin dashboard actions.';

COMMIT;
```

**Decision**: Include `admin_actions` in MVP migration. It's low cost and provides the foundation for activity logging in the dashboard. The dashboard will display recent actions.

---

## Admin Dashboard Template Structure

### Directory Structure

```
internal/templates/admin/
├── layout.html           — Admin layout with sidebar navigation
├── dashboard.html        — Overview stats
├── users.html            — User list table
├── user_detail.html      — Single user edit
├── invite_result.html    — HTMX fragment for invite generation result
├── vineyard.html         — Vineyard settings
└── forbidden.html        — 403 page for non-admin access
```

### Layout Template (`admin/layout.html`)

```html
{{define "admin/layout.html"}}
<!DOCTYPE html>
<html lang="sv" class="h-full">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{block "title" .}}{{.Title}}{{end}}</title>
    <script src="https://unpkg.com/htmx.org@2.0.4"></script>
    <script defer src="https://cdn.jsdelivr.net/npm/alpinejs@3.14.8/dist/cdn.min.js"></script>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="h-full bg-gray-50">
    <div class="flex h-full">
        <!-- Sidebar -->
        <aside class="w-64 bg-white border-r border-gray-200 flex flex-col">
            <div class="p-4 border-b border-gray-200">
                <a href="/" class="text-[#2d6a2d] font-bold text-lg no-underline">🍷 Svenskt Vin</a>
                <p class="text-xs text-gray-500 mt-1">Adminpanel</p>
            </div>
            <nav class="flex-1 p-2">
                <a href="/admin"
                   class="block px-3 py-2 rounded text-sm {{if .IsAdminDashboard}}bg-green-50 text-[#2d6a2d] font-medium{{else}}text-gray-600 hover:bg-gray-100{{end}}">
                    📊 Översikt
                </a>
                <a href="/admin/users"
                   class="block px-3 py-2 rounded text-sm {{if .IsAdminUsers}}bg-green-50 text-[#2d6a2d] font-medium{{else}}text-gray-600 hover:bg-gray-100{{end}}">
                    👥 Användare
                </a>
                <a href="/admin/vineyard"
                   class="block px-3 py-2 rounded text-sm {{if .IsAdminVineyard}}bg-green-50 text-[#2d6a2d] font-medium{{else}}text-gray-600 hover:bg-gray-100{{end}}">
                    🍇 Vingård
                </a>
            </nav>
            <div class="p-4 border-t border-gray-200">
                <div class="text-sm text-gray-500 mb-2">{{.User.Name}}</div>
                <form method="POST" action="/logout">
                    <button type="submit" class="text-sm text-gray-500 hover:text-gray-700">Logga ut</button>
                </form>
            </div>
        </aside>

        <!-- Main content -->
        <main class="flex-1 overflow-auto">
            <header class="bg-white border-b border-gray-200 px-6 py-4">
                <h1 class="text-xl font-semibold">{{block "page_title" .}}{{.Title}}{{end}}</h1>
            </header>
            <div class="p-6">
                {{template "flash" .}}
                {{block "content" .}}{{end}}
            </div>
        </main>
    </div>
</body>
</html>
{{end}}
```

### Dashboard Template (`admin/dashboard.html`)

```html
{{template "admin/layout.html" .}}
{{define "page_title"}}Adminpanel — Översikt{{end}}
{{define "IsAdminDashboard"}}true{{end}}

{{define "content"}}
<div class="grid gap-4 md:grid-cols-3 mb-8">
    <div class="bg-white border border-gray-200 rounded p-4">
        <p class="text-sm text-gray-500 mb-1">Totala användare</p>
        <p class="text-3xl font-bold">{{.UserCount}}</p>
    </div>
    <div class="bg-white border border-gray-200 rounded p-4">
        <p class="text-sm text-gray-500 mb-1">Administratörer</p>
        <p class="text-3xl font-bold">{{.AdminCount}}</p>
    </div>
    <div class="bg-white border border-gray-200 rounded p-4">
        <p class="text-sm text-gray-500 mb-1">Aktiva</p>
        <p class="text-3xl font-bold">{{.ActiveCount}}</p>
    </div>
</div>

<h2 class="text-lg font-semibold mb-3">Senaste inloggningar</h2>
{{if .RecentLogins}}
<table class="w-full text-sm">
    <thead>
        <tr class="text-left text-gray-500 border-b">
            <th class="pb-2">Namn</th>
            <th class="pb-2">E-post</th>
            <th class="pb-2">Senast inloggad</th>
            <th class="pb-2">Roll</th>
        </tr>
    </thead>
    <tbody>
        {{range .RecentLogins}}
        <tr class="border-b border-gray-100">
            <td class="py-2">{{.Name}}</td>
            <td class="py-2">{{.Email}}</td>
            <td class="py-2">
                {{if .LastLogin}}
                    {{.LastLogin.Format "2006-01-02 15:04"}}
                {{else}}
                    <span class="text-gray-400">Aldrig</span>
                {{end}}
            </td>
            <td class="py-2">
                {{if .IsAdmin}}
                    <span class="bg-green-100 text-green-800 px-2 py-0.5 rounded text-xs">Admin</span>
                {{else}}
                    <span class="text-gray-500">Användare</span>
                {{end}}
            </td>
        </tr>
        {{end}}
    </tbody>
</table>
{{else}}
<p class="text-gray-500">Inga inloggningar registrerade.</p>
{{end}}

<h2 class="text-lg font-semibold mb-3 mt-8">Senaste admin-åtgärder</h2>
{{if .RecentActions}}
<!-- Render recent admin actions -->
{{end}}
{{end}}
```

### Users Template (`admin/users.html`)

```html
{{template "admin/layout.html" .}}
{{define "page_title"}}Användare — Svenskt Vin{{end}}
{{define "IsAdminUsers"}}true{{end}}

{{define "content"}}
<!-- Invite section -->
<div class="bg-white border border-gray-200 rounded p-4 mb-6">
    <h2 class="text-lg font-semibold mb-3">Bjud in användare</h2>
    <form hx-post="/admin/users/invite" hx-swap="outerHTML" class="flex gap-3 items-end">
        <div class="flex-1">
            <label class="block text-sm text-gray-600 mb-1">E-postadress</label>
            <input type="email" name="email" required
                   class="w-full p-2 border border-gray-300 rounded text-sm" />
        </div>
        <div>
            <label class="block text-sm text-gray-600 mb-1">Roll</label>
            <select name="role" class="p-2 border border-gray-300 rounded text-sm">
                <option value="owner">Ägare</option>
                <option value="editor">Redigerare</option>
            </select>
        </div>
        <input type="hidden" name="csrf_token" value="{{.CSRFToken}}" />
        <button type="submit"
                class="px-4 py-2 bg-[#2d6a2d] text-white rounded text-sm cursor-pointer">
            Skicka inbjudan
        </button>
    </form>
    <div id="invite-result"></div>
</div>

<!-- User list -->
<table class="w-full text-sm bg-white border border-gray-200 rounded">
    <thead>
        <tr class="text-left text-gray-500 border-b">
            <th class="p-3">Namn</th>
            <th class="p-3">E-post</th>
            <th class="p-3">Roll</th>
            <th class="p-3">Status</th>
            <th class="p-3">Senast inloggad</th>
            <th class="p-3">Åtgärder</th>
        </tr>
    </thead>
    <tbody>
        {{range .Users}}
        <tr class="border-b border-gray-100">
            <td class="p-3"><a href="/admin/users/{{.ID}}" class="text-[#2d6a2d] no-underline hover:underline">{{.Name}}</a></td>
            <td class="p-3">{{.Email}}</td>
            <td class="p-3">
                {{if .IsAdmin}}
                    <span class="bg-green-100 text-green-800 px-2 py-0.5 rounded text-xs">Admin</span>
                {{else}}
                    <span class="text-gray-500">Användare</span>
                {{end}}
            </td>
            <td class="p-3">
                {{if .Active}}
                    <span class="text-green-600">● Aktive</span>
                {{else}}
                    <span class="text-red-600">● Inaktiv</span>
                {{end}}
            </td>
            <td class="p-3">
                {{if .LastLogin}}
                    {{.LastLogin.Format "2006-01-02"}}
                {{else}}
                    <span class="text-gray-400">—</span>
                {{end}}
            </td>
            <td class="p-3">
                <a href="/admin/users/{{.ID}}"
                   class="text-sm text-[#2d6a2d] no-underline hover:underline">Redigera →</a>
            </td>
        </tr>
        {{end}}
    </tbody>
</table>
{{end}}
```

### User Detail Template (`admin/user_detail.html`)

```html
{{template "admin/layout.html" .}}
{{define "page_title"}}{{.User.Name}} — Användarinställningar{{end}}
{{define "IsAdminUsers"}}true{{end}}

{{define "content"}}
<div class="max-w-xl">
    <div class="bg-white border border-gray-200 rounded p-6 mb-6">
        <h2 class="text-lg font-semibold mb-4">Användarinformation</h2>
        <dl class="space-y-3">
            <dt class="text-sm text-gray-500">Namn</dt>
            <dd class="font-medium">{{.User.Name}}</dd>

            <dt class="text-sm text-gray-500">E-post</dt>
            <dd class="font-medium">{{.User.Email}}</dd>

            <dt class="text-sm text-gray-500">Roll</dt>
            <dd>
                {{if .User.IsAdmin}}
                    <span class="bg-green-100 text-green-800 px-2 py-0.5 rounded text-sm">Administratör</span>
                {{else}}
                    <span class="text-gray-500">Användare</span>
                {{end}}
            </dd>

            <dt class="text-sm text-gray-500">Status</dt>
            <dd>
                {{if .User.Active}}
                    <span class="text-green-600">Aktive</span>
                {{else}}
                    <span class="text-red-600">Inaktiv</span>
                {{end}}
            </dd>

            <dt class="text-sm text-gray-500">Skapad</dt>
            <dd>{{.User.CreatedAt.Format "2006-01-02 15:04"}}</dd>

            <dt class="text-sm text-gray-500">Senast inloggad</dt>
            <dd>
                {{if .User.LastLogin}}
                    {{.User.LastLogin.Format "2006-01-02 15:04"}}
                {{else}}
                    <span class="text-gray-400">Aldrig inloggad</span>
                {{end}}
            </dd>
        </dl>
    </div>

    <div class="bg-white border border-gray-200 rounded p-6">
        <h2 class="text-lg font-semibold mb-4">Åtgärder</h2>
        <form method="POST" class="space-y-3">
            <input type="hidden" name="csrf_token" value="{{.CSRFToken}}" />

            {{if .User.Active}}
            <button type="submit" name="action" value="deactivate"
                    class="px-4 py-2 bg-red-600 text-white rounded text-sm cursor-pointer">
                Inaktivera konto
            </button>
            {{else}}
            <button type="submit" name="action" value="activate"
                    class="px-4 py-2 bg-green-600 text-white rounded text-sm cursor-pointer">
                Aktivera konto
            </button>
            {{end}}

            <button type="submit" name="action" value="reset_password"
                    class="px-4 py-2 bg-white border border-gray-300 text-gray-700 rounded text-sm cursor-pointer">
                Återställ lösenord
            </button>

            {{if not .User.IsAdmin}}
            <button type="submit" name="action" value="toggle_admin"
                    class="px-4 py-2 bg-white border border-gray-300 text-gray-700 rounded text-sm cursor-pointer">
                Gör till administratör
            </button>
            {{else}}
            <button type="submit" name="action" value="toggle_admin"
                    class="px-4 py-2 bg-white border border-yellow-300 text-yellow-700 rounded text-sm cursor-pointer">
                Ta bort administratörsrättigheter
            </button>
            {{end}}
        </form>
    </div>
</div>
{{end}}
```

### Forbidden Template (`admin/forbidden.html`)

```html
{{template "admin/layout.html" .}}
{{define "page_title"}}Åtkomst nekad{{end}}

{{define "content"}}
<div class="max-w-md text-center py-12">
    <h1 class="text-3xl mb-4">Åtkomst nekad</h1>
    <p class="text-gray-600 mb-6">{{.Error}}</p>
    <a href="/vineyard" class="text-[#2d6a2d] no-underline">← Tillbaka till vingården</a>
</div>
{{end}}
```

---

## Nav Integration

The existing nav template (`internal/templates/nav.html`) needs to show an admin link when the user is an admin.

**Modified `nav.html`** — add after the owner settings link:

```html
{{if .IsAdmin}}
    <div class="flex gap-1 items-center">
        <a href="/admin"
           class="text-gray-500 text-sm no-underline px-3 py-2 rounded hover:bg-gray-100">
            ⚙️ Admin
        </a>
    </div>
{{end}}
```

The `IsAdmin` flag is passed from each handler's data map. For the admin dashboard handlers, `IsAdmin: true` is always set. For regular vineyard handlers, it's set based on `user.IsAdmin`.

---

## HTMX Integration Strategy

All admin actions use HTMX for SPA-like navigation:

| Action | Method | URL | Swap |
|--------|--------|-----|------|
| Generate invite | POST | `/admin/users/invite` | `outerHTML` on `#invite-result` |
| Deactivate user | POST | `/admin/users/:id` | `innerHTML` on `#user-detail` |
| Activate user | POST | `/admin/users/:id` | `innerHTML` on `#user-detail` |
| Reset password | POST | `/admin/users/:id` | `innerHTML` on `#user-detail` |
| Toggle admin | POST | `/admin/users/:id` | `innerHTML` on `#user-detail` |

**Flash messages** for admin actions: Use the existing `{{template "flash" .}}` partial. Set flash via session cookie or query param (`?flash=success&message=...`).

---

## SMTP Dependency Handling

The admin invite flow must handle two SMTP states gracefully:

### Case 1: SMTP configured and working
- Invite email sent successfully
- Admin sees: "✅ Inbjudan skickad till <email>"

### Case 2: SMTP not configured or failed
- Create `pending_invite` record anyway
- Admin sees the invite link with a **copy-to-clipboard** button:
  ```
  ⚠ SMTP inte konfigurerad. Kopiera länken och skicka manuellt:
  [http://localhost:8080/invite/confirm?token=abc123  📋 Kopiera]
  ```

### Implementation
```go
// In HandleInviteGeneratePOST:
emailErr := h.emailSender.SendInviteWithEmail(email, appHost, vineyardName, token)

if emailErr != nil {
    // SMTP failed — show manual link
    data.EmailSent = false
    data.InviteURL = fmt.Sprintf("%s/invite/confirm?token=%s", appHost, token)
} else {
    data.EmailSent = true
    data.InviteURL = ""
}
```

The invite link format reuses the existing `/invite/confirm` endpoint that the invitee visits.

---

## Session Cookie Integration

Our recent fix centralized cookie configuration via `config.CookieConfig`. The admin dashboard uses the same session cookie (`session_id`) as all other authenticated pages:

- **Cookie name**: `session_id`
- **Cookie path**: `/`
- **HttpOnly**: `true`
- **Secure**: depends on `APP_ENV` (false for dev, true for prod)
- **SameSite**: `Lax` (default) or configured via YAML

No changes needed — the admin dashboard simply reuses the existing cookie-based session.

---

## Integration Points

### 1. With Existing Invite Flow

The admin's "generate invite" action reuses the existing `pending_invites` table and `SendInviteWithEmail` method. The invitee experiences the same `/invite/confirm` flow regardless of whether the invite was admin-generated or auto-generated through the membership request flow.

**Key difference**: Admin-generated invites specify a role (`owner` or `editor`). The existing membership request flow doesn't pre-assign a role.

### 2. With Magic Link Authentication

Password reset for admin-managed users creates a `magic_link_token` record, which the existing `/auth/set-password` endpoint handles. The admin can either:
- Generate the magic link URL and show it to copy
- (Future) Send it via email

### 3. With Session Management

When an admin deactivates a user, `DeleteSessionsByUser` is called to invalidate all their sessions. The user is immediately logged out on the next request.

### 4. With CSRF Protection

All admin POST forms include the CSRF token, validated via the existing `validateCSRFToken` function from `auth.go`.

---

## Route Registration

**File**: `cmd/web/main.go` (modify)

```go
// In main(), after initializing handlers:

adminHandler := pages.NewAdminHandler(
    store, sessionMgr, cfg.Cookie, emailSender, cfg.AppHost, templates,
)

// Admin routes (require admin authentication)
mux.HandleFunc("GET /admin", adminHandler.RequireAdmin(adminHandler.HandleDashboardGET()))
mux.HandleFunc("GET /admin/", adminHandler.RequireAdmin(adminHandler.HandleDashboardGET()))
mux.HandleFunc("GET /admin/users", adminHandler.RequireAdmin(adminHandler.HandleUsersGET()))
mux.HandleFunc("GET /admin/users/invite", adminHandler.RequireAdmin(adminHandler.HandleUsersGET()))
mux.HandleFunc("GET /admin/vineyard", adminHandler.RequireAdmin(adminHandler.HandleVineyardGET()))
mux.HandleFunc("GET /admin/users/{id}", adminHandler.RequireAdmin(adminHandler.HandleUserDetailGET()))
mux.HandleFunc("POST /admin/users/{id}", adminHandler.RequireAdmin(adminHandler.HandleUserDetailPOST()))
mux.HandleFunc("POST /admin/users/invite", adminHandler.RequireAdmin(adminHandler.HandleInviteGeneratePOST()))
```

**Note on Go 1.22+ routing**: The `/{id}` syntax works in Go's `net/http` mux. For the `POST /admin/users/{id}` route, the `{id}` segment captures the user ID.

---

## Migration Plan

### Phase 1: Database Migration
1. Create `db/migrations/020_admin_dashboard.sql` — adds `admin_actions` table
2. Ensure `users.created_at` exists (it should from migration 002)
3. Ensure `pending_invites.role` constraint includes needed values

### Phase 2: Store Layer
1. Add `ListAllUsers()`, `GetFirstAdmin()`, `UpdateUserActive()`, `CountUsers()` to `db.Store`
2. Add `CreatedAt` field to `User` struct if not present

### Phase 3: Bootstrap CLI
1. Create `cmd/web/admin_bootstrap.go`
2. Add subcommand detection in `main.go`
3. Implement idempotent admin creation logic

### Phase 4: Admin Middleware
1. Add `RequireAdmin` method to `SessionManager` in `auth/session.go`
2. Add `IsAdmin` flag to template data in all page handlers

### Phase 5: Admin Templates
1. Create `internal/templates/admin/` directory
2. Create `layout.html`, `dashboard.html`, `users.html`, `user_detail.html`
3. Create `forbidden.html` for 403 responses

### Phase 6: Admin Handler
1. Create `internal/handlers/pages/admin.go`
2. Implement all handler methods
3. Wire up route registration in `main.go`

### Phase 7: Nav Integration
1. Modify `internal/templates/nav.html` to show admin link
2. Ensure `IsAdmin` flag is set in all page handler data maps

### Phase 8: Testing
1. Test bootstrap CLI end-to-end
2. Test admin middleware (403 for non-admin)
3. Test invite generation with and without SMTP
4. Test user deactivation (session invalidation)
5. Test password reset flow

---

## File Summary

### New Files (8)
| File | Purpose | Lines (est.) |
|------|---------|--------------|
| `cmd/web/admin_bootstrap.go` | Bootstrap CLI command | ~80 |
| `internal/handlers/pages/admin.go` | Admin handler with all endpoints | ~350 |
| `internal/templates/admin/layout.html` | Admin layout with sidebar | ~50 |
| `internal/templates/admin/dashboard.html` | Dashboard overview | ~40 |
| `internal/templates/admin/users.html` | User list table | ~50 |
| `internal/templates/admin/user_detail.html` | User edit detail | ~60 |
| `internal/templates/admin/invite_result.html` | HTMX invite result fragment | ~20 |
| `internal/templates/admin/forbidden.html` | 403 page | ~15 |
| `db/migrations/020_admin_dashboard.sql` | Admin actions audit table | ~25 |

### Modified Files (4)
| File | Change | Lines (est.) |
|------|--------|--------------|
| `cmd/web/main.go` | Add admin handler init + route registration + subcommand detection | ~30 |
| `internal/auth/session.go` | Add `RequireAdmin` middleware | ~25 |
| `internal/db/store.go` | Add 4 new store methods | ~60 |
| `internal/db/models.go` | Add `CreatedAt` to `User` struct (if missing) | ~3 |
| `internal/templates/nav.html` | Add admin nav link | ~5 |
| `internal/handlers/pages/auth.go` | Add `IsAdmin` flag to template data | ~10 |

**Total: ~750 lines of new code, ~130 lines modified.**

---

## Risks & Trade-offs

### 1. Role Constraint Extension
**Risk**: The `pending_invites.role` CHECK constraint currently only allows `('owner', 'editor')`. If MVP+ needs `viewer`, the migration must drop and recreate the constraint.
**Mitigation**: MVP only offers `owner` and `editor`. The admin UI reflects this.

### 2. SMTP Failure Visibility
**Risk**: Admin may not notice that invite emails are failing.
**Mitigation**: When SMTP fails, the admin sees the invite link with a copy button. The `invite_result.html` fragment clearly indicates whether the email was sent or needs manual distribution.

### 3. Password Hash vs Magic Link
**Risk**: Bootstrap creates a user with a password hash. The existing authentication flow supports both magic-link (NULL password_hash) and password-based auth (non-NULL password_hash). This dual mode is already tested in the existing codebase.
**Mitigation**: No new auth code needed. The existing `doLogin` in auth.go already handles both cases.

### 4. Session Invalidation on Deactivation
**Risk**: Deactivating a user who is currently logged in should immediately revoke their session.
**Mitigation**: `UpdateUserActive` calls `DeleteSessionsByUser` atomically. The `VerifySession` query already filters by `u.active = true`.

### 5. CSRF in Admin Context
**Risk**: Admin actions are destructive (deactivate users, change roles). CSRF protection is critical.
**Mitigation**: All admin POST forms include CSRF tokens, validated via the existing `validateCSRFToken` function.

### 6. Admin Self-Deactivation
**Risk**: Admin deactivates themselves, locking out all admins.
**Mitigation**: The bootstrap CLI command can always recover (re-run with `--password`). In the UI, we could add a confirmation dialog for self-deactivation, but that requires Alpine.js. For MVP, the CLI recovery path is sufficient.

---

## Open Questions

1. **Should the bootstrap command create a default vineyard?** Yes — the spec includes creating "Min vingård" as the default vineyard if none exists.

2. **Should non-admin users see any admin link?** No — the nav template only shows the admin link when `IsAdmin == true`.

3. **Should there be a "reset all passwords" emergency command?** No — the bootstrap command already resets the admin password. Individual password resets are handled via admin user detail page.

4. **Should the admin dashboard use the core-api or the web app?** The web app (`cmd/web/`) is the right layer for HTMX page rendering. The core-api is a separate service. The admin dashboard stays in the web app.

5. **Should invite emails include the vineyard name?** Yes — `SendInviteWithEmail` already takes `vineyardName` as a parameter. The handler looks up the first vineyard's name.

---

## Acceptance Criteria

1. ✅ `go run ./cmd/web/ admin bootstrap --email=admin@example.com --password=Test123!` creates first admin
2. ✅ After bootstrap, the admin can log in at `/login` with email + password
3. ✅ After login, admin sees "⚙️ Admin" link in navigation
4. ✅ `/admin` shows dashboard with user count, admin count, active count, and recent logins
5. ✅ `/admin/users` lists all users with action links
6. ✅ Admin can generate an invite link for a new user with owner/editor role
7. ✅ If SMTP fails, admin sees the invite link to copy manually
8. ✅ Admin can deactivate/reactivate users
9. ✅ Admin can toggle admin role for other users
10. ✅ Non-admin users see 403 at `/admin`
11. ✅ Bootstrap command is idempotent — re-running updates the admin password
12. ✅ Deactivating a user invalidates all their sessions
13. ✅ Default vineyard "Min vingård" is created during bootstrap if none exists
