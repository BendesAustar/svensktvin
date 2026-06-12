-- 005_blocks.sql
-- Named parcels within a vineyard. One variety per block.

CREATE TABLE IF NOT EXISTS blocks (
  id              serial PRIMARY KEY,
  vineyard_id     integer NOT NULL REFERENCES vineyards(id) ON DELETE CASCADE,
  variety_id      integer NOT NULL REFERENCES varieties(id),
  block_name      text NOT NULL,
  area_ha         numeric(6,3) NOT NULL,
  vine_count      integer,
  planting_year   integer,
  training_system text,
  aspect          text CHECK (aspect IN ('N','NE','E','SE','S','SW','W','NW')),
  slope_degrees   numeric(4,1),
  elevation_m     integer,
  is_active       boolean NOT NULL DEFAULT true,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now(),
  deleted_at      timestamptz
);

-- Case-insensitive unique block name per vineyard
CREATE UNIQUE INDEX IF NOT EXISTS idx_blocks_name_ci
  ON blocks (vineyard_id, LOWER(block_name)) WHERE deleted_at IS NULL;
