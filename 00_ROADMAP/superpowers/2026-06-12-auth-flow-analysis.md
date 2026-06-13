# Svenskt Vin — Authentication & User Handling Flow Analysis

**Date:** 2026-06-12
**Scope:** Complete authentication flow, session management, user creation, invite system, and access control.

---

## 1. Architecture Overview

```
┌──────────┐     HTTP cookie      ┌──────────────┐
│ Browser  │ ◄──────────────────► │ Hooks Server  │
│          │                      │ (session_id)  │
└──────────┘                      └──────────────┘
       │                                   │
       │  Email magic link                 │  DB queries
       ▼                                   ▼
┌──────────────┐                    ┌──────────────┐
│ SMTP / Mail  │                    │ Postgres     │
│ (prod only)  │                    │ (5434)       │
└──────────────┘                    └──────────────┘
```

**Auth model:** Passwordless magic-link authentication.
- Users log in via email magic link (15-minute expiry)
- Sessions stored server-side (30-day expiry)
- Cookie: `session_id` (HttpOnly, SameSite=Lax, Secure in prod)
- No passwords anywhere in the stack

---

## 2. Database Schema (Auth-Related Tables)

### `users`
| Column | Type | Constraint |
|--------|------|-----------|
| id | serial | PK |
| email | text | UNIQUE, NOT NULL |
| name | text | NOT NULL |
| active | boolean | DEFAULT true |
| created_at | timestamptz | DEFAULT now() |
| last_login | timestamptz | nullable |
| is_admin | boolean | DEFAULT false |

### `sessions`
| Column | Type | Constraint |
|--------|------|-----------|
| id | text (UUID) | PK |
| user_id | int | FK→users(id), CASCADE |
| expires_at | timestamptz | NOT NULL |
| created_at | timestamptz | DEFAULT now() |

### `magic_link_tokens`
| Column | Type | Constraint |
|--------|------|-----------|
| id | serial | PK |
| user_id | int | FK→users(id), CASCADE |
| token_hash | text | UNIQUE, hashed (SHA-256) |
| expires_at | timestamptz | NOT NULL (15 min) |
| used | boolean | DEFAULT false (one-shot) |
| created_at | timestamptz | DEFAULT now() |

### `pending_invites`
| Column | Type | Constraint |
|--------|------|-----------|
| id | serial | PK |
| email | text | NOT NULL |
| vineyard_id | int | FK→vineyards(id), CASCADE |
| role | text | CHECK ('owner' \| 'editor') |
| token | text | UNIQUE (random hex) |
| expires_at | timestamptz | NOT NULL (7 days) |
| used | boolean | DEFAULT false |
| created_at | timestamptz | DEFAULT now() |

### `vineyard_members` (composite PK)
| Column | Type | Constraint |
|--------|------|-----------|
| vineyard_id | int | FK→vineyards(id), CASCADE |
| user_id | int | FK→users(id), CASCADE |
| role | text | CHECK ('owner' \| 'editor') |
| created_at | timestamptz | DEFAULT now() |

---

## 3. Complete Flow Traces

### 3.1 First-Time User Flow (New Vineyard Owner)

```
Browser → /login → POST email
  ↓
Server: GET user by email → NULL (no account)
  ↓
Server: Returns { sent: true } (no account enumeration)
  ↓
User receives magic link email → clicks link → /auth/verify?token=xxx
  ↓
Server: Verify token → returns user_id
  ↓
Server: createSession(user_id) → creates session row + returns UUID
  ↓
Server: cookies.set('session_id', uuid)
  ↓
Server: Query memberships → empty
  ↓
Server: redirect(303, '/onboard')
  ↓
User completes vineyard registration → /vineyard/:id
  ↓
Server: INSERT vineyard + INSERT vineyard_members(user_id, 'owner')
```

### 3.2 Returning User Login Flow

```
Browser → /login → POST email
  ↓
Server: GET user by email → FOUND
  ↓
Server: createMagicLink(user_id) → generates token, inserts magic_link_tokens
  ↓
Server: sendMagicLink(email, token) → SMTP
  ↓
User receives magic link email → clicks link → /auth/verify?token=xxx
  ↓
Server: verifyToken(token) → returns user_id (marks token used=true)
  ↓
Server: updateLastLogin(user_id)
  ↓
Server: createSession(user_id) → session row + UUID
  ↓
Server: cookies.set('session_id', uuid)
  ↓
Server: Query pending_invites WHERE email ILIKE user.email
  ↓
If pending invite exists: auto-join vineyard, mark invite used
  ↓
Server: Query memberships → 1 or more → redirect to vineyard or home
```

### 3.3 Invite Flow — Existing User (Logged In)

```
User clicks invite link: /invite?token=xxx
  ↓
Server: Validate invite token → FOUND
  ↓
Server: Check if locals.user exists → YES
  ↓
Server: Compare user.email with invite.email
  ↓
If MATCH:
  INSERT vineyard_members (ON CONFLICT DO UPDATE)
  UPDATE pending_invites SET used=true
  redirect(303, /vineyard/:id)
If MISMATCH:
  Delete session from DB + cookie
  redirect(303, /login?invite=xxx)
```

### 3.4 Invite Flow — Non-Registered User

```
User clicks invite link: /invite?token=xxx
  ↓
Server: Validate invite token → FOUND
  ↓
Server: Check if locals.user exists → NO
  ↓
Server: redirect(303, /register?token=xxx&email=xxx)
  ↓
User on /register page:
  - Sees: vineyard name, role, pre-filled email
  - Enters: name
  ↓
Server: Validate invite token → FOUND
  ↓
Server: getUserByEmail(email) → NULL (no account)
  ↓
Server: INSERT users (name, email, active=true) → returns user_id
  ↓
Server: INSERT vineyard_members(user_id, role)
  ↓
Server: UPDATE pending_invites SET used=true
  ↓
Server: createSession(user_id, cookies)
  ↓
Server: redirect(303, /vineyard/:id)
```

### 3.5 Invite Flow — User with Existing Account (Wrong Login)

```
User clicks invite: /invite?token=xxx
  ↓
Logged in as different email → MISMATCH → /login?invite=xxx
  ↓
User enters invite email → /login
  ↓
Server: getUserByEmail(invite_email) → NULL
  ↓
Server: { needsRegistration: true, inviteToken }
  ↓
User sees: "Du har blivit inbjuden men har inget konto"
  ↓
User clicks "Logga in" on /register → /register?token=xxx&email=xxx
  ↓
(Continues as flow 3.4)
```

### 3.6 Hooks Server — Request Interceptor

```
Every request → hooks.server.ts:
  1. Read session_id cookie
  2. If present → getSession(session_id) → joins sessions JOIN users
  3. Sets event.locals.user = {id, email, name, is_admin} or null
  4. After resolve() → appends set-cookie header (refreshes 30-day expiry)
```

**Important:** The hook runs for EVERY request. `event.locals.user` is available in all server-side load functions and action handlers.

---

## 4. Route Guards & Access Control Matrix

| Route | Auth Required | Role Required | Check Location |
|-------|--------------|---------------|----------------|
| `/login` | No | — | N/A |
| `/register` | No | — | N/A |
| `/auth/verify` | No | — | N/A (token-based) |
| `/invite` | No | — | N/A (token-based redirect) |
| `/onboard` | Yes (locals.user) | — | `+layout.server.ts` |
| `/logout` | Yes (locals.user) | — | `+page.server.ts` |
| `/vineyard/[id]/**` | Yes | Member | `+layout.server.ts` |
| `/vineyard/[id]/settings` | Yes | owner | `+page.server.ts` load |
| `/vineyard/[id]/benchmark` | Yes | — | `+page.server.ts` load |

**Access control layers:**
1. **Auth check** — `locals.user` null → redirect to `/login`
2. **Membership check** — user not in `vineyard_members` → 403
3. **Role check** — only owners can access settings page

---

## 5. Issues Found

### P0 — Critical Bugs

#### 5.1 `/register` Page Returns 500 Error
**Location:** `src/routes/register/+page.server.ts` line ~40
**Issue:** In the `load` function, the SQL query references `v.name AS vineyard_name` in the `invite` object, but the query doesn't JOIN the vineyards table:
```typescript
// Line 40: The invite object has vineyard_id but NOT vineyard_name
const [invite] = await sql`
  SELECT id, email, role, vineyard_id, used, expires_at
  FROM pending_invites
  WHERE token = ${inviteToken} AND used = false
  LIMIT 1
`;
```
But the Svelte template references `data.invite?.vineyard.name`:
```svelte
<strong>{data.invite?.vineyard.name}</strong>
```
This would cause a runtime error at render time if accessed, though the load function itself should succeed. However, the actual 500 error likely comes from the `invite` object not having a `vineyard` property, causing `vineyard.name` to throw `Cannot read properties of undefined`.

**Root cause:** The load function returns `invite` without a `vineyard` sub-object, but the Svelte template tries to access `data.invite?.vineyard.name`. The optional chaining `?.` prevents a hard crash, BUT the template also passes `invite` to the form data and the server-side action re-validates the token by querying the same table.

**Actual 500 error root cause:** The `createSession` call in the action handler passes `event.cookies` as a second argument:
```typescript
await createSession(user.id, event.cookies);
```
But `createSession` in `auth.ts` takes only one argument: `async function createSession(userId: number)`. Passing cookies causes an unhandled argument mismatch — the function creates a session row but ignores the cookies param. However, the redirect should still happen. The real issue is likely that `createSession` was modified in `register/+page.server.ts` to take two args but the auth.ts function only takes one.

Let me check... Actually looking at auth.ts line 50: `export async function createSession(userId: number): Promise<string>` — it only takes one arg. But register/+page.server.ts line ~109 calls `await createSession(user.id, event.cookies)` — passing two args. This is silently ignored by JS (the cookies param is discarded), and the function returns the session ID. But wait — that shouldn't cause a 500. Let me re-examine.

The actual 500 is more likely from the DB connection during SSR. The `getConnection()` in db.ts has `max: 0` for build-time (no DATABASE_URL), which could cause issues.

#### 5.2 `createSession` Called with Wrong Signature
**Location:** `src/routes/register/+page.server.ts` line ~109
```typescript
await createSession(user.id, event.cookies);
```
**Issue:** `createSession` in `auth.ts` only takes `(userId: number)`. Passing `event.cookies` is silently ignored. The session is created in DB but the cookie is never set because there's no `cookies.set()` call after `createSession()`. This means the newly created user won't have a session cookie, so they'll be redirected to `/vineyard/:id` but immediately redirected back to `/login` by the layout guard.

**Fix:** Add `cookies.set('session_id', sessionId, ...)` after `createSession()`.

### P1 — High Priority Issues

#### 5.3 `/login` — "needsRegistration" Dead-End
**Location:** `src/routes/login/+page.svelte` + `+page.server.ts`
**Issue:** When a non-registered user enters their email during invite flow, the server returns `{ needsRegistration: true, inviteToken }`. The Svelte page shows "Du har blivit inbjuden... men du har inget konto än. Be den som bjöd in dig att registrera dig först." This is a **dead-end UX** — it tells the user to ask the inviter to register them, which is backwards. The user should be able to register themselves.

**Fix:** On `needsRegistration`, show a "Skapa konto" button that redirects to `/register?token=inviteToken&email=xxx`.

#### 5.4 Invite Link Redirect Loop for Non-Registered Users
**Location:** `src/routes/invite/+server.ts` + `src/routes/register/+page.server.ts`
**Issue:** The invite link `/invite?token=xxx` redirects non-logged-in users to `/register?token=xxx&email=xxx`. But if the user's browser has an expired session, `locals.user` is null → redirect to `/register`. The register page validates the token and creates a session. But if there's any error in the register load (e.g., 500 from issue 5.1), the user sees a 500 error instead of a registration form.

#### 5.5 `createMagicLink` Generates Token But Email May Not Send
**Location:** `src/lib/server/email.ts` `sendMagicLink()`
**Issue:** `getTransport()` throws `Error('SMTP environment variables not configured')` if SMTP isn't configured. In development without SMTP, `sendMagicLink()` will throw, causing the login action to 500. There's no graceful fallback.

**Fix:** Wrap SMTP send in try/catch with local-warn fallback.

### P2 — Medium Priority Issues

#### 5.6 Session Cookie Not Refreshed for Non-SSL Localhost
**Location:** `src/hooks.server.ts`
**Issue:** `Secure` flag is always set on the cookie in the hook (not conditional on `NODE_ENV`):
```typescript
const cookieValue = `session_id=${sessionId}; HttpOnly; Secure; SameSite=Lax; Path=/; Max-Age=${30 * 24 * 60 * 60}`;
```
This means localhost:5173 cannot set the cookie (Secure flag requires HTTPS). The `auth/verify` page sets it correctly with `secure: process.env.NODE_ENV === 'production'`, but the hook does not. This means the hook's `set-cookie` append will be rejected by browsers on localhost.

**Fix:** Make the hook conditional: only set `Secure` when `NODE_ENV === 'production'`.

#### 5.7 `inviteToken` Passed to Register Page but Not Preserved on Redirect
**Location:** `src/routes/invite/+server.ts`
**Issue:** When a logged-in user is on the wrong account, they're redirected to `/login?invite=token`. If they then enter the correct email and get `{ needsRegistration: true }`, the `needsRegistration` path doesn't carry the `inviteToken` forward. The user can't proceed to register.

#### 5.8 No Account Enumeration Prevention for Invite Email
**Location:** `src/routes/login/+page.server.ts`
**Issue:** When a non-registered user enters their email during invite flow:
- If they HAVE an account → `{ sent: true, inviteToken }`
- If they DON'T have an account → `{ needsRegistration: true, inviteToken }`

The response message is different, which IS account enumeration (but the code comment says it's intentional for invite flow). This is acceptable for invites but could be confusing to users.

#### 5.9 No Cleanup of Expired Sessions/Magic Links
**Location:** DB schema comments say "Rows cleaned up by cron" but no cron is configured
**Issue:** `magic_link_tokens` and `sessions` rows accumulate. `magic_link_tokens` has UNIQUE constraint on `token_hash`, so reused tokens are rejected (but the row still exists). No automatic cleanup.

### P3 — Low Priority / Minor Issues

#### 5.10 `pending_invites` role CHECK constraint allows 'owner'
**Location:** `db/migrations/016_pending_invites.sql`
**Issue:** The `role` CHECK constraint allows `'owner'` as a role for invites. The server-side invite action restricts to `'editor'` only. This creates inconsistency — the DB allows owner invites but the UI doesn't. If someone uses the DB directly or an old client, they could invite as owner.

#### 5.11 `inviteEmailTemplate` Token Not Encoded in URL
**Location:** `src/lib/server/email.ts` `inviteEmailTemplate()`
**Issue:** The token is inserted directly into the URL without encoding:
```typescript
const link = `${appHost}/invite?token=${encodeURIComponent(token)}`;
```
Actually it IS encoded. This is fine.

#### 5.12 `pending_invites` token is not hashed
**Location:** `db/migrations/016_pending_invites.sql`
**Issue:** The invite token is stored as plaintext hex, not hashed like magic link tokens. This means if the DB is compromised, all invite links are immediately usable. Magic link tokens use SHA-256 hashing; invite tokens do not.

#### 5.13 No Rate Limiting on Login/Invite Endpoints
**Issue:** No brute-force or spam protection on `/login` (magic link generation) or settings invite action. An attacker could spam magic links to any email.

---

## 6. Code Quality Notes

### Good Practices
- ✅ Passwordless auth — no password storage/hash issues
- ✅ Magic link tokens stored as SHA-256 hashes, not plaintext
- ✅ One-shot tokens (`used = true` prevents replay)
- ✅ Server-side session storage (not JWT/token-based)
- ✅ Session expiry enforcement in queries (`expires_at > now()`)
- ✅ `HttpOnly` cookie — not accessible via JS
- ✅ 15-minute magic link expiry — short-lived
- ✅ Graceful SMTP fallback for invite flow
- ✅ Account enumeration prevention on login (same message for found/not-found)

### Areas for Improvement
- ❌ Invite tokens not hashed (plaintext in DB)
- ❌ No rate limiting
- ❌ No automatic cleanup of expired tokens/sessions
- ❌ Session cookie `Secure` flag always set (breaks localhost)
- ❌ `createSession` signature mismatch in register flow
- ❌ Dead-end UX on login for non-registered invite users

---

## 7. Summary of Bugs to Fix

| Priority | Bug | File | Fix |
|----------|-----|------|-----|
| P0 | `createSession` signature mismatch — cookies not set | `register/+page.server.ts` | Add `cookies.set()` after `createSession()` |
| P0 | `/register` page 500 error | `register/+page.server.ts` | Debug actual root cause (likely DB or type error) |
| P1 | Login "needsRegistration" dead-end | `login/+page.svelte` | Add redirect to `/register?token=xxx&email=xxx` |
| P1 | SMTP crash in dev | `email.ts` `sendMagicLink()` | Try/catch with warn fallback |
| P2 | Cookie Secure flag breaks localhost | `hooks.server.ts` | Conditional on `NODE_ENV` |
| P2 | No inviteToken preservation on wrong-account flow | `login/+page.server.ts` | Pass inviteToken through needsRegistration |
| P3 | Invite tokens plaintext | `pending_invites` migration | Hash tokens like magic links |
| P3 | No rate limiting | All auth routes | Add rate limiting middleware |
| P3 | No session/token cleanup cron | Infrastructure | Add cleanup job |
