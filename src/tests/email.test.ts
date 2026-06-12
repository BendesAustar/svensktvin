// src/tests/email.test.ts
import { describe, it, expect } from 'vitest';
import { loginEmailTemplate, welcomeEmailTemplate } from '$lib/server/email';

describe('loginEmailTemplate', () => {
  it('should generate a valid HTML email', () => {
    const { html, text } = loginEmailTemplate('https://example.com', 'token123');
    expect(html).toContain('https://example.com/auth/verify?token=token123');
    expect(html).toContain('Svenskt Vin');
    expect(text).toContain('token123');
    expect(text).toContain('15 minuter');
  });

  it('should include security warning', () => {
    const { html, text } = loginEmailTemplate('https://example.com', 'abc');
    expect(html).toContain('Om du inte begärde detta, ignorera detta mejl.');
    expect(text).toContain('Om du inte begärde detta, ignorera detta mejl.');
  });
});

describe('welcomeEmailTemplate', () => {
  it('should generate a welcome email with personalization', () => {
    const { html, text } = welcomeEmailTemplate('Testanvändare', 'https://example.com/vineyard/1');
    expect(html).toContain('Testanvändare');
    expect(html).toContain('Välkommen');
    expect(text).toContain('Testanvändare');
    expect(text).toContain('https://example.com/vineyard/1');
  });
});
