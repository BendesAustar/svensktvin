# Council Critique: Centralized Cookie Configuration

**Date:** 2026-06-15
**Subject:** Fix P0 — Replace 4 scattered `Secure: true` with centralized config
**Author:** Architectural review by DesignBrain + EngineeringBrain

## Problem

All 4 cookie-setting locations hardcode `Secure: true`:

| # | File | Function | Cookie |
|---|------|----------|--------|
| 1 | `internal/auth/session.go:157` | `SetSessionCookie()` | `session_id` |
| 2 | `internal/auth/session.go:170` | `ClearSessionCookie()` | `session_id` (expired) |
| 3 | `internal/handlers/pages/auth.go:63` | `setCSRFCookie()` | `csrf_token` |
| 4 | `internal/handlers/api/account.go:86` | `setRememberMeCookie()` | `remember_me` |

Browsers **never** send `Secure` cookies over HTTP. Localhost login is impossible.

## Proposed Solution

### 1. Add `CookieConfig` struct to `config.go`

```go
// CookieConfig holds cookie security settings.
type CookieConfig struct {
    Secure bool // false in dev, true in production
}

// Cookie returns the resolved cookie config.
// Auto-detects dev mode from APP_ENV if Cookie.Secure is not explicitly set.
func (c *Config) Cookie() CookieConfig {
    if c.Cookie.Secure == c.Cookie.Secure && os.Getenv("APP_ENV") != "" {
        // explicit config present
    }
    return CookieConfig{
        Secure: c.Cookie.Secure || os.Getenv("APP_ENV") != "development",
    }
}
```

### 2. Pass `CookieConfig` to all cookie-setters

- `SessionManager`: new field `cookieCfg CookieConfig`, updated constructor
- `AuthHandler`: new field `cookieCfg CookieConfig`, updated constructor
- `setCSRFCookie`: takes `secure bool` parameter
- `AccountHandler`: new field `cookieCfg CookieConfig`
- `setRememberMeCookie`: takes `secure bool` parameter

### 3. `main.go` wiring

```go
cookieCfg := cfg.Cookie()
sessionMgr := auth.NewSessionManager(store, cfg.Auth.SessionExpiry, cookieCfg)
authHandler := pages.NewAuthHandler(store, sessionMgr, magicLinkMgr, rateLimiter, cfg, emailSender, cookieCfg)
vineyardHandler := pages.NewVineyardHandler(store, sessionMgr)  // inherits cookieCfg from sessionMgr? No.
```

## Open Questions for Council

1. **Should `CookieConfig` have more fields?** (SameSite, Domain, MaxAge) — Better to plan ahead or keep it minimal for now?
2. **Should `APP_ENV` auto-detection be the default, or explicit-only?** Auto-detection is convenient for dev but explicit config is safer for auditability.
3. **Should `SessionManager` own the cookie config, or should each handler own its own?** If each handler owns it, we get duplication in constructors. If SessionManager owns it, vineyard/harvest handlers (which only use SessionManager) don't need changes.

## Constraints

- Must work for: localhost dev (HTTP), Docker compose (HTTP behind nginx), production HTTPS
- Must not require changes in more than 2 locations for a future config change
- Must compile and build
- Must not change cookie names, SameSite settings, or HttpOnly behavior
