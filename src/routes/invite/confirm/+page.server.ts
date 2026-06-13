// src/routes/invite/confirm/+page.server.ts
import { redirect, error, fail } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { sql } from '$lib/server/db.js';

export const load: PageServerLoad = async ({ url, locals }) => {
  const token = url.searchParams.get('token');
  const inviteEmail = url.searchParams.get('invite_email');
  const currentEmail = url.searchParams.get('current_email');

  if (!token || !inviteEmail || !currentEmail) {
    throw error(400, 'Ogiltig inbjudan.');
  }

  // Validate the invite
  const [invite] = await sql`
    SELECT pi.id, pi.vineyard_id, pi.role, v.name AS vineyard_name
    FROM pending_invites pi
    JOIN vineyards v ON v.id = pi.vineyard_id
    WHERE pi.token = ${token}
      AND pi.used = false
      AND pi.expires_at > now()
      AND pi.email ILIKE ${inviteEmail}
    LIMIT 1
  `;

  if (!invite) {
    throw error(400, 'Inbjudan har gått ut eller är ogiltig.');
  }

  return {
    inviteEmail,
    currentEmail,
    vineyardName: invite.vineyard_name,
    role: invite.role,
  };
};

export const actions: Actions = {
  default: async ({ request, url, locals }) => {
    if (!locals.user) throw redirect(303, '/login');

    const token = url.searchParams.get('token');
    const inviteEmail = url.searchParams.get('invite_email');

    if (!token || !inviteEmail) {
      return fail(400, { error: 'Ogiltig inbjudan.' });
  }

    // Validate the invite
    const [invite] = await sql`
      SELECT pi.id, pi.vineyard_id, pi.role
      FROM pending_invites pi
      WHERE pi.token = ${token}
        AND pi.used = false
        AND pi.expires_at > now()
        AND pi.email ILIKE ${inviteEmail}
      LIMIT 1
    `;

    if (!invite) {
      return fail(400, { error: 'Inbjudan har gått ut eller är ogiltig.' });
    }

    // Add to vineyard (idempotent)
    try {
      await sql`
        INSERT INTO vineyard_members (vineyard_id, user_id, role)
        VALUES (${invite.vineyard_id}, ${locals.user.id}, ${invite.role})
        ON CONFLICT (vineyard_id, user_id) DO UPDATE SET role = EXCLUDED.role
      `;

      // Mark invite as used
      await sql`UPDATE pending_invites SET used = true WHERE id = ${invite.id}`;

      // Redirect to vineyard
      throw redirect(303, `/vineyard/${invite.vineyard_id}`);
    } catch (err) {
      console.error('Failed to add to vineyard:', err);
      return fail(500, { error: 'Kunde inte gå med i vingården. Försök igen.' });
    }
  },
};
