// src/routes/api/account/delete/+server.ts
import { json, redirect } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { sql } from '$lib/server/db.js';
import { createHash } from 'crypto';

export const POST: RequestHandler = async ({ locals, request, url }) => {
  if (!locals.user) throw redirect(303, '/login');

  // CSRF protection: reject cross-origin requests
  const origin = request.headers.get('origin');
  const host = url.host;
  if (origin && origin !== `https://${host}` && origin !== `http://${host}`) {
    return json({ error: 'Ogiltig förfrågan.' }, { status: 403 });
  }

  const body = await request.json();
  const confirm = body?.confirm;

  if (confirm !== true) {
    return json({ error: 'Bekräfta radering med { "confirm": true }.' }, { status: 400 });
  }

  const userId = locals.user.id;

  // Check if user is the last owner of any vineyard
  const vineyards = await sql`
    SELECT v.id, v.name
    FROM vineyard_members vm
    JOIN vineyards v ON v.id = vm.vineyard_id
    WHERE vm.user_id = ${userId} AND vm.role = 'owner'
  `;

  if (vineyards.length > 0) {
    return json({
      error: 'Du måste överlåta eller radera dina vingårdar först.',
      vineyards: vineyards.map((v) => ({ id: v.id, name: v.name }))
    }, { status: 409 });
  }

  // Delete sessions
  await sql`DELETE FROM sessions WHERE user_id = ${userId}`;

  // Delete magic-link tokens
  await sql`DELETE FROM magic_link_tokens WHERE user_id = ${userId}`;

  // Delete all vineyard memberships (user is no longer linked to any vineyard)
  await sql`
    DELETE FROM vineyard_members
    WHERE user_id = ${userId}
  `;

  // Delete user blocks and harvest records (all user data is removed)
  // Note: Harvest records belonging to this user's blocks are cascade-deleted
  // via ON DELETE CASCADE on blocks table. This means the user's contribution
  // to benchmarks is removed along with their account — consistent with GDPR
  // "right to be forgotten" where the user chooses full deletion.
  // If they want to keep benchmark data, they should use export first.

  // Delete user (hard delete — no soft delete column)
  await sql`DELETE FROM users WHERE id = ${userId}`;

  return json({ success: true });
};
