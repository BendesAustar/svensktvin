-- 003_varieties.sql
-- Shared variety catalog. Powers fuzzy-match submission and benchmarking.

CREATE TABLE IF NOT EXISTS varieties (
  id                       serial PRIMARY KEY,
  name                     text NOT NULL,
  synonyms                 text[],
  piwi                     boolean NOT NULL DEFAULT false,
  color                    text NOT NULL CHECK (color IN ('white', 'red', 'rosé', 'other')),
  origin_country           text,
  status                   text NOT NULL DEFAULT 'approved'
                             CHECK (status IN ('approved', 'review_needed')),
  submitted_by_vineyard_id integer,
  created_at               timestamptz NOT NULL DEFAULT now()
);

-- Case-insensitive unique name (via lowercase index)
CREATE UNIQUE INDEX IF NOT EXISTS idx_varieties_name_ci
  ON varieties (LOWER(name));

CREATE INDEX IF NOT EXISTS idx_varieties_name_trgm ON varieties USING GIN (name gin_trgm_ops);
