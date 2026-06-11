-- 004_blocks.sql
-- Seed block fixtures for vineyard A.

DO $$
DECLARE
  vineyard_id integer;
  solaris_id integer;
  regent_id integer;
BEGIN
  SELECT id INTO vineyard_id FROM vineyards WHERE name = 'Vingård A' LIMIT 1;
  SELECT id INTO solaris_id FROM varieties WHERE name = 'Solaris' LIMIT 1;
  SELECT id INTO regent_id FROM varieties WHERE name = 'Regent' LIMIT 1;

  INSERT INTO blocks (vineyard_id, variety_id, block_name, area_ha, vine_count, planting_year, training_system, aspect)
  VALUES
    (vineyard_id, solaris_id, 'Sol söder', 0.5, 2000, 2016, 'VSP', 'S'),
    (vineyard_id, solaris_id, 'Sol norr', 0.4, 1600, 2017, 'VSP', 'N'),
    (vineyard_id, regent_id, 'Regent mitt', 0.6, 2400, 2016, 'GDC', 'SE');
END $$;
