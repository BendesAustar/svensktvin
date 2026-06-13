// src/routes/auth/set-password/+page.server.ts
import { redirect, fail } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { getUserByEmail, verifyToken, hashPassword, updateUserPassword, createSession, passwordValidation } from '$lib/server/auth.js';
import { sql } from '$lib/server/db.js';

export const load: PageServerLoad = async ({ url, cookies }) => {
  const token = url.searchParams.get('token');
  const email = url.searchParams.get('email');

  // If user is already logged in, redirect to settings
  const sessionId = cookies.get('session_id');
  if (sessionId) {
    const [session] = await sql`
      SELECT id FROM sessions WHERE id = ${sessionId} AND expires_at > now() LIMIT 1
    `;
    if (session) {
      throw redirect(303, '/vineyard');
    }
  }

  // If token exists, validate it to get the user
  let userId: number | null = null;
  if (token) {
    userId = await verifyToken(token);
    if (!userId) {
      return { form: { error: 'Ogiltig eller utgången länk. Begära en ny.', token: null }, email: null };
    }
  }

  // If email exists in params, look up the user
  if (email && !token) {
    const user = await getUserByEmail(email);
    if (!user) {
      return { form: { error: 'Inget konto hittades för den e-postadressen.' }, email };
    }
    if (user.password_hash) {
      // Already has a password — redirect to login
      throw redirect(303, `/login?email=${encodeURIComponent(email)}&error=existing_password`);
    }
    userId = user.id;
  }

  return { form: { token, email, password: '', confirmPassword: '' }, email };
};

export const actions: Actions = {
  default: async ({ request, cookies }) => {
    const data = await request.formData();
    const token = data.get('token') as string;
    const email = (data.get('email') as string)?.trim().toLowerCase();
    const password = data.get('password') as string;
    const confirmPassword = data.get('confirmPassword') as string;

    // Validate passwords match
    if (password !== confirmPassword) {
      return fail(400, { error: 'Lösenorden matchar inte.', password, confirmPassword, token });
    }

    // Validate password strength
    const validation = passwordValidation(password);
    if (!validation.valid) {
      return fail(400, { error: 'Lösenordet uppfyller inte kraven.', password, confirmPassword, passwordErrors: validation.errors, token });
    }

    let userId: number | null = null;

    // Get user from token or email
    if (token) {
      userId = await verifyToken(token);
      if (!userId) {
        return fail(400, { error: 'Ogiltig eller utgången länk. Begära en ny.' });
      }
    } else if (email) {
      const user = await getUserByEmail(email);
      if (!user || !user.password_hash) {
        return fail(400, { error: 'Ogiltig begäran.' });
      }
      userId = user.id;
    } else {
      return fail(400, { error: 'Ogiltig begäran.' });
    }

    // Hash and set password
    const hash = await hashPassword(password);
    await updateUserPassword(userId!, hash);

    // Create session and redirect
    const sessionId = await createSession(userId!);
    cookies.set('session_id', sessionId, {
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      sameSite: 'lax',
      path: '/',
      maxAge: 30 * 24 * 60 * 60,
    });

    throw redirect(303, '/vineyard');
  },
};
