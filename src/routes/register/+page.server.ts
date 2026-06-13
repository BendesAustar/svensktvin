import { fail, redirect } from '@sveltejs/kit';
import { createMagicLink, createSession, getUserByEmail } from '$lib/server/auth';
import { randomBytes } from 'node:crypto';
import type { Actions, PageServerLoad, ServerLoadEvent } from './$types';
import { sql } from '$lib/server/db';

// ---------------------------------------------------------------------------
// Load: validate invite token, pre-fill email
// ---------------------------------------------------------------------------
export const load: PageServerLoad = async ({ url, cookies }) => {
  const inviteToken = url.searchParams.get('token');
  const email = url.searchParams.get('email') ?? '';

  if (!inviteToken || !email) {
    throw redirect(303, '/login');
  }

  // Validate invite token
  const [invite] = await sql<{
    id: number;
    email: string;
    role: string;
    vineyard_id: number;
    used: boolean;
    expires_at: Date;
  }>`
    SELECT id, email, role, vineyard_id, used, expires_at
    FROM pending_invites
    WHERE token = ${inviteToken} AND used = false
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
    invite,
    email: invite.email,
    hasAccount: !!existingUser,
    inviteToken
  };
};

// ---------------------------------------------------------------------------
// Actions: create account and auto-join vineyard
// ---------------------------------------------------------------------------
export const actions: Actions = {
  default: async (event: ServerLoadEvent) => {
    const data = await event.request.formData();
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
    const [invite] = await sql<{
      id: number;
      vineyard_id: number;
      role: string;
      used: boolean;
      expires_at: Date;
    }>`
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
    const [user] = await sql<{ id: number }>`
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

    // Create session
    await createSession(user.id, event.cookies);

    // Redirect to vineyard
    throw redirect(303, `/vineyard/${invite.vineyard_id}`);
  }
};
