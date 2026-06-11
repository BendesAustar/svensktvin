// src/tests/auth.test.ts
import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { createHash, randomBytes } from 'crypto';

// Test pure crypto functions (no DB needed)
describe('generateToken', () => {
  it('returns a 64-char hex string for raw token', () => {
    const raw = randomBytes(32).toString('hex');
    expect(raw).toHaveLength(64);
    expect(raw).toMatch(/^[0-9a-f]+$/);
  });

  it('hashes deterministically', () => {
    const raw = 'test-token';
    const h1 = createHash('sha256').update(raw).digest('hex');
    const h2 = createHash('sha256').update(raw).digest('hex');
    expect(h1).toBe(h2);
  });

  it('two different raw tokens produce different hashes', () => {
    const h1 = createHash('sha256').update(randomBytes(32)).digest('hex');
    const h2 = createHash('sha256').update(randomBytes(32)).digest('hex');
    expect(h1).not.toBe(h2);
  });
});

// DB-dependent tests — require running dev DB
describe('auth DB functions', () => {
  let sql: Awaited<ReturnType<typeof import('postgres').default>>;
  let testUserId: number;

  beforeAll(async () => {
    const postgres = (await import('postgres')).default;
    sql = postgres(process.env.DATABASE_URL!);

    // Insert a test user
    const [user] = await sql`
      INSERT INTO users (email, name)
      VALUES ('test-auth@svensktvin.test', 'Test User')
      RETURNING id
    `;
    testUserId = user.id;
  });

  afterAll(async () => {
    // Clean up
    await sql`DELETE FROM users WHERE email = 'test-auth@svensktvin.test'`;
    await sql.end();
  });

  it('createMagicLink inserts a token and returns raw string', async () => {
    const { createMagicLink } = await import('../lib/server/auth.js');
    const raw = await createMagicLink(testUserId);
    expect(raw).toHaveLength(64);

    const [row] = await sql`
      SELECT used, expires_at FROM magic_link_tokens
      WHERE user_id = ${testUserId}
      ORDER BY created_at DESC LIMIT 1
    `;
    expect(row.used).toBe(false);
    expect(new Date(row.expires_at) > new Date()).toBe(true);
  });

  it('verifyToken returns user_id and marks token used', async () => {
    const { createMagicLink, verifyToken } = await import('../lib/server/auth.js');
    const raw = await createMagicLink(testUserId);
    const userId = await verifyToken(raw);
    expect(userId).toBe(testUserId);

    // Second call with same token returns null (already used)
    const again = await verifyToken(raw);
    expect(again).toBeNull();
  });

  it('verifyToken returns null for unknown token', async () => {
    const { verifyToken } = await import('../lib/server/auth.js');
    const result = await verifyToken('deadbeef'.repeat(8));
    expect(result).toBeNull();
  });

  it('createSession returns a UUID and is retrievable', async () => {
    const { createSession, getSession } = await import('../lib/server/auth.js');
    const sessionId = await createSession(testUserId);
    expect(sessionId).toMatch(
      /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/
    );

    const user = await getSession(sessionId);
    expect(user?.id).toBe(testUserId);
    expect(user?.email).toBe('test-auth@svensktvin.test');
    expect(user?.is_admin).toBe(false);
  });

  it('deleteSession removes the session', async () => {
    const { createSession, deleteSession, getSession } = await import('../lib/server/auth.js');
    const sessionId = await createSession(testUserId);
    await deleteSession(sessionId);
    const user = await getSession(sessionId);
    expect(user).toBeNull();
  });
});
