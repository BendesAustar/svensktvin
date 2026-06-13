-- Migration: 019_pending_invites_company_fields.sql
-- Adds company_name and owner_name to pending_invites
-- Adds password_hash to users table for future authentication support.
-- This migration is idempotent.

BEGIN;

-- Add company_name column to pending_invites if it doesn't exist.
-- This allows tracking the company name associated with a pending invitation.
ALTER TABLE pending_invites
    ADD COLUMN IF NOT EXISTS company_name TEXT;

COMMENT ON COLUMN pending_invites.company_name IS 'The name of the company associated with the pending invitation.';

-- Add owner_name column to pending_invites if it doesn't exist.
-- Add owner_name column to pending_invites if it doesn't exist.
ALTER TABLE pending_invites
    ADD COLUMN IF NOT EXISTS owner_name TEXT;

COMMENT ON COLUMN pending_invites.owner_name IS 'The owner associated with the pending invitation.';

-- Add password_hash column to users if it doesn't exist.
-- This is nullable because existing users do not have passwords set yet.
-- Passwords should be hashed using bcrypt or argon2 before storage.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS password_hash TEXT;

COMMENT ON COLUMN users.password_hash IS 'The hashed password for the user. NULL indicates no password is set.';

COMMIT;
