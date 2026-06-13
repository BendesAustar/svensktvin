// src/hooks.server.ts
import type { Handle } from '@sveltejs/kit';
import { getSession } from '$lib/server/auth.js';

export const handle: Handle = async ({ event, resolve }) => {
  const sessionId = event.cookies.get('session_id');
  event.locals.user = sessionId ? await getSession(sessionId) : null;
  const response = await resolve(event);
  return response;
};
