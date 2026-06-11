// src/lib/server/db.ts
import postgres from 'postgres';

let connection: ReturnType<typeof postgres> | null = null;

function getConnection(): ReturnType<typeof postgres> {
  if (!connection) {
    const url = process.env.DATABASE_URL;
    if (!url) throw new Error('DATABASE_URL environment variable is not set');
    connection = postgres(url, {
      max: 10,
      idle_timeout: 20,
      connect_timeout: 10
    });
  }
  return connection;
}

export const sql = getConnection();
