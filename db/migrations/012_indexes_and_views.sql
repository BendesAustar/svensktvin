-- 012_indexes_and_views.sql
-- Performance indexes and soft-delete view for vineyards.

-- Indexes
CREATE INDEX IF NOT EXISTS idx_blocks_vineyard_id ON blocks (vineyard_id);
CREATE INDEX IF NOT EXISTS idx_blocks_variety_id ON blocks (variety_id);
CREATE INDEX IF NOT EXISTS idx_blocks_vineyard_name ON blocks (vineyard_id, block_name);

CREATE INDEX IF NOT EXISTS idx_harvest_records_block_id ON harvest_records (block_id);
CREATE INDEX IF NOT EXISTS idx_harvest_records_block_year ON harvest_records (block_id, harvest_year);

CREATE INDEX IF NOT EXISTS idx_harvest_records_year ON harvest_records (harvest_year);
CREATE INDEX IF NOT EXISTS idx_harvest_records_block_year_yields
  ON harvest_records (block_id, harvest_year)
  INCLUDE (yield_kg);

CREATE INDEX IF NOT EXISTS idx_vineyard_members_user_id ON vineyard_members (user_id);

CREATE INDEX IF NOT EXISTS idx_varieties_status ON varieties (status);

-- Soft-delete view: hides deleted vineyards
CREATE OR REPLACE VIEW active_vineyards AS
SELECT * FROM vineyards
WHERE deleted_at IS NULL;

-- Blocks visible view: joins active vineyards, excludes deleted blocks
CREATE OR REPLACE VIEW active_blocks AS
SELECT b.*
FROM blocks b
JOIN active_vineyards av ON av.id = b.vineyard_id
WHERE b.deleted_at IS NULL;

-- Harvest records visible view: only for active blocks
CREATE OR REPLACE VIEW active_harvest_records AS
SELECT hr.*
FROM harvest_records hr
JOIN active_blocks ab ON ab.id = hr.block_id
WHERE hr.deleted_at IS NULL;
