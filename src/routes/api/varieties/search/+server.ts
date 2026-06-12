// src/routes/api/varieties/search/+server.ts
import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { sql } from '$lib/server/db.js';

export const GET: RequestHandler = async ({ url, locals }) => {
  if (!locals.user) {
    return json({ error: 'Unauthorized' }, { status: 401 });
  }

  const q = url.searchParams.get('q');
  if (!q || q.length < 2) {
    return json({ query: q, matches: [], high_confidence: false });
  }

  const matches = await sql`
    SELECT id, name, piwi, color,
           round(similarity(name, ${q})::numeric, 2) AS score
    FROM varieties
    WHERE similarity(name, ${q}) > 0.4
      AND status = 'approved'
    ORDER BY similarity(name, ${q}) DESC
    LIMIT 3
  `;

  return json({
    query: q,
    matches: matches.map((m: Record<string, unknown>) => ({
      id: m.id,
      name: m.name,
      score: m.score,
      piwi: m.piwi,
      color: m.color
    })),
    high_confidence: matches.length > 0 && (matches[0].score as number) >= 0.8
  });
};
