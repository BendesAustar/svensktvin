-- 017_block_locks.sql
-- Pessimistic lock for block-level harvest editing.
-- Ensures only one user edits harvest data for a block at a time.
-- Locks expire after 30 minutes automatically.

CREATE TABLE IF NOT EXISTS block_locks (
  id          serial      PRIMARY KEY,
  block_id    integer     NOT NULL REFERENCES blocks(id) ON DELETE CASCADE,
  user_id     integer     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  locked_at   timestamptz NOT NULL DEFAULT now(),
  expires_at  timestamptz NOT NULL
);

-- Only one active lock per block at a time
CREATE UNIQUE INDEX IF NOT EXISTS uix_block_locks_block
  ON block_locks (block_id);
