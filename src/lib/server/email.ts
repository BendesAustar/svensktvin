// src/lib/server/email.ts
import nodemailer from 'nodemailer';

function getTransport() {
  const host = process.env.SMTP_HOST;
  const port = Number(process.env.SMTP_PORT ?? 587);
  const user = process.env.SMTP_USER;
  const pass = process.env.SMTP_PASS;
  if (!host || !user || !pass) {
    throw new Error('SMTP environment variables not configured');
  }
  return nodemailer.createTransport({ host, port, auth: { user, pass } });
}

export async function sendMagicLink(to: string, rawToken: string): Promise<void> {
  const appHost = process.env.APP_HOST ?? 'http://localhost:5173';
  const from = process.env.SMTP_FROM ?? 'noreply@svensktvin.se';
  const link = `${appHost}/auth/verify?token=${rawToken}`;

  const transport = getTransport();
  await transport.sendMail({
    from,
    to,
    subject: 'Logga in på Svenskt Vin',
    text: `Klicka på länken nedan för att logga in. Länken är giltig i 15 minuter.\n\n${link}\n\nOm du inte begärde detta, ignorera detta mejl.`,
    html: `
      <p>Klicka på knappen nedan för att logga in på Svenskt Vin.</p>
      <p><a href="${link}" style="display:inline-block;padding:12px 24px;background:#2d6a2d;color:#fff;text-decoration:none;border-radius:4px;">Logga in</a></p>
      <p>Länken är giltig i 15 minuter.</p>
      <p style="color:#888;font-size:12px;">Om du inte begärde detta, ignorera detta mejl.</p>
    `
  });
}
