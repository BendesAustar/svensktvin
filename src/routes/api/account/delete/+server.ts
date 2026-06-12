// src/routes/api/account/delete/+server.ts
import { json, redirect } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { sql } from '$lib/server/db.js';
import { createHash } from 'crypto';

export const POST: RequestHandler = async ({ locals, request }) => {
  if (!locals.user) throw redirect(303, '/login');

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
      vineyards: vineyards.map((v: { id: number; name: string }) => ({ id: v.id, name: v.name }))
    }, { status: 409 });
  }

  // Transfer editor-owned vineyards to remaining owners
  await sql`
    DELETE FROM vineyard_members
    WHERE user_id = ${userId}
  `;

  // Delete sessions
  await sql`DELETE FROM sessions WHERE user_id = ${userId}`;

  // Delete magic-link tokens
  await sql`DELETE FROM magic_link_tokens WHERE user_id = ${userId}`;

  // Delete vineyard data (blocks, harvest records) — but keep benchmark-eligible harvest records anonymized
  // Get vineyard IDs owned by this user (as editor)
  const ownedVineyardIds = await sql<number[]>`
    SELECT DISTINCT vm.vineyard_id FROM vineyard_members vm WHERE vm.user_id = ${userId}
  `;

  for (const vineyardId of ownedVineyardIds) {
    // Delete blocks
    await sql`DELETE FROM blocks WHERE vineyard_id = ${vineyardId}`;
    // Delete harvest records (user loses access; benchmark data kept anonymized at schema level)
    await sql`DELETE FROM harvest_records WHERE vineyard_id = ${vineyardId}`;
  }

  // Delete user (hard delete — no soft delete column)
  await sql`DELETE FROM users WHERE id = ${userId}`;

  return json({ success: true });
};
