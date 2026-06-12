-- 003_users.sql
-- Seed user fixtures.

DO $$
DECLARE
  admin_user_id integer;
  owner_id integer;
  editor_id integer;
BEGIN
  INSERT INTO users (email, name, is_admin)
  VALUES
    ('admin@svensktvin.se', 'Admin Användare', true)
  RETURNING id INTO admin_user_id;

  INSERT INTO users (email, name, is_admin)
  VALUES
    ('owner@example.se', 'Värd Ägare', false)
  RETURNING id INTO owner_id;

  INSERT INTO users (email, name, is_admin)
  VALUES
    ('editor@example.se', 'Redigerare Exempel', false)
  RETURNING id INTO editor_id;

  -- Link users to vineyard A
  INSERT INTO vineyard_members (vineyard_id, user_id, role)
  SELECT (SELECT id FROM vineyards WHERE name = 'Vingård A' LIMIT 1), admin_user_id, 'owner';

  INSERT INTO vineyard_members (vineyard_id, user_id, role)
  SELECT (SELECT id FROM vineyards WHERE name = 'Vingård A' LIMIT 1), owner_id, 'owner';

  INSERT INTO vineyard_members (vineyard_id, user_id, role)
  SELECT (SELECT id FROM vineyards WHERE name = 'Vingård A' LIMIT 1), editor_id, 'editor';
END $$;
