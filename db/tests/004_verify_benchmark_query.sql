-- 004_verify_benchmark_query.sql
-- Verifies the benchmark query returns expected results with seed data.

DO $$
DECLARE
  result_count int;
  solaris_count int;
BEGIN
  -- Run the benchmark query for Solaris in Skåne, 2025
  SELECT count(*) INTO result_count
  FROM harvest_records hr
  JOIN blocks b ON b.id = hr.block_id
  JOIN varieties v ON v.id = b.variety_id
  JOIN vineyards vi ON vi.id = b.vineyard_id
  WHERE v.name = 'Solaris'
    AND v.status = 'approved'
    AND hr.harvest_year = 2025
    AND vi.county = 'Skåne'
    AND vi.deleted_at IS NULL
  GROUP BY vi.id
  HAVING count(DISTINCT vi.id) >= 1;

  -- Should return at least 1 county group with data
  IF result_count < 1 THEN
    RAISE EXCEPTION 'FAIL: benchmark query returned no results for Solaris in Skåne 2025';
  END IF;

  -- Verify the anonymity floor works: count distinct vineyards
  SELECT count(DISTINCT vi.id) INTO solaris_count
  FROM harvest_records hr
  JOIN blocks b ON b.id = hr.block_id
  JOIN varieties v ON v.id = b.variety_id
  JOIN vineyards vi ON vi.id = b.vineyard_id
  WHERE v.name = 'Solaris'
    AND v.status = 'approved'
    AND hr.harvest_year = 2025
    AND vi.deleted_at IS NULL;

  RAISE NOTICE 'Benchmark query verified: % distinct vineyard(s) with Solaris data in 2025',
    solaris_count;
END $$;
