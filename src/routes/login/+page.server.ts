// src/routes/login/+page.server.ts
import { redirect, fail } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { getUserByEmail, createMagicLink } from '$lib/server/auth.js';
import { sendMagicLink } from '$lib/server/email.js';
import { sql } from '$lib/server/db.js';

interface InviteVineyard {
  vineyard_name: string;
}

function getInviteVineyard(inviteToken: string | null) {
  if (!inviteToken) return Promise.resolve(null);
  return sql<InviteVineyard[]>`
    SELECT v.name AS vineyard_name
    FROM pending_invites pi
    JOIN vineyards v ON v.id = pi.vineyard_id
    WHERE pi.token = ${inviteToken} AND pi.used = false AND pi.expires_at > now()
    LIMIT 1
  `;
}

export const load: PageServerLoad = async ({ url }) => {
  const inviteToken = url.searchParams.get('invite');
  let vineyard: { name: string } | null = null;
  const rows = await getInviteVineyard(inviteToken);
  if (rows?.[0]) vineyard = { name: rows[0].vineyard_name };
  return { inviteToken, vineyard };
};

export const actions: Actions = {
  default: async ({ request, url }) => {
    const data = await request.formData();
    const email = (data.get('email') as string)?.trim().toLowerCase();

    if (!email || !email.includes('@')) {
      return fail(400, { error: 'Ange en giltig e-postadress.', email });
    }

    // Preserve invite context
    const inviteToken = url.searchParams.get('invite');
    const inviteRows = await getInviteVineyard(inviteToken);
    const vineyard = inviteRows?.[0] ? { name: inviteRows[0].vineyard_name } : null;

    // Look up user — always return the same message to avoid account enumeration
    const user = await getUserByEmail(email);
    if (user) {
      const token = await createMagicLink(user.id);
      await sendMagicLink(email, token);
      return { sent: true, inviteToken, vineyard };
    }

    // Non-registered user — redirect to register page with invite context
    if (inviteToken) {
      throw redirect(303, `/register?token=${encodeURIComponent(inviteToken)}&email=${encodeURIComponent(email)}`);
    }

    return { sent: true, vineyard };
  }
};
