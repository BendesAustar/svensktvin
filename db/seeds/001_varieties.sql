-- 001_varieties.sql
-- Curated variety catalog seed. All approved, no submissions.

INSERT INTO varieties (name, synonyms, piwi, color, origin_country, status) VALUES
  ('Solaris', '{"SOL-65-20"}', true, 'white', 'Germany', 'approved'),
  ('Souvignier Gris', '{"SOU-55-42"}', true, 'rosé', 'Germany', 'approved'),
  ('Regent', '{"SAL-45-18"}', true, 'red', 'Germany', 'approved'),
  ('Cabernet Carbon', '{"SAL-139-21"}', true, 'red', 'Germany', 'approved'),
  ('Rondo', '{"SAL-16-6"}', true, 'red', 'Switzerland', 'approved'),
  ('Donaurieser', '{"SOL-120-23"}', true, 'white', 'Austria', 'approved'),
  ('Bacchus', NULL, true, 'white', 'Germany', 'approved'),
  ('Kerner', NULL, false, 'white', 'Germany', 'approved'),
  ('Müller-Thurgau', NULL, false, 'white', 'Germany', 'approved'),
  ('Chardonnay', NULL, false, 'white', 'France', 'approved'),
  ('Pinot Noir', '{"Spätburgunder, Black Burgundy"}', false, 'red', 'France', 'approved'),
  ('Riesling', NULL, false, 'white', 'Germany', 'approved'),
  ('Gewürztraminer', NULL, false, 'white', 'France', 'approved'),
  ('Seyval Blanc', NULL, true, 'white', 'France', 'approved'),
  ('Orion', '{"SOL-152-20"}', true, 'white', 'Germany', 'approved'),
  ('Himrod', NULL, true, 'white', 'USA', 'approved'),
  ('Frontenac', NULL, true, 'red', 'USA', 'approved'),
  ('Marquette', NULL, true, 'red', 'USA', 'approved'),
  ('Lambrusco', NULL, false, 'red', 'Italy', 'approved'),
  ('Pinot Gris', '{"Tokaji, Malvasia Nera"}', false, 'rosé', 'France', 'approved')
ON CONFLICT (LOWER(name)) DO NOTHING;
