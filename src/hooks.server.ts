// src/hooks.server.ts
import type { Handle } from '@sveltejs/kit';
import { getSession } from '$lib/server/auth.js';

export const handle: Handle = async ({ event, resolve }) => {
  const sessionId = event.cookies.get('session_id');
  event.locals.user = sessionId ? await getSession(sessionId) : null;
  const response = await resolve(event);
  // Secure cookie attributes — must use headers.append() after resolve
  // Secure flag only in production (localhost dev needs it without Secure)
  if (sessionId) {
    const secure = process.env.NODE_ENV === 'production' ? 'Secure' : '';
    const cookieValue = `session_id=${sessionId}; HttpOnly; ${secure} SameSite=Lax; Path=/; Max-Age=${30 * 24 * 60 * 60}`;
    response.headers.append('set-cookie', cookieValue);
  }
  return response;
};
