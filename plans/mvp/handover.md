# Handover — mvp

## State

T4 landed (session 6): `internal/store` (pgxpool + WithTx + repositories +
audit; session TimeZone=UTC) and `internal/app` (21 write-path services per
operations.md, Actor for provenance, injected clock, FieldError detail).
Migration 0002 made `audit_log.entity_id` text (parameter keys). Deps pinned:
pgx v5.11.0, x/crypto v0.57.0, google/uuid v1.6.0 (recorded decision).
Integration suite green from a fresh test DB (`go test -count=1
./internal/...`). Cluster up; test DB migrated (empty). Role enforcement is
NOT in yet — T6 threads Actor and enforces the matrix.

## Next

**T5. Reads, SQL aggregation, cost, and the 3-second rule**
(`plans/mvp/plan.md`).

- Step 0: baseline green (below); T4 services callable from a test (they
  are — the suite drives them).
- Plan (read): `docs/spec/operations.md` (reads and aggregation section),
  `docs/spec/business-rules.md` (RN04, RN07, performance section),
  `docs/spec/data-model.md` (indexes).
- Do: read services — GetRoute, ListRoutes, GetDashboardByDay/Month/Period,
  each backed by ONE SQL aggregation query returning plain structs. Cost and
  journey percent computed in SQL numeric, rounded once. Driver scoping
  (own routes only) as a query-level filter. `db/seed/gen_year.sql` (or a
  Go test helper) generating 12 months of synthetic routes. Tests: golden
  series from the golden seed (day 161, month 161, period 161); a timed
  test asserting the 12-month dashboard answers under 3s with EXPLAIN
  ANALYZE showing index scans (assert on the plan, print it in the log).
- Verify: `nix develop -c bash -c 'cd backend && go test -count=1
  ./internal/...'` green including the perf test, run fresh in this
  session, from `scripts/testdb.sh` (+ `--seed` for the golden series).
- Stop-when: W5 green in this session, committed, pushed, handover
  rewritten.

## Baseline commands

```
git status                                            # clean tree
nix develop -c bash -c 'scripts/testdb.sh --seed'     # fresh migrated+seeded test DB
nix develop -c bash -c 'eval "$(scripts/db-up.sh)" && cd backend && go build ./... && go vet ./... && go test -count=1 ./internal/...'
```

`-count=1` matters: go's cache cannot see the DB rebuild.

## Facts this task needs

- Parity commands **cd into `backend/`**; integration tests need
  `TEST_DATABASE_URL` in the environment (the eval line exports it).
- Decimal feeds for the domain oracle: scan numeric as text, never
  float64 (`notes.md` "T3 session"); money/percent computed in SQL
  numeric, rounded once — the store returns the SQL result, the domain
  mirrors it for tests.
- numeric text round-trips are scale-normalized: `6.09` → `"6.0900"`
  (parameter), `100` → `"100.00"` (distance), journey percent scale is
  whatever the SQL pins — document it.
- pgx returns timestamptz in the PROCESS timezone — compare instants in
  tests, format only at the UI edge (later).
- Golden fixture (testdb --seed): three routes on 2026-06-15 → day 161,
  month 161, period 161; driver emails driver-a/b/c@stoptime.dev; the
  app suite's TestMain truncates, so golden-seed tests must re-seed or
  use their own generator (gen_year).
- RN05 filter and driver scoping are query-level (`WHERE driver_user_id
  = $1`), never post-filtering (operations.md).
- Indexes in 0001: `route(route_date)`, `route(driver_user_id,
  route_date)` unique, `route_stop(route_id, stop_order)` unique,
  `audit_log(at)`. The perf test asserts index scans on these.
- Errors: `ErrNotFound`, `ErrForbidden` (define when scoping lands) —
  same sentinel pattern; adapters map in T7.

## Open risks (subset relevant to T5)

- The perf escape hatch: if synthetic generation makes the timing flaky,
  lower the row count and record the number; the pattern guard (index
  scans, no per-row loops) matters more than the absolute time.
- `db/seed/gen_year.sql` ordering with golden.sql under the en_US
  collation (punctuation ignored): if gen_year must run after golden,
  name it so glob order guarantees it (see `notes.md` T2 session) or make
  the generator independent of golden rows.

## Out of scope

No HTTP (T7), no auth/roles (T6 — but driver scoping's query-level filter
IS this card, keyed on a driver_user_id input). No UI. No CSV export
(comes with the screens card T9 per the plan; ExportPeriodCSV's SQL may
land here only if trivial — it is NOT in this card's Do).
