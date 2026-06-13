// src/routes/vineyard/[id]/blocks/[blockId]/harvest/lock/+server.ts
import { json, error, redirect } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { sql } from '$lib/server/db.js';

const LOCK_TTL_MS = 30 * 60 * 1000; // 30 minutes

export const GET: RequestHandler = async ({ params, locals }) => {
  if (!locals.user) throw error(401, 'Olognad.');

  const vineyardId = Number(params.id);
  const blockId = Number(params.blockId);

  // Verify user has access to this vineyard
  const [member] = await sql`
    SELECT role FROM vineyard_members
    WHERE vineyard_id = ${vineyardId} AND user_id = ${locals.user.id}
  `;
  if (!member) throw error(403, 'Du har inte tillgång till denna vingård.');

  // Verify block belongs to vineyard
  const [block] = await sql`
    SELECT id FROM blocks WHERE id = ${blockId} AND vineyard_id = ${vineyardId}
  `;
  if (!block) throw error(404, 'Blocket hittades inte.');

  // Check for active lock
  const [lock] = await sql`
    SELECT bl.id, bl.user_id, bl.locked_at, bl.expires_at, u.name, u.email
    FROM block_locks bl
    JOIN users u ON u.id = bl.user_id
    WHERE bl.block_id = ${blockId}
      AND bl.expires_at > now()
    LIMIT 1
  `;

  if (!lock) {
    return json({ locked: false, blockId, expiresAt: null });
  }

  return json({
    locked: true,
    blockId,
    holder: {
      id: lock.user_id,
      name: lock.name,
      email: lock.email
    },
    expiresAt: lock.expires_at
  });
};

export const POST: RequestHandler = async ({ params, locals }) => {
  if (!locals.user) throw error(401, 'Olognad.');

  const vineyardId = Number(params.id);
  const blockId = Number(params.blockId);

  // Verify user has access to this vineyard
  const [member] = await sql`
    SELECT role FROM vineyard_members
    WHERE vineyard_id = ${vineyardId} AND user_id = ${locals.user.id}
  `;
  if (!member) throw error(403, 'Du har inte tillgång till denna vingård.');

  // Verify block belongs to vineyard
  const [block] = await sql`
    SELECT id FROM blocks WHERE id = ${blockId} AND vineyard_id = ${vineyardId}
  `;
  if (!block) throw error(404, 'Blocket hittades inte.');

  // Check for active lock
  const [lock] = await sql`
    SELECT bl.id, bl.user_id, bl.locked_at, bl.expires_at
    FROM block_locks bl
    WHERE bl.block_id = ${blockId}
      AND bl.expires_at > now()
    LIMIT 1
  `;

  if (lock) {
    if (lock.user_id === locals.user!.id) {
      // User already holds the lock — release it
      await sql`DELETE FROM block_locks WHERE id = ${lock.id}`;
      return json({ unlocked: true, blockId, message: 'Blocket är nu olåst.' });
    }
    // Locked by someone else
    const [holder] = await sql`SELECT name FROM users WHERE id = ${lock.user_id}`;
    throw error(409, `Blocket redigeras just nu av ${holder?.name || 'någon'}. Försök igen senare.`);
  }

  // No active lock — acquire
  const expiresAt = new Date(Date.now() + LOCK_TTL_MS);
  try {
    await sql`
      INSERT INTO block_locks (block_id, user_id, expires_at)
      VALUES (${blockId}, ${locals.user.id}, ${expiresAt})
    `;
    return json({ locked: true, blockId, expiresAt, message: 'Blocket är nu låst.' });
  } catch (err) {
    // Unique constraint violated — someone acquired it between our check and insert
    const [holder] = await sql`
      SELECT name FROM users WHERE id = (
        SELECT user_id FROM block_locks WHERE block_id = ${blockId} AND expires_at > now() LIMIT 1
      )
    `;
    throw error(409, `Blocket redigeras just nu av ${holder?.name || 'någon'}. Försök igen senare.`);
  }
};

export const DELETE: RequestHandler = async ({ params, locals }) => {
  if (!locals.user) throw error(401, 'Olognad.');

  const vineyardId = Number(params.id);
  const blockId = Number(params.blockId);

  // Verify user has access
  const [member] = await sql`
    SELECT role FROM vineyard_members
    WHERE vineyard_id = ${vineyardId} AND user_id = ${locals.user.id}
  `;
  if (!member) throw error(403, 'Du har inte tillgång till denna vingård.');

  // Verify block belongs to vineyard
  const [block] = await sql`
    SELECT id FROM blocks WHERE id = ${blockId} AND vineyard_id = ${vineyardId}
  `;
  if (!block) throw error(404, 'Blocket hittades inte.');

  // Check if user holds the lock
  const [lock] = await sql`
    SELECT id FROM block_locks
    WHERE block_id = ${blockId}
      AND user_id = ${locals.user.id}
      AND expires_at > now()
    LIMIT 1
  `;

  if (!lock) {
    throw error(409, 'Du har inget aktivt lås på detta block.');
  }

  await sql`DELETE FROM block_locks WHERE id = ${lock.id}`;
  return json({ unlocked: true, blockId });
};
