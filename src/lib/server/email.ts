// src/lib/server/email.ts
import nodemailer from 'nodemailer';

export function loginEmailTemplate(appHost: string, token: string): { text: string; html: string } {
  const link = `${appHost}/auth/verify?token=${token}`;
  return {
    text: `Klicka på länken nedan för att logga in. Länken är giltig i 15 minuter.\n\n${link}\n\nOm du inte begärde detta, ignorera detta mejl.`,
    html: `
      <p>Klicka på knappen nedan för att logga in på Svenskt Vin.</p>
      <p><a href="${link}" style="display:inline-block;padding:12px 24px;background:#2d6a2d;color:#fff;text-decoration:none;border-radius:4px;">Logga in</a></p>
      <p>Länken är giltig i 15 minuter.</p>
      <p style="color:#888;font-size:12px;">Om du inte begärde detta, ignorera detta mejl.</p>
    `
  };
}

export function welcomeEmailTemplate(name: string, vineyardUrl: string): { text: string; html: string } {
  return {
    text: `Välkommen, ${name}!\n\nDin vingård är nu registrerad:\n${vineyardUrl}\n\nVänliga hälsningar,\nSvenskt Vin`,
    html: `
      <h2>Välkommen, ${name}!</h2>
      <p>Din vingård är nu registrerad.</p>
      <p><a href="${vineyardUrl}" style="display:inline-block;padding:12px 24px;background:#2d6a2d;color:#fff;text-decoration:none;border-radius:4px;">Gå till din vingård</a></p>
      <p style="color:#888;font-size:12px;">Vänliga hälsningar, Svenskt Vin</p>
    `
  };
}

export function inviteEmailTemplate(appHost: string, vineyardName: string, token: string): { text: string; html: string } {
  const link = `${appHost}/invite?token=${encodeURIComponent(token)}`;
  return {
    text: `Du har blivit inbjuden att gå med i vingården ${vineyardName}.\n\nKlicka på länken nedan för att acceptera inbjudan:\n${link}\n\nLänken är giltig i 7 dagar.\n\nOm du inte blev inbjuden, ignorera detta mejl.`,
    html: `
      <h2>Inbjudan till ${vineyardName}</h2>
      <p>Du har blivit inbjuden att gå med som medlem i <strong>${vineyardName}</strong>.</p>
      <p><a href="${link}" style="display:inline-block;padding:12px 24px;background:#2d6a2d;color:#fff;text-decoration:none;border-radius:4px;">Acceptera inbjudan</a></p>
      <p style="color:#888;font-size:12px;">Länken är giltig i 7 dagar. Om du inte blev inbjuden, ignorera detta mejl.</p>
    `
  };
}

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

  const transport = getTransport();
  const { text, html } = loginEmailTemplate(appHost, rawToken);
  await transport.sendMail({
    from,
    to,
    subject: 'Logga in på Svenskt Vin',
    text,
    html
  });
}
