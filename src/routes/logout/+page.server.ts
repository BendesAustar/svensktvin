// src/routes/logout/+page.server.ts
import { redirect } from '@sveltejs/kit';
import type { Actions } from './$types';
import { deleteSession } from '$lib/server/auth.js';

export const actions: Actions = {
  default: async ({ cookies }) => {
    const sessionId = cookies.get('session_id');
    if (sessionId) {
      await deleteSession(sessionId);
      cookies.delete('session_id', { path: '/' });
    }
    throw redirect(303, '/login');
  }
};
