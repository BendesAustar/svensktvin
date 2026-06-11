-- 009_magic_link_tokens.sql
-- One-time login tokens. Rows cleaned up by cron (used = true AND expires_at < now()).

CREATE TABLE IF NOT EXISTS magic_link_tokens (
  id         serial      PRIMARY KEY,
  user_id    integer     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash text        NOT NULL UNIQUE,
  expires_at timestamptz NOT NULL,
  used       boolean     NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_magic_link_tokens_token_hash ON magic_link_tokens (token_hash);
CREATE INDEX IF NOT EXISTS idx_magic_link_tokens_expires_at ON magic_link_tokens (expires_at);
CREATE INDEX IF NOT EXISTS idx_magic_link_tokens_used ON magic_link_tokens (used);
