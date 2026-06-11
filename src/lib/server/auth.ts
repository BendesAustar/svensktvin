// src/lib/server/auth.ts
import { randomBytes, createHash } from 'crypto';
import { sql } from './db.js';

export async function createMagicLink(userId: number): Promise<string> {
  const raw = randomBytes(32).toString('hex');
  const hash = createHash('sha256').update(raw).digest('hex');
  const expiresAt = new Date(Date.now() + 15 * 60 * 1000); // 15 min
  await sql`
    INSERT INTO magic_link_tokens (user_id, token_hash, expires_at)
    VALUES (${userId}, ${hash}, ${expiresAt})
  `;
  return raw;
}

export async function verifyToken(rawToken: string): Promise<number | null> {
  const hash = createHash('sha256').update(rawToken).digest('hex');
  const [token] = await sql`
    UPDATE magic_link_tokens
    SET used = true
    WHERE token_hash = ${hash}
      AND used = false
      AND expires_at > now()
    RETURNING user_id
  `;
  return token?.user_id ?? null;
}

export async function createSession(userId: number): Promise<string> {
  const id = crypto.randomUUID();
  const expiresAt = new Date(Date.now() + 30 * 24 * 60 * 60 * 1000); // 30 days
  await sql`
    INSERT INTO sessions (id, user_id, expires_at)
    VALUES (${id}, ${userId}, ${expiresAt})
  `;
  return id;
}

export async function getSession(
  sessionId: string
): Promise<{ id: number; email: string; name: string; is_admin: boolean } | null> {
  const [row] = await sql`
    SELECT u.id, u.email, u.name, u.is_admin
    FROM sessions s
    JOIN users u ON u.id = s.user_id
    WHERE s.id = ${sessionId}
      AND s.expires_at > now()
      AND u.active = true
  `;
  return row ?? null;
}

export async function deleteSession(sessionId: string): Promise<void> {
  await sql`DELETE FROM sessions WHERE id = ${sessionId}`;
}

export async function getUserByEmail(
  email: string
): Promise<{ id: number } | null> {
  const [user] = await sql`
    SELECT id FROM users WHERE email = ${email} AND active = true
  `;
  return user ?? null;
}

export async function updateLastLogin(userId: number): Promise<void> {
  await sql`UPDATE users SET last_login = now() WHERE id = ${userId}`;
}
