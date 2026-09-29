# Handover — mvp

## State

T3 landed (session 5): `backend/internal/domain` — the business rules as
pure, unit-tested functions (`StopSeconds`, `WholeMinutes`,
`RouteTotalSeconds`, `JourneyPercent`, `EstimatedCost`, `RoundMoney`,
`Validate*`), exact-rationale decimals (`math/big.Rat`), sentinel errors.
27 test pins green including the golden numbers (A/B/C = 75/41/45 min,
day 161, 15.625%, B = 36 at min_stop_minutes 6). The Go module has **no
dependencies yet** — pgx v5 arrives in T4 per the architecture.md budget.
Cluster up; `stoptime_test` golden-seeded; dev DB migrated.

## Next

**T4. Store + write operations — the service layer can run a whole day**
(`plans/mvp/plan.md`).

- Step 0: baseline green (below); `scripts/testdb.sh` re-applies T2's
  migration to a fresh test DB; pgx v5 added per the dependency budget.
- Plan (read): `docs/spec/operations.md` (directories, route composition,
  time recording sections), `docs/spec/data-model.md` (audit section).
- Do: `internal/store` — pgxpool setup, tx helper, repositories, audit
  writes inside the same transaction. `internal/app` — services for:
  CreateDriver, UpdateDriver, ListDrivers, CreateManager, ListManagers,
  CreateLocation, UpdateLocation, ListLocations, CreateRoute, AddStop,
  RemoveStop, ReorderStops, StartRoute, CloseRoute, ReopenRoute,
  RecordArrival, RecordDeparture, UpdateStopTimes, SetRouteDistance,
  GetParams, UpdateParam. Sentinel errors per the error model. No HTTP
  anywhere in these packages. Integration tests against `stoptime_test`
  driving a full day for route A and asserting: RN05 conflict on
  duplicate, RN02 rejection, audit rows on corrections/params,
  closed-route rejection.
- Verify: `nix develop -c bash -c 'cd backend && go build ./... && go vet
  ./... && go test ./internal/...'` green from a fresh `scripts/testdb.sh`.
- Stop-when: W4 green in this session, committed, pushed, handover
  rewritten.

## Baseline commands

```
git status                                      # clean tree
nix develop -c bash -c 'scripts/testdb.sh --seed'   # fresh test DB (also proves migrations apply)
nix develop -c bash -c 'cd backend && go build ./... && go vet ./... && go test ./...'
```

## Facts this task needs

- Parity commands **cd into `backend/`** (module root; `notes.md`).
- Feeding the domain oracle: money/percent inputs are `*big.Rat`; scan
  pgx numerics as strings (`big.Rat.SetString` parses "6.09", "8", and
  fractions) — never through float64 (`notes.md` "T3 session").
- Domain sentinel errors exist already: `ErrDepartureBeforeArrival`,
  `ErrInvalidStopOrder`, `ErrInvalidDistance`, `ErrInvalidKmPerL`,
  `ErrInvalidParameter`. Services return them; adapters map statuses in
  T7. New store-level errors (RN05 conflict, closed route) join the same
  pattern per `operations.md`'s error model.
- Golden fixture facts (deterministic ids, emails like
  `driver-a@stoptime.dev`, route A = driver-a on 2026-06-15 with 4
  stops): `notes.md` "T2 session" and `db/seed/golden.sql`.
- Integration tests connect to `TEST_DATABASE_URL` only; `eval
  "$(scripts/db-up.sh)"` exports both URLs. Never the dev database.
- Audit rows commit inside the mutation's transaction (data-model.md,
  RNF05) — if a pgx tx pattern fights, simplify the repository shape,
  never the guarantee (card escape hatch).
- Purity/layering guards that must stay green: no `time.Now`/sql strings
  in domain; no SQL outside `internal/store`.

## Open risks (subset relevant to T4)

- pgx v5 pin: record the exact version in `notes.md` when added. The
  dependency budget is closed — anything beyond pgx needs a recorded
  decision first.
- `scripts/testdb.sh` rebuilds `stoptime_test` from scratch —
  integration tests must create their own data and never depend on the
  golden seed being present (unless they call testdb --seed themselves).

## Out of scope

No HTTP (T7), no read/aggregation services beyond the card's list (T5),
no auth/session/role checks (T6), no UI. No schema changes — the
migration set is closed for T4; a discovered schema bug is a new
migration + spec fix in the same commit.
