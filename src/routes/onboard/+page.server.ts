// src/routes/onboard/+page.server.ts
import { redirect, fail } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { sql } from '$lib/server/db.js';

export const load: PageServerLoad = ({ locals }) => {
  if (!locals.user) throw redirect(303, '/login');
  return {};
};

export const actions: Actions = {
  default: async ({ request, locals }) => {
    if (!locals.user) throw redirect(303, '/login');

    const data = await request.formData();
    const name = (data.get('name') as string)?.trim();
    const organic = data.get('organic') === 'on';
    const biodynamic = data.get('biodynamic') === 'on';
    const legal_id = (data.get('legal_id') as string)?.trim() || null;
    const legal_id_type = (data.get('legal_id_type') as string) || null;
    const legal_name = (data.get('legal_name') as string)?.trim() || null;
    const established_year = data.get('established_year') ? Number(data.get('established_year')) : null;
    const total_area_ha = data.get('total_area_ha') ? Number(data.get('total_area_ha')) : null;
    const county = (data.get('county') as string)?.trim();
    const municipality = (data.get('municipality') as string)?.trim();
    const lat = data.get('lat') ? Number(data.get('lat')) : null;
    const lon = data.get('lon') ? Number(data.get('lon')) : null;

    // Validation
    if (!name) return fail(400, { error: 'Vingårdsnamn krävs.' });
    if (!county) return fail(400, { error: 'Län krävs.' });
    if (!municipality) return fail(400, { error: 'Kommun krävs.' });

    if (legal_id_type === 'ab' && legal_id) {
      // Swedish organisationsnummer check digit (mod-10)
      const cleanId = legal_id.replace(/[-\s]/g, '');
      if (!/^\d{6,10}$/.test(cleanId)) {
        return fail(400, { error: 'Organisationsnumret måste vara 6–10 siffror.' });
      }
    }

    let vineyardId: number;
    try {
      const [row] = await sql`
        INSERT INTO vineyards (
          name, county, municipality, lat, lon,
          established_year, total_area_ha,
          organic, biodynamic,
          legal_id, legal_id_type, legal_name
        ) VALUES (
          ${name}, ${county}, ${municipality}, ${lat}, ${lon},
          ${established_year}, ${total_area_ha},
          ${organic}, ${biodynamic},
          ${legal_id}, ${legal_id_type}, ${legal_name}
        )
        RETURNING id
      `;
      vineyardId = row.id;
    } catch (err) {
      console.error('Failed to create vineyard:', err);
      return fail(500, { error: 'Kunde inte skapa vingård. Försök igen.' });
    }

    // Assign current user as owner
    await sql`
      INSERT INTO vineyard_members (vineyard_id, user_id, role)
      VALUES (${vineyardId}, ${locals.user.id}, 'owner')
    `;

    throw redirect(303, `/vineyard/${vineyardId}`);
  }
};
