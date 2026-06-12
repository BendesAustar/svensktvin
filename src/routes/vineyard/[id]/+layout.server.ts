// src/routes/vineyard/[id]/+layout.server.ts
import { redirect, error } from '@sveltejs/kit';
import type { LayoutServerLoad } from './$types';
import { sql } from '$lib/server/db.js';

export const load: LayoutServerLoad = async ({ params, locals }) => {
  if (!locals.user) throw redirect(303, '/login');

  const vineyardId = Number(params.id);

  const [member] = await sql`
    SELECT role FROM vineyard_members
    WHERE vineyard_id = ${vineyardId} AND user_id = ${locals.user.id}
  `;
  if (!member) throw error(403, 'Du har inte tillgång till denna vingård.');

  const [vineyard] = await sql`
    SELECT id, name, county, municipality FROM vineyards
    WHERE id = ${vineyardId}
  `;
  if (!vineyard) throw error(404, 'Vingården hittades inte.');

  return { vineyard, role: member.role };
};
