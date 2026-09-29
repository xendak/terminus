# Handover — mvp

## State

T7 landed (session 9): the HTTP shell exists — session middleware,
template engine with labels maps, sentinel error mapping, flash
cookies, Login + Home + Directories screens (drivers/managers/locations
with list/create/edit states) and their JSON mirrors. Assets vendored
(htmx 2.0.6, Chart.js 4.4.9, Pico 2.1.1) and embedded. main.go wires
store → app → httpapi. All guards hold (no SQL in httpapi, no URLs in
templates, no cdn refs in backend/). Cluster up, test DB migrated.

## Next

**T8. Route builder + tracker — a full day runs from the UI contract**
(`plans/mvp/plan.md`).

- Step 0: baseline green (below); T7 shell compiles and login works.
- Plan (read): `docs/spec/screens.md` (Route builder §2, Route tracker
  §3 — already read this conversation; re-check on disk),
  `docs/spec/operations.md` (route composition + time recording
  transports — in context).
- Do: htmx flows for the two screens' state tables: builder (empty →
  adding → reordering → invalid → saved), tracker (not started →
  active → arrived → departed → completed → closed → error). "Mark
  arrival/departure now" buttons; manual timestamp entry for null
  fields; no stopwatch UI on stop 1 (RN01 visible in the product).
  JSON mirrors for every operation involved.
- Verify: build/vet/test green; curl walkthrough creates route A via
  the builder endpoints, records every arrival/departure via the
  tracker endpoints, closes, and `GET /api/routes/{id}` shows total
  75 minutes.
- Stop-when: W8 green in this session, committed, pushed, handover
  rewritten.

## Baseline commands

```
git status                                            # clean tree
nix develop -c bash -c 'scripts/testdb.sh'            # fresh migrated test DB
nix develop -c bash -c 'eval "$(scripts/db-up.sh)" && cd backend && go build ./... && go vet ./... && go test -count=1 ./internal/...'
```

## Facts this task needs

- **The T7 flip point:** `loginRedirect(role)` in
  `internal/httpapi/auth.go` maps every role to "/" today. T8 owns the
  driver side: point driver → the tracker route (e.g. /routes/today)
  when the tracker exists; T9 flips manager/admin → /dashboard.
- Transports for the involved operations (operations.md): builder —
  `POST /routes` (+`POST /api/routes`), `POST /routes/{id}/stops`,
  `POST /routes/{id}/stops/{order}/remove`,
  `POST /routes/{id}/stops/{order}/move`; tracker —
  `POST /routes/{id}/start`, `POST /routes/{id}/close`,
  `POST /routes/{id}/stops/{order}/arrive|depart`,
  `POST /routes/{id}/distance` (+ the /api mirrors).
- **GetRoute output** already carries everything the tracker renders:
  stops with counted flag (stop 1 shows NO stopwatch), stop_seconds,
  totals, journey percent, cost (`svc.GetRoute` + RouteView).
- Driver scoping is enforced in the service — the tracker page for a
  driver actor fetches their own route; GetRoute as driver on another's
  route is 403. The tracker needs "my route for today": there is no
  ListRoutes-by-today operation — ListRoutes with from=to=today + the
  driver's forced scope is the query; pick the first row.
- htmx partials: the state tables say what each state shows — the
  partial id contract lives in the screens' state tables; swap
  fragments server-rendered from the same templates.
- Times: manual entry parses "2006-01-02 15:04" (server clock on
  record; the injected clock drives defaults — `svc.Now`). Record*
  take `at`; htmx "now" posts without the field (service default).
- The curl walkthrough must end asserting `GET /api/routes/{id}`\n  total_stopped_minutes = 75 (route A shape: 4 stops, 15/10/50 min,
  stop 1 contributes 0).
- httpapi tests see the golden users after the app suite — the
  walkthrough page flows can reuse golden drivers (driver-a owns no
  route on fresh dates; create fixtures as needed).

## Open risks (subset relevant to T8)

- Tracker as driver: the driver actor cannot ListDrivers/ListLocations
  (matrix) — the tracker page must not need them; build the stop list
  from GetRoute only.
- Manual time strings: parse errors → FieldError → the screens.md
  "error" state inline; keep formats to one (see above), document in
  labels.

## Out of scope

No dashboard/history/params/audit/export screens (T9). No CSV. No
optimization. No new services — if the screens demand one (e.g. a
today-route lookup), that is a spec-first addition recorded like
migration 0002 was.
