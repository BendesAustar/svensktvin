// src/routes/login/+page.server.ts
import { fail } from '@sveltejs/kit';
import type { Actions } from './$types';
import { getUserByEmail, createMagicLink } from '$lib/server/auth.js';
import { sendMagicLink } from '$lib/server/email.js';

export const actions: Actions = {
  default: async ({ request }) => {
    const data = await request.formData();
    const email = (data.get('email') as string)?.trim().toLowerCase();

    if (!email || !email.includes('@')) {
      return fail(400, { error: 'Ange en giltig e-postadress.' });
    }

    // Look up user — always return the same message to avoid account enumeration
    const user = await getUserByEmail(email);
    if (user) {
      const token = await createMagicLink(user.id);
      await sendMagicLink(email, token);
    }

    return { sent: true };
  }
};
