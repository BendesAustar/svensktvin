-- 002_verify_data_integrity.sql
-- Verifies seed data integrity and referential consistency.

DO $$
DECLARE
  variety_count int;
  vineyard_count int;
  block_count int;
  harvest_count int;
  member_count int;
BEGIN
  SELECT count(*) INTO variety_count FROM varieties;
  IF variety_count < 10 THEN
    RAISE EXCEPTION 'FAIL: expected 10+ seeded varieties, got %', variety_count;
  END IF;

  SELECT count(*) INTO vineyard_count FROM vineyards WHERE deleted_at IS NULL;
  IF vineyard_count < 3 THEN
    RAISE EXCEPTION 'FAIL: expected 3 seeded vineyards, got %', vineyard_count;
  END IF;

  SELECT count(*) INTO block_count FROM blocks JOIN vineyards ON vineyards.id = blocks.vineyard_id
    WHERE vineyards.name = 'Vingård A' AND vineyards.deleted_at IS NULL;
  IF block_count < 3 THEN
    RAISE EXCEPTION 'FAIL: expected 3 blocks for Vingård A, got %', block_count;
  END IF;

  SELECT count(*) INTO harvest_count FROM harvest_records
    JOIN blocks ON blocks.id = harvest_records.block_id
    JOIN vineyards ON vineyards.id = blocks.vineyard_id
    WHERE vineyards.name = 'Vingård A' AND harvest_records.deleted_at IS NULL;
  IF harvest_count < 3 THEN
    RAISE EXCEPTION 'FAIL: expected 3 harvest records for Vingård A, got %', harvest_count;
  END IF;

  SELECT count(*) INTO member_count FROM vineyard_members
    JOIN users ON users.id = vineyard_members.user_id
    JOIN vineyards ON vineyards.id = vineyard_members.vineyard_id
    WHERE vineyards.name = 'Vingård A' AND vineyards.deleted_at IS NULL;
  IF member_count < 3 THEN
    RAISE EXCEPTION 'FAIL: expected 3 members for Vingård A, got %', member_count;
  END IF;

  RAISE NOTICE 'All data integrity checks passed.';
END $$;
