-- 014_varieties_extended.sql
-- Extended variety catalog. Focus: PIWI releases from DE/CH/AT, cold-hardy
-- North American crosses, and missing classics. All pre-approved.
-- Source: breeder catalogs (JKI, Agroscope, Staatliche Weinbauinstitut),
-- University of Minnesota releases, ENTAV-INRAE.
-- Replace with VIVC import when available.

INSERT INTO varieties (name, synonyms, piwi, color, origin_country, status) VALUES

  -- PIWI whites — Germany (JKI / Staatliches Weinbauinstitut Freiburg)
  ('Muscaris',        '{"Gm 6494-5"}',                true, 'white', 'Germany',      'approved'),
  ('Johanniter',      '{"FR 177-83"}',                true, 'white', 'Germany',      'approved'),
  ('Helios',          '{"JK-SB 18-95"}',              true, 'white', 'Germany',      'approved'),
  ('Hibernal',        '{"Gm 7652-2"}',                true, 'white', 'Germany',      'approved'),
  ('Bronner',         '{"Gm 6473-1"}',                true, 'white', 'Germany',      'approved'),
  ('Sirius',          '{"Gm 6767-3"}',                true, 'white', 'Germany',      'approved'),
  ('Merzling',        '{"Gm 6918-6"}',                true, 'white', 'Germany',      'approved'),
  ('Palatina',        '{"Gm 8027-8"}',                true, 'white', 'Germany',      'approved'),
  ('Calardis Blanc',  '{"FR 522-99"}',                true, 'white', 'Germany',      'approved'),
  ('Calardis Musqué', '{"FR 525-99"}',                true, 'white', 'Germany',      'approved'),
  ('Aris',            '{"Gm 322-58"}',                true, 'white', 'Germany',      'approved'),
  ('Kernling',        NULL,                           true, 'white', 'Germany',      'approved'),
  ('Aromera',         NULL,                           true, 'white', 'Germany',      'approved'),
  ('Felicia',         NULL,                           true, 'white', 'Germany',      'approved'),
  ('Monarch',         NULL,                           true, 'white', 'Germany',      'approved'),

  -- PIWI whites — Switzerland (Agroscope)
  ('Sauvignac',       '{"RAC 2286"}',                 true, 'white', 'Switzerland',  'approved'),
  ('Bianca',          NULL,                           true, 'white', 'Hungary',      'approved'),

  -- PIWI reds — Germany / Switzerland
  ('Pinotin',         '{"SW 21.209"}',                true, 'red',   'Switzerland',  'approved'),
  ('Cabernet Cortis', '{"Gm 756-4"}',                 true, 'red',   'Germany',      'approved'),
  ('Cabernet Cantor', '{"Gm 6473-4"}',                true, 'red',   'Germany',      'approved'),
  ('Prior',           '{"Gm 6493-2"}',                true, 'red',   'Germany',      'approved'),
  ('Bolero',          NULL,                           true, 'red',   'Germany',      'approved'),
  ('Dakapo',          NULL,                           true, 'red',   'Germany',      'approved'),
  ('Leon Millot',     NULL,                           true, 'red',   'France',       'approved'),
  ('Maréchal Foch',   '{"Kuhlmann 188-2"}',           true, 'red',   'France',       'approved'),
  ('Dornfelder',      NULL,                           false,'red',   'Germany',      'approved'),
  ('Cabernet Dorsa',  NULL,                           true, 'red',   'Germany',      'approved'),

  -- Cold-hardy — University of Minnesota / Cornell
  ('La Crescent',     NULL,                           true, 'white', 'USA',          'approved'),
  ('Itasca',          NULL,                           true, 'white', 'USA',          'approved'),
  ('Brianna',         NULL,                           true, 'white', 'USA',          'approved'),
  ('St. Pepin',       NULL,                           true, 'white', 'USA',          'approved'),
  ('Swenson White',   NULL,                           true, 'white', 'USA',          'approved'),
  ('Petite Pearl',    NULL,                           true, 'red',   'USA',          'approved'),
  ('St. Croix',       NULL,                           true, 'red',   'USA',          'approved'),
  ('Swenson Red',     NULL,                           true, 'red',   'USA',          'approved'),

  -- Classics not yet in catalog
  ('Cabernet Franc',      NULL,                       false,'red',   'France',       'approved'),
  ('Cabernet Sauvignon',  NULL,                       false,'red',   'France',       'approved'),
  ('Merlot',              NULL,                       false,'red',   'France',       'approved'),
  ('Sauvignon Blanc',     NULL,                       false,'white', 'France',       'approved'),
  ('Pinot Blanc',         '{"Weißburgunder"}',        false,'white', 'France',       'approved'),
  ('Pinot Meunier',       '{"Schwarzriesling"}',      false,'red',   'France',       'approved'),
  ('Gamay',               '{"Gamay Noir"}',           false,'red',   'France',       'approved'),
  ('Viognier',            NULL,                       false,'white', 'France',       'approved'),
  ('Zweigelt',            '{"Blauer Zweigelt"}',      false,'red',   'Austria',      'approved'),
  ('Blaufränkisch',       '{"Lemberger, Kékfrankos"}',false,'red',   'Austria',      'approved')

ON CONFLICT (LOWER(name)) DO NOTHING;
