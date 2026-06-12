// src/routes/benchmarks/+page.server.ts
import type { PageServerLoad } from './$types';
import { redirect } from '@sveltejs/kit';
import { sql } from '$lib/server/db.js';

interface BenchmarkRow {
  variety_name: string;
  county: string;
  harvest_year: number;
  avg_yield_kg_ha: number;
  min_yield_kg_ha: number;
  max_yield_kg_ha: number;
  vineyard_count: number;
  user_yield_kg_ha: number | null;
}

export const load: PageServerLoad = async ({ url, locals }) => {
  if (!locals.user) throw redirect(303, '/login');

  const selectedYear = url.searchParams.get('year')
    ? parseInt(url.searchParams.get('year')!)
    : null;
  const selectedVariety = url.searchParams.get('variety') ?? null;

  // Get user's vineyard IDs for highlighting
  const memberRows = await sql<{ vineyard_id: number }[]>`
    SELECT vineyard_id FROM vineyard_members WHERE user_id = ${locals.user.id}
  `;
  const vineyardIds = memberRows.map((r) => r.vineyard_id);

  const rows = await sql<BenchmarkRow[]>`
    WITH county_benchmarks AS (
      SELECT
        var.name                                           AS variety_name,
        vi.county,
        hr.harvest_year,
        round(avg(hr.yield_kg / b.area_ha)::numeric, 0)   AS avg_yield_kg_ha,
        round(min(hr.yield_kg / b.area_ha)::numeric, 0)   AS min_yield_kg_ha,
        round(max(hr.yield_kg / b.area_ha)::numeric, 0)   AS max_yield_kg_ha,
        count(DISTINCT vi.id)::int                         AS vineyard_count
      FROM harvest_records hr
      JOIN blocks b      ON b.id    = hr.block_id
      JOIN varieties var ON var.id  = b.variety_id
      JOIN vineyards vi  ON vi.id   = b.vineyard_id
      WHERE var.status = 'approved'
        AND (${selectedYear}::int IS NULL OR hr.harvest_year = ${selectedYear}::int)
        AND (${selectedVariety}::text IS NULL OR var.name = ${selectedVariety}::text)
      GROUP BY var.name, vi.county, hr.harvest_year
      HAVING count(DISTINCT vi.id) >= 3
    ),
    user_yields AS (
      SELECT
        var.name                                           AS variety_name,
        vi.county,
        hr.harvest_year,
        round(avg(hr.yield_kg / b.area_ha)::numeric, 0)   AS user_yield_kg_ha
      FROM harvest_records hr
      JOIN blocks b      ON b.id    = hr.block_id
      JOIN varieties var ON var.id  = b.variety_id
      JOIN vineyards vi  ON vi.id   = b.vineyard_id
      WHERE b.vineyard_id = ANY(${vineyardIds.length > 0 ? vineyardIds : [0]}::int[])
        AND var.status = 'approved'
      GROUP BY var.name, vi.county, hr.harvest_year
    )
    SELECT
      cb.variety_name, cb.county, cb.harvest_year,
      cb.avg_yield_kg_ha, cb.min_yield_kg_ha, cb.max_yield_kg_ha,
      cb.vineyard_count, uy.user_yield_kg_ha
    FROM county_benchmarks cb
    LEFT JOIN user_yields uy USING (variety_name, county, harvest_year)
    ORDER BY cb.variety_name, cb.county, cb.harvest_year DESC
  `;

  const years = [...new Set(rows.map((r) => r.harvest_year))].sort((a, b) => b - a);
  const varieties = [...new Set(rows.map((r) => r.variety_name))].sort();

  return { rows, years, varieties, selectedYear, selectedVariety };
};
