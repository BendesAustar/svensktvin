// src/lib/server/db.ts
import postgres from 'postgres';

let connection: ReturnType<typeof postgres> | null = null;

function getConnection(): ReturnType<typeof postgres> {
  if (!connection) {
    const url = process.env.DATABASE_URL;
    if (!url) {
      // Allow build-time analysis without DATABASE_URL
      connection = postgres('postgresql://localhost/svensktvin', {
        max: 0, // No connections at build time
        onnotice: () => {}
      });
    } else {
      connection = postgres(url, {
        max: 10,
        idle_timeout: 20,
        connect_timeout: 10
      });
    }
  }
  return connection;
}

export const sql = getConnection();
