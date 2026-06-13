# Svenskt Vin — Architecture

## North Star

Svenskt Vin is a **standalone SvelteKit web application** for viticulture community management. It is an **artifact** of the Cinerarium platform, not a platform component. It has zero dependency on Cinerarium's TAP-Core, MCP Gateway, or agent infrastructure.

**Scope:** Invite-only vineyard management — users join via invite, create/manage vineyards, collaborate on wine production.
**Not in scope:** AI agent routing, knowledge graphs, multi-brain orchestration, Cinerarium platform services.

## Stack

| Layer | Technology |
|-------|-----------|
| Framework | SvelteKit (server routes + page components) |
| Language | TypeScript (strict) |
| Database | PostgreSQL (standalone — own connection string, own schema) |
| ORM | Drizzle ORM (Postgres driver only) |
| Auth | bcrypt (≥10 rounds), httpOnly/secure session cookies |
| Email | Nodemailer (or equivalent) |

## Data Model

### Core Tables (all in standalone Postgres)

**`users`**
- `id` (UUID PK), `email` (unique), `password_hash` (TEXT, nullable), `name` (TEXT), `role` (TEXT, default 'member')
- `password_hash` is nullable — users created via invite set it during registration; legacy users get prompted to set it on first login.
- `created_at`, `updated_at`

**`vineyards`**
- `id` (UUID PK), `name` (TEXT), `description` (TEXT), `owner_id` (FK → users), `settings` (JSONB)
- `created_at`, `updated_at`

**`vineyard_members`**
- `vineyard_id` (FK → vineyards), `user_id` (FK → users), `role` (TEXT: owner/admin/editor)
- Unique on (vineyard_id, user_id)

**`pending_invites`**
- `id` (UUID PK), `email` (unique), `vineyard_id` (FK → vineyards), `token` (unique TEXT),
  `company_name` (TEXT), `owner_name` (TEXT), `role` (TEXT), `expires_at` (TIMESTAMP), `created_at`
- `company_name` and `owner_name` are pre-filled by the inviting admin and editable on first registration.
- Invites expire after 7 days (configurable).

**`sessions`**
- `id` (TEXT PK), `user_id` (FK → users), `created_at`, `expires_at`

## Route Conventions

### Auth Flow

```
GET  /invite?token=xxx       → validate token, show /register pre-filled
POST /register               → create user+session, auto-join vineyard
GET  /login                  → email + password form
POST /login                  → verify password, create session
GET  /auth/forgot-password   → email form for password reset
POST /auth/forgot-password   → send magic-link (password_reset_email)
GET  /auth/set-password?token=xxx  → validate token, show password form
POST /auth/set-password      → hash + set password, create session
```

### Vineyard

```
GET  /vineyard/[id]           → vineyard detail
GET  /vineyard/[id]/settings  → settings (role-gated)
  - owner/admin: all settings + member management
  - editor: settings only (no member management)
```

### Layout

```
GET  /                        → landing / login redirect
```

### getSession (routes/+layout.server.ts)

**CRITICAL:** `getSession` must NEVER return `password_hash` in the session payload. SvelteKit auto-serializes the return value of `load` into the session cookie. Extract only: `id`, `email`, `name`, `role`, `vineyard_id`. Use Drizzle `.select()` with explicit columns.

## Security Model

1. **Password hashing:** bcrypt, minimum 10 rounds (use 12). Never store plaintext.
2. **Session cookies:** `httpOnly`, `secure`, `SameSite=Lax`. Session ID is cryptographically random.
3. **password_hash never exposed:** Never returned in API responses, session payloads, or log output.
4. **Rate limiting:** Login and password reset endpoints rate-limited per IP (token bucket, ~5 attempts per 15 min).
5. **No enumeration:** "Forgot password" returns same message for existing/non-existing email.
6. **Standalone DB:** Svenskt Vin uses its own PostgreSQL instance. No connection to Cinerarium's TAP-Core.
7. **No MCP tools:** SvelteKit server routes use Drizzle ORM directly. The MCP `db_query`/`db_exec` tools are for Cinerarium agents only, not for the web app.

## Invitation Lifecycle

1. Admin creates invite (with `company_name`, `owner_name`, `email`) → row in `pending_invites` with JWT-like token
2. Recipient clicks `/invite?token=xxx` → token validated, page pre-fills data
3. Recipient registers (name, password, confirms company) → user created, auto-joined, invite consumed
4. If recipient already has account → redirect to `/login?invite=xxx` (prompt to set password)
5. Expired invites (past `expires_at`) → show error, offer "request new invite"

## Data Migration

- **Existing users without password_hash:** On first login, if `password_hash` is NULL → redirect to `/auth/set-password` (create password via magic-link)
- **New columns:** `ALTER TABLE ... ADD COLUMN` with `DROP NOT NULL` for existing data compatibility
- **Migrations are idempotent** — check for existence before ALTER

## Swedish UI

All user-visible text must be in Swedish. Do not translate to English. Preserve existing Swedish text verbatim when modifying pages.

## Generated File Paths

| File | Purpose |
|------|---------|
| `src/lib/server/db.ts` | Postgres connection, schema definitions |
| `src/lib/server/auth.ts` | bcrypt helpers, session management, `getUserByEmail`, `createSession`, `getSession` |
| `src/lib/server/email.ts` | Email template functions (magic_link, password_reset) |
| `src/routes/+layout.server.ts` | `getSession`, global load |
| `src/routes/invite/+server.ts` | Invite token validation (GET) |
| `src/routes/register/+page.server.ts` | Register action (POST) — rewrite for password |
| `src/routes/register/+page.svelte` | Registration form — rewrite for password |
| `src/routes/login/+page.server.ts` | Login action (POST) — rewrite for email+password |
| `src/routes/login/+page.svelte` | Login form — rewrite |
| `src/routes/auth/forgot-password/+page.server.ts` | New: forgot password (POST) |
| `src/routes/auth/set-password/+page.server.ts` | New: set password via magic-link token (POST) |
| `src/routes/vineyard/[id]/settings/+page.server.ts` | Settings actions — add password change |
| `src/routes/vineyard/[id]/settings/+page.svelte` | Settings UI — add password section |
| `db/migrations/*.sql` | Database migrations |

## Implementation Plan — Invite + Password Onboarding

Tasks are ordered by dependency. Each task should be reviewed before implementation.

| # | Task | Output Path | Depends On |
|---|------|-------------|------------|
| 1 | Add `password_hash` column to users table, add `company_name`/`owner_name` to pending_invites, add `expires_at` | `db/migrations/019_pending_invites_company_fields.sql` | — |
| 2 | Rewrite `src/lib/server/auth.ts`: add `hashPassword`, `verifyPassword`, update `getUserByEmail` to return `password_hash`, update `createSession` to accept password for first-time users | `src/lib/server/auth.ts` | 1 |
| 3 | Update `src/lib/server/email.ts`: add `passwordResetEmailTemplate` function | `src/lib/server/email.ts` | 2 |
| 4 | Rewrite `src/routes/register/+page.server.ts`: accept password, hash it, create user + session, handle existing account redirect | `src/routes/register/+page.server.ts` | 1, 2 |
| 5 | Rewrite `src/routes/register/+page.svelte`: add password + confirm password fields, show pre-filled company/owner, Swedish UI | `src/routes/register/+page.svelte` | 4 |
| 6 | Rewrite `src/routes/login/+page.server.ts`: email+password login, check password_hash, redirect if missing | `src/routes/login/+page.server.ts` | 2 |
| 7 | Rewrite `src/routes/login/+page.svelte`: email + password form, "Forgot password?" link, Swedish UI | `src/routes/login/+page.svelte` | 6 |
| 8 | New: `src/routes/auth/forgot-password/+page.server.ts` — send magic-link for password reset | `src/routes/auth/forgot-password/+page.server.ts` | 2, 3 |
| 9 | New: `src/routes/auth/forgot-password/+page.svelte` — email form | `src/routes/auth/forgot-password/+page.svelte` | 8 |
| 10 | New: `src/routes/auth/set-password/+page.server.ts` — validate token, hash + set password | `src/routes/auth/set-password/+page.server.ts` | 2, 3 |
| 11 | New: `src/routes/auth/set-password/+page.svelte` — password form | `src/routes/auth/set-password/+page.svelte` | 10 |
| 12 | Update `src/routes/+layout.server.ts`: ensure `getSession` excludes `password_hash` from return object | `src/routes/+layout.server.ts` | 2 |
| 13 | Update `src/routes/vineyard/[id]/settings/+page.server.ts`: add password change action, invite with company/owner fields | `src/routes/vineyard/[id]/settings/+page.server.ts` | 2 |
| 14 | Update `src/routes/vineyard/[id]/settings/+page.svelte`: add password section, editor restrictions | `src/routes/vineyard/[id]/settings/+page.svelte` | 12, 13 |
| 15 | Update `src/routes/invite/+server.ts`: validate new invite token fields | `src/routes/invite/+server.ts` | 1 |
| 16 | Migration data script: prompt legacy users (NULL password_hash) to set password on first login | (integrated into login/register flow) | 2 |

## Constraints (non-negotiable)

1. Zero Cinerarium / TAP-Core dependency
2. Own PostgreSQL database
3. bcrypt >= 10 rounds (default 12)
4. Session cookies: httpOnly + secure + SameSite=Lax
5. password_hash NEVER in session/API responses
6. Rate limit login/reset endpoints
7. Preserve Swedish UI text
8. No MCP tools in SvelteKit routes — use Drizzle directly
9. All TypeScript — strict mode, no `.js` files
10. Drizzle ORM for all database access
11. SvelteKit server routes for all backend logic
