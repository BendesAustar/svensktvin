// src/routes/invite/+server.ts
import { redirect, error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { sql } from '$lib/server/db.js';
import { createSession } from '$lib/server/auth.js';

export const GET: RequestHandler = async ({ url, cookies, locals }) => {
  const token = url.searchParams.get('token');
  if (!token) throw error(400, 'Ogiltig inbjudan.');

  // Validate the invite
  const [invite] = await sql`
    SELECT pi.id, pi.email, pi.vineyard_id, pi.role, pi.expires_at
    FROM pending_invites pi
    WHERE pi.token = ${token}
      AND pi.used = false
      AND pi.expires_at > now()
    LIMIT 1
  `;

  if (!invite) {
    return new Response(
      JSON.stringify({ error: 'Inbjudan har gått ut eller är ogiltig.' }),
      { status: 400, headers: { 'Content-Type': 'application/json' } }
    );
  }

  const isLoggedIn = !!locals.user;

  if (isLoggedIn) {
    // User is logged in — check if their email matches the invite
    const userEmail = locals.user?.email ?? '';
    if (userEmail.toLowerCase() !== invite.email.toLowerCase()) {
      throw error(403, 'Ditt konto matchar inte inbjudan. Logga in med rätt e-postadress.');
    }

    // Add to vineyard
    try {
      const userId = locals.user!.id;
      await sql`
        INSERT INTO vineyard_members (vineyard_id, user_id, role)
        VALUES (${invite.vineyard_id}, ${userId}, ${invite.role})
        ON CONFLICT (vineyard_id, user_id) DO UPDATE SET role = EXCLUDED.role
      `;

      // Mark invite as used
      await sql`UPDATE pending_invites SET used = true WHERE id = ${invite.id}`;

      // Redirect to vineyard
      throw redirect(303, `/vineyard/${invite.vineyard_id}`);
    } catch (err) {
      console.error('Failed to add to vineyard:', err);
      throw error(500, 'Kunde inte gå med i vingården. Försök igen.');
    }
  }

  // Not logged in — redirect to login with invite token for callback
  throw redirect(303, `/login?invite=${encodeURIComponent(token)}`);
};
