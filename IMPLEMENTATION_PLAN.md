# Implementation Plan: Svenskt Vin HTMX Migration

## Overview

Complete restructure of the Svenskt Vin application from a dual-runtime architecture (Go REST API + SvelteKit frontend) into a single Go binary serving Go HTML templates with HTMX for partial updates and Alpine.js for client-side state. The existing Go API layer (`core-api/`) — 2,495 lines across 11 handler/middleware/routing files — is restructured from JSON-only API handlers into a hybrid template + JSON renderer. The SvelteKit frontend (~4,950 lines across ~20 pages) is discarded entirely, with all logic consolidated into Go templates and handlers. Database schema, migrations (19 files), and seeds (5 files) remain unchanged. Target: single Dockerfile, single Go binary on port 8080, zero Node.js dependency.

## Dependencies

- **PostgreSQL 16 with PostGIS** — existing DB container unchanged
- **Nominatim API** — for `/api/geo/reverse` GPS→location lookup
- **NPM/Tailwind CLI** — one-time `npx tailwindcss -o static/css/app.css` for production CSS build (not required in dev)
- **DesignBrain ARCHITECTURE.md** — complete template examples and route mapping reference

## Tasks

### Phase 1 — Foundation

#### Task P1-1
**Type:** infra
**Component:** infra
**Prerequisites:** none
**Description:** Create new project directory structure: `cmd/web/`, `internal/auth/`, `internal/handlers/` (pages + api subdirs), `internal/templates/` (with all subdirectories), `static/css/`, `internal/db/`, `internal/config/`
**Verification:** `find . -type d | grep -E '(cmd|internal|static)' | sort`
**Rollback:** `rm -rf cmd/ internal/ static/`

#### Task P1-2
**Type:** implementation
**Component:** Go
**Prerequisites:** P1-1
**Description:** Port existing Go DB layer — copy `internal/db/pool.go` and `db/store.go` (if any store methods exist) from `core-api/internal/db/` to new `internal/db/`. Create a `Store` wrapper in the new package that provides typed query helpers (GetVineyard, ListBlocks, CreateBlock, etc.) by refactoring query patterns from the existing handler files into the store layer.
**Verification:** `cd /home/neurograft/Techstack/svensktvin && go build ./internal/db/`
**Rollback:** Delete the new `internal/db/` package and its files

#### Task P1-3
**Type:** implementation
**Component:** Go
**Prerequisites:** P1-2
**Description:** Port existing auth logic into `internal/auth/` — port session management from `core-api/internal/api/middleware.go` `requireAuth` pattern and SvelteKit `lib/server/auth.ts` (password verify + session create) into Go files. Port magic link token generation/verification from `core-api/internal/api/handlers_auth.go` (`sendMagicLink`, `verifyMagicLink`) into `auth/magic_link.go`. Port bcrypt verify/hash from SvelteKit `lib/server/auth.ts` into `auth/password.go`. Port `randomHex` utility. Port rate limiter from `middleware.go` into a standalone `internal/ratelimit/` package.
**Verification:** `go vet ./internal/auth/` and `go vet ./internal/ratelimit/`
**Rollback:** Delete the new auth packages

#### Task P1-4
**Type:** implementation
**Component:** Go
**Prerequisites:** P1-2, P1-3
**Description:** Create `cmd/web/main.go` — entry point that loads config, connects to DB, instantiates `Store`, loads all Go templates at startup with `html/template.New()`, builds the `http.ServeMux` router with all route registrations, creates `http.Server` on port 8080, and sets up graceful shutdown on SIGINT/SIGTERM. Port the existing `main.go` structure but remove CORS middleware and replace JSON router with template-aware handler registrations.
**Verification:** `go build ./cmd/web/ && ./svensktvin --help` (should print port 8080)
**Rollback:** Delete `cmd/web/main.go` and the new main package

#### Task P1-5
**Type:** implementation
**Component:** Go
**Prerequisites:** P1-4
**Description:** Create `internal/config/config.go` — reuse the existing `core-api/internal/config/config.go` Config struct (API port, database URL, auth session expiry, rate limits). Update to add new config fields needed for the template-based approach: `SessionSecret`, `SMTP` fields (`Host`, `User`, `Pass`, `From`), `APP_HOST`, and template render mode flag. Update `Load()` to read from environment variables as fallback.
**Verification:** `go vet ./internal/config/`
**Rollback:** Delete the new config package

### Phase 2 — Auth Pages

#### Task P2-6
**Type:** implementation
**Component:** Go
**Prerequisites:** P1-4, P1-5
**Description:** Implement `handleLoginGET` in `internal/handlers/pages/auth.go` — renders `auth/login.html` template. Handle `invite_token` query parameter. Check if user is already authenticated → redirect to `/vineyard`. Set up CSRF token generation. Load vineyard context if invite token is present.
**Verification:** `curl -s http://localhost:8080/login | grep -q "Svenskt Vin" && echo "PASS" || echo "FAIL"`
**Rollback:** Delete handler function and route registration

#### Task P2-7
**Type:** implementation
**Component:** Go
**Prerequisites:** P2-6
**Description:** Implement `handleLoginPOST` in `auth.go` — processes two actions: `login_password` (email + password → verify with bcrypt → create session cookie → redirect) and `request_membership` (email + name → insert into pending_invites → send confirmation email → show success page). Validate CSRF token. Set flash messages. Return appropriate HX-Redirect or re-render form with errors.
**Verification:** `curl -s -X POST http://localhost:8080/login -d 'action=login_password&email=test@example.com&password=wrong' -w '%{http_code}'` → expect 400 or 401
**Rollback:** Delete handler, revert route registration

#### Task P2-8
**Type:** implementation
**Component:** Go
**Prerequisites:** P2-6
**Description:** Implement `handleLogoutPOST` in `auth.go` — delete session cookie from database (invalidate session_id), clear session cookie from response (httpOnly, secure, SameSite=Lax, path=/), set flash message "Du har loggats ut", redirect to `/login` with HX-Redirect header.
**Verification:** After login, `curl -s -X POST http://localhost:8080/logout -w '%{http_code}'` → 303 redirect to /login
**Rollback:** Delete handler, revert route

#### Task P2-9
**Type:** implementation
**Component:** Go
**Prerequisites:** P2-6
**Description:** Implement `handleRegisterGET/POST` in `auth.go` — GET validates invite token from query string, renders register form with pre-filled data if token is valid. POST creates user with email, name, password (bcrypt ≥ 12), creates session, sets cookie, redirects to first vineyard or landing. If no invite token, show "register with invite" error.
**Verification:** `curl -s http://localhost:8080/register?token=validtoken | grep -q "Skapa konto" && echo "PASS"`
**Rollback:** Delete handler, revert route

#### Task P2-10
**Type:** implementation
**Component:** Go
**Prerequisites:** P2-6, P2-7
**Description:** Implement `handleForgotPasswordGET/POST` in `auth.go` — GET renders forgot-password form. POST generates magic link token (random 32-byte hex, SHA256 hash stored), upserts user if email not found (account enumeration safe), inserts magic_link_token into DB with 15-minute TTL, sends email via `net/smtp` (or logs if SMTP not configured), returns success page with "check your email" message.
**Verification:** `curl -s -X POST http://localhost:8080/auth/forgot-password -d 'email=test@example.com' | grep -q "inloggningslänk" && echo "PASS"`
**Rollback:** Delete handler, revert route

#### Task P2-11
**Type:** implementation
**Component:** Go
**Prerequisites:** P2-6
**Description:** Implement `handleSetPasswordGET/POST` in `auth.go` — GET validates token from query string, renders set-password form with password + confirm fields (HTML5 minlength="8"). POST validates password match, hashes with bcrypt ≥ 12, updates user's `password_hash`, invalidates all sessions for the user, creates new session, sets cookie, redirects to vineyard.
**Verification:** `curl -s http://localhost:8080/auth/set-password?token=token123 | grep -q "Lösenord" && echo "PASS"`
**Rollback:** Delete handler, revert route

#### Task P2-12
**Type:** implementation
**Component:** Go
**Prerequisites:** P2-6
**Description:** Implement `handleInviteConfirmGET/POST` in `auth.go` — GET validates pending_invite token, shows confirmation page. POST accepts the invite, creates vineyard_member record with appropriate role, redirects to vineyard dashboard with success flash.
**Verification:** `curl -s http://localhost:8080/invite/confirm?token=t | grep -q "Bekräfta" && echo "PASS"`
**Rollback:** Delete handler, revert route

### Phase 3 — Vineyard Dashboard

#### Task P3-13
**Type:** implementation
**Component:** Go
**Prerequisites:** P1-4
**Description:** Implement `handleLandingGET` in `internal/handlers/pages/vineyard.go` — renders landing page showing list of user's vineyards (or redirect to first vineyard if exactly one). For unauthenticated users, show login/register CTAs. For authenticated users, show vineyard list or redirect.
**Verification:** After login, `curl -s http://localhost:8080/ | grep -q "Svenskt Vin" && echo "PASS"`
**Rollback:** Delete handler, revert route

#### Task P3-14
**Type:** implementation
**Component:** Go
**Prerequisites:** P1-4, P3-13
**Description:** Implement `handleVineyardGET` in `vineyard.go` — renders `vineyard/dashboard.html`. Queries vineyard details (name, county, municipality, organic, biodynamic, established_year, total_area_ha), user's role, blocks with latest harvest and variety info, benchmark teaser (if any harvest exists for this county). Passes all data to template.
**Verification:** `curl -s http://localhost:8080/vineyard/1 | grep -q "Block" && echo "PASS"`
**Rollback:** Delete handler, revert route

#### Task P3-15
**Type:** implementation
**Component:** Go
**Prerequisites:** P3-14
**Description:** Implement `handleBenchmarkGET` in `vineyard.go` — renders `vineyard/benchmark.html`. Queries three datasets: user yields (aggregated by variety + year), regional benchmarks (county-level averages, min 3 vineyards), timeline (chronological harvest records). Passes all to template with 3 tables.
**Verification:** `curl -s http://localhost:8080/vineyard/1/benchmark | grep -q "Jämförelse" && echo "PASS"`
**Rollback:** Delete handler, revert route

#### Task P3-16
**Type:** implementation
**Component:** templates
**Prerequisites:** P1-4
**Description:** Create `internal/templates/base.html` — full HTML5 document with Go template inheritance (`{{define}}` blocks for title, content, head, scripts). Include HTMX CDN (`htmx.org@2.x`), Alpine.js CDN (`alpinejs@3.x`), Tailwind CDN (`cdn.tailwindcss.com`), custom CSS link. Render nav conditionally (logged-in vs logged-out). Render cookie consent bar with Alpine.js `x-data`. Define nav, flash, form-errors as template includes.
**Verification:** `go build ./cmd/web/ && echo "Templates parse OK"` (Go will error on template parse failure at startup)
**Rollback:** Delete `internal/templates/base.html`

#### Task P3-17
**Type:** implementation
**Component:** templates
**Prerequisites:** P3-16
**Description:** Create `internal/templates/nav.html` — navigation partial with logged-in links: vineyard home, skörd, jämförelse (benchmark), inställningar (settings, owner-only). Active state based on URL comparison.
**Verification:** Template parses correctly: included in `go build`
**Rollback:** Delete the file

#### Task P3-18
**Type:** implementation
**Component:** templates
**Prerequisites:** P3-16
**Description:** Create `internal/templates/flash.html` — renders `.Flashes` array with success/error/info styling. Create `internal/templates/form-errors.html` — renders `.Error` string and `.FieldErrors` slice.
**Verification:** Template parses correctly
**Rollback:** Delete files

### Phase 4 — Block CRUD

#### Task P4-19
**Type:** implementation
**Component:** Go
**Prerequisites:** P3-14
**Description:** Implement `handleBlockNewGET` in `internal/handlers/pages/blocks.go` — renders `vineyard/blocks/new.html`. Requires vineyard_id from URL path. Fetches vineyard details for context. Passes empty form data to template. Alpine variety search component embedded in template.
**Verification:** `curl -s http://localhost:8080/vineyard/1/blocks/new | grep -q "Nytt block" && echo "PASS"`
**Rollback:** Delete handler, revert route

#### Task P4-20
**Type:** implementation
**Component:** Go
**Prerequisites:** P4-19
**Description:** Implement `handleBlockNewPOST` in `blocks.go` — parses form values, validates required fields (block_name, area_ha, variety_id OR variety_name). Creates variety if name provided (review_needed status). Creates block record. On success: set flash, redirect to vineyard dashboard with HX-Redirect. On error: re-render form with values and errors.
**Verification:** `curl -s -X POST http://localhost:8080/vineyard/1/blocks/new -d 'block_name=test&area_ha=1.5&variety_id=1' -w '%{http_code}'` → expect 303
**Rollback:** Delete handler, revert route

#### Task P4-21
**Type:** implementation
**Component:** Go
**Prerequisites:** P4-20
**Description:** Implement `handleBlockEditGET` — renders `vineyard/blocks/edit.html` with pre-filled values. Validates block belongs to vineyard.
**Verification:** `curl -s http://localhost:8080/vineyard/1/blocks/1/edit | grep -q "Redigera block" && echo "PASS"`
**Rollback:** Delete handler, revert route

#### Task P4-22
**Type:** implementation
**Component:** Go
**Prerequisites:** P4-21
**Description:** Implement `handleBlockEditPOST` — same validation as new POST, but UPDATE instead of INSERT. Redirect on success. Re-render on error.
**Verification:** POST to edit URL with valid data → 303 redirect
**Rollback:** Delete handler, revert route

#### Task P4-23
**Type:** implementation
**Component:** Go
**Prerequisites:** P3-15
**Description:** Implement `handleVarietySearchGET` in `internal/handlers/api/` — JSON endpoint `GET /api/varieties/search?q=xxx`. Returns JSON with `matches` array (id, name, color, piwi) and `high_confidence` boolean. Uses pg_trgm similarity > 0.4, limited to 3 results. High confidence when score ≥ 0.8.
**Verification:** `curl -s 'http://localhost:8080/api/varieties/search?q=Pinot' | python3 -c "import sys,json; d=json.load(sys.stdin); assert 'matches' in d and 'high_confidence' in d; print('PASS')"`
**Rollback:** Delete handler, revert route

#### Task P4-24
**Type:** implementation
**Component:** Go
**Prerequisites:** P4-23
**Description:** Implement `handleGeoReversePOST` — JSON endpoint `POST /api/geo/reverse`. Accepts JSON `{lat, lon}`. Calls Nominatim reverse geocoding API. Returns `{name, county, municipality}` JSON. Gracefully handles Nominatim failures (return empty result, no error).
**Verification:** `curl -s -X POST http://localhost:8080/api/geo/reverse -d '{"lat":59.3293,"lon":18.0686}' | grep -q "name" && echo "PASS"`
**Rollback:** Delete handler, revert route

#### Task P4-25
**Type:** implementation
**Component:** templates
**Prerequisites:** P4-19
**Description:** Create `internal/templates/vineyard/blocks/new.html` — block creation form with Alpine.js variety search component. Form fields: block_name, area_ha, vine_count, planting_year, training_system, aspect (select), slope_degrees, elevation_m. Alpine `x-data="varietySearch()"` with debounce. Hidden inputs for variety_id and variety_name. Submit via `hx-post` with `hx-swap="none"`.
**Verification:** Template parses in Go build; HTML contains `x-data="varietySearch()"` and all expected form fields
**Rollback:** Delete file

### Phase 5 — Harvest CRUD

#### Task P5-26
**Type:** implementation
**Component:** Go
**Prerequisites:** P4-20
**Description:** Implement `handleHarvestNewGET` in `internal/handlers/pages/` — renders `vineyard/harvest/new.html`. Requires vineyard_id and optional block_id from URL. Checks block lock status. Renders harvest form with block selector dropdown (if no block_id provided). Alpine lock timer component if block is locked.
**Verification:** `curl -s http://localhost:8080/vineyard/1/harvest/new | grep -q "Ny skörd" && echo "PASS"`
**Rollback:** Delete handler, revert route

#### Task P5-27
**Type:** implementation
**Component:** Go
**Prerequisites:** P5-26
**Description:** Implement `handleHarvestNewPOST` — validates all required fields (block_id, harvest_year, yield_kg). Verifies block belongs to vineyard. Checks no duplicate harvest for this block+year. Verifies block not locked (or lock not expired). Creates harvest record. On success: set flash, redirect to vineyard dashboard. On conflict: re-render form with error.
**Verification:** POST with valid data → 303 redirect; duplicate year → 409
**Rollback:** Delete handler, revert route

#### Task P5-28
**Type:** implementation
**Component:** Go
**Prerequisites:** P5-27
**Description:** Implement `handleHarvestEditGET` — renders `vineyard/harvest/edit.html` with pre-filled harvest record data. Validates record belongs to vineyard.
**Verification:** `curl -s http://localhost:8080/vineyard/1/harvest/1/edit | grep -q "Redigera" && echo "PASS"`
**Rollback:** Delete handler, revert route

#### Task P5-29
**Type:** implementation
**Component:** Go
**Prerequisites:** P5-28
**Description:** Implement `handleHarvestEditPOST` — same validation as new POST, UPDATE harvest record. Redirect on success.
**Verification:** POST to edit → 303 redirect
**Rollback:** Delete handler, revert route

#### Task P5-30
**Type:** implementation
**Component:** Go
**Prerequisites:** P5-27
**Description:** Implement `handleHarvestLockPOST` — `POST /vineyard/{id}/blocks/{blockId}/harvest/lock`. Checks block not already locked. Creates block_lock record with current time + TTL (default 30 minutes). Redirects to harvest new form with block_id pre-filled. HX-Redirect header for HTMX.
**Verification:** `curl -s -X POST http://localhost:8080/vineyard/1/blocks/1/harvest/lock -w '%{http_code}'` → 303
**Rollback:** Delete handler, revert route

#### Task P5-31
**Type:** implementation
**Component:** Go
**Prerequisites:** P5-30
**Description:** Implement `handleHarvestUnlockPOST` — `DELETE /vineyard/{id}/blocks/{blockId}/harvest/lock`. Deletes block_lock record. Redirects to vineyard dashboard.
**Verification:** `curl -s -X DELETE http://localhost:8080/vineyard/1/blocks/1/harvest/lock -w '%{http_code}'` → 303
**Rollback:** Delete handler, revert route

#### Task P5-32
**Type:** implementation
**Component:** Go
**Prerequisites:** P5-30
**Description:** Implement `handleHarvestExtendPOST` — `POST /vineyard/{id}/blocks/{blockId}/harvest/lock/extend`. Updates block_lock `expires_at` to now + 30 min. Returns `HX-Trigger: lock-extended` header for HTMX to refresh the countdown timer.
**Verification:** POST to extend → 200 with HX-Trigger header
**Rollback:** Delete handler, revert route

#### Task P5-33
**Type:** implementation
**Component:** templates
**Prerequisites:** P5-26
**Description:** Create `vineyard/harvest/new.html` and `vineyard/harvest/edit.html` — harvest form with fields: block_id (select dropdown or hidden), harvest_date, harvest_year, yield_kg, brix, acid_g_l, vine_health_rating (1-5), notes, still_wine_l, sparkling_l, juice_l, sold_kg, discarded_kg. Alpine lock countdown timer component. Submit via `hx-post` with `hx-swap="none"`.
**Verification:** Template parses correctly; contains all expected form fields
**Rollback:** Delete template files

### Phase 6 — Settings & Static Pages

#### Task P6-34
**Type:** implementation
**Component:** Go
**Prerequisites:** P3-14
**Description:** Implement `handleSettingsGET` in `internal/handlers/pages/` — renders `vineyard/settings.html`. Requires owner role. Loads vineyard details, county list (for select), members list (with roles), current user's password info. Shows invite form. Role-gated: show/hide owner-only sections.
**Verification:** `curl -s http://localhost:8080/vineyard/1/settings | grep -q "Inställningar" && echo "PASS"`; non-owner → 403
**Rollback:** Delete handler, revert route

#### Task P6-35
**Type:** implementation
**Component:** Go
**Prerequisites:** P6-34
**Description:** Implement `handleSettingsPOST` — dispatches on `action` field: `update_vineyard` (update vineyard details), `change_password` (verify old + bcrypt new), `invite_member` (add vineyard_member), `remove_member` (delete member). Returns flash messages and HX-Redirect or re-renders with errors. Owner-only action validation.
**Verification:** POST with action=update_vineyard → 200 with flash; action=change_password → flash success/error
**Rollback:** Delete handler, revert route

#### Task P6-36
**Type:** implementation
**Component:** Go
**Prerequisites:** P2-8
**Description:** Implement `handleAccountExportGET` — `GET /api/account/export`. Queries user data (profile + vineyard membership + blocks + harvests), returns JSON with Content-Type application/json and Content-Disposition attachment header for download.
**Verification:** `curl -s http://localhost:8080/api/account/export -w '%{content_type}'` → `application/json`
**Rollback:** Delete handler, revert route

#### Task P6-37
**Type:** implementation
**Component:** Go
**Prerequisites:** P6-36
**Description:** Implement `handleAccountDeletePOST` — `POST /api/account/delete`. Requires `confirm=true` in form. Soft-deletes all user's vineyards. Removes all vineyard_members. Invalidates all sessions. Deletes password_hash (set to NULL or empty). Redirects to `/login` with deletion confirmation flash.
**Verification:** POST with confirm → 303 redirect to /login
**Rollback:** Delete handler, revert route (data can be recovered from DB backups)

#### Task P6-38
**Type:** implementation
**Component:** templates
**Prerequisites:** P6-34
**Description:** Create static page templates: `onboard.html` (vineyard registration form with GPS), `privacy.html` (static privacy policy), `terms.html` (static terms of service). Create `invite/confirm.html` (invite confirmation page) and `invite/success.html` (post-acceptance page). Create `error.html` (global error page with 404/500 variants).
**Verification:** `curl -s http://localhost:8080/privacy | grep -q "Svenskt Vin" && echo "PASS"`; `curl -s http://localhost:8080/terms | grep -q "Svenskt Vin" && echo "PASS"`
**Rollback:** Delete template files

### Phase 7 — Infrastructure

#### Task P7-39
**Type:** infra
**Component:** infra
**Prerequisites:** P1-4
**Description:** Create new `Dockerfile` — multi-stage build: builder stage (`golang:1.22-bookworm`) compiles Go binary with `-ldflags="-w -s"`, runtime stage (`debian:bookworm-slim`) copies binary, templates, and static files. Health check on `/health`. Expose port 8080.
**Verification:** `docker build -t svensktvin:test . && docker run --rm svensktvin:test wget -qO- http://localhost:8080/health || exit 0` (container exits, health check not running)
**Rollback:** `git checkout Dockerfile`

#### Task P7-40
**Type:** infra
**Component:** infra
**Prerequisites:** P7-39
**Description:** Update `docker-compose.yml` — keep PostgreSQL (PostGIS) container, add `app` service (build from Dockerfile, port 8080, depends on db). Remove old Node.js app service. Update environment variables for SMTP, session secret, app host.
**Verification:** `docker compose config` → validates YAML; `docker compose up -d` starts app + DB
**Rollback:** `git checkout docker-compose.yml`

#### Task P7-41
**Type:** infra
**Component:** infra
**Prerequisites:** none
**Description:** Create `static/css/app.css` — either generate via `npx tailwindcss -o static/css/app.css` using the Tailwind color mapping from ARCHITECTURE.md Appendix B, or document that production uses CDN-only approach (acceptable for internal use). Create `static/js/app.js` for any custom JS (Alpine event handlers, HTMX init, cookie consent localStorage).
**Verification:** `ls -la static/css/app.css static/js/app.js`
**Rollback:** Delete files

#### Task P7-42
**Type:** documentation
**Component:** docs
**Prerequisites:** P7-39
**Description:** Update `CLAUDE.md` — rewrite project description for new architecture. Update project structure section to show new layout. Update working commands (remove npm commands, add Go build/run commands, update Docker commands). Update behavioral principles (remove SvelteKit-specific notes, add Go template/HTMX notes).
**Verification:** `grep -q "Go templates" CLAUDE.md && grep -q "HTMX" CLAUDE.md && echo "PASS"`
**Rollback:** `git checkout CLAUDE.md`

### Phase 8 — Verification

#### Task P8-43
**Type:** verification
**Component:** verification
**Prerequisites:** P2-6, P2-7, P2-8, P2-9, P2-10, P2-11, P2-12
**Description:** Verify complete auth flow: register with invite token → login with password → verify session cookie → request membership → forgot-password → set-password → logout. Test CSRF rejection (missing token → 403). Test expired token behavior.
**Verification:** `curl -s -X POST http://localhost:8080/logout -v 2>&1 | grep -q "Set-Cookie.*session_id.*; Path=/; HttpOnly" && echo "Auth flow PASS"`
**Rollback:** N/A (verification only)

#### Task P8-44
**Type:** verification
**Component:** verification
**Prerequisites:** P4-20, P5-27, P5-30
**Description:** Verify vineyard flow: dashboard loads → create block → create harvest → lock block → extend lock → unlock block → edit block → edit harvest. Test lock expiration (wait > 30 min, try to create harvest). Test duplicate harvest year conflict.
**Verification:** Full pipeline: `curl` through create block, create harvest, lock/extend/unlock sequence
**Rollback:** N/A (verification only)

#### Task P8-45
**Type:** verification
**Component:** verification
**Prerequisites:** P4-23, P4-24, P3-18
**Description:** Verify HTMX flows: variety search autocomplete (AJAX JSON endpoint returns matches), form validation errors (submit invalid form → HTMX swaps form with error messages), flash messages (submit valid form → flash appears on redirect), cookie consent (Alpine `x-data` dismisses and stores in localStorage).
**Verification:** `curl -s 'http://localhost:8080/api/varieties/search?q=Pinot' | python3 -c "import sys,json; d=json.load(sys.stdin); print('Variety search:', 'PASS' if d.get('matches') else 'OK (no results)')"`
**Rollback:** N/A (verification only)

#### Task P8-46
**Type:** verification
**Component:** verification
**Prerequisites:** P7-39, P7-40
**Description:** Verify Docker build and deployment: `docker compose up --build` succeeds. App starts and health check passes. Database migrations run on startup (or verify migrations already applied). Test full request lifecycle through Docker: health check → login → dashboard → block CRUD.
**Verification:** `docker compose up --build -d && sleep 5 && docker compose ps` → all services healthy; `curl -s http://localhost:8080/health | grep -q "ok"`
**Rollback:** `docker compose down --volumes` and revert code
**Manual confirmation required:** This task involves deploying to Docker which may affect running services. Confirm with project owner before running.

## Execution Order

```
P1-1 → P1-2 → P1-3 → P1-4 → P1-5
    ↓
P2-6 ← P1-4,P1-5    P3-13 ← P1-4    P6-34 ← P3-14    P4-19 ← P3-14
    ↓                   ↓                ↓                  ↓
P2-7,P2-8,P2-9     P3-14            P6-35             P4-20,P4-21
    ↓                   ↓                ↓                  ↓
P2-10,P2-11,P2-12   P3-15           P6-36,P6-37       P4-22,P4-23,P4-24
    ↓                   ↓                ↓                  ↓
                  P3-16,P3-17,P3-18        P6-38        P4-25
    ↓                   ↓                ↓                  ↓
                              P7-39,P7-40,P7-41,P7-42
    ↓
                              P8-43 ← P2-*
                              P8-44 ← P4-*,P5-*
                              P8-45 ← P4-23,P4-24,P3-18
                              P8-46 ← P7-39,P7-40
```

**Parallelizable phases:** Phase 2 (auth pages) and Phase 6 (settings/static) are independent after Phase 1 completes. Phase 3 (vineyard dashboard) can start after Phase 1. Phase 4 (block CRUD) depends on Phase 3. Phase 5 (harvest CRUD) depends on Phase 4.

**Estimated timeline:** 13-15 days total, parallelized across 2-3 developers.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Auth cookie format mismatch (SvelteKit → Go) | Low | High | Same cookie name `session_id`, httpOnly/secure/SameSite=Lax, 30-day expiry. Test migration: create session via Go with existing SvelteKit session format |
| Template rendering performance | Low | Medium | Pre-compile all templates at startup with `template.ParseGlob()`, cache with `sync.Map` |
| CSRF protection gaps | Medium | High | Implement CSRF middleware early (P1-3 or P1-5), include `_csrf` hidden input in ALL forms, validate in every POST handler |
| HTMX `hx-target` misidentification | Medium | Medium | Use explicit `id` attributes on all `hx-target` elements; test each `hx-post`/`hx-swap`/`hx-target` combination |
| Alpine.js loading state race conditions | Low | Low | Use `x-cloak` for flash/error state; debounce search inputs at 300ms |
| Nominatim API reliability | Medium | Low | Graceful degradation: if Nominatim fails, allow manual lat/lon entry; log errors but don't block form |
| Tailwind CDN vs prebuilt CSS | Medium | Medium | Dev uses CDN (`cdn.tailwindcss.com`), prod uses `npx tailwindcss -o static/css/app.css` — document both |
| Swedish text drift | Low | Medium | ARCHITECTURE.md Section 7 migration table maps every Svelte text to Go template — use as reference |
| CORS removal | Low | Low | Single server = same-origin; remove all CORS config from `main.go` |
| pg_trgm extension availability | Low | High | Migration `001_extensions.sql` installs `pg_trgm`; verify extension is enabled before variety search |

## Verification Commands by Phase

### Phase 1 Foundation
```bash
# Verify directory structure
find /home/neurograft/Techstack/svensktvin/cmd /home/neurograft/Techstack/svensktvin/internal -type d | sort

# Verify Go compilation
cd /home/neurograft/Techstack/svensktvin && go build ./internal/db/ && go build ./internal/auth/ && go build ./internal/config/ && go build ./cmd/web/

# Verify config loads
cd /home/neurograft/Techstack/svensktvin && DATABASE_URL="postgres://sv_app:sv_dev_pass@localhost:5434/svensktvin" go run ./cmd/web/ --help 2>&1 | head -5
```

### Phase 2 Auth
```bash
# Health check
curl -s http://localhost:8080/health

# Login page renders
curl -s http://localhost:8080/login | grep -c "Svenskt Vin"

# Login POST with invalid data
curl -s -X POST http://localhost:8080/login -d 'action=login_password&email=test@example.com&password=wrong' -w '\nHTTP_CODE:%{http_code}\n'

# CSRF rejection (no token)
curl -s -X POST http://localhost:8080/login -d 'action=login_password&email=t@e.com&password=x&_csrf=bad' -w '\nHTTP_CODE:%{http_code}\n'

# Register with invite token
curl -s "http://localhost:8080/register?token=invite123" | grep -c "Skapa konto"

# Forgot password
curl -s -X POST http://localhost:8080/auth/forgot-password -d 'email=test@example.com' | grep -c "inloggningslänk"
```

### Phase 3 Vineyard
```bash
# Landing page (redirects to first vineyard or shows list)
curl -s http://localhost:8080/ -D - | grep -E "(HTTP/|Location:|Svenskt Vin)"

# Dashboard
curl -s http://localhost:8080/vineyard/1 | grep -c "Block"

# Benchmark
curl -s http://localhost:8080/vineyard/1/benchmark | grep -c "Jämförelse"

# Session cookie format
curl -s http://localhost:8080/vineyard/1 -D - | grep "Set-Cookie"
```

### Phase 4 Blocks
```bash
# New block form
curl -s http://localhost:8080/vineyard/1/blocks/new | grep -c "Nytt block"

# Create block with invalid data (should return errors)
curl -s -X POST http://localhost:8080/vineyard/1/blocks/new -d 'block_name=&area_ha=' -w '\nHTTP_CODE:%{http_code}\n'

# Create block with valid data
curl -s -X POST http://localhost:8080/vineyard/1/blocks/new -d 'block_name=Block%20A&area_ha=1.5&variety_id=1' -D - | grep "Location:"

# Variety search
curl -s 'http://localhost:8080/api/varieties/search?q=Pinot' | python3 -c "import sys,json; print(json.dumps(json.load(sys.stdin), indent=2))"

# Geo reverse
curl -s -X POST http://localhost:8080/api/geo/reverse -d '{"lat":59.3293,"lon":18.0686}' | python3 -c "import sys,json; print(json.dumps(json.load(sys.stdin), indent=2))"
```

### Phase 5 Harvest
```bash
# New harvest form
curl -s http://localhost:8080/vineyard/1/harvest/new | grep -c "Ny skörd"

# Create harvest (should fail without login, or succeed with session)
curl -s -X POST http://localhost:8080/vineyard/1/harvest/new -d 'block_id=1&harvest_year=2024&yield_kg=5000' -w '\nHTTP_CODE:%{http_code}\n'

# Harvest lock
curl -s -X POST http://localhost:8080/vineyard/1/blocks/1/harvest/lock -D - | grep -E "(Location:|HX-Redirect:)"

# Harvest unlock
curl -s -X DELETE http://localhost:8080/vineyard/1/blocks/1/harvest/lock -D - | grep -E "(Location:|HX-Redirect:)"

# Harvest extend
curl -s -X POST http://localhost:8080/vineyard/1/blocks/1/harvest/lock/extend -D - | grep "HX-Trigger"
```

### Phase 6 Settings
```bash
# Settings page (owner-only)
curl -s http://localhost:8080/vineyard/1/settings | grep -c "Inställningar"

# Settings POST (update vineyard)
curl -s -X POST http://localhost:8080/vineyard/1/settings -d 'action=update_vineyard&name=Test+Vineyard' -w '\nHTTP_CODE:%{http_code}\n'

# Account export
curl -s http://localhost:8080/api/account/export -D - | grep "Content-Type"

# Privacy/Terms
curl -s http://localhost:8080/privacy | grep -c "Svenskt Vin"
curl -s http://localhost:8080/terms | grep -c "Svenskt Vin"
```

## Rollback Strategy

### Code Rollback
- All new code lives in separate directory structure (`cmd/web/`, `internal/`) from existing `core-api/`
- SvelteKit frontend (`src/`) remains untouched until verification passes
- Git branch: `feat/htmx-migration` — merge to master only after P8-46 passes
- Existing `core-api/` directory can be deleted after migration is complete and verified

### Docker Rollback
- Keep `docker-compose.yml` with both services during transition: old Node.js app on `:3000` and new Go app on `:8080`
- Nginx reverse proxy can route to either service
- `docker compose down && git checkout HEAD~1 && docker compose up -d` for full rollback

### Database Rollback
- No schema changes → zero data risk
- Database migrations (19 files) and seeds (5 files) are idempotent and unchanged
- If any issues, `docker compose down --volumes` and restore from backup

### Partial Rollback
- If specific pages are broken: the SvelteKit versions remain available on the old port
- Use feature flag query param `?migrate=0` to force SvelteKit frontend when Go is deployed
- Health check at `/health` returns both Go and SvelteKit service status

## Cross-Phase Dependencies Summary

```
Phase 1 (Foundation) ─────┬────→ Phase 2 (Auth Pages) ──┐
                          ├────→ Phase 3 (Vineyard Dash) ─┤
                          ├────→ Phase 6 (Settings/Static)│
                          │                                ├──→ Phase 8 (Verification)
                          └────→ Phase 7 (Infra) ─────────┘
Phase 4 (Block CRUD) ────────────────────────────────────┘
Phase 5 (Harvest CRUD) ──────────────────────────────────┘
```

All verification tasks (P8-43 through P8-46) depend on completion of their respective phase tasks.
