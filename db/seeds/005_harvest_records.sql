-- 005_harvest_records.sql
-- Seed harvest records for blocks in vineyard A.

DO $$
DECLARE
  block_sol_soder integer;
  block_regent_mitt integer;
BEGIN
  SELECT blocks.id INTO block_sol_soder
  FROM blocks JOIN vineyards ON vineyards.id = blocks.vineyard_id
  WHERE vineyards.name = 'Vingård A' AND blocks.block_name = 'Sol söder'
  LIMIT 1;

  SELECT blocks.id INTO block_regent_mitt
  FROM blocks JOIN vineyards ON vineyards.id = blocks.vineyard_id
  WHERE vineyards.name = 'Vingård A' AND blocks.block_name = 'Regent mitt'
  LIMIT 1;

  INSERT INTO harvest_records (block_id, harvest_year, harvest_date, yield_kg, brix, acid_g_l, vine_health_rating, notes, still_wine_l, sparkling_l, sold_kg)
  VALUES
    (block_sol_soder, 2025, '2025-09-15', 3500.00, 22.0, 7.5, 4, 'Torrare år, bra syra.', 2400, 200, 1800),
    (block_sol_soder, 2024, '2024-09-12', 4200.00, 21.5, 7.8, 4, 'Bra skörd, jämn mognad.', 2900, 300, 2200),
    (block_regent_mitt, 2025, '2025-09-20', 4800.00, 23.0, 6.9, 5, 'Exceptionell mognad.', 3200, 500, 2800);
END $$;
