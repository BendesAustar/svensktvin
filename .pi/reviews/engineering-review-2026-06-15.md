# Svenskt Vin — Security & System Architecture Review
**Date:** 2026-06-15  
**Author:** EngineeringBrain (Cinerarium)  
**Scope:** Auth flow, security controls, rate limiting, session management, configuration, deployment pipeline

---

## Executive Summary

The SvensktVin application has solid foundational security: bcrypt hashing, magic link auth, rate limiting, CSRF tokens, and typed DB models. However, there are critical gaps in production readiness:

1. **P0 CRITICAL:** `Secure: true` on all cookies blocks localhost login (confirmed by DesignBrain)
2. **P1 HIGH:** Missing `SESSION_SECRET` despite explicit security requirement in architecture spec
3. **P2 MEDIUM:** No HTTPS enforcement in production configuration
4. **P2 MEDIUM:** Configuration loaded from env vars only — no validation of required fields at startup
5. **P1 HIGH:** No `Content-Security-Policy` header
6. **P2 MEDIUM:** No `X-Frame-Options` or `X-Content-Type-Options` middleware (only set on specific responses)
7. **P3 LOW:** Rate limiting uses in-memory store — not suitable for multi-instance deployment

---

## P0 — CRITICAL: Cookie Secure Flag

**Files:** `internal/auth/session.go:157,170`, `internal/handlers/pages/auth.go:63`, `internal/handlers/api/account.go:86`

### The Bug
All cookies (`csrf_token`, `session_id`, `remember_me`) are set with `Secure: true`, which means browsers **never** send them over HTTP connections.

### Reproduction
```bash
curl -s http://localhost:8090/login -c /tmp/cookies.txt  # Sets csrf_token with Secure flag
curl -s -b /tmp/cookies.txt -d "action=login_password&email=x@y.se&password=z&csrf_token=abc" http://localhost:8090/login
# → 400 "Ogiltig begäran." — csrf_token cookie never sent over HTTP
```

### Fix
```go
// In all cookie-setting locations:
secure := os.Getenv("APP_ENV") != "development"
http.SetCookie(w, &http.Cookie{
    Name:     "session_id",
    Value:    sessionID,
    Path:     "/",
    HttpOnly: true,
    Secure:   secure,  // ← dynamic based on env
    SameSite: http.SameSiteLaxMode,
    Expires:  time.Now().Add(sm.sessionExpiry),
})
```

**Locations to fix:**
1. `internal/auth/session.go` — `SetSessionCookie()` (line ~157)
2. `internal/auth/session.go` — `ClearSessionCookie()` (line ~170)
3. `internal/handlers/pages/auth.go` — `setCSRFCookie()` (line ~63)
4. `internal/handlers/api/account.go` — `setRememberMeCookie()` (line ~86)

### Risk
- **Without fix:** Zero users can log in on localhost. Beta testing impossible.
- **With fix:** Development workflow enabled; production HTTPS continues to work.

---

## P1 — HIGH: Missing SESSION_SECRET

**File:** `cmd/web/main.go` startup, `.env.example`

### Finding
The application logs `WARN svensktvin: no SESSION_SECRET set` but continues to run. The architecture spec says:

> "SESSION_SECRET — 256-bit secret for cookie encryption"

However, there's no startup validation that this is set. The session manager falls back to a zero-value secret, which means:
- All sessions are cryptographically identical across restarts
- Session tokens are predictable if the algorithm is known

### Fix
```go
if os.Getenv("SESSION_SECRET") == "" {
    slog.Fatal("SESSION_SECRET must be set in production")
}
```

For development, generate one:
```bash
SESSION_SECRET=$(openssl rand -hex 32)
```

---

## P1 — HIGH: Missing Content-Security-Policy

**File:** `internal/handlers/pages/auth.go` middleware

### Finding
No CSP header is set on any response. The app loads scripts from CDN (HTMX, Alpine.js, Tailwind Play). Without CSP, these are vulnerable to injection attacks.

### Current Script Loading
```html
<script src="https://unpkg.com/htmx.org@2.0.4" ...></script>
<script src="https://cdn.jsdelivr.net/npm/@alpinejs/intersect@3.14.8/dist/cdn.min.js"></script>
<script src="https://cdn.tailwindcss.com"></script>
```

### Fix
```go
w.Header().Set("Content-Security-Policy", 
    "default-src 'self'; "+
    "script-src 'self' https://unpkg.com https://cdn.jsdelivr.net 'unsafe-inline'; "+
    "style-src 'self' https://cdn.jsdelivr.net 'unsafe-inline'; "+
    "connect-src 'self' https://api.open-meteo.com; "+
    "img-src 'self' data: https:; "+
    "frame-ancestors 'none'; "+
    "form-action 'self';")
```

---

## P2 — MEDIUM: Missing X-Frame-Options

**File:** `internal/handlers/pages/auth.go`

### Finding
`X-Content-Type-Options: nosniff` is set on individual responses (error pages, file servers) but not as a middleware applied to all responses.

`X-Frame-Options: DENY` is not set anywhere, leaving the app vulnerable to clickjacking.

### Fix
```go
// In authMiddleware:
w.Header().Set("X-Content-Type-Options", "nosniff")
w.Header().Set("X-Frame-Options", "DENY")
w.Header().Set("X-XSS-Protection", "0") // Modern browsers use CSP instead
```

---

## P2 — MEDIUM: Configuration Validation

**File:** `cmd/web/main.go` startup

### Finding
The app starts even with missing required configuration. Only `SESSION_SECRET` shows a warning. No validation of `DATABASE_URL`, `SMTP_HOST`, or other required fields.

### Fix
```go
requiredVars := []string{
    "DATABASE_URL",
    "SESSION_SECRET",
    "SMTP_HOST",
    "SMTP_USER",
    "SMTP_PASS",
}
for _, v := range requiredVars {
    if os.Getenv(v) == "" {
        slog.Warn("missing required env var", "var", v)
    }
}
```

---

## P2 — MEDIUM: Rate Limiting Storage

**File:** `internal/auth/ratelimit.go`

### Finding
Uses an in-memory map for rate limiting:

```go
type RateLimiter struct {
    mu      sync.Mutex
    users   map[string]*userRate
}
```

This is fine for single-instance deployment but means:
- Rate limits are not shared across multiple app instances
- Restarting the app clears all rate limit counters
- Load balancing could allow bypassing limits

### Current State
The architecture spec says "in-memory rate limiter" — so this is intentional for now.

### Recommendation
Keep in-memory for now. Flag as future work: "Upgrade to Redis-based rate limiting for multi-instance deployments."

---

## P2 — MEDIUM: Docker Build Pipeline

**File:** `Dockerfile`, `docker-compose.yml`, `.env.example`

### Findings
1. Dockerfile uses `CGO_ENABLED=0` — good for static binary
2. No `.dockerignore` specified — may include `.git`, `node_modules`, etc.
3. `docker-compose.yml` references `postgres:17-alpine` but local dev uses different port
4. No healthcheck defined in docker-compose

### Recommendations
```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
    CMD wget -qO- http://localhost:8090/health || exit 1
```

```yaml
# docker-compose.yml
services:
  app:
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8090/health"]
      interval: 30s
      timeout: 3s
      retries: 3
```

---

## P3 — LOW: Password Hashing Round Count

**File:** `internal/auth/password.go`

### Finding
Uses `bcrypt.DefaultCost` (10) which is the Go bcrypt default. The architecture spec says "bcrypt ≥12 rounds for production."

### Current Code
```go
func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(bytes), err
}
```

### Fix
```go
const bcryptCost = 12 // ≥12 per security spec

func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
    return string(bytes), err
}
```

---

## P3 — LOW: Missing HTTPS Enforcement

**File:** `cmd/web/main.go` server config

### Finding
No HTTP → HTTPS redirect in production. The server listens on plain HTTP by default.

### Fix
Add a secondary HTTP server that redirects to HTTPS:
```go
go func() {
    http.ListenAndServe(":8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Location", "https://"+r.Host+r.URL.Path)
        w.WriteHeader(http.StatusMovedPermanently)
    }))
}()
```

---

## Summary Table

| Priority | Issue | File | Fix Effort |
|----------|-------|------|------------|
| P0 | Cookie `Secure` flag | 4 files | 30 min |
| P1 | Missing `SESSION_SECRET` validation | `cmd/web/main.go` | 15 min |
| P1 | Missing CSP header | `internal/handlers/pages/auth.go` | 30 min |
| P2 | Missing `X-Frame-Options` | Same | 15 min |
| P2 | Config validation | `cmd/web/main.go` | 30 min |
| P2 | Docker healthcheck | `Dockerfile`, `docker-compose.yml` | 30 min |
| P2 | Rate limiter in-memory | `internal/auth/ratelimit.go` | Future work |
| P3 | bcrypt cost < 12 | `internal/auth/password.go` | 15 min |
| P3 | No HTTPS redirect | `cmd/web/main.go` | 30 min |

**Total estimated effort for P0-P2 fixes: ~3 hours**

---

## Architecture Review: Go + HTMX Pattern

### Current Architecture
```
cmd/web/main.go          → Single entry point, HTTP server
internal/auth/           → Session, magic link, password, rate limit
internal/handlers/       → Auth handlers (auth.go), vineyard handlers
internal/templates/      → 16 templates, standalone pattern
internal/db/             → Typed models, store pattern with query helpers
internal/config/         → YAML + env config
internal/email/          → SMTP email sender
static/css/              → Minimal CSS
```

### Strengths
1. **Single binary** — simple deployment
2. **Typed DB models** — query safety
3. **CSRF + SameSite cookies** — solid CSRF protection
4. **HTMX integration** — progressive enhancement, minimal JS
5. **Alpine.js for reactivity** — variety search, harvest countdown
6. **Magic link primary auth** — no password storage for primary flow
7. **Password fallback** — good for admin/recovery scenarios

### Gaps
1. **No template inheritance** — 399 lines of boilerplate duplication
2. **No HTTPS** — must be behind reverse proxy (nginx/cloudflare)
3. **No metrics** — no Prometheus, no request timing
4. **No request ID** — hard to trace requests in logs
5. **No graceful shutdown timeout** — `SIGTERM` handler exists but no deadline

### Recommended Docker Architecture
```
Internet → Cloudflare/nginx (TLS termination) → SvensktVin Go binary (:8090)
                                                   ↓
                                             PostgreSQL :5432 (separate)
```

The Go binary should listen on **HTTP only** (behind TLS-terminating proxy). The `Secure` flag fix enables localhost HTTP development without breaking production HTTPS.
