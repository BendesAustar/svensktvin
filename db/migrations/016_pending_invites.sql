-- 016_pending_invites.sql
-- Stores vineyard invitations sent to emails that don't yet have an account.

CREATE TABLE IF NOT EXISTS pending_invites (
  id             serial      PRIMARY KEY,
  email          text        NOT NULL,
  vineyard_id    integer     NOT NULL REFERENCES vineyards(id) ON DELETE CASCADE,
  role           text        NOT NULL CHECK (role IN ('owner', 'editor')),
  token          text        NOT NULL UNIQUE,
  expires_at     timestamptz NOT NULL,
  used           boolean     NOT NULL DEFAULT false,
  created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_pending_invites_email ON pending_invites (email);
CREATE INDEX IF NOT EXISTS idx_pending_invites_token ON pending_invites (token);
CREATE INDEX IF NOT EXISTS idx_pending_invites_expires_at ON pending_invites (expires_at);
CREATE INDEX IF NOT EXISTS idx_pending_invites_used ON pending_invites (used);
