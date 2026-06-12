// src/routes/vineyard/[id]/benchmark/+page.server.ts
import { redirect, error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { sql } from '$lib/server/db.js';

export const load: PageServerLoad = async ({ params, locals }) => {
  if (!locals.user) throw redirect(303, '/login');

  const vineyardId = Number(params.id);

  const [member] = await sql`
    SELECT role FROM vineyard_members
    WHERE vineyard_id = ${vineyardId} AND user_id = ${locals.user.id}
  `;
  if (!member) throw error(403, 'Du har inte tillgång till denna vingård.');

  // Load vineyard info
  const [vineyard] = await sql`
    SELECT name, county FROM vineyards WHERE id = ${vineyardId} AND deleted_at IS NULL
  `;

  // Load user's variety-level yields by year
  const userYields = await sql`
    SELECT
      v.name AS variety_name,
      hr.harvest_year,
      round(avg(hr.yield_kg / b.area_ha), 0)::int AS avg_yield_kg_ha,
      round(avg(hr.brix::numeric), 1) AS avg_brix,
      count(*) AS harvest_count
    FROM harvest_records hr
    JOIN blocks b ON b.id = hr.block_id
    JOIN varieties v ON v.id = b.variety_id
    WHERE b.vineyard_id = ${vineyardId}
      AND hr.harvest_year IS NOT NULL
    GROUP BY v.name, hr.harvest_year
    ORDER BY hr.harvest_year DESC, v.name
  `;

  // Load regional benchmark: for each variety, compare user to other vineyards in same county
  const regionalBenchmarks = await sql`
    SELECT
      var.name AS variety_name,
      hr.harvest_year,
      round(avg(hr.yield_kg / b.area_ha), 0)::int AS county_avg_kg_ha,
      round(avg(hr.brix::numeric), 1) AS county_avg_brix,
      count(DISTINCT vi.id) AS vineyard_count
    FROM harvest_records hr
    JOIN blocks b ON b.id = hr.block_id
    JOIN varieties var ON var.id = b.variety_id
    JOIN vineyards vi ON vi.id = b.vineyard_id
    WHERE vi.county = ${vineyard.county}
      AND var.status = 'approved'
      AND hr.harvest_year IS NOT NULL
    GROUP BY var.name, hr.harvest_year
    HAVING count(DISTINCT vi.id) >= 3
    ORDER BY hr.harvest_year DESC, var.name
  `;

  // Load all-harvests timeline
  const timeline = await sql`
    SELECT
      hr.harvest_year,
      hr.harvest_date,
      b.block_name,
      v.name AS variety_name,
      hr.yield_kg,
      hr.brix,
      hr.vine_health_rating,
      b.area_ha
    FROM harvest_records hr
    JOIN blocks b ON b.id = hr.block_id
    JOIN varieties v ON v.id = b.variety_id
    WHERE b.vineyard_id = ${vineyardId}
      AND hr.harvest_year IS NOT NULL
    ORDER BY hr.harvest_year DESC, hr.harvest_date DESC
  `;

  return {
    vineyard,
    userYields: userYields.map((r: Record<string, unknown>) => ({
      variety_name: r.variety_name,
      harvest_year: r.harvest_year,
      avg_yield_kg_ha: r.avg_yield_kg_ha,
      avg_brix: r.avg_brix,
      harvest_count: r.harvest_count
    })),
    regionalBenchmarks: regionalBenchmarks.map((r: Record<string, unknown>) => ({
      variety_name: r.variety_name,
      harvest_year: r.harvest_year,
      county_avg_kg_ha: r.county_avg_kg_ha,
      county_avg_brix: r.county_avg_brix,
      vineyard_count: r.vineyard_count
    })),
    timeline: timeline.map((r: Record<string, unknown>) => ({
      harvest_year: r.harvest_year,
      harvest_date: r.harvest_date,
      block_name: r.block_name,
      variety_name: r.variety_name,
      yield_kg: r.yield_kg,
      brix: r.brix,
      vine_health_rating: r.vine_health_rating,
      area_ha: r.area_ha
    }))
  };
};
