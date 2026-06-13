# Rewrite: Invite + Password Onboarding

## Goal
Replace magic-link authentication with password-based auth for invite-only access.
Invite link pre-fills company details; user sets password on first use.

## Current Flow
```
Admin sends invite → user clicks /invite?token=xxx → /register → creates account (no password) →
magic link login → auto-join vineyard → /onboard (register vineyard/company)
```

## New Flow
```
Admin sends invite (with company_name, owner_name) → user clicks /invite?token=xxx →
/register → enters name (preset), sets password → account created + auto-join →
/login with email+password → vineyard
```

## DB Changes

### 1. Add password_hash to users
```sql
ALTER TABLE users ADD COLUMN password_hash TEXT;
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL; -- optional for existing users
```

### 2. Add company_name and owner_name to pending_invites
```sql
ALTER TABLE pending_invites ADD COLUMN company_name TEXT;
ALTER TABLE pending_invites ADD COLUMN owner_name TEXT;
```

## File Changes

### auth.ts
- Add `hashPassword(password)` — bcrypt hash
- Add `verifyPassword(password, hash)` — bcrypt compare
- Update `getUserByEmail` to return password_hash
- Keep magic-link functions for password reset only

### register/+page.server.ts (new)
- Load: validate invite token, return company_name, owner_name, email, role
- Action: hash password, create user, create session, redirect to vineyard
- Handle existing account: redirect to /login?invite=xxx

### register/+page.svelte (new)
- Show company_name (from invite, editable)
- Show owner_name as user name field (editable)
- Email field (readonly from invite)
- Password + confirm password fields
- Submit → create account

### login/+page.server.ts (changed)
- Load: return invite context if ?invite= present
- Action: email + password login
  - If user has password_hash → verify and create session
  - If user has no password_hash → redirect to /register?invite=xxx (set password)
- "Forgot password?" → send magic-link to set password

### login/+page.svelte (changed)
- Email + password form
- "Forgot password?" link → /auth/forgot-password

### auth/verify/+page.server.ts (unchanged for now)
- Still handles magic-link verification
- Used only for password reset flow

### auth/forgot-password/+page.server.ts (new)
- GET: show email form
- POST: send magic-link to set new password

### auth/set-password/+page.server.ts (new)
- GET: validate magic-link token, show password form
- POST: hash new password, update user, create session, redirect

### routes/+layout.server.ts
- Update `getSession` to return password_hash (for checking if user needs to set password)

### vineyard/[id]/settings/+page.server.ts
- Invite creation: accept company_name + owner_name fields
- Member list visibility:
  - owner/admin → show all members (current)
  - editor → show self only
- Add "change password" action
- Remove "remove member" for non-owners (already owner-only)

### vineyard/[id]/settings/+page.svelte
- Show all members only if owner/admin
- If editor: show "Min profil" page only
- Add "Change password" section (all roles)
- Hide member management if editor

### email.ts
- Add `passwordResetEmailTemplate` for magic-link password reset
- Keep existing templates

### migrate existing users
- For existing users without password_hash: show "Set your password" prompt on next login

## Edge Cases
1. Existing user with no password_hash → redirect to set password
2. Invite token used but user already has account → redirect to /login
3. Invite token expired → show error
4. Password reset for non-existent email → same message as success (no enumeration)
