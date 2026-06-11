-- 007_vineyard_members.sql
-- User-vineyard membership with role-based access.

CREATE TABLE IF NOT EXISTS vineyard_members (
  vineyard_id integer NOT NULL REFERENCES vineyards(id) ON DELETE CASCADE,
  user_id     integer NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role        text NOT NULL CHECK (role IN ('owner', 'editor')),
  created_at  timestamptz NOT NULL DEFAULT now(),

  PRIMARY KEY (vineyard_id, user_id)
);
