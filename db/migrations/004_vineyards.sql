-- 004_vineyards.sql
-- Legal vineyard entities. Soft-deleted rows are preserved for historical data.

CREATE TABLE IF NOT EXISTS vineyards (
  id               serial PRIMARY KEY,
  name             text NOT NULL,
  county           text NOT NULL,
  municipality     text NOT NULL,
  lat              numeric(9,6) NOT NULL,
  lon              numeric(9,6) NOT NULL,
  established_year integer,
  total_area_ha    numeric(8,2),
  organic          boolean NOT NULL DEFAULT false,
  biodynamic       boolean NOT NULL DEFAULT false,
  active           boolean NOT NULL DEFAULT true,
  legal_id         text UNIQUE,
  legal_id_type    text CHECK (legal_id_type IN (
    'ab',
    'enskild',
    'handelsbolag',
    'swealagsbolag',
    'stiftelse',
    'kommun',
    'other'
  )),
  legal_name       text,
  deleted_at       timestamptz,
  updated_at       timestamptz NOT NULL DEFAULT now(),
  created_at       timestamptz NOT NULL DEFAULT now()
);
