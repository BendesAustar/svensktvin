// src/lib/server/db.ts
import postgres from 'postgres';
import { loadEnv } from 'vite';

// Load .env files properly for server-side code (Vite doesn't auto-load
// non-VITE_ prefixed vars into process.env in dev mode)
const env = loadEnv('development', process.cwd(), '');
const url = env.DATABASE_URL || process.env.DATABASE_URL;

function getConnection(): ReturnType<typeof postgres> {
  if (!url) {
    // Allow build-time analysis without DATABASE_URL
    return postgres('postgresql://localhost/svensktvin', {
      max: 0, // No connections at build time
      onnotice: () => {}
    });
  }

  return postgres(url, {
    max: 10,
    idle_timeout: 20,
    connect_timeout: 5
  });
}

export const sql = getConnection();
