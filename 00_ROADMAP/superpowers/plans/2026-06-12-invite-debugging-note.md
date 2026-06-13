# Invite Flow — Debugging Notes & Plan

## Current State (2026-06-12)

### Changes Committed
- **Server-side**: Only "editor" role allowed for invites (`role !== 'editor'` rejected with error)
- **UI**: Dropdown only shows "Redaktör" option, "Ägare" removed
- **Graceful fallback**: SMTP not configured → saves to DB, logs token, no 500 error
- **New `/register` route**: Creates account + auto-joins vineyard from invite token
- **Invite link redirect**: Non-auth users now go to `/register` instead of dead-end `/login`

### Known Issues (still unresolved)

#### 1. 500 Error on /register Page
The register page returns HTTP 500. The dev server log output was unreliable (timeouts, no stdout capture), so the exact error couldn't be captured. Suspected causes:
- `postgres` import path issue (fixed in `+page.server.ts` — now uses `$lib/server/db` instead of `'postgres'`)
- Database connection failure during SSR load
- Missing vineyard lookup (invite route now does `JOIN vineyards v ON v.id = pi.vineyard_id`)
- Some other runtime error during SSR evaluation

**Next steps for debugging:**
1. Start dev server in foreground, watch real-time logs: `npm run dev -- --host 0.0.0.0 --port 5173`
2. Visit `/register?token=xxx&email=xxx` while watching terminal output
3. Note the exact error message and stack trace

#### 2. Browser Password Autofill
The register page has a `name` field that browsers may interpret as a password field (since email is pre-filled and readonly). User reports "it wants a password".
- **Possible fix**: Add `autocomplete="new-name"` or `autocomplete="name"` attributes to the name input
- **Also**: Add `autocomplete="new-password"` to any hidden password field if we add one later
- **Or**: The browser's password manager may be offering to fill credentials — this is a client-side issue, dismissible

#### 3. SMTP Not Configured Locally
SMTP fallback saves invites to DB and logs tokens. Works for local testing but email won't be sent.
- Set `SMTP_HOST`, `SMTP_USER`, `SMTP_PASS`, `SMTP_FROM` in `.env` to enable email

### Files Modified (this session)
- `src/routes/vineyard/[id]/settings/+page.server.ts` — role restriction, SMTP fallback
- `src/routes/vineyard/[id]/settings/+page.svelte` — dropdown update, error/success display
- `src/routes/invite/+server.ts` — include vineyard name, redirect to `/register`
- `src/routes/register/+page.server.ts` (NEW) — account creation + auto-join
- `src/routes/register/+page.svelte` (NEW) — registration form

### Debug Plan (for next session)
1. Start dev server in a persistent terminal window
2. Reproduce the 500 error and capture the server-side stack trace
3. If it's a database error: check if `vineyards` table has the invited vineyard's data
4. If it's a template/rendering error: check Svelte compilation output
5. Fix the root cause, test full flow end-to-end
