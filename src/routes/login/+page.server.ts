// src/routes/login/+page.server.ts
import { redirect, fail } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { getUserByEmail, createMagicLink } from '$lib/server/auth.js';
import { sendMagicLink } from '$lib/server/email.js';

export const load: PageServerLoad = ({ url }) => ({
  inviteToken: url.searchParams.get('invite')
});

export const actions: Actions = {
  default: async ({ request, url }) => {
    const data = await request.formData();
    const email = (data.get('email') as string)?.trim().toLowerCase();

    if (!email || !email.includes('@')) {
      return fail(400, { error: 'Ange en giltig e-postadress.', email });
    }

    // Look up user — always return the same message to avoid account enumeration
    const user = await getUserByEmail(email);
    if (user) {
      const token = await createMagicLink(user.id);
      await sendMagicLink(email, token);
      return { sent: true, inviteToken: url.searchParams.get('invite') };
    }

    // Non-registered user — redirect to register page with invite context
    const inviteToken = url.searchParams.get('invite');
    if (inviteToken) {
      throw redirect(303, `/register?token=${encodeURIComponent(inviteToken)}&email=${encodeURIComponent(email)}`);
    }

    return { sent: true };
  }
};
