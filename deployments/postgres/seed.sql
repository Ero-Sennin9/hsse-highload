-- Seed для Iteration 0 baseline (read-heavy профиль).
-- Создаёт N опубликованных объявлений с детерминированными UUID вида
-- '00000000-0000-0000-0000-XXXXXXXXXXXX', где X — zero-padded порядковый номер.
-- k6 берёт случайный i из [1..N] и собирает такой же UUID на клиенте (lib/config.js).

\set N 50000

INSERT INTO ads (id, title, description, category, region, price, status, created_at, published_at)
SELECT
  ('00000000-0000-0000-0000-' || lpad(to_hex(i), 12, '0'))::uuid AS id,
  'seed ad #' || i                                                 AS title,
  'Тестовое объявление для нагрузочного теста, позиция ' || i     AS description,
  CASE (i % 5)
    WHEN 0 THEN 'transport'
    WHEN 1 THEN 'electronics'
    WHEN 2 THEN 'realty'
    WHEN 3 THEN 'clothing'
    ELSE 'other'
  END                                                              AS category,
  CASE (i % 3)
    WHEN 0 THEN 'moscow'
    WHEN 1 THEN 'spb'
    ELSE 'other'
  END                                                              AS region,
  (i * 100)                                                        AS price,
  'published'                                                      AS status,
  now() - (i || ' seconds')::interval                             AS created_at,
  now() - (i || ' seconds')::interval                             AS published_at
FROM generate_series(1, :N) AS s(i)
ON CONFLICT (id) DO NOTHING;

ANALYZE ads;

-- Контрольный вывод
SELECT count(*) AS published_ads FROM ads WHERE status = 'published';
