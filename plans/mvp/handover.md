# Handover — mvp

## State

T2 landed (session 4): `db/migrations/0001_init.sql` (full schema per
`data-model.md` — app_user, driver_profile, location, route, route_stop with
the RN01/RN02 generated `stop_seconds`, parameter, audit_log, indexes),
`db/seed/golden.sql` (5 demo users — admin, manager, drivers A/B/C, password
`stoptime-dev`; 12 locations with tp.md §5 addresses verbatim; routes A/B/C
on 2026-06-15; parameters at defaults), `db/seed/golden_check.sql`
(assertions + printed proof, exits 0 printing 75/41/45/161/15.625).
`testdb.sh` skips `*_check.sql` when seeding. The cluster is **up**; dev DB
and `stoptime_test` are migrated; the test DB is seeded. The Go module is
still just `cmd/server` with `/healthz` — `internal/domain` does not exist.

## Next

**T3. Domain package — the rules as pure, unit-tested functions**
(`plans/mvp/plan.md`).

- Step 0: baseline green (below); `business-rules.md` golden table present;
  `backend/internal/domain` does not exist yet.
- Plan (read): `docs/spec/business-rules.md` (whole file).
- Do: `backend/internal/domain` — plain types (`Route`, `Stop`, timestamps)
  and pure functions: `StopSeconds` (RN01+RN02), `RouteTotalSeconds` (RN03
  with `min_stop_minutes`), `JourneyPercent` (RN04), `EstimatedCost` (RN07),
  and validation (departure >= arrival, stop order rules, distance > 0).
  No IO, no clock, no imports beyond stdlib. Unit tests pin the golden
  numbers (75/41/45; 15.625% for A; min_stop_minutes=6 → B totals 36) and the
  edge cases (midnight span, open stop, floor-of-sum vs sum-of-floors).
- Verify: `nix develop -c bash -c 'cd backend && go test ./internal/domain/ -v'`
  green; `go vet ./...` clean;
  `grep -rn "time.Now\|sql\|http" backend/internal/domain/` empty (purity guard).
- Stop-when: W3 green in this session, committed, pushed, handover rewritten.

## Baseline commands

```
git status                                        # clean tree
nix develop -c bash -c 'cd backend && go build ./... && go vet ./... && go test ./...'
```

Cluster (already up; restart with `nix develop -c bash -c 'scripts/db-up.sh'`
if needed). T3 is pure Go — the DB is not touched; golden numbers come from
`notes.md` / `business-rules.md`.

## Facts this task needs

- Parity commands **cd into `backend/`** first (module root, `notes.md`).
- Golden numbers, do not re-derive: route A=75, B=41, C=45 minutes
  (4500/2460/2700 seconds); stop 1 contributes 0 (RN01) even with timestamps;
  day total 161; journey percent A = 15.625%; with `min_stop_minutes=6`
  route B = 36 minutes (5-minute stop drops). Floor semantics: sum seconds,
  floor ONCE for display (59+59+59 s → 2 min, not 0).
- Domain functions mirror the SQL as a validation oracle: SQL is the
  persistence truth (generated column), the domain package is the tested
  mirror (architecture.md layering rule 4; the grep guard enforces purity).
- `stop_seconds` DB semantics for the mirror: order 1 → always 0; either
  timestamp NULL → NULL (contributes 0 to totals); else
  `floor(extract(epoch FROM departure_at - arrival_at))`.

## Open risks (subset relevant to T3)

- If a golden number fails and the spec's arithmetic turns out wrong, fix
  spec and test together in the same commit and record it (card bisection
  note). The SQL layer passed these numbers in T2, so divergence is more
  likely in the Go mirror than the spec.

## Out of scope

No store, no IO, no HTTP, no clock reads in domain (T4+). No
`internal/domain` subpackages — one flat package. No SQL strings anywhere in
domain.
