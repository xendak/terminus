# Seed data

SQL fixtures applied by `scripts/testdb.sh --seed` against `stoptime_test`,
in filename order, after a drop/create/migrate cycle.

- `golden.sql` (T2): the tp.md section 5 example — routes A/B/C with the
  Portuguese addresses verbatim, totals 75/41/45 minutes, stop 1 always 0
  (plans/mvp/notes.md, "Golden fixture").
- `gen_year.sql` (T5): 12-month generated history for the dashboard
  performance test.

Seeds assume an empty, freshly migrated `stoptime_test` — `testdb.sh`
guarantees that on every run.
