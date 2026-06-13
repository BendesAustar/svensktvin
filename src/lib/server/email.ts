// src/lib/server/email.ts
import { createTransport } from 'nodemailer';

// Configuration for email transport
const transporter = createTransport({
  host: process.env.SMTP_HOST || 'localhost',
  port: parseInt(process.env.SMTP_PORT || '587', 10),
  secure: process.env.SMTP_SECURE === 'true',
  auth: {
    user: process.env.SMTP_USER || '',
    pass: process.env.SMTP_PASS || '',
  },
});

// Existing email template for welcome emails
export function welcomeEmailTemplate(email: string): { subject: string; body: string } {
  const subject = 'Välkommen!';
  const body = `
    <html>
      <body>
        <h1>Välkommen!</h1>
        <p>Tack för att du registrerade dig på vår plattform.</p>
        <p>Mvh, <strong>Ditt Företag</strong></p>
      </body>
    </html>
  `;
  return { subject, body };
}

// Email template for password reset
export function passwordResetEmailTemplate(token: string, email: string): { subject: string; body: string } {
  const resetLink = `/auth/set-password?token=${token}`;
  const subject = 'Återställ ditt lösenord';
  const body = `
    <html>
      <body>
        <h1>Återställ ditt lösenord</h1>
        <p>Klicka på länken nedan för att återställa ditt lösenord:</p>
        <p>
          <a href="${resetLink}" style="background-color: #4CAF50; color: white; padding: 14px 20px; text-align: center; text-decoration: none; display: inline-block; border-radius: 4px;">
            Återställ lösenord
          </a>
        </p>
        <p>Om du inte begärde detta, ignorera detta e-postmeddelande.</p>
        <p>Denna länk är endast giltig i 1 timme.</p>
        <p>Mvh, <strong>Ditt Företag</strong></p>
      </body>
    </html>
  `;
  return { subject, body };
}

// Send welcome email
export async function sendWelcomeEmail(email: string): Promise<void> {
  const template = welcomeEmailTemplate(email);
  await transporter.sendMail({
    from: `"Ditt Företag" <${process.env.SMTP_FROM || 'no-reply@yourdomain.com'}>`,
    to: email,
    subject: template.subject,
    html: template.body,
  });
}

// Invite email template for settings
export function inviteEmailTemplate(
  appHost: string,
  vineyardName: string,
  token: string
): { text: string; html: string } {
  const acceptLink = `${appHost}/invite?token=${encodeURIComponent(token)}`;
  const text = `Du har blivit inbjuden att gå med i ${vineyardName} på Svenskt Vin.\n\nKlicka på länken nedan för att acceptera inbjudan:\n${acceptLink}\n\nDen här länken är giltig i 7 dagar.`;
  const html = `
    <html>
      <body>
        <h1>Inbjudan till ${vineyardName}</h1>
        <p>Du har blivit inbjuden att gå med i <strong>${vineyardName}</strong> på Svenskt Vin.</p>
        <p>
          <a href="${acceptLink}" style="background-color: #2d6a2d; color: white; padding: 14px 20px; text-align: center; text-decoration: none; display: inline-block; border-radius: 4px;">
            Acceptera inbjudan
          </a>
        </p>
        <p>Den här länken är giltig i 7 dagar.</p>
        <p>Mvh, <strong>Svenskt Vin</strong></p>
      </body>
    </html>
  `;
  return { text, html };
}

// Send magic link email
export async function sendMagicLink(email: string, token: string): Promise<void> {
  const magicLink = `/auth/verify?token=${token}`;
  const subject = 'Din inloggningslänk';
  const body = `
    <html>
      <body>
        <h1>Inloggning — Svenskt Vin</h1>
        <p>Klicka på länken nedan för att logga in:</p>
        <p>
          <a href="${magicLink}" style="background-color: #2d6a2d; color: white; padding: 14px 20px; text-align: center; text-decoration: none; display: inline-block; border-radius: 4px;">
            Logga in
          </a>
        </p>
        <p>Om du inte begärde detta, ignorera detta e-postmeddelande.</p>
        <p>Denna länk är endast giltig i 15 minuter.</p>
        <p>Mvh, <strong>Svenskt Vin</strong></p>
      </body>
    </html>
  `;
  await transporter.sendMail({
    from: `"Svenskt Vin" <${process.env.SMTP_FROM || 'no-reply@yourdomain.com'}>`,
    to: email,
    subject,
    html: body,
  });
}

// Send password reset email
export async function sendPasswordResetEmail(email: string, token: string): Promise<void> {
  const template = passwordResetEmailTemplate(token, email);
  await transporter.sendMail({
    from: `"Svenskt Vin" <${process.env.SMTP_FROM || 'no-reply@yourdomain.com'}>`,
    to: email,
    subject: template.subject,
    html: template.body,
  });
}
