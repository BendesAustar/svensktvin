-- 010_varieties_submitted_by_fk.sql
-- FK from varieties.submitted_by_vineyard_id to vineyards.id.
-- Created after vineyards table exists (circular dependency workaround).

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.table_constraints
    WHERE constraint_name = 'varieties_submitted_by_fk'
  ) THEN
    ALTER TABLE varieties
      ADD CONSTRAINT varieties_submitted_by_fk
      FOREIGN KEY (submitted_by_vineyard_id) REFERENCES vineyards(id) ON DELETE SET NULL;
  END IF;
END $$;
