-- 002_vineyards.sql
-- Seed vineyard fixtures.

DO $$
DECLARE
  vineyard_a_id integer;
  vineyard_b_id integer;
  vineyard_c_id integer;
BEGIN
  INSERT INTO vineyards (name, county, municipality, lat, lon, established_year, total_area_ha, organic, biodynamic)
  VALUES ('Vingård A', 'Skåne', 'Malmö', 55.6059, 13.0007, 2015, 2.5, false, false)
  RETURNING id INTO vineyard_a_id;

  INSERT INTO vineyards (name, county, municipality, lat, lon, established_year, total_area_ha, organic, biodynamic)
  VALUES ('Vingård B', 'Skåne', 'Helsingborg', 56.0465, 12.6945, 2018, 1.8, true, false)
  RETURNING id INTO vineyard_b_id;

  INSERT INTO vineyards (name, county, municipality, lat, lon, established_year, total_area_ha, organic, biodynamic)
  VALUES ('Vingård C', 'Blekinge', 'Karlskrona', 56.1615, 15.5867, 2020, 3.2, false, false)
  RETURNING id INTO vineyard_c_id;
END $$;
