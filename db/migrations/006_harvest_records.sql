-- 006_harvest_records.sql
-- One record per block per year. Required + optional fate-of-fruit fields.

CREATE TABLE IF NOT EXISTS harvest_records (
  id                 serial PRIMARY KEY,
  block_id           integer NOT NULL REFERENCES blocks(id) ON DELETE CASCADE,
  harvest_year       integer NOT NULL,
  harvest_date       date,
  yield_kg           numeric(10,2) NOT NULL CHECK (yield_kg >= 0),
  brix               numeric(4,1) CHECK (brix >= 0),
  acid_g_l           numeric(4,2) CHECK (acid_g_l >= 0),
  vine_health_rating integer CHECK (vine_health_rating BETWEEN 1 AND 5),
  notes              text,
  still_wine_l       numeric(10,2) CHECK (still_wine_l >= 0),
  sparkling_l        numeric(10,2) CHECK (sparkling_l >= 0),
  juice_l            numeric(10,2) CHECK (juice_l >= 0),
  sold_kg            numeric(10,2) CHECK (sold_kg >= 0),
  discarded_kg       numeric(10,2) CHECK (discarded_kg >= 0),
  created_at         timestamptz NOT NULL DEFAULT now(),
  updated_at         timestamptz NOT NULL DEFAULT now(),
  deleted_at         timestamptz,

  UNIQUE (block_id, harvest_year)
);
