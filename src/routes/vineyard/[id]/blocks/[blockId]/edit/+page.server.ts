// src/routes/vineyard/[id]/blocks/[blockId]/edit/+page.server.ts
import { redirect, error, fail } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { sql } from '$lib/server/db.js';

export const load: PageServerLoad = async ({ params, locals }) => {
  if (!locals.user) throw redirect(303, '/login');

  const vineyardId = Number(params.id);
  const blockId = Number(params.blockId);

  const [member] = await sql`
    SELECT role FROM vineyard_members
    WHERE vineyard_id = ${vineyardId} AND user_id = ${locals.user.id}
  `;
  if (!member) throw error(403, 'Du har inte tillgång till denna vingård.');
  if (member.role !== 'owner' && member.role !== 'editor') {
    throw error(403, 'Endast ägare eller redaktör kan redigera block.');
  }

  const [block] = await sql`
    SELECT b.*, v.name AS variety_name, v.color, v.piwi, v.status AS variety_status
    FROM blocks b
    JOIN varieties v ON v.id = b.variety_id
    WHERE b.id = ${blockId} AND b.vineyard_id = ${vineyardId} AND b.deleted_at IS NULL
  `;
  if (!block) throw error(404, 'Blocket hittades inte.');

  const varieties = await sql`
    SELECT id, name, piwi, color, status
    FROM varieties
    ORDER BY name
  `;

  return {
    block,
    varieties: varieties.map((v: Record<string, unknown>) => ({
      id: v.id,
      name: v.name,
      color: v.color,
      piwi: v.piwi,
      status: v.status
    })),
    role: member.role
  };
};

export const actions: Actions = {
  default: async ({ request, locals, params }) => {
    if (!locals.user) throw redirect(303, '/login');

    const vineyardId = Number(params.id);
    const blockId = Number(params.blockId);
    const data = await request.formData();

    const block_name = (data.get('block_name') as string)?.trim();
    const variety_id = data.get('variety_id') ? Number(data.get('variety_id')) : null;
    const area_ha = data.get('area_ha') ? Number(data.get('area_ha')) : null;
    const vine_count = data.get('vine_count') ? Number(data.get('vine_count')) : null;
    const planting_year = data.get('planting_year') ? Number(data.get('planting_year')) : null;
    const training_system = (data.get('training_system') as string)?.trim() || null;
    const aspect = (data.get('aspect') as string) || null;
    const slope_degrees = data.get('slope_degrees') ? Number(data.get('slope_degrees')) : null;
    const elevation_m = data.get('elevation_m') ? Number(data.get('elevation_m')) : null;

    if (!block_name) return fail(400, { error: 'Blocknamn krävs.' });
    if (!variety_id) return fail(400, { error: 'Välj en sort.' });
    if (!area_ha || area_ha <= 0) return fail(400, { error: 'Area måste vara större än 0.' });

    try {
      await sql`
        UPDATE blocks SET
          block_name = ${block_name},
          variety_id = ${variety_id},
          area_ha = ${area_ha},
          vine_count = ${vine_count},
          planting_year = ${planting_year},
          training_system = ${training_system},
          aspect = ${aspect},
          slope_degrees = ${slope_degrees},
          elevation_m = ${elevation_m}
        WHERE id = ${blockId} AND vineyard_id = ${vineyardId}
      `;
    } catch (err) {
      if (err?.code === '23505') {
        return fail(400, { error: 'Ett block med det namnet finns redan i denna vingård.' });
      }
      console.error('Failed to update block:', err);
      return fail(500, { error: 'Kunde inte uppdatera block. Försök igen.' });
    }

    throw redirect(303, `/vineyard/${vineyardId}`);
  }
};
