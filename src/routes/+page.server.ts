// src/routes/+page.server.ts
import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { sql } from '$lib/server/db.js';

export const load: PageServerLoad = async ({ locals }) => {
  if (!locals.user) throw redirect(303, '/login');

  const memberships = await sql`
    SELECT vineyard_id, role FROM vineyard_members WHERE user_id = ${locals.user.id}
  `;

  if (memberships.length === 0) throw redirect(303, '/onboard');
  if (memberships.length === 1) throw redirect(303, `/vineyard/${memberships[0].vineyard_id}`);

  return { vineyards: memberships };
};
