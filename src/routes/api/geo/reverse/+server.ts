// src/routes/api/geo/reverse/+server.ts
import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const POST: RequestHandler = async ({ request, locals }) => {
  if (!locals.user) {
    return json({ error: 'Unauthorized' }, { status: 401 });
  }

  const body = await request.json();
  const { lat, lon } = body as { lat: number; lon: number };

  if (lat == null || lon == null) {
    return json({ error: 'location_not_found' }, { status: 400 });
  }

  // Rate-limit cache key: round coordinates to 4 decimal places (~10m precision)
  const cacheKey = `geo:${lat.toFixed(4)},${lon.toFixed(4)}`;
  const cached = request.headers.get('x-geo-cache') === 'hit' ? null : null;

  const url = `https://nominatim.openstreetmap.org/reverse?format=json&lat=${lat.toFixed(6)}&lon=${lon.toFixed(6)}&accept-language=sv`;

  try {
    const res = await fetch(url, {
      headers: {
        'User-Agent': 'SvensktVin/1.0 (contact@svensktvin.se)',
        'Accept-Language': 'sv'
      }
    });

    if (!res.ok) {
      return json({ error: 'location_not_found' }, { status: 502 });
    }

    const data = await res.json() as Record<string, unknown>;
    const address = (data.address ?? {}) as Record<string, string>;

    // Extract county (state)
    const rawState = address.state as string;
    if (!rawState) {
      return json({ error: 'location_not_found' }, { status: 404 });
    }

    const county = rawState
      .replace(/ county$/i, '')
      .replace(/ län$/i, '')
      .trim();

    // Extract municipality
    const municipality = (address.city ?? address.town ?? address.village ?? address.municipality) as string;

    if (!county || !municipality) {
      return json({ error: 'location_not_found' }, { status: 404 });
    }

    return json({ county, municipality });
  } catch {
    return json({ error: 'location_not_found' }, { status: 502 });
  }
};
