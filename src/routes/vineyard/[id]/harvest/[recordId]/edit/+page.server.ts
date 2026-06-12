// src/routes/vineyard/[id]/harvest/[recordId]/edit/+page.server.ts
import { redirect, error, fail } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { sql } from '$lib/server/db.js';

export const load: PageServerLoad = async ({ params, locals }) => {
  if (!locals.user) throw redirect(303, '/login');

  const vineyardId = Number(params.id);
  const recordId = Number(params.recordId);

  const [member] = await sql`
    SELECT role FROM vineyard_members
    WHERE vineyard_id = ${vineyardId} AND user_id = ${locals.user.id}
  `;
  if (!member) throw error(403, 'Du har inte tillgång till denna vingård.');

  const [record] = await sql`
    SELECT hr.*, b.block_name, b.variety_id, v.name AS variety_name
    FROM harvest_records hr
    JOIN blocks b ON b.id = hr.block_id
    JOIN varieties v ON v.id = b.variety_id
    WHERE hr.id = ${recordId}
      AND b.vineyard_id = ${vineyardId}
  `;
  if (!record) throw error(404, 'Skörden hittades inte.');

  const blocks = await sql`
    SELECT b.id, b.block_name, v.name AS variety_name
    FROM blocks b
    JOIN varieties v ON v.id = b.variety_id
    WHERE b.vineyard_id = ${vineyardId} AND b.is_active = true
    ORDER BY b.block_name
  `;

  return {
    record,
    blocks: blocks.map((b: Record<string, unknown>) => ({
      id: b.id,
      block_name: b.block_name,
      variety_name: b.variety_name
    }))
  };
};

export const actions: Actions = {
  default: async ({ request, locals, params }) => {
    if (!locals.user) throw redirect(303, '/login');

    const vineyardId = Number(params.id);
    const recordId = Number(params.recordId);
    const data = await request.formData();

    const harvest_date = (data.get('harvest_date') as string) || null;
    const yield_kg = data.get('yield_kg') ? Number(data.get('yield_kg')) : null;
    const brix = data.get('brix') ? Number(data.get('brix')) : null;
    const acid_g_l = data.get('acid_g_l') ? Number(data.get('acid_g_l')) : null;
    const vine_health_rating = data.get('vine_health_rating') ? Number(data.get('vine_health_rating')) : null;
    const notes = (data.get('notes') as string)?.trim() || null;
    const still_wine_l = data.get('still_wine_l') ? Number(data.get('still_wine_l')) : null;
    const sparkling_l = data.get('sparkling_l') ? Number(data.get('sparkling_l')) : null;
    const juice_l = data.get('juice_l') ? Number(data.get('juice_l')) : null;
    const sold_kg = data.get('sold_kg') ? Number(data.get('sold_kg')) : null;
    const discarded_kg = data.get('discarded_kg') ? Number(data.get('discarded_kg')) : null;

    if (!yield_kg || yield_kg <= 0) return fail(400, { error: 'Skördevikt (kg) måste vara större än 0.' });

    try {
      await sql`
        UPDATE harvest_records SET
          harvest_date       = ${harvest_date},
          yield_kg           = ${yield_kg},
          brix               = ${brix},
          acid_g_l           = ${acid_g_l},
          vine_health_rating = ${vine_health_rating},
          notes              = ${notes},
          still_wine_l       = ${still_wine_l},
          sparkling_l        = ${sparkling_l},
          juice_l            = ${juice_l},
          sold_kg            = ${sold_kg},
          discarded_kg       = ${discarded_kg},
          updated_at         = now()
        WHERE id = ${recordId}
          AND block_id IN (SELECT id FROM blocks WHERE vineyard_id = ${vineyardId})
      `;
    } catch (err) {
      console.error('Failed to update harvest record:', err);
      return fail(500, { error: 'Kunde inte uppdatera skörd. Försök igen.' });
    }

    throw redirect(303, `/vineyard/${vineyardId}`);
  }
};
