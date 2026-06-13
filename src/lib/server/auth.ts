// src/lib/server/auth.ts
import { randomBytes, createHash } from 'crypto';
import { sql } from './db.js';
import bcrypt from 'bcrypt';

// ---------------------------------------------------------------------------
// Password hashing / verification — new for password-based auth
// ---------------------------------------------------------------------------

/**
 * Hashes a password using bcrypt.
 * @param password - The plain text password.
 * @param rounds - The number of salt rounds (default 12).
 * @returns The hashed password string.
 */
export async function hashPassword(password: string, rounds: number = 12): Promise<string> {
	return bcrypt.hash(password, rounds);
}

/**
 * Verifies a password against a bcrypt hash.
 * @param password - The plain text password to verify.
 * @param hash - The bcrypt hash to compare against.
 * @returns True if the password matches, false otherwise.
 */
export async function verifyPassword(password: string, hash: string): Promise<boolean> {
	return bcrypt.compare(password, hash);
}

/**
 * Validates password strength.
 * @param password - The password to validate.
 * @returns An object containing validity status and an array of error messages.
 */
export function passwordValidation(password: string): { valid: boolean; errors: string[] } {
	const errors: string[] = [];

	if (password.length < 8) {
		errors.push('Lösenordet måste vara minst 8 tecken.');
	}

	if (!/[A-Z]/.test(password)) {
		errors.push('Lösenordet måste innehålla minst en stor bokstav.');
	}

	if (!/[a-z]/.test(password)) {
		errors.push('Lösenordet måste innehålla minst en liten bokstav.');
	}

	if (!/[0-9]/.test(password)) {
		errors.push('Lösenordet måste innehålla minst en siffra.');
	}

	return { valid: errors.length === 0, errors };
}

/**
 * Updates the user's password hash in the database.
 * @param userId - The ID of the user.
 * @param hash - The new hashed password.
 */
export async function updateUserPassword(userId: number, hash: string): Promise<void> {
	await sql`
		UPDATE users SET password_hash = ${hash} WHERE id = ${userId}
	`;
}

/**
 * Checks if a user needs password setup (i.e., has no password_hash yet).
 * @param userId - The ID of the user.
 * @returns True if the user has no password_hash.
 */
export async function needsPasswordSetup(userId: number): Promise<boolean> {
	const [{ has_password }] = await sql`
		SELECT (password_hash IS NOT NULL) AS has_password
		FROM users
		WHERE id = ${userId}
		LIMIT 1
	`;
	return has_password === false;
}

/**
 * Retrieves a user by their email address (includes password_hash for login).
 * @param email - The email address of the user.
 * @returns A user object containing id, email, password_hash, name, and is_admin.
 */
export async function getUserByEmail(
	email: string
): Promise<{ id: number; email: string; password_hash: string | null; name: string; is_admin: boolean } | null> {
	const [user] = await sql`
		SELECT id, email, password_hash, name, is_admin
		FROM users
		WHERE email = ${email} AND active = true
	`;
	return user ?? null;
}

// ---------------------------------------------------------------------------
// Session / magic-link functions — preserved from original
// ---------------------------------------------------------------------------

/**
 * Creates a cryptographically random session token.
 * @returns The token string.
 */
export function createSessionToken(): string {
	return randomBytes(64).toString('hex');
}

/**
 * Creates a magic-link token pair (raw token + hashed token).
 * @returns { token: string; hash: string }
 */
export function createMagicLinkToken(): { token: string; hash: string } {
	const raw = randomBytes(32).toString('hex');
	const hash = createHash('sha256').update(raw).digest('hex');
	return { token: raw, hash };
}

/**
 * Creates a magic-link entry in the database.
 * @param userId - The user ID to associate the link with.
 * @returns The raw token (to be sent via email).
 */
export async function createMagicLink(userId: number): Promise<string> {
	const raw = randomBytes(32).toString('hex');
	const hash = createHash('sha256').update(raw).digest('hex');
	const expiresAt = new Date(Date.now() + 15 * 60 * 1000); // 15 min
	await sql`
		INSERT INTO magic_link_tokens (user_id, token_hash, expires_at)
		VALUES (${userId}, ${hash}, ${expiresAt})
	`;
	return raw;
}

/**
 * Verifies a magic-link token.
 * @param rawToken - The raw token from the email link.
 * @returns The user_id if valid (and marks it used), or null.
 */
export async function verifyToken(rawToken: string): Promise<number | null> {
	const hash = createHash('sha256').update(rawToken).digest('hex');
	const [token] = await sql`
		UPDATE magic_link_tokens
		SET used = true
		WHERE token_hash = ${hash}
		  AND used = false
		  AND expires_at > now()
		RETURNING user_id
	`;
	return token?.user_id ?? null;
}

/**
 * Creates a session record in the database.
 * @param userId - The user ID to create a session for.
 * @returns The session ID.
 */
export async function createSession(userId: number): Promise<string> {
	const id = crypto.randomUUID();
	const expiresAt = new Date(Date.now() + 30 * 24 * 60 * 60 * 1000); // 30 days
	await sql`
		INSERT INTO sessions (id, user_id, expires_at)
		VALUES (${id}, ${userId}, ${expiresAt})
	`;
	return id;
}

/**
 * Gets the latest active session by user ID.
 * @param userId - The user ID.
 * @returns Session object or null.
 */
export async function getSessionByUserId(
	userId: number
): Promise<{ id: string; expires_at: Date } | null> {
	const [session] = await sql`
		SELECT id, expires_at
		FROM sessions
		WHERE user_id = ${userId}
		  AND expires_at > now()
		ORDER BY created_at DESC
		LIMIT 1
	`;
	return session ?? null;
}

/**
 * Gets session details by session ID.
 * Excludes password_hash — session payloads must never contain it.
 * @param sessionId - The session ID.
 * @returns User object or null.
 */
export async function getSession(
	sessionId: string
): Promise<{ id: number; email: string; name: string; is_admin: boolean } | null> {
	const [row] = await sql`
		SELECT u.id, u.email, u.name, u.is_admin
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.id = ${sessionId}
		  AND s.expires_at > now()
		  AND u.active = true
	`;
	return row ?? null;
}

/**
 * Deletes a session by session ID.
 * @param sessionId - The session ID to delete.
 */
export async function deleteSession(sessionId: string): Promise<void> {
	await sql`DELETE FROM sessions WHERE id = ${sessionId}`;
}

/**
 * Updates the user's last login timestamp.
 * @param userId - The user ID.
 */
export async function updateLastLogin(userId: number): Promise<void> {
	await sql`UPDATE users SET last_login = now() WHERE id = ${userId}`;
}
