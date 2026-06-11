-- 001_verify_schema.sql
-- Verifies all tables, columns, constraints, and triggers exist.

DO $$
DECLARE
  tables_found text[] := ARRAY[]::text[];
  checks int := 0;
  failed int := 0;
BEGIN
  -- Check all tables exist
  FOR t IN SELECT table_name FROM information_schema.tables
    WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
  LOOP
    IF t NOT IN (
      'users', 'varieties', 'vineyards', 'blocks',
      'harvest_records', 'vineyard_members',
      'sessions', 'magic_link_tokens', '_migrations_applied'
    ) THEN
      RAISE NOTICE 'UNEXPECTED TABLE: %', t;
    END IF;
  END LOOP;

  -- Check required tables
  IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'users') THEN
    RAISE EXCEPTION 'FAIL: users table not found';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'varieties') THEN
    RAISE EXCEPTION 'FAIL: varieties table not found';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'vineyards') THEN
    RAISE EXCEPTION 'FAIL: vineyards table not found';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'blocks') THEN
    RAISE EXCEPTION 'FAIL: blocks table not found';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'harvest_records') THEN
    RAISE EXCEPTION 'FAIL: harvest_records table not found';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'vineyard_members') THEN
    RAISE EXCEPTION 'FAIL: vineyard_members table not found';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'sessions') THEN
    RAISE EXCEPTION 'FAIL: sessions table not found';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'magic_link_tokens') THEN
    RAISE EXCEPTION 'FAIL: magic_link_tokens table not found';
  END IF;

  -- Check required columns
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'vineyards' AND column_name = 'updated_at'
  ) THEN
    RAISE EXCEPTION 'FAIL: vineyards.updated_at not found';
  END IF;
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'vineyards' AND column_name = 'deleted_at'
  ) THEN
    RAISE EXCEPTION 'FAIL: vineyards.deleted_at not found';
  END IF;
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'vineyards' AND column_name = 'legal_id_type'
  ) THEN
    RAISE EXCEPTION 'FAIL: vineyards.legal_id_type not found';
  END IF;
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'blocks' AND column_name = 'updated_at'
  ) THEN
    RAISE EXCEPTION 'FAIL: blocks.updated_at not found';
  END IF;
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'harvest_records' AND column_name = 'updated_at'
  ) THEN
    RAISE EXCEPTION 'FAIL: harvest_records.updated_at not found';
  END IF;

  -- Check CHECK constraints exist
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.table_constraints
    WHERE table_name = 'vineyards' AND constraint_type = 'CHECK'
      AND constraint_name = 'vineyards_legal_id_type_check'
  ) THEN
    RAISE EXCEPTION 'FAIL: vineyards legal_id_type CHECK constraint not found';
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM information_schema.table_constraints
    WHERE table_name = 'harvest_records' AND constraint_type = 'CHECK'
      AND constraint_name = 'harvest_records_yield_kg_check'
  ) THEN
    RAISE EXCEPTION 'FAIL: harvest_records yield_kg CHECK constraint not found';
  END IF;

  -- Check triggers exist
  IF NOT EXISTS (
    SELECT 1 FROM pg_trigger WHERE tgname = 'set_vineyards_updated_at'
  ) THEN
    RAISE EXCEPTION 'FAIL: vineyards updated_at trigger not found';
  END IF;
  IF NOT EXISTS (
    SELECT 1 FROM pg_trigger WHERE tgname = 'set_blocks_updated_at'
  ) THEN
    RAISE EXCEPTION 'FAIL: blocks updated_at trigger not found';
  END IF;
  IF NOT EXISTS (
    SELECT 1 FROM pg_trigger WHERE tgname = 'set_harvest_records_updated_at'
  ) THEN
    RAISE EXCEPTION 'FAIL: harvest_records updated_at trigger not found';
  END IF;

  -- Check extensions
  IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_trgm') THEN
    RAISE EXCEPTION 'FAIL: pg_trgm extension not found';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'postgis') THEN
    RAISE EXCEPTION 'FAIL: postgis extension not found';
  END IF;

  RAISE NOTICE 'All schema checks passed.';
END $$;
