// src/routes/vineyard/[id]/settings/+page.server.ts
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
  if (member.role !== 'owner') throw error(403, 'Endast ägare kan ändra inställningar.');

  const [vineyard] = await sql`
    SELECT id, name, county, municipality, organic, biodynamic,
           established_year, total_area_ha, legal_id, legal_id_type, legal_name
    FROM vineyards WHERE id = ${vineyardId} AND deleted_at IS NULL
  `;
  if (!vineyard) throw error(404, 'Vingården hittades inte.');

  const members = await sql`
    SELECT um.id, um.role, u.email, u.name
    FROM vineyard_members um
    JOIN users u ON u.id = um.user_id
    WHERE um.vineyard_id = ${vineyardId}
    ORDER BY um.role DESC, u.name
  `;

  return {
    vineyard,
    members: members.map((m: Record<string, unknown>) => ({
      id: m.id,
      role: m.role,
      email: m.email,
      name: m.name
    }))
  };
};

export const actions: Actions = {
  default: async ({ request, locals, params }) => {
    if (!locals.user) throw redirect(303, '/login');

    const vineyardId = Number(params.id);
    const data = await request.formData();

    const action = data.get('action') as string;

    // Edit vineyard details
    if (action === 'update_vineyard') {
      const name = (data.get('name') as string)?.trim();
      const county = (data.get('county') as string)?.trim();
      const municipality = (data.get('municipality') as string)?.trim();
      const legal_id = (data.get('legal_id') as string)?.trim() || null;
      const legal_id_type = (data.get('legal_id_type') as string) || null;
      const legal_name = (data.get('legal_name') as string)?.trim() || null;
      const organic = data.get('organic') === 'on';
      const biodynamic = data.get('biodynamic') === 'on';
      const established_year = data.get('established_year') ? Number(data.get('established_year')) : null;
      const total_area_ha = data.get('total_area_ha') ? Number(data.get('total_area_ha')) : null;

      if (!name) return fail(400, { error: 'Vingårdsnamn krävs.' });
      if (!county) return fail(400, { error: 'Län krävs.' });

      try {
        await sql`
          UPDATE vineyards SET
            name = ${name}, county = ${county}, municipality = ${municipality},
            legal_id = ${legal_id}, legal_id_type = ${legal_id_type}, legal_name = ${legal_name},
            organic = ${organic}, biodynamic = ${biodynamic},
            established_year = ${established_year}, total_area_ha = ${total_area_ha}
          WHERE id = ${vineyardId}
        `;
      } catch (err) {
        console.error('Failed to update vineyard:', err);
        return fail(500, { error: 'Kunde inte uppdatera vingård. Försök igen.' });
      }
    }

    return { success: true };
  }
};
