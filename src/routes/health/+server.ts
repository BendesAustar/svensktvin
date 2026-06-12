// src/routes/health/+server.ts
import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { sql } from '$lib/server/db.js';

export const GET: RequestHandler = async () => {
  const checks: Record<string, 'ok' | 'fail'> = {
    app: 'ok'
  };

  try {
    await sql`SELECT 1`;
    checks.db = 'ok';
  } catch {
    checks.db = 'fail';
  }

  const status = Object.values(checks).every(v => v === 'ok') ? 'healthy' : 'degraded';

  return json({
    status,
    checks,
    uptime: process.uptime(),
    timestamp: new Date().toISOString()
  }, {
    status: status === 'healthy' ? 200 : 503
  });
};
