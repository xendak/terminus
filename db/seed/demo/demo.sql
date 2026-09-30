-- demo.sql — a believable live-demo dataset for the DEV database (T11).
--
-- Applied by scripts/dev-seed.sh AFTER db/seed/golden.sql, on the dev
-- database `stoptime` only. It lives in db/seed/demo/ so testdb.sh --seed
-- (which globs db/seed/*.sql) never loads it: tests see the golden
-- fixture alone.
--
-- What it adds, relative to CURRENT_DATE (so the dashboard's default
-- windows always have data):
--   * 16 Belo Horizonte locations (1 distribution center + 15 customers);
--   * closed routes for drivers A, B, C on the weekdays of the last 8
--     weeks (a few days off each), 4–7 stops, stopped times 3–45 min,
--     distances 25–120 km, working hours in America/Sao_Paulo;
--   * today's route for driver A, `active`, first two delivery stops done;
--   * tomorrow's route for driver B, `draft`, no times.
--
-- Reproducible: every "random" choice is a hash of (driver, date, slot),
-- and route/stop ids are md5-derived uuids, so rerunning on the same day
-- yields the same rows. Stop 1 (departure point, RN01) gets a departure
-- time only. The golden date 2026-06-15 is skipped (RN05). No audit rows:
-- seeds are not user actions. Dates are Sao Paulo calendar days (SET
-- TimeZone below), so the result does not depend on the server zone;
-- today's done stops are never later than now.

-- Calendar days are Sao Paulo days whatever the server or client zone:
-- CURRENT_DATE, and date/timestamp casts below, read this setting.
SET TimeZone = 'America/Sao_Paulo';

-- h(key) — deterministic pseudo-random integer in [0, 2^28).
CREATE FUNCTION pg_temp.h(k text) RETURNS int
LANGUAGE sql IMMUTABLE AS $$
  SELECT ('x' || substr(md5(k), 1, 7))::bit(28)::int
$$;

-- Locations. Registered by the manager.
INSERT INTO location (id, label, address, latitude, longitude, created_by) VALUES
  ('aa000000-0000-4000-8000-000000000401', 'CD Terminus União',          'Av. Cristiano Machado, 4000 - União, Belo Horizonte - MG',               -19.873800, -43.932200, 'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-000000000402', 'Mercado Central',            'Av. Augusto de Lima, 744 - Centro, Belo Horizonte - MG',                 -19.922500, -43.943700, 'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-000000000403', 'Drogaria Savassi',           'Rua Pernambuco, 1000 - Savassi, Belo Horizonte - MG',                    -19.937600, -43.934000, 'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-000000000404', 'Padaria Lourdes',            'Rua Curitiba, 2010 - Lourdes, Belo Horizonte - MG',                      -19.931100, -43.948000, 'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-000000000405', 'Supermercado Pampulha',      'Av. Otacílio Negrão de Lima, 3000 - Pampulha, Belo Horizonte - MG',      -19.852000, -43.975000, 'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-000000000406', 'Empório Buritis',            'Av. Professor Mário Werneck, 1500 - Buritis, Belo Horizonte - MG',       -19.975000, -43.969000, 'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-000000000407', 'Clínica Funcionários',       'Av. Afonso Pena, 2300 - Funcionários, Belo Horizonte - MG',              -19.932000, -43.931000, 'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-000000000408', 'Restaurante Santa Tereza',   'Rua Mármore, 310 - Santa Tereza, Belo Horizonte - MG',                   -19.915000, -43.917000, 'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-000000000409', 'Armazém Barreiro',           'Av. Afonso Vaz de Melo, 640 - Barreiro, Belo Horizonte - MG',            -19.978000, -44.018000, 'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-00000000040a', 'Farmácia Venda Nova',        'Av. Vilarinho, 1200 - Venda Nova, Belo Horizonte - MG',                  -19.814000, -43.953000, 'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-00000000040b', 'Mercearia Sagrada Família',  'Rua Itajubá, 1510 - Sagrada Família, Belo Horizonte - MG',               -19.903000, -43.924000, 'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-00000000040c', 'Distribuidora Cidade Nova',  'Av. José Cândido da Silveira, 1700 - Cidade Nova, Belo Horizonte - MG', -19.893000, -43.927000, 'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-00000000040d', 'Papelaria Sion',             'Rua Haiti, 220 - Sion, Belo Horizonte - MG',                             -19.953000, -43.934000, 'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-00000000040e', 'Pet Shop Castelo',           'Av. dos Engenheiros, 900 - Castelo, Belo Horizonte - MG',                -19.887000, -44.000000, 'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-00000000040f', 'Hortifruti Prado',           'Rua Platina, 520 - Prado, Belo Horizonte - MG',                          -19.927000, -43.960000, 'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-000000000410', 'Mercado do Cruzeiro',        'Rua Opala, 150 - Cruzeiro, Belo Horizonte - MG',                         -19.943000, -43.927000, 'aa000000-0000-4000-8000-000000000002');

-- The demo plan: one row per route (driver × date), with its size.
CREATE TEMP TABLE demo_route ON COMMIT DROP AS
SELECT md5('demo-route/' || drv || '/' || d)::uuid AS id,
       drv::uuid                                  AS driver_user_id,
       d                                          AS route_date,
       status,
       n_stops
  FROM (
        -- History: weekdays of the last 8 weeks, ~1 day in 8 off per driver.
        -- Hash keys use the date text (YYYY-MM-DD), never a timestamptz,
        -- so the data does not depend on the session time zone.
        SELECT drv, d, 'closed' AS status,
               4 + pg_temp.h('n/' || drv || '/' || d) % 4 AS n_stops
          FROM unnest(ARRAY['aa000000-0000-4000-8000-000000000003',
                            'aa000000-0000-4000-8000-000000000004',
                            'aa000000-0000-4000-8000-000000000005']) AS drv,
               (SELECT g::date AS d
                  FROM generate_series(CURRENT_DATE - 56, CURRENT_DATE - 1, interval '1 day') AS g) AS days
         WHERE extract(isodow FROM d) < 6
           AND d <> DATE '2026-06-15'
           AND pg_temp.h('off/' || drv || '/' || d) % 8 <> 0
        UNION ALL
        -- Today: drivers A, B and C all on the road (first two deliveries
        -- done), so the per-driver timeline has more than one row.
        SELECT 'aa000000-0000-4000-8000-000000000003', CURRENT_DATE, 'active', 6
        UNION ALL
        SELECT 'aa000000-0000-4000-8000-000000000004', CURRENT_DATE, 'active', 5
        UNION ALL
        SELECT 'aa000000-0000-4000-8000-000000000005', CURRENT_DATE, 'active', 7
        UNION ALL
        -- Tomorrow: driver B planned, not started.
        SELECT 'aa000000-0000-4000-8000-000000000004', CURRENT_DATE + 1, 'draft', 5
       ) plan
 WHERE d <> DATE '2026-06-15';

INSERT INTO route (id, driver_user_id, route_date, distance_km, status, created_by)
SELECT id, driver_user_id, route_date,
       CASE WHEN status = 'closed'
            THEN 25 + (pg_temp.h('km/' || id) % 951) / 10.0 END, -- 25.0–120.0 km
       status,
       'aa000000-0000-4000-8000-000000000002'
  FROM demo_route;

-- Stops: stop 1 is always the distribution center; stops 2..n are
-- distinct customers picked by hash rank.
CREATE TEMP TABLE demo_stop ON COMMIT DROP AS
WITH customers AS (
  SELECT r.id AS route_id, r.route_date, r.status, r.n_stops, l.id AS location_id,
         row_number() OVER (PARTITION BY r.id ORDER BY pg_temp.h(r.id || '/' || l.id)) + 1 AS stop_order
    FROM demo_route r
   CROSS JOIN location l
   WHERE l.id BETWEEN 'aa000000-0000-4000-8000-000000000402'
                  AND 'aa000000-0000-4000-8000-000000000410'
), picked AS (
  SELECT route_id, route_date, status, stop_order, location_id
    FROM customers WHERE stop_order <= n_stops
  UNION ALL
  SELECT id, route_date, status, 1, 'aa000000-0000-4000-8000-000000000401'::uuid
    FROM demo_route
)
SELECT route_id, route_date, status, stop_order, location_id,
       -- leave the depot 07:30–08:29 local time
       CASE WHEN stop_order = 1 THEN 30 + pg_temp.h('start/' || route_id) % 60 ELSE 0 END AS start_min,
       -- drive 15–45 min to reach this stop
       CASE WHEN stop_order = 1 THEN 0 ELSE 15 + pg_temp.h('drive/' || route_id || '/' || stop_order) % 31 END AS drive_min,
       -- stopped 3–45 min, skewed towards short stops
       CASE WHEN stop_order = 1 THEN 0
            ELSE 3 + (pg_temp.h('stopA/' || route_id || '/' || stop_order) % 43)
                   * (pg_temp.h('stopB/' || route_id || '/' || stop_order) % 43) / 42 END AS stop_min
  FROM picked;

-- Local clock offsets (minutes after 07:00 local) via running sums:
-- arrival_k = depart_1 + Σ_{j≤k} drive_j + Σ_{j<k} stop_j.
-- Today's active route must never show times in the future: when the
-- planned schedule of its done stops ends later than 5 minutes ago, the
-- whole route shifts earlier by the difference (seeding at 08:00 puts
-- its last recorded departure at 07:55).
INSERT INTO route_stop (id, route_id, stop_order, location_id, arrival_at, departure_at)
SELECT md5('demo-stop/' || route_id || '/' || stop_order)::uuid,
       route_id, stop_order, location_id,
       CASE WHEN stop_order > 1 AND done THEN planned_arrival - shift END,
       CASE WHEN done THEN planned_departure - shift END
  FROM (
        SELECT t.*,
               CASE WHEN status = 'active'
                    THEN greatest(interval '0',
                                  max(planned_departure) FILTER (WHERE done) OVER (PARTITION BY route_id)
                                  + interval '5 minutes' - now())
                    ELSE interval '0'
               END AS shift
          FROM (
                SELECT s.*,
                       ((route_date + time '07:00') + interval '1 minute' * arrive_off)
                           AT TIME ZONE 'America/Sao_Paulo' AS planned_arrival,
                       ((route_date + time '07:00') + interval '1 minute' * (arrive_off + stop_min))
                           AT TIME ZONE 'America/Sao_Paulo' AS planned_departure
                  FROM (
                        SELECT s.*,
                               sum(start_min + drive_min) OVER w + coalesce(sum(stop_min) OVER w_prev, 0) AS arrive_off,
                               CASE status
                                    WHEN 'closed' THEN true
                                    WHEN 'active' THEN stop_order <= 3 -- departed + first two deliveries
                                    ELSE false
                               END AS done
                          FROM demo_stop s
                        WINDOW w      AS (PARTITION BY route_id ORDER BY stop_order),
                               w_prev AS (PARTITION BY route_id ORDER BY stop_order
                                          ROWS BETWEEN UNBOUNDED PRECEDING AND 1 PRECEDING)
                       ) s
               ) t
       ) u;
