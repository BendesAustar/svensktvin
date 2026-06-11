-- 011_updated_at_triggers.sql
-- Auto-update updated_at on BEFORE UPDATE for tables that have it.

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
BEGIN
  -- vineyards
  IF NOT EXISTS (
    SELECT 1 FROM pg_trigger
    WHERE tgname = 'set_vineyards_updated_at'
  ) THEN
    CREATE TRIGGER set_vineyards_updated_at
      BEFORE UPDATE ON vineyards
      FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
  END IF;

  -- blocks
  IF NOT EXISTS (
    SELECT 1 FROM pg_trigger
    WHERE tgname = 'set_blocks_updated_at'
  ) THEN
    CREATE TRIGGER set_blocks_updated_at
      BEFORE UPDATE ON blocks
      FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
  END IF;

  -- harvest_records
  IF NOT EXISTS (
    SELECT 1 FROM pg_trigger
    WHERE tgname = 'set_harvest_records_updated_at'
  ) THEN
    CREATE TRIGGER set_harvest_records_updated_at
      BEFORE UPDATE ON harvest_records
      FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
  END IF;
END $$;
