-- golden_check.sql — asserts the seeded tp.md section 5 fixture
-- (docs/spec/business-rules.md, "Golden fixture": A=75, B=41, C=45;
-- stop 1 contributes 0; day total 161; journey percent 15.625;
-- min_stop_minutes=6 makes route B total 36).
--
-- Run against a freshly seeded test database:
--   psql "$TEST_DATABASE_URL" -f db/seed/golden_check.sql
--
-- ON_ERROR_STOP makes any failed assertion exit psql nonzero; a green run
-- prints the route totals (75/41/45), the day total (161) and the journey
-- percent (15.625) and exits 0.

\set ON_ERROR_STOP on

DO $$
DECLARE
    golden_date   date    := DATE '2026-06-15';
    a_secs        integer;
    b_secs        integer;
    c_secs        integer;
    day_secs      bigint;
    stop1_secs    bigint;
    journey_hours numeric;
    a_pct         numeric;
    b_secs_min6   bigint;
BEGIN
    -- RN03 totals: sum seconds over counted stops, floor once for display.
    SELECT coalesce(sum(rs.stop_seconds), 0)
      INTO a_secs
      FROM route_stop rs
      JOIN route r       ON r.id = rs.route_id
      JOIN app_user u    ON u.id = r.driver_user_id
     WHERE u.email = 'driver-a@stoptime.dev'
       AND r.route_date = golden_date
       AND rs.stop_order > 1;

    SELECT coalesce(sum(rs.stop_seconds), 0)
      INTO b_secs
      FROM route_stop rs
      JOIN route r       ON r.id = rs.route_id
      JOIN app_user u    ON u.id = r.driver_user_id
     WHERE u.email = 'driver-b@stoptime.dev'
       AND r.route_date = golden_date
       AND rs.stop_order > 1;

    SELECT coalesce(sum(rs.stop_seconds), 0)
      INTO c_secs
      FROM route_stop rs
      JOIN route r       ON r.id = rs.route_id
      JOIN app_user u    ON u.id = r.driver_user_id
     WHERE u.email = 'driver-c@stoptime.dev'
       AND r.route_date = golden_date
       AND rs.stop_order > 1;

    day_secs := a_secs + b_secs + c_secs;

    -- RN01: stop 1 carries seeded timestamps yet contributes exactly 0.
    SELECT coalesce(sum(rs.stop_seconds), 0)
      INTO stop1_secs
      FROM route_stop rs
     WHERE rs.stop_order = 1
       AND rs.arrival_at IS NOT NULL
       AND rs.departure_at IS NOT NULL;

    -- RN04: route A journey percent at the standard journey hours parameter.
    SELECT p.value
      INTO journey_hours
      FROM parameter p
     WHERE p.key = 'standard_journey_hours';

    a_pct := round(a_secs / (journey_hours * 3600) * 100, 3);

    -- RF10/RN03 with min_stop_minutes = 6, against a local threshold
    -- (the parameter itself stays 0 in the seed): route B's 5-minute stop
    -- drops out, so B totals 10 + 26 = 36 minutes.
    SELECT coalesce(sum(rs.stop_seconds), 0)
      INTO b_secs_min6
      FROM route_stop rs
      JOIN route r       ON r.id = rs.route_id
      JOIN app_user u    ON u.id = r.driver_user_id
     WHERE u.email = 'driver-b@stoptime.dev'
       AND r.route_date = golden_date
       AND rs.stop_order > 1
       AND floor(rs.stop_seconds / 60) >= 6;

    IF a_secs <> 4500 THEN
        RAISE EXCEPTION 'route A total % seconds, expected 4500 (75 min)', a_secs;
    END IF;
    IF b_secs <> 2460 THEN
        RAISE EXCEPTION 'route B total % seconds, expected 2460 (41 min)', b_secs;
    END IF;
    IF c_secs <> 2700 THEN
        RAISE EXCEPTION 'route C total % seconds, expected 2700 (45 min)', c_secs;
    END IF;
    IF day_secs <> 9660 THEN
        RAISE EXCEPTION 'day total % seconds, expected 9660 (161 min)', day_secs;
    END IF;
    IF stop1_secs <> 0 THEN
        RAISE EXCEPTION 'stop-1 stops carry % seconds, expected 0 (RN01)', stop1_secs;
    END IF;
    IF a_pct <> 15.625 THEN
        RAISE EXCEPTION 'route A journey percent %, expected 15.625', a_pct;
    END IF;
    IF b_secs_min6 <> 2160 THEN
        RAISE EXCEPTION 'route B with min_stop_minutes=6 totals % seconds, expected 2160 (36 min)', b_secs_min6;
    END IF;
END $$;

-- Printed proof (W2 verify): per-route totals.
SELECT r.note                                   AS route,
       floor(sum(rs.stop_seconds) / 60)::int   AS total_minutes
  FROM route_stop rs
  JOIN route r ON r.id = rs.route_id
 WHERE r.route_date = DATE '2026-06-15'
   AND rs.stop_order > 1
 GROUP BY r.note
 ORDER BY r.note;

-- Day total (by-day series entry for the golden date).
SELECT floor(sum(rs.stop_seconds) / 60)::int   AS day_total_minutes
  FROM route_stop rs
  JOIN route r ON r.id = rs.route_id
 WHERE r.route_date = DATE '2026-06-15'
   AND rs.stop_order > 1;

-- Journey percent of route A at the default 8h day (RN04).
SELECT round(sum(rs.stop_seconds)
              / ((SELECT p.value FROM parameter p
                   WHERE p.key = 'standard_journey_hours') * 3600)
              * 100, 3)                        AS route_a_journey_percent
  FROM route_stop rs
  JOIN route r ON r.id = rs.route_id
 WHERE r.route_date = DATE '2026-06-15'
   AND rs.stop_order > 1
   AND r.driver_user_id = 'aa000000-0000-4000-8000-000000000003';
