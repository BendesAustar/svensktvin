// src/routes/vineyard/[id]/harvest/new/+page.server.ts
import { redirect, error, fail } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { sql } from '$lib/server/db.js';

export const load: PageServerLoad = async ({ params, locals }) => {
  if (!locals.user) throw redirect(303, '/login');

  const vineyardId = Number(params.id);

  const [member] = await sql`
    SELECT role FROM vineyard_members
    WHERE vineyard_id = ${vineyardId} AND user_id = ${locals.user.id}
  `;
  if (!member) throw error(403, 'Du har inte tillgång till denna vingård.');

  const blocks = await sql`
    SELECT b.id, b.block_name, v.name AS variety_name
    FROM blocks b
    JOIN varieties v ON v.id = b.variety_id
    WHERE b.vineyard_id = ${vineyardId} AND b.is_active = true
    ORDER BY b.block_name
  `;

  const currentYear = new Date().getFullYear();
  const years = Array.from({ length: 5 }, (_, i) => currentYear - i);

  return {
    blocks: blocks.map((b: Record<string, unknown>) => ({
      id: b.id,
      block_name: b.block_name,
      variety_name: b.variety_name
    })),
    years
  };
};

export const actions: Actions = {
  default: async ({ request, locals, params }) => {
    if (!locals.user) throw redirect(303, '/login');

    const vineyardId = Number(params.id);
    const data = await request.formData();

    const block_id = data.get('block_id') ? Number(data.get('block_id')) : null;
    const harvest_year = data.get('harvest_year') ? Number(data.get('harvest_year')) : null;
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

    // Validation
    if (!block_id) return fail(400, { error: 'Välj ett block.' });
    if (!harvest_year) return fail(400, { error: 'Skördeår krävs.' });
    if (!yield_kg || yield_kg <= 0) return fail(400, { error: 'Skördevikt (kg) måste vara större än 0.' });

    // Verify block belongs to vineyard
    const [block] = await sql`
      SELECT id FROM blocks WHERE id = ${block_id} AND vineyard_id = ${vineyardId}
    `;
    if (!block) throw error(404, 'Blocket hittades inte.');

    // Check for duplicate year
    const [existing] = await sql`
      SELECT id FROM harvest_records WHERE block_id = ${block_id} AND harvest_year = ${harvest_year}
    `;
    if (existing) {
      return fail(400, { error: `En skörd för ${harvest_year} finns redan.` });
    }

    try {
      await sql`
        INSERT INTO harvest_records (
          block_id, harvest_year, harvest_date, yield_kg,
          brix, acid_g_l, vine_health_rating, notes,
          still_wine_l, sparkling_l, juice_l,
          sold_kg, discarded_kg
        ) VALUES (
          ${block_id}, ${harvest_year}, ${harvest_date}, ${yield_kg},
          ${brix}, ${acid_g_l}, ${vine_health_rating}, ${notes},
          ${still_wine_l}, ${sparkling_l}, ${juice_l},
          ${sold_kg}, ${discarded_kg}
        )
      `;
    } catch (err) {
      console.error('Failed to create harvest record:', err);
      return fail(500, { error: 'Kunde inte spara skörd. Försök igen.' });
    }

    throw redirect(303, `/vineyard/${vineyardId}`);
  }
};
