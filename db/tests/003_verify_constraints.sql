-- 003_verify_constraints.sql
-- Tests that CHECK constraints and unique indexes work correctly.

DO $$
BEGIN
  -- Test vineyard legal_id_type constraint
  BEGIN
    INSERT INTO vineyards (name, county, municipality, lat, lon, legal_id_type)
    VALUES ('Test', 'Skåne', 'Malmö', 55.6, 13.0, 'invalid_type');
    RAISE EXCEPTION 'FAIL: vineyards should reject invalid legal_id_type';
  EXCEPTION WHEN check_violation THEN
    RAISE NOTICE 'PASS: vineyards legal_id_type CHECK constraint works';
  END;

  -- Test harvest yield_kg non-negative constraint
  BEGIN
    INSERT INTO harvest_records (block_id, harvest_year, yield_kg)
    VALUES (1, 2025, -100);
    RAISE EXCEPTION 'FAIL: harvest_records should reject negative yield_kg';
  EXCEPTION WHEN check_violation THEN
    RAISE NOTICE 'PASS: harvest_records yield_kg CHECK constraint works';
  END;

  -- Test harvest brix non-negative constraint
  BEGIN
    INSERT INTO harvest_records (block_id, harvest_year, yield_kg, brix)
    VALUES (1, 2026, 100, -1.0);
    RAISE EXCEPTION 'FAIL: harvest_records should reject negative brix';
  EXCEPTION WHEN check_violation THEN
    RAISE NOTICE 'PASS: harvest_records brix CHECK constraint works';
  END;

  -- Test variety name case-insensitive uniqueness
  BEGIN
    INSERT INTO varieties (name, color, piwi) VALUES ('Solaris', 'white', true);
    RAISE EXCEPTION 'FAIL: varieties should reject duplicate name (case-insensitive)';
  EXCEPTION WHEN unique_violation THEN
    RAISE NOTICE 'PASS: varieties name uniqueness is case-insensitive';
  END;
END $$;
