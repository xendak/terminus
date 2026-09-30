-- 0005_stop_threshold_params.sql
--
-- The dashboard colours single stops by two thresholds (minutes): at or
-- above stop_warn_minutes a stop is "long", at or above
-- stop_alert_minutes it is "too long". They were hardcoded 15/45 in the
-- UI; parameters are data, not code (AGENTS.md, business-rules.md
-- "Parameters").
--
-- parameter.updated_by is NOT NULL (FK app_user). On a database that
-- already has users (dev), the rows are attributed to the first admin.
-- A freshly created database has no users yet, so nothing is inserted
-- here: the seed that creates the users (db/seed/golden.sql) inserts
-- the parameters with the rest of the defaults.

INSERT INTO parameter (key, value, unit, updated_by)
SELECT v.key, v.value, 'minutes', a.id
  FROM (VALUES ('stop_warn_minutes', 15), ('stop_alert_minutes', 45)) AS v (key, value)
 CROSS JOIN (SELECT id FROM app_user WHERE role = 'admin' ORDER BY created_at, id LIMIT 1) AS a
ON CONFLICT (key) DO NOTHING;
