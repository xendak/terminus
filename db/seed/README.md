# Seed data

SQL fixtures applied by `scripts/testdb.sh --seed` against `stoptime_test`,
in filename order, after a drop/create/migrate cycle.

- `golden.sql` (T2): the tp.md section 5 example — routes A/B/C with the
  Portuguese addresses verbatim, totals 75/41/45 minutes, stop 1 always 0
  (plans/mvp/notes.md, "Golden fixture").
- `gen_year.sql` (T5): 12-month generated history for the dashboard
  performance test.

Files ending in `_check.sql` are **assertion scripts, not data** —
`testdb.sh` skips them; they run explicitly after seeding, e.g.
`psql "$TEST_DATABASE_URL" -f db/seed/golden_check.sql`.

Seeds assume an empty, freshly migrated `stoptime_test` — `testdb.sh`
guarantees that on every run.

Note: shell glob "filename order" follows the locale collation
(`en_US.UTF-8` ignores punctuation), not byte order — one reason migrations
use zero-padded `000N_` prefixes. If a future seed must run before another,
prefix it the same way.

## Demo data (dev database only)

`demo/demo.sql` is the live-demo dataset (T11): 16 Belo Horizonte locations
and ~8 weeks of closed routes for drivers A/B/C relative to `CURRENT_DATE`,
plus today's `active` route for driver A and tomorrow's `draft` for driver B.
It is applied by `scripts/dev-seed.sh` to the dev database `stoptime`, after
`golden.sql`, and never by `testdb.sh`: it lives in a subdirectory precisely
so the `db/seed/*.sql` glob does not see it. Rerun `scripts/dev-seed.sh`
(it truncates first) to move the demo's "today" to the current date.
