# Handover — mvp

## State

T8 landed (session 10): builder + tracker as htmx state machines over
the #route-body fragment; drivers log in to /routes/today and run the
day (RN01 has no stopwatch on stop 1); JSON mirrors for every involved
operation. Parity now runs `-p 1` (packages share the test DB). All
guards hold. Cluster up, test DB migrated.

## Next

**T9. Dashboard, history, params, audit, export — the client's
screens** (`plans/mvp/plan.md`).

- Step 0: baseline green (below); aggregation services exist (T5 —
  GetDashboardByDay/Month/Period, ListRoutes, GetRoute, GetParams,
  UpdateParam all shipped).
- Plan (read): `docs/spec/screens.md` (Dashboard §4, History §5,
  Parameters §7, Audit §8), `docs/spec/business-rules.md` (Parameters).
- Do: dashboard page with three tabs fed by the aggregate series
  (Chart.js renders, never computes); history list + route detail +
  corrections form (audited, manager/admin only); parameters screen;\n  audit list (admin); CSV export per operations.md (UTF-8 BOM, RFC
  4180). A Go test parses the exported CSV back and asserts rows.
- Verify: build/vet/test green (`-count=1 -p 1`); a scripted end-to-end
  run demonstrates acceptance criteria 2–4 of `tp.md` section 10; the
  CSV test passes.
- Stop-when: W9 green in this session, committed, pushed, handover
  rewritten.

## Baseline commands

```
git status                                            # clean tree
nix develop -c bash -c 'scripts/testdb.sh --seed'     # fresh migrated+seeded test DB
nix develop -c bash -c 'eval "$(scripts/db-up.sh)" && cd backend && go build ./... && go vet ./... && go test -count=1 -p 1 ./internal/...'
```

`-count=1 -p 1` both matter: the cache can't see DB rebuilds, and
packages must not run in parallel on the shared test DB.

## Facts this task needs

- **Remaining transports** (operations.md): history — `GET /history` +
  `GET /api/routes` (ListRoutes), corrections form `POST
  /routes/{id}/stops/{order}/times` (UpdateStopTimes — service exists,
  transport new); dashboards — `GET /dashboard?from&to` + `/api/\n  dashboard/{day,month,period}`; params — `GET /params`,
  `POST /params/{key}`, `GET|PUT /api/params[...]`; audit — `GET\n  /audit` + `/api/audit` (ListAudit: **service does not exist yet** —
  it is an operations.md op whose store query must be written; entity
  filtering + window at query level); export — `GET /history/export` +
  `/api/export` (CSV stream, UTF-8 BOM first three bytes, RFC 4180:
n  route date, driver, stop order, address, arrival, departure, stop
  minutes, route total minutes, route cost).
- **loginRedirect flip for T9:** manager/admin → `/dashboard`
  (httpapi/auth.go — the single flip point; drivers stay on
  /routes/today).
- Chart.js is vendored (`/static/chart.umd.js`, 4.4.9) — pages load it
  and receive AGGREGATE series only (architecture: charts never sum
  rows; the SQL series is the data).
- Corrections (UpdateStopTimes) are manager/admin — the history route\n  detail shows the form for those roles; drivers read-only.
- The goldens for dashboard assertions: day/month/period 161 on\n  2026-06-15 (fresh `--seed`); route detail shows 75/15.625/NULL-cost\n  (no distance) for route A.
- httpapi tests see app-suite leftovers (fuel_price_brl = 6.19 at the\n  end of the app suite) — reset params via the service (`svc` is\n  package-level in httpapi_test.go, adminID available) when asserting\n  cost-dependent values.
- CSV test: parse with encoding/csv; assert BOM bytes 0xEF 0xBB 0xBF
  first; assert header + the route A row's stop minutes and total.
- Guard reminder: SQL only in store; the CSV/audit endpoints are\n  handlers + store queries.

## Open risks (subset relevant to T9)

- ListAudit needs a new store query + service (matrix: admin only) —\n  operations.md defines the op; implement it per the operation-first\n  contract and record the shape.
- Chart tab switching with htmx: the screens.md state table allows\n  simple links/anchors per tab — no SPA; one page, three sections.\n- Excel BOM: write the BOM before any header byte; test asserts the\n  first three bytes literally.

## Out of scope

No route builder/tracker changes (done in T8). No new business rules.\nNo SPA. T10 (document final review) and T11 (name/campaign/demo) come\nafter.
