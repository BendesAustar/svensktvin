// src/routes/login/+page.server.ts
import { redirect, fail } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { getUserByEmail, createMagicLink, verifyPassword, getSessionByUserId } from '$lib/server/auth.js';
import { sendMagicLink } from '$lib/server/email.js';
import { sql } from '$lib/server/db.js';

// ─── Rate limiting (simple in-memory, per-IP) ───────────────────────────
const loginAttempts = new Map<string, { count: number; resetAt: number }>();
const MAX_ATTEMPTS = 10;
const WINDOW_MS = 15 * 60 * 1000; // 15 minutes

function checkRateLimit(ip: string): boolean {
  const now = Date.now();
  const entry = loginAttempts.get(ip);
  if (!entry) return true;
  if (now > entry.resetAt) {
    loginAttempts.delete(ip);
    return true;
  }
  return entry.count < MAX_ATTEMPTS;
}

function recordAttempt(ip: string) {
  const entry = loginAttempts.get(ip) || { count: 0, resetAt: Date.now() + WINDOW_MS };
  entry.count++;
  loginAttempts.set(ip, entry);
}

// ─── Helper: vineyard info from invite token ────────────────────────────
function getInviteVineyard(inviteToken: string | null) {
  if (!inviteToken) return Promise.resolve(null);
  return sql<{ vineyard_name: string }[]>`
    SELECT v.name AS vineyard_name
    FROM pending_invites pi
    JOIN vineyards v ON v.id = pi.vineyard_id
    WHERE pi.token = ${inviteToken} AND pi.used = false AND pi.expires_at > now()
    LIMIT 1
  `;
}

// ─── Load: show vineyard context ────────────────────────────────────────
export const load: PageServerLoad = async ({ url }) => {
  const inviteToken = url.searchParams.get('invite');
  let vineyard: { name: string } | null = null;
  const rows = await getInviteVineyard(inviteToken);
  if (rows?.[0]) vineyard = { name: rows[0].vineyard_name };
  return { inviteToken, vineyard };
};

// ─── Actions ────────────────────────────────────────────────────────────
export const actions: Actions = {
  // ── Password-based login ──
  login_password: async ({ request, cookies, getClientAddress }) => {
    const ip = getClientAddress();

    // Rate limit
    if (!checkRateLimit(ip)) {
      return fail(429, { error: 'För många inloggningsförsök. Försök igen om 15 minuter.', sent: false });
    }

    const data = await request.formData();
    const email = (data.get('email') as string)?.trim().toLowerCase();
    const password = data.get('password') as string;
    const inviteToken = request.url.searchParams.get('invite') ?? undefined;

    if (!email || !password) {
      return fail(400, { error: 'Ange e-postadress och lösenord.', email, sent: false });
    }

    const user = await getUserByEmail(email);
    if (!user) {
      // Don't reveal whether account exists — same message as magic link
      try {
        await sendMagicLink(email, 'placeholder');
      } catch {
        // SMTP not configured — still return success to avoid enumeration
      }
      return { sent: true, vineyard: null, showPassword: false, membershipSent: false };
    }

    const valid = await verifyPassword(password, user.password_hash!);
    if (!valid) {
      // User has no password yet — send magic link instead
      const token = await createMagicLink(user.id);
      try {
        await sendMagicLink(email, token);
      } catch {
        // SMTP not configured
      }
      return { sent: true, vineyard: null, showPassword: false, membershipSent: false };
    }

    // Password OK — create session
    const session = await getSessionByUserId(user.id);
    if (!session || session.expires_at < new Date()) {
      // Create new session
      const expiresAt = new Date();
      expiresAt.setDate(expiresAt.getDate() + 30);
      const sessionId = crypto.randomUUID();

      await sql`
        INSERT INTO sessions (id, user_id, expires_at)
        VALUES (${sessionId}, ${user.id}, ${expiresAt.toISOString()})
      `;

      cookies.set('session_id', sessionId, {
        httpOnly: true,
        secure: process.env.NODE_ENV === 'production',
        sameSite: 'lax',
        path: '/',
        maxAge: 30 * 24 * 60 * 60, // 30 days
      });
    } else {
      // Re-authenticate existing session
      cookies.set('session_id', session.id, {
        httpOnly: true,
        secure: process.env.NODE_ENV === 'production',
        sameSite: 'lax',
        path: '/',
        maxAge: 30 * 24 * 60 * 60,
      });
    }

    // If user has no password_hash (old magic-link-only account), force setup
    if (!user.password_hash) {
      throw redirect(303, `/auth/set-password?email=${encodeURIComponent(email)}`);
    }

    // Redirect: invite context → register, otherwise vineyard
    if (inviteToken) {
      throw redirect(303, `/register?invite=${encodeURIComponent(inviteToken)}&email=${encodeURIComponent(email)}`);
    }
    throw redirect(303, '/vineyard');
  },

  // ── Request membership (no account yet) ──
  request_membership: async ({ request }) => {
    const data = await request.formData();
    const email = (data.get('email') as string)?.trim().toLowerCase();
    const name = (data.get('name') as string)?.trim();

    if (!email || !email.includes('@')) {
      return fail(400, { error: 'Ange en giltig e-postadress.', membershipEmail: email });
    }
    if (!name) {
      return fail(400, { error: 'Ange ditt namn.', membershipEmail: email });
    }

    // Check if user already exists
    const existing = await getUserByEmail(email);
    if (existing) {
      // Already has an account — send magic link instead
      const token = await createMagicLink(existing.id);
      try {
        await sendMagicLink(email, token);
      } catch {
        // SMTP not configured — still return success
      }
      return { sent: true, vineyard: null, showPassword: false, membershipSent: true };
    }

    // TODO: In production, store the membership request and notify admin
    // For now, just acknowledge — admin manually creates account + sends invite
    console.log(`[Membership request] ${name} <${email}> — awaiting admin approval`);

    return { membershipSent: true, showPassword: false, sent: false };
  },
};
