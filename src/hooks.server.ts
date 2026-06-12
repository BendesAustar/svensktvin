// src/hooks.server.ts
import type { Handle } from '@sveltejs/kit';
import { getSession } from '$lib/server/auth.js';

export const handle: Handle = async ({ event, resolve }) => {
  const sessionId = event.cookies.get('session_id');
  event.locals.user = sessionId ? await getSession(sessionId) : null;
  const response = await resolve(event);
  // Secure cookie attributes
  if (sessionId) {
    event.cookies.set('session_id', sessionId, {
      httpOnly: true,
      secure: true,
      sameSite: 'lax',
      path: '/',
      maxAge: 30 * 24 * 60 * 60
    });
  }
  return response;
};
