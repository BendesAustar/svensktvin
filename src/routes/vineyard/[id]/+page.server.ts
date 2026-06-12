// src/routes/vineyard/[id]/+page.server.ts
import { redirect, error, fail } from '@sveltejs/kit';
import type { PageServerLoad, Actions } from './$types';
import { sql } from '$lib/server/db.js';

export const load: PageServerLoad = async ({ params, locals }) => {
  if (!locals.user) throw redirect(303, '/login');

  const vineyardId = Number(params.id);

  // Verify membership
  const [member] = await sql`
    SELECT role FROM vineyard_members
    WHERE vineyard_id = ${vineyardId} AND user_id = ${locals.user.id}
  `;
  if (!member) throw error(403, 'Du har inte tillgång till denna vingård.');

  // Load vineyard
  const [vineyard] = await sql`
    SELECT id, name, county, municipality, organic, biodynamic,
           established_year, total_area_ha, legal_name
    FROM vineyards WHERE id = ${vineyardId} AND deleted_at IS NULL
  `;
  if (!vineyard) throw error(404, 'Vingården hittades inte.');

  // Load blocks with latest harvest
  const blocks = await sql`
    SELECT
      b.id, b.block_name, b.area_ha, b.is_active,
      v.name AS variety_name,
      v.status AS variety_status,
      hr.id AS latest_harvest_id,
      hr.harvest_year,
      hr.yield_kg,
      hr.brix
    FROM blocks b
    JOIN varieties v ON v.id = b.variety_id
    LEFT JOIN LATERAL (
      SELECT id, harvest_year, yield_kg, brix
      FROM harvest_records
      WHERE block_id = b.id AND deleted_at IS NULL
      ORDER BY harvest_year DESC
      LIMIT 1
    ) hr ON true
    WHERE b.vineyard_id = ${vineyardId}
      AND b.deleted_at IS NULL
    ORDER BY b.block_name
  `;

  // Load benchmark teaser (user's most-planted variety)
  const [mostPlanted] = await sql`
    SELECT v.name AS variety_name, count(*) AS block_count
    FROM blocks b
    JOIN varieties v ON v.id = b.variety_id
    WHERE b.vineyard_id = ${vineyardId}
      AND b.is_active = true
    GROUP BY v.name
    ORDER BY block_count DESC
    LIMIT 1
  `;

  let benchmarkTeaser = null;
  if (mostPlanted) {
    const [teaser] = await sql`
      SELECT
        v.county,
        round(avg(hr.yield_kg / b.area_ha), 0)::int AS avg_yield_kg_ha,
        count(DISTINCT vi.id) AS vineyard_count
      FROM harvest_records hr
      JOIN blocks b ON b.id = hr.block_id
      JOIN varieties var ON var.id = b.variety_id
      JOIN vineyards vi ON vi.id = b.vineyard_id
      WHERE var.name = ${mostPlanted.variety_name}
        AND var.status = 'approved'
        AND hr.harvest_year = EXTRACT(YEAR FROM now())::int
        AND vi.county = ${vineyard.county}
      GROUP BY v.county
      HAVING count(DISTINCT vi.id) >= 3
    `;

    if (teaser && teaser.avg_yield_kg_ha) {
      // Get user's own yield for this variety
      const [userYield] = await sql`
        SELECT round(avg(hr.yield_kg / b.area_ha), 0)::int AS yield_kg_ha
        FROM harvest_records hr
        JOIN blocks b ON b.id = hr.block_id
        JOIN varieties v ON v.id = b.variety_id
        WHERE b.vineyard_id = ${vineyardId}
          AND v.name = ${mostPlanted.variety_name}
          AND hr.harvest_year = EXTRACT(YEAR FROM now())::int
      `;

      benchmarkTeaser = {
        variety_name: teaser.variety_name,
        user_yield_kg_ha: userYield?.yield_kg_ha ?? 0,
        county_avg_kg_ha: teaser.avg_yield_kg_ha,
        vineyard_count: teaser.vineyard_count
      };
    }
  }

  return {
    vineyard,
    blocks: blocks.map((b: Record<string, unknown>) => ({
      id: b.id,
      block_name: b.block_name,
      variety_name: b.variety_name,
      variety_status: b.variety_status,
      area_ha: b.area_ha,
      is_active: b.is_active,
      latest_harvest_id: b.latest_harvest_id,
      latest_harvest: b.latest_harvest_id ? {
        harvest_year: b.harvest_year,
        yield_kg: b.yield_kg
      } : null
    })),
    benchmarkTeaser,
    role: member.role
  };
};
