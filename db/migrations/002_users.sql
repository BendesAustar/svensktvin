-- 002_users.sql
-- User accounts. Auth is magic-link only; no passwords.

CREATE TABLE IF NOT EXISTS users (
  id         serial PRIMARY KEY,
  email      text NOT NULL UNIQUE,
  name       text NOT NULL,
  active     boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  last_login timestamptz,
  is_admin   boolean NOT NULL DEFAULT false
);
