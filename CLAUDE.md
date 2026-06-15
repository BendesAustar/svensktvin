# Svenskt Vin

Standalone SvelteKit web application for viticulture community management. **Artifact of Cinerarium — zero dependency on the platform.**

## Architecture Reference

For full architecture, data model, and route conventions, see [`architecture.md`](architecture.md). Key facts:

- **Frontend:** SvelteKit 2.x (TypeScript, strict), Svelte 5, Vite
- **Backend:** Go REST API (`core-api/`) on `:9091`
- **Database:** PostgreSQL standalone, port `5434` (not `5433` — collision with Cinerarium)
- **Auth:** bcrypt ≥ 12 rounds, httpOnly/secure session cookies, magic links
- **Email:** Nodemailer (SMTP required for forgot-password flow)

## Behavioral Principles

### Standalone — No Cinerarium Dependencies

- **Never import Cinerarium packages, tools, or infrastructure.** Svenskt Vin has zero dependency on TAP-Core, MCP Gateway, or agent infrastructure.
- **No MCP tools in SvelteKit routes.** Use Drizzle ORM directly for all database access. The MCP `db_query`/`db_exec` tools are for Cinerarium agents only.
- **Own PostgreSQL database.** Never connect to `cinerarium` database or `tap_core` schema.

### Swedish UI — Preserve Existing Text

All user-visible text is in Swedish. Do not translate to English. Preserve existing Swedish text verbatim when modifying pages.

### Security — Never Expose password_hash

- `password_hash` is **never** returned in session payloads, API responses, or log output.
- `getSession()` must extract only: `id`, `email`, `name`, `role`, `vineyard_id`.
- SvelteKit auto-serializes the return value of `load` into the session cookie — be explicit with columns.

### TypeScript — Strict Mode

- All TypeScript — no `.js` files.
- Strict mode enabled. No `any` types unless absolutely necessary.
- Use Drizzle ORM for all database access — raw SQL only in migrations.

### SvelteKit 2.x — Named Actions

- SvelteKit 2.x supports named form actions (`action=login_password`, `action=request_membership`).
- When using `use:enhance`, ensure the action name matches the exported function.
- If a form POST returns 404, check that the action is exported and the `enhance` directive isn't intercepting incorrectly.

### Password Auth — Complete Flow

- Login: email + password → verify → session → `/vineyard`
- Forgot password: email → magic-link → set password → session
- Registration: invite token → pre-filled form → password → user + session
- Password change: authenticated → verify old → hash new → store
- **CRITICAL:** `login_password` and `request_membership` actions on `/login` — both must coexist in the same `+page.server.ts`

## Project Structure

```
svensktvin/
├── src/                          # SvelteKit frontend (TypeScript)
│   ├── lib/server/               # Shared server utilities
│   │   ├── db.ts                 # Drizzle schema + connection
│   │   ├── auth.ts               # bcrypt, sessions, getUserByEmail
│   │   └── email.ts              # Email template functions
│   └── routes/                   # SvelteKit pages + actions
│       ├── login/                # Email + password login
│       ├── register/             # Invite → user creation
│       ├── invite/               # Token validation
│       ├── auth/                 # forgot-password, set-password
│       └── vineyard/[id]/        # Vineyard pages + settings
├── core-api/                     # Go REST API (:9091)
│   ├── cmd/api/                  # Entry point
│   └── internal/                 # Handlers, middleware, DB
├── db/                           # PostgreSQL migrations + seeds
│   ├── migrations/               # 12 numbered, idempotent
│   └── seeds/                    # 5 seed scripts
├── scripts/                      # migrate.sh, seed.sh, test.sh
├── docker-compose.yml            # Database + app containers
└── architecture.md               # Full architecture reference
```

## Working Commands

```bash
# Start database only
docker compose up -d

# Apply migrations (idempotent)
./scripts/migrate.sh

# Load seed data
./scripts/seed.sh

# Run tests
./scripts/test.sh

# Dev server (with DB)
DATABASE_URL=postgres://sv_app:sv_dev_pass@localhost:5434/svensktvin npm run dev

# Build (requires DATABASE_URL in env)
DATABASE_URL=postgres://sv_app:sv_dev_pass@localhost:5434/svensktvin npm run build

# Core API (Go)
cd core-api && go run ./cmd/api
```

## Current State

**In progress:** Password authentication flow (login, register, forgot-password, password change).

**Blocking:** POST `/login` returns 404 — likely SvelteKit 2.x named action routing issue. Check browser Network tab for actual POST endpoint.

**Unimplemented:**
- DesignBrain review of new UI
- Production adapter-node ESM fix (`__dirname` issue)
- Real SMTP integration (current: graceful fallback with console.log)
- Magic link email template (currently reusing `sendMagicLink` for forgot-password)

## See Also

- [`architecture.md`](architecture.md) — data model, routes, security model, constraints
- [`SESSION_LOG_2026-06-13.md`](SESSION_LOG_2026-06-13.md) — current session state, blocking issues
