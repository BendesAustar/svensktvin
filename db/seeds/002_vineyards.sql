-- 002_vineyards.sql
-- Seed vineyard fixtures. IDs are auto-generated via RETURNING.

DO $$
DECLARE
  vineyard_a_id integer;
  vineyard_b_id integer;
  vineyard_c_id integer;
BEGIN
  INSERT INTO vineyards (name, county, municipality, lat, lon, established_year, total_area_ha, organic, biodynamic)
  VALUES
    ('Vingård A', 'Skåne', 'Malmö', 55.6059, 13.0007, 2015, 2.5, false, false),
    ('Vingård B', 'Skåne', 'Helsingborg', 56.0465, 12.6945, 2018, 1.8, true, false),
    ('Vingård C', 'Blekinge', 'Karlskrona', 56.1615, 15.5867, 2020, 3.2, false, false)
  RETURNING id INTO vineyard_a_id;

  -- We need three separate statements because RETURNING only captures the last one
  SELECT id INTO vineyard_b_id FROM vineyards WHERE name = 'Vingård B' LIMIT 1;
  SELECT id INTO vineyard_c_id FROM vineyards WHERE name = 'Vingård C' LIMIT 1;

  -- Store for seed usage
  PERFORM set_config('sv.vineyard_a_id', vineyard_a_id::text, true);
  PERFORM set_config('sv.vineyard_b_id', vineyard_b_id::text, true);
  PERFORM set_config('sv.vineyard_c_id', vineyard_c_id::text, true);
END $$;
