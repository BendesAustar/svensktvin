// src/routes/auth/verify/+page.server.ts
import { redirect, error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { verifyToken, createSession, updateLastLogin } from '$lib/server/auth.js';
import { sql } from '$lib/server/db.js';

export const load: PageServerLoad = async ({ url, cookies }) => {
  const token = url.searchParams.get('token');
  if (!token) throw error(400, 'Token saknas.');

  const userId = await verifyToken(token);
  if (!userId) {
    throw error(400, 'Länken är ogiltig eller har gått ut. Begär en ny länk.');
  }

  await updateLastLogin(userId);
  const sessionId = await createSession(userId);

  cookies.set('session_id', sessionId, {
    path: '/',
    httpOnly: true,
    secure: process.env.NODE_ENV === 'production',
    sameSite: 'lax',
    maxAge: 30 * 24 * 60 * 60  // 30 days in seconds
  });

  // Auto-join any pending invites matching this user's email
  const [user] = await sql<{ email: string }[]>`SELECT email FROM users WHERE id = ${userId}`;

  if (user) {
    const [invite] = await sql`
      SELECT pi.vineyard_id, pi.role, pi.id
      FROM pending_invites pi
      WHERE pi.email ILIKE ${user.email}
        AND pi.used = false
        AND pi.expires_at > now()
      LIMIT 1
    `;

    if (invite) {
      try {
        await sql`
          INSERT INTO vineyard_members (vineyard_id, user_id, role)
          VALUES (${invite.vineyard_id}, ${userId}, ${invite.role})
          ON CONFLICT (vineyard_id, user_id) DO UPDATE SET role = EXCLUDED.role
        `;
        await sql`UPDATE pending_invites SET used = true WHERE id = ${invite.id}`;
      } catch (err) {
        console.error('Failed to auto-join vineyard from invite:', err);
      }
    }
  }

  // Determine redirect target
  const memberships = await sql`
    SELECT vineyard_id FROM vineyard_members WHERE user_id = ${userId}
  `;

  if (memberships.length === 0) throw redirect(303, '/onboard');
  if (memberships.length === 1) throw redirect(303, `/vineyard/${memberships[0].vineyard_id}`);
  throw redirect(303, '/');
};
