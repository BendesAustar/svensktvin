-- 018_users_password_hash.sql
-- Add password_hash column for password-based auth.
-- Existing users (created via magic-link) have NULL and must set one on next login.

ALTER TABLE users ADD COLUMN password_hash TEXT;
