-- 001_extensions.sql
-- Required extensions: pg_trgm for fuzzy variety matching, postgis for future block polygons (Phase 2)

CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS postgis;
