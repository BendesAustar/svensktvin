// src/tests/auth.test.ts
import { describe, it, expect } from 'vitest';
import { createSessionToken, createMagicLinkToken } from '$lib/server/auth';

describe('createSessionToken', () => {
  it('should generate a 128-character hex token', () => {
    const token = createSessionToken();
    expect(token.length).toBe(128); // 64 bytes = 128 hex chars
    expect(/^[0-9a-f]+$/.test(token)).toBe(true);
  });

  it('should generate unique tokens', () => {
    const tokens = new Set<string>();
    for (let i = 0; i < 1000; i++) {
      tokens.add(createSessionToken());
    }
    expect(tokens.size).toBe(1000);
  });
});

describe('createMagicLinkToken', () => {
  it('should generate a valid token/hash pair', () => {
    const result = createMagicLinkToken();
    expect(result.token.length).toBe(64); // 32 bytes hex
    expect(result.hash.length).toBe(64); // SHA-256 hex
    expect(result.token).not.toBe(result.hash);
  });

  it('should generate unique token/hash pairs', () => {
    const seen = new Set<string>();
    for (let i = 0; i < 1000; i++) {
      const { token, hash } = createMagicLinkToken();
      const key = `${token}:${hash}`;
      expect(seen.has(key)).toBe(false);
      seen.add(key);
    }
  });
});
