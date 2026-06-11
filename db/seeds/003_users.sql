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
    ('admin@svensktvin.se', 'Admin Användare', true),
    ('owner@example.se', 'Värd Ägare', false),
    ('editor@example.se', 'Redigerare Exempel', false)
  RETURNING id INTO admin_user_id;

  SELECT id INTO owner_id FROM users WHERE email = 'owner@example.se' LIMIT 1;
  SELECT id INTO editor_id FROM users FROM users WHERE email = 'editor@example.se' LIMIT 1;

  -- Link to vineyard A: admin is owner, owner is owner, editor is editor
  INSERT INTO vineyard_members (vineyard_id, user_id, role)
  SELECT (SELECT id FROM vineyards WHERE name = 'Vingård A' LIMIT 1), u.id,
         CASE WHEN u.is_admin OR u.email = 'owner@example.se' THEN 'owner' ELSE 'editor' END
  FROM users u
  WHERE u.email IN ('admin@svensktvin.se', 'owner@example.se', 'editor@example.se');
END $$;
