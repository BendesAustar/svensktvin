// src/tests/email.test.ts
import { describe, it, expect, vi, beforeEach } from 'vitest';

describe('sendMagicLink', () => {
  beforeEach(() => {
    vi.resetModules();
  });

  it('calls nodemailer sendMail with correct fields', async () => {
    // Mock nodemailer before importing email module
    const sendMailMock = vi.fn().mockResolvedValue({ messageId: 'test-id' });
    vi.mock('nodemailer', async (importOriginal) => {
      const actual = await importOriginal<typeof import('nodemailer')>();
      return {
        default: {
          createTransport: vi.fn(() => ({ sendMail: sendMailMock }))
        }
      };
    });

    // Set env vars before importing
    process.env.SMTP_HOST = 'smtp.test.local';
    process.env.SMTP_PORT = '587';
    process.env.SMTP_USER = 'test';
    process.env.SMTP_PASS = 'pass';
    process.env.SMTP_FROM = 'no-reply@test.local';
    process.env.APP_HOST = 'http://localhost:5173';

    const { sendMagicLink } = await import('../lib/server/email.js');

    await sendMagicLink('user@example.com', 'abc123token');

    expect(sendMailMock).toHaveBeenCalledOnce();
    const call = sendMailMock.mock.calls[0][0];
    expect(call.to).toBe('user@example.com');
    expect(call.subject).toBe('Logga in på Svenskt Vin');
    expect(call.text).toContain('abc123token');
    expect(call.html).toContain('abc123token');
  });
});
