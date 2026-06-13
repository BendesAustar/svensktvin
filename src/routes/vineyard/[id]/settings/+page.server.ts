// src/routes/vineyard/[id]/settings/+page.server.ts
import { redirect, error, fail } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { sql } from '$lib/server/db.js';
import { randomBytes, createHash } from 'crypto';
import { inviteEmailTemplate } from '$lib/server/email.js';
import { verifyPassword, hashPassword, updateUserPassword, passwordValidation } from '$lib/server/auth.js';

export const load: PageServerLoad = async ({ params, locals }) => {
  if (!locals.user) throw redirect(303, '/login');

  const vineyardId = Number(params.id);

  const [member] = await sql`
    SELECT role FROM vineyard_members
    WHERE vineyard_id = ${vineyardId} AND user_id = ${locals.user.id}
  `;
  if (!member) throw error(403, 'Du har inte tillgång till denna vingård.');
  if (member.role !== 'owner') throw error(403, 'Endast ägare kan ändra inställningar.');

  const [vineyard] = await sql`
    SELECT id, name, county, municipality, organic, biodynamic,
           established_year, total_area_ha, legal_id, legal_id_type, legal_name
    FROM vineyards WHERE id = ${vineyardId}
  `;
  if (!vineyard) throw error(404, 'Vingården hittades inte.');

  const members = await sql`
    SELECT um.user_id AS id, um.role, u.email, u.name
    FROM vineyard_members um
    JOIN users u ON u.id = um.user_id
    WHERE um.vineyard_id = ${vineyardId}
    ORDER BY um.role DESC, u.name
  `;

  // Count total owners for per-member delete protection
  const [{ count: ownerCount }] = await sql<{ count: number }[]>`
    SELECT count(*)::int AS count FROM vineyard_members
    WHERE vineyard_id = ${vineyardId} AND role = 'owner'
  `;

  return {
    vineyard,
    ownerCount,
    members: members.map((m: Record<string, unknown>) => ({
      id: m.id,
      role: m.role,
      email: m.email,
      name: m.name
    }))
  };
};

export const actions: Actions = {
  default: async ({ request, locals, params }) => {
    if (!locals.user) throw redirect(303, '/login');

    const vineyardId = Number(params.id);
    const data = await request.formData();

    const action = data.get('action') as string;

    // Edit vineyard details
    if (action === 'update_vineyard') {
      const name = (data.get('name') as string)?.trim();
      const county = (data.get('county') as string)?.trim();
      const municipality = (data.get('municipality') as string)?.trim();
      const legal_id = (data.get('legal_id') as string)?.trim() || null;
      const legal_id_type = (data.get('legal_id_type') as string) || null;
      const legal_name = (data.get('legal_name') as string)?.trim() || null;
      const organic = data.get('organic') === 'on';
      const biodynamic = data.get('biodynamic') === 'on';
      const established_year = data.get('established_year') ? Number(data.get('established_year')) : null;
      const total_area_ha = data.get('total_area_ha') ? Number(data.get('total_area_ha')) : null;

      if (!name) return fail(400, { error: 'Vingårdsnamn krävs.' });
      if (!county) return fail(400, { error: 'Län krävs.' });

      try {
        await sql`
          UPDATE vineyards SET
            name = ${name}, county = ${county}, municipality = ${municipality},
            legal_id = ${legal_id}, legal_id_type = ${legal_id_type}, legal_name = ${legal_name},
            organic = ${organic}, biodynamic = ${biodynamic},
            established_year = ${established_year}, total_area_ha = ${total_area_ha}
          WHERE id = ${vineyardId}
        `;
      } catch (err) {
        console.error('Failed to update vineyard:', err);
        return fail(500, { error: 'Kunde inte uppdatera vingård. Försök igen.' });
      }
    }

    // Invite a new member by email (editors only — only owners can invite)
    if (action === 'invite_member') {
      const email = (data.get('email') as string)?.trim().toLowerCase();
      const role = data.get('role') as string;

      if (!email || !role || role !== 'editor') {
        return fail(400, { error: 'Välj en e-postadress och roll (endast redaktör).' });
      }

      // Check if already a member
      const [existing] = await sql`
        SELECT 1 FROM vineyard_members
        WHERE vineyard_id = ${vineyardId} AND user_id = (
          SELECT id FROM users WHERE email = ${email} AND active = true LIMIT 1
        )
      `;
      if (existing) {
        return fail(400, { error: 'Denna användare är redan medlem.' });
      }

      // Check for existing pending invite
      const [pending] = await sql`
        SELECT id FROM pending_invites
        WHERE email = ${email} AND vineyard_id = ${vineyardId} AND used = false
      `;
      if (pending) {
        return fail(400, { error: 'En inbjudan till denna e-postadress är redan under behandling.' });
      }

      // Create pending invite
      const token = randomBytes(32).toString('hex');
      const expiresAt = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000); // 7 days

      try {
        const [vineyard] = await sql`SELECT name FROM vineyards WHERE id = ${vineyardId}`;

        await sql`
          INSERT INTO pending_invites (email, vineyard_id, role, token, expires_at)
          VALUES (${email}, ${vineyardId}, ${role}, ${token}, ${expiresAt})
        `;

        // Send invite email (graceful fallback if SMTP not configured)
        if (!process.env.SMTP_HOST || !process.env.SMTP_USER || !process.env.SMTP_PASS) {
          console.warn(`[INVITE] SMTP not configured. Invite to ${email} for ${vineyard.name} (${role}) stored in DB but email not sent.`);
          console.warn(`[INVITE] Token: ${token}`);
          console.warn(`[INVITE] Accept at: ${process.env.APP_HOST ?? 'http://localhost:5173'}/invite?token=${token}`);
        } else {
          const appHost = process.env.APP_HOST ?? 'http://localhost:5173';
          const from = process.env.SMTP_FROM ?? 'noreply@svensktvin.se';
          const nodemailer = await import('nodemailer');
          const transport = nodemailer.createTransport({
            host: process.env.SMTP_HOST,
            port: Number(process.env.SMTP_PORT ?? 587),
            auth: { user: process.env.SMTP_USER, pass: process.env.SMTP_PASS }
          });
          const { text, html } = inviteEmailTemplate(appHost, vineyard.name, token);

          await transport.sendMail({
            from,
            to: email,
            subject: `Inbjudan till ${vineyard.name} på Svenskt Vin`,
            text,
            html
          });
        }
      } catch (err) {
        console.error('Failed to send invite:', err);
        return fail(500, { error: 'Kunde inte skicka inbjudan. Kontrollera SMTP-inställningarna.' });
      }
    }

    // Remove a member
    if (action === 'remove_member') {
      const targetUserId = Number(data.get('user_id'));

      if (targetUserId === locals.user!.id) {
        return fail(400, { error: 'Du kan inte ta bort dig själv.' });
      }

      const [{ count: ownerCount }] = await sql<{ count: number }[]>`
        SELECT count(*)::int AS count FROM vineyard_members
        WHERE vineyard_id = ${vineyardId} AND role = 'owner'
      `;
      const [targetMember] = await sql`
        SELECT role FROM vineyard_members
        WHERE vineyard_id = ${vineyardId} AND user_id = ${targetUserId}
      `;
      if (targetMember?.role === 'owner' && ownerCount <= 1) {
        return fail(400, { error: 'Kan inte ta bort den siste ägaren. Det måste alltid finnas minst en ägare.' });
      }

      await sql`
        DELETE FROM vineyard_members
        WHERE vineyard_id = ${vineyardId} AND user_id = ${targetUserId}
      `;
    }

    // Change password
    if (action === 'change_password') {
      const currentPassword = data.get('current_password') as string;
      const newPassword = data.get('new_password') as string;
      const confirmPassword = data.get('confirm_password') as string;

      if (!currentPassword || !newPassword || !confirmPassword) {
        return fail(400, { passwordError: 'Fyll i alla lösenordsfält.' });
      }

      if (newPassword !== confirmPassword) {
        return fail(400, { passwordError: 'De nya lösenorden matchar inte.' });
      }

      const validation = passwordValidation(newPassword);
      if (!validation.valid) {
        return fail(400, { passwordError: validation.errors.join(' ') });
      }

      // Verify current password
      const user = await sql`
        SELECT password_hash FROM users WHERE id = ${locals.user!.id} LIMIT 1
      `;
      if (!user?.[0]?.password_hash) {
        return fail(400, { passwordError: 'Ditt konto har inget lösenord. Logga in med inloggningslänk eller ställ in ett lösenord.' });
      }

      const valid = await verifyPassword(currentPassword, user[0].password_hash);
      if (!valid) {
        return fail(400, { passwordError: 'Nuvarande lösenord är felaktigt.' });
      }

      // Update password
      const hash = await hashPassword(newPassword);
      await updateUserPassword(locals.user!.id, hash);

      return { passwordSuccess: true, success: true };
    }

    return { success: true };
  }
};
