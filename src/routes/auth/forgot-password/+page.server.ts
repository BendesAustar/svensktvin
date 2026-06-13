// src/routes/auth/forgot-password/+page.server.ts
import { fail } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { getUserByEmail, createMagicLink } from '$lib/server/auth.js';
import { sendMagicLink } from '$lib/server/email.js';

// Rate limiting
const resetAttempts = new Map<string, { count: number; resetAt: number }>();
const MAX_ATTEMPTS = 5;
const WINDOW_MS = 15 * 60 * 1000;

function checkRateLimit(ip: string): boolean {
  const now = Date.now();
  const entry = resetAttempts.get(ip);
  if (!entry) return true;
  if (now > entry.resetAt) {
    resetAttempts.delete(ip);
    return true;
  }
  return entry.count < MAX_ATTEMPTS;
}

function recordAttempt(ip: string) {
  const entry = resetAttempts.get(ip) || { count: 0, resetAt: Date.now() + WINDOW_MS };
  entry.count++;
  resetAttempts.set(ip, entry);
}

export const actions: Actions = {
  default: async ({ request, getClientAddress }) => {
    const ip = getClientAddress();
    if (!checkRateLimit(ip)) {
      return fail(429, { error: 'För många försök. Försök igen om 15 minuter.' });
    }

    const data = await request.formData();
    const email = (data.get('email') as string)?.trim().toLowerCase();

    if (!email || !email.includes('@')) {
      return fail(400, { error: 'Ange en giltig e-postadress.', email });
    }

    const user = await getUserByEmail(email);
    if (user) {
      const token = await createMagicLink(user.id);
      recordAttempt(ip);

      // Send email with graceful fallback — if SMTP isn't configured, log token to console
      try {
        await sendMagicLink(email, token);
      } catch (err) {
        console.warn(`[FORGOT-PASSWORD] SMTP not available for ${email}, token: ${token}`);
      }

      return { sent: true };
    }

    // Same message to avoid account enumeration
    return { sent: true };
  },
};
