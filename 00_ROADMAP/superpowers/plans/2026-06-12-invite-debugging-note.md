# Invite Flow — Debugging Notes & Plan

## Current State (2026-06-12)

### Changes Committed
- **Server-side**: Only "editor" role allowed for invites (`role !== 'editor'` rejected with error)
- **UI**: Dropdown only shows "Redaktör" option, "Ägare" removed
- **Graceful fallback**: SMTP not configured → saves to DB, logs token, no 500 error
- **New `/register` route**: Creates account + auto-joins vineyard from invite token

### ✅ RESOLVED: 500 Error on /register Page

**Root Cause:** Two separate bugs.

1. **`DATABASE_URL` not loaded in dev mode** — Vite loads `.env` into `import.meta.env` but NOT into `process.env`. Server-side code (`db.ts`) used `process.env.DATABASE_URL` which was always `undefined`, causing the postgres driver to fall back to `postgresql://localhost/svensktvin` with `max: 0` (no connections). Requests that hit the DB hung silently (no error, just timeout).

   **Fix:** `db.ts` now uses Vite's `loadEnv('development', ...)` to properly load `.env` files.

2. **Session cookie never set after registration** — `createSession(user.id, event.cookies)` passed a second argument but `auth.ts`'s `createSession(userId)` only accepts one. The session was created in the DB but the cookie was never set, so the newly registered user was immediately kicked back to login by the layout guard.

   **Fix:** Call `createSession(user.id)` and explicitly set the `session_id` cookie in the action handler, with conditional `Secure` flag (only in production).

3. **Register page load function accessed undefined `data.invite.vineyard.name`** — The load function returned `invite` with `{ id, email, role, vineyard_id, ... }` but the Svelte template accessed `invite.vineyard.name`. The load function was updated to JOIN the vineyards table and return `{ vineyard: { name } }`.

### ✅ RESOLVED: Browser Password Autofill

**Root Cause:** The browser's password manager detected a form with email + name fields and offered to autofill a saved password. The `name` input lacked proper autocomplete hints.

**Fix:** Added `autocomplete="name"` to the name input field. Browsers should now treat it as a name field, not a password field.

### 🔄 Remaining Items

#### 1. SMTP Not Configured Locally
SMTP fallback saves invites to DB and logs tokens. Works for local testing but email won't be sent.
- Set `SMTP_HOST`, `SMTP_USER`, `SMTP_PASS`, `SMTP_FROM` in `.env` to enable email

### Files Modified (this fix session)
- `src/lib/server/db.ts` — Added `loadEnv()` for proper .env loading in dev mode
- `src/routes/register/+page.server.ts` — Fixed createSession call, cookie set, vineyard JOIN, type imports, .js extension
- `src/routes/register/+page.svelte` — Added `form` export for error display
- `src/hooks.server.ts` — Conditional `Secure` flag on cookie (dev-friendly)

### Debug Plan (for next session)
1. Test full invite-to-registration flow end-to-end
2. Verify the `name` input no longer triggers password autofill
3. Configure SMTP for production email sending
