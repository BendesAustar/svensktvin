// src/routes/vineyard/[id]/blocks/new/+page.server.ts
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
  if (member.role !== 'owner' && member.role !== 'editor') {
    throw error(403, 'Endast ägare eller redaktör kan skapa block.');
  }

  return { role: member.role };
};

export const actions: Actions = {
  default: async ({ request, locals, params }) => {
    if (!locals.user) throw redirect(303, '/login');

    const vineyardId = Number(params.id);
    const data = await request.formData();

    const block_name = (data.get('block_name') as string)?.trim();
    const variety_id = data.get('variety_id') ? Number(data.get('variety_id')) : null;
    const variety_name = (data.get('variety_name') as string)?.trim() || null;
    const area_ha = data.get('area_ha') ? Number(data.get('area_ha')) : null;
    const vine_count = data.get('vine_count') ? Number(data.get('vine_count')) : null;
    const planting_year = data.get('planting_year') ? Number(data.get('planting_year')) : null;
    const training_system = (data.get('training_system') as string)?.trim() || null;
    const aspect = (data.get('aspect') as string) || null;
    const slope_degrees = data.get('slope_degrees') ? Number(data.get('slope_degrees')) : null;
    const elevation_m = data.get('elevation_m') ? Number(data.get('elevation_m')) : null;

    // Validation
    if (!block_name) return fail(400, { error: 'Blocknamn krävs.' });
    if (!variety_id && !variety_name) return fail(400, { error: 'Välj eller sök efter en sort.' });
    if (!area_ha || area_ha <= 0) return fail(400, { error: 'Area måste vara större än 0.' });

    // Verify membership
    const [member] = await sql`
      SELECT role FROM vineyard_members
      WHERE vineyard_id = ${vineyardId} AND user_id = ${locals.user.id}
    `;
    if (!member) throw error(403, 'Du har inte tillgång till denna vingård.');

    let finalVarietyId = variety_id;

    // If variety_name provided (new review_needed variety)
    if (!variety_id && variety_name) {
      // Normalize to Title Case (initcap) to prevent ALL CAPS entries
      const [result] = await sql`
        INSERT INTO varieties (name, piwi, color, status, submitted_by_vineyard_id)
        VALUES ((SELECT string_agg(initcap(word), ' ') FROM regexp_split_to_table(${variety_name}, '\s+'))), false, 'other', 'review_needed', ${vineyardId})
        ON CONFLICT (LOWER(name)) DO NOTHING
        RETURNING id
      `;
      if (!result) {
        // Variety already exists (duplicate name)
        const [existing] = await sql`
          SELECT id FROM varieties WHERE LOWER(name) = LOWER(${variety_name})
        `;
        if (existing) {
          finalVarietyId = existing.id;
        } else {
          return fail(500, { error: 'Kunde inte skapa sort. Försök igen.' });
        }
      } else {
        finalVarietyId = result.id;
      }
    }

    if (!finalVarietyId) {
      return fail(500, { error: 'Kunde inte skapa block.' });
    }

    try {
      await sql`
        INSERT INTO blocks (
          vineyard_id, variety_id, block_name, area_ha,
          vine_count, planting_year, training_system, aspect,
          slope_degrees, elevation_m
        ) VALUES (
          ${vineyardId}, ${finalVarietyId}, ${block_name}, ${area_ha},
          ${vine_count}, ${planting_year}, ${training_system}, ${aspect},
          ${slope_degrees}, ${elevation_m}
        )
      `;
    } catch (err) {
      if ((err as { code?: string })?.code === '23505') {
        return fail(400, { error: 'Ett block med det namnet finns redan i denna vingård.' });
      }
      console.error('Failed to create block:', err);
      return fail(500, { error: 'Kunde inte skapa block. Försök igen.' });
    }

    throw redirect(303, `/vineyard/${vineyardId}`);
  }
};
