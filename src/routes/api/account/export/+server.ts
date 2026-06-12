// src/routes/api/account/export/+server.ts
import { json, redirect } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { sql } from '$lib/server/db.js';

export const GET: RequestHandler = async ({ locals }) => {
  if (!locals.user) throw redirect(303, '/login');

  const userId = locals.user.id;

  // Gather all user data
  const [user] = await sql`
    SELECT id, email, name, is_admin, created_at, last_login
    FROM users WHERE id = ${userId}
  `;

  const vineyardMembers = await sql`
    SELECT v.id as vineyard_id, v.name as vineyard_name, vm.role
    FROM vineyard_members vm
    JOIN vineyards v ON v.id = vm.vineyard_id
    WHERE vm.user_id = ${userId}
  `;

  const vineyards = await Promise.all(
    vineyardMembers.map(async (member: { vineyard_id: number; vineyard_name: string; role: string }) => {
      const [vineyard] = await sql`
        SELECT id, name, county, municipality, organic, biodynamic,
               established_year, total_area_ha, legal_id, legal_id_type, legal_name
        FROM vineyards WHERE id = ${member.vineyard_id}
      `;

      const blocks = await sql`
        SELECT id, name, variety, area_ha, planted_year, notes
        FROM blocks WHERE vineyard_id = ${member.vineyard_id}
      `;

      const harvests = await sql`
        SELECT id, block_id, yield_kg, brix, acidity, ph, harvest_date, notes
        FROM harvest_records WHERE vineyard_id = ${member.vineyard_id}
      `;

      return {
        vineyard_id: vineyard?.id,
        name: vineyard?.name,
        county: vineyard?.county,
        municipality: vineyard?.municipality,
        organic: vineyard?.organic,
        biodynamic: vineyard?.biodynamic,
        established_year: vineyard?.established_year,
        total_area_ha: vineyard?.total_area_ha,
        role: member.role,
        blocks: blocks.map((b: Record<string, unknown>) => ({
          id: b.id,
          name: b.name,
          variety: b.variety,
          area_ha: b.area_ha,
          planted_year: b.planted_year,
          notes: b.notes
        })),
        harvests: harvests.map((h: Record<string, unknown>) => ({
          id: h.id,
          block_id: h.block_id,
          yield_kg: h.yield_kg,
          brix: h.brix,
          acidity: h.acidity,
          ph: h.ph,
          harvest_date: h.harvest_date,
          notes: h.notes
        }))
      };
    })
  );

  return json({
    user: {
      id: user.id,
      email: user.email,
      name: user.name,
      is_admin: user.is_admin,
      created_at: user.created_at,
      last_login: user.last_login
    },
    vineyards
  });
};
