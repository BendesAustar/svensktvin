import { fail, redirect } from '@sveltejs/kit';
import { createSession, getUserByEmail } from '$lib/server/auth';
import type { Actions, PageServerLoad } from './$types';
import { sql } from '$lib/server/db.js';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
interface InviteRow {
  id: number;
  email: string;
  role: string;
  vineyard_id: number;
  used: boolean;
  expires_at: Date;
  vineyard_name: string;
}

// ---------------------------------------------------------------------------
// Load: validate invite token, pre-fill email
// ---------------------------------------------------------------------------
export const load: PageServerLoad = async ({ url }) => {
  const inviteToken = url.searchParams.get('token');
  const email = url.searchParams.get('email') ?? '';

  if (!inviteToken || !email) {
    throw redirect(303, '/login');
  }

  // Validate invite token + load vineyard name
  const [invite] = await sql<InviteRow[]>`
    SELECT pi.id, pi.email, pi.role, pi.vineyard_id, pi.used, pi.expires_at,
           v.name AS vineyard_name
    FROM pending_invites pi
    JOIN vineyards v ON v.id = pi.vineyard_id
    WHERE pi.token = ${inviteToken} AND pi.used = false
    LIMIT 1
  `;

  if (!invite) {
    return { error: 'Inbjudan är ogiltig eller har gått ut.' };
  }

  if (invite.expires_at && invite.expires_at < new Date()) {
    return { error: 'Inbjudan har gått ut.' };
  }

  // Check if email already has an account
  const existingUser = await getUserByEmail(invite.email);

  return {
    invite: {
      id: invite.id,
      email: invite.email,
      role: invite.role,
      vineyard_id: invite.vineyard_id,
      vineyard: { name: invite.vineyard_name }
    },
    email: invite.email,
    hasAccount: !!existingUser,
    inviteToken
  };
};

// ---------------------------------------------------------------------------
// Actions: create account and auto-join vineyard
// ---------------------------------------------------------------------------
export const actions: Actions = {
  default: async ({ request, cookies }) => {
    const data = await request.formData();
    const email = (data.get('email') as string)?.trim().toLowerCase();
    const name = (data.get('name') as string)?.trim();
    const inviteToken = (data.get('invite_token') as string) ?? '';
    const action = (data.get('action') as string) ?? '';

    if (action !== 'register') {
      return fail(400, { error: 'Ogiltig åtgärd.' });
    }

    if (!name || name.length < 2) {
      return fail(400, { error: 'Ange ditt namn (minst 2 tecken).' });
    }
    if (!email || !email.includes('@')) {
      return fail(400, { error: 'Ange en giltig e-postadress.' });
    }

    // Validate invite token
    const [invite] = await sql<InviteRow[]>`
      SELECT id, vineyard_id, role, used, expires_at
      FROM pending_invites
      WHERE token = ${inviteToken} AND used = false
      LIMIT 1
    `;

    if (!invite) {
      return fail(400, { error: 'Inbjudan är ogiltig eller har gått ut.' });
    }

    if (invite.expires_at && invite.expires_at < new Date()) {
      return fail(400, { error: 'Inbjudan har gått ut.' });
    }

    // Check if user already exists
    const existingUser = await getUserByEmail(email);
    if (existingUser) {
      return fail(400, {
        error:
          'Det finns redan ett konto för den e-postadressen. Logga in istället.',
        email
      });
    }

    // Create user account
    const [user] = await sql<{ id: number }[]>`
      INSERT INTO users (name, email, active)
      VALUES (${name}, ${email}, true)
      RETURNING id
    `;

    // Auto-join vineyard
    await sql`
      INSERT INTO vineyard_members (vineyard_id, user_id, role)
      VALUES (${invite.vineyard_id}, ${user.id}, ${invite.role})
    `;

    // Mark invite as used
    await sql`
      UPDATE pending_invites SET used = true WHERE id = ${invite.id}
    `;

    // Create session + set cookie
    const sessionId = await createSession(user.id);
    cookies.set('session_id', sessionId, {
      path: '/',
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      sameSite: 'lax',
      maxAge: 30 * 24 * 60 * 60
    });

    // Redirect to vineyard
    throw redirect(303, `/vineyard/${invite.vineyard_id}`);
  }
};
