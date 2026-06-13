// src/routes/register/+page.server.ts
import { redirect, fail } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import {
  getUserByEmail,
  hashPassword,
  verifyPassword,
  passwordValidation,
  createSession,
  updateLastLogin,
} from '$lib/server/auth.js';
import { sql } from '$lib/server/db.js';

interface InviteVineyard {
  vineyard_name: string;
  role: string;
}

function getInviteData(inviteToken: string | null) {
  if (!inviteToken) return Promise.resolve(null);
  return sql<{
    vineyard_id: number;
    vineyard_name: string;
    role: string;
    company_name: string | null;
    owner_name: string | null;
    email: string;
    used: boolean;
    expires_at: string;
  }[]>`
    SELECT pi.vineyard_id, v.name AS vineyard_name, pi.role,
           pi.company_name, pi.owner_name, pi.email, pi.used, pi.expires_at
    FROM pending_invites pi
    JOIN vineyards v ON v.id = pi.vineyard_id
    WHERE pi.token = ${inviteToken} AND pi.used = false AND pi.expires_at > now()
    LIMIT 1
  `;
}

export const load: PageServerLoad = async ({ url }) => {
  const inviteToken = url.searchParams.get('invite');
  const rows = await getInviteData(inviteToken);
  const invite = rows?.[0] ?? null;

  // Check if user already has an account with the invite email
  let hasAccount = false;
  if (invite) {
    const existing = await sql<{ id: number }[]>`
      SELECT id FROM users WHERE email = ${invite.email} AND active = true
    `;
    hasAccount = existing.length > 0;
  }

  return {
    inviteToken,
    invite: invite
      ? {
          vineyard: { name: invite.vineyard_name },
          role: invite.role,
          companyName: invite.company_name ?? undefined,
          ownerName: invite.owner_name ?? undefined,
          email: invite.email,
        }
      : undefined,
    hasAccount,
  };
};

export const actions: Actions = {
  default: async ({ request, url, cookies }) => {
    const data = await request.formData();
    const name = (data.get('name') as string)?.trim();
    const password = (data.get('password') as string) ?? '';
    const confirmPassword = (data.get('confirm_password') as string) ?? '';
    const inviteToken = (data.get('invite_token') as string)?.trim();

    // Validate required fields
    if (!name || !password || !confirmPassword || !inviteToken) {
      return fail(400, { error: 'Alla fält är obligatoriska.' });
    }

    // Validate invite
    const inviteRows = await sql<{
      vineyard_id: number;
      used: boolean;
      expires_at: string;
      email: string;
    }[]>`
      SELECT vineyard_id, used, expires_at, email
      FROM pending_invites
      WHERE token = ${inviteToken} AND used = false AND expires_at > now()
      LIMIT 1
    `;

    if (inviteRows.length === 0) {
      return fail(404, { error: 'Ogiltig eller utgången inbjudningskod.' });
    }

    const inviteData = inviteRows[0];

    if (inviteData.used) {
      return fail(400, { error: 'Inbjudningen har redan använts.' });
    }

    if (new Date(inviteData.expires_at) < new Date()) {
      return fail(400, { error: 'Inbjudningen har utgått.' });
    }

    // Check if user already has an account with this email
    const existingUser = await sql<{ id: number }[]>`
      SELECT id FROM users WHERE email = ${inviteData.email} AND active = true
    `;
    if (existingUser.length > 0) {
      // User already exists — redirect to login with invite context
      throw redirect(303, `/login?invite=${encodeURIComponent(inviteToken)}&email=${encodeURIComponent(inviteData.email)}`);
    }

    // Validate password strength
    const validation = passwordValidation(password);
    if (!validation.valid) {
      return fail(400, { error: validation.errors[0] });
    }

    // Check password match
    if (password !== confirmPassword) {
      return fail(400, { error: 'Lösenorden matchar inte.' });
    }

    // Hash password and create user
    const hashedPassword = await hashPassword(password);

    try {
      await sql`
        BEGIN;
        INSERT INTO users (email, name, password_hash, active)
        VALUES (${inviteData.email}, ${name}, ${hashedPassword}, true)
        RETURNING id;
        INSERT INTO vineyard_members (vineyard_id, user_id, role)
        VALUES (${inviteData.vineyard_id}, (SELECT id FROM users WHERE email = ${inviteData.email}), ${inviteData.role === 'editor' ? 'editor' : 'owner'});
        UPDATE pending_invites SET used = true WHERE token = ${inviteToken};
        COMMIT;
      `;
    } catch (e) {
      console.error('Registration error:', e);
      return fail(500, { error: 'Ett fel uppstod under registreringen. Försök igen senare.' });
    }

    // Create session
    const [user] = await sql<{ id: number }[]>`
      SELECT id FROM users WHERE email = ${inviteData.email}
    `;
    const sessionId = await createSession(user.id);
    await updateLastLogin(user.id);

    // Set secure session cookie
    cookies.set('session_id', sessionId, {
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      sameSite: 'lax',
      path: '/',
      maxAge: 30 * 24 * 60 * 60,
    });

    throw redirect(302, '/vineyard');
  },
};
