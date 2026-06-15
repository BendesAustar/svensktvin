-- Migration 020: Admin Dashboard Support
-- Adds admin_actions audit table for the admin dashboard.
BEGIN;

-- Ensure created_at exists on users (from migration 002)
-- This is a no-op if the column already exists.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'users' AND column_name = 'created_at'
    ) THEN
        ALTER TABLE users ADD COLUMN created_at timestamptz NOT NULL DEFAULT now();
    END IF;
END $$;

-- Create admin_actions table for audit trail
CREATE TABLE IF NOT EXISTS admin_actions (
  id           serial PRIMARY KEY,
  admin_user_id integer NOT NULL REFERENCES users(id),
  target_user_id integer REFERENCES users(id),
  action       text NOT NULL,
  details      jsonb DEFAULT '{}',
  created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_admin_actions_admin ON admin_actions (admin_user_id);
CREATE INDEX IF NOT EXISTS idx_admin_actions_target ON admin_actions (target_user_id);
CREATE INDEX IF NOT EXISTS idx_admin_actions_created ON admin_actions (created_at DESC);

COMMENT ON TABLE admin_actions IS 'Audit log for admin dashboard actions.';

COMMIT;
