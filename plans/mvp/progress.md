# Progress — mvp

Newest entry on top. Append-only.

## Session 13 — T11: branding, campaign, final acceptance (2026-09-29)

Step 0: tree clean at 318daff (main, ahead of origin by 23, not pushed);
`frontend/` committed by its owner; Go server on 127.0.0.1:8080 and Next
dev on :3210 up; `scripts/dev-seed.sh` → `105 closed, 1 active, 1 draft
routes`.

**What landed:**

- Branding: the legacy htmx layout's `<title>` and wordmark, the htmx home
  label, and prose/comments (flake.nix description, .env.example,
  `backend/web/embed.go`, `cmd/server/main.go` header + startup log line
  "terminus listening on …", `scripts/lib.sh`, `scripts/db-init.sh` conf
  marker for new clusters) now say Terminus; the httpapi test that asserts
  the app name follows. `docs/spec/product.md` header rewritten (was
  "StopTime (working title)") and its deliverables row points at the
  campaign. Left untouched on purpose: identifiers (module, DBs, cookie,
  `@stoptime.dev`, `stoptime-dev`, `UpdateStopTimes`), the applied
  migration `0001_init.sql` header comment (append-only), old progress
  entries. The frontend already said Terminus.
- Campaign: `docs/campanha.md` (pt-BR) — name rationale, tagline "Cada
  minuto parado tem um endereço.", pitch, three screenshots + the `/sobre`
  page, Instagram and LinkedIn post texts, how the shots were made.
  Screenshots in `docs/campanha/` (Playwright, pt-BR, America/Sao_Paulo,
  light, reduced motion, clean demo seed): `painel-gestor.png` (63 KB,
  manager, "Este mês", Por dia), `motorista-hoje-mobile.png` (145 KB,
  driver-a /hoje at 375 px @2x, active route), `roteiro-fechado.png`
  (118 KB, closed route of driver C with departure "não conta" and cost),
  `sobre.png` (106 KB). Each one was looked at: no half-faded animation,
  no skeletons.
- README final: pt-BR summary on top, what Terminus is, stack with
  versions, layout incl. `frontend/` and `docs/campanha*`, run path (nix
  and Ubuntu), URLs incl. `/sobre`, demo logins, Tests section (Go parity
  + `pnpm lint && pnpm build && pnpm test:e2e`, reseed afterwards).

**What was discovered (must not rediscover):** Next 16 dev writes into
`.next/dev`, so `pnpm build` runs next to a live `next dev` without
conflict (the dev server answered 200 right after the build and the e2e suite ran on it). The Go
server had to be restarted (by exact PID) to serve the new template.

**Acceptance — `tp.md` §10, literal output, this session:**

(1) Departure point never counts stopped time. Route of Carla Camargo,
2026-09-28, `GET /api/routes/cdf8118a-…` via :3210 as manager:

```
"stop_order": 1, "counted": false, "label": "CD Terminus União",
"arrival_at": null, "departure_at": "2026-09-28T08:02:00-03:00", "stop_seconds": 0
"stop_order": 2, "counted": true, ... "stop_seconds": 1140
... total_stopped_seconds 4140 (= 1140+360+420+420+1800), total_stopped_minutes 69
```

psql on the dev DB (`stop_seconds` is a generated column, `WHEN
stop_order = 1 THEN 0`), then a forced arrival on stop 1 inside a rolled
back transaction:

```
 departure_stops | with_arrival | departure_seconds_total
             107 |            3 |                       0
 stop_order |       arrival_at       |      departure_at      | stop_seconds
          1 | 2026-09-28 07:17:00-03 | 2026-09-28 08:02:00-03 |            0
ROLLBACK
```

(2) Dashboard presents day, month and period cuts
(`from=2026-08-01&to=2026-09-29`, manager):

```
GET /api/dashboard/day    → {"series":[{"date":"2026-08-04","total_stopped_minutes":180,"journey_percent":"12.500"}, …]}  HTTP 200 0.004242s
GET /api/dashboard/month  → {"series":[{"month":"2026-08","total_stopped_minutes":3163,"journey_percent":"12.433"},{"month":"2026-09","total_stopped_minutes":2644,"journey_percent":"11.017"}],"standard_journey_hours":"8.0000"}  HTTP 200 0.003976s
GET /api/dashboard/period → {"standard_journey_hours":"8.0000","total_stopped_minutes":5807,"journey_percent":"11.746","routes_count":103,"by_driver":[…3 drivers…]}  HTTP 200 0.004141s
```

The Next.js `/painel` renders the three tabs "Por dia / Por mês /
Período" (screenshot `docs/campanha/painel-gestor.png`; e2e
dashboard.spec.ts green below).

(3) Every stopped time shown is tied to an address and a date/time:

```
 counted_stops_with_time | without_address_or_timestamps
                     471 |                             0
GET /api/export?from=2026-09-28&to=2026-09-28 (first rows):
Data,Motorista,Ordem,Endereço,Chegada,Saída,Minutos parados,Total do roteiro (min),Custo do roteiro (R$)
28/09/2026,Bianca Batista,2,"Rua Curitiba, 2010 - Lourdes, Belo Horizonte - MG",28/09/2026 07:56,28/09/2026 08:02,6,22,16.18
```

(4) Cost and journey params change without code change (admin, via
:3210, same route):

```
-- before
route cost_brl = 34.29 | journey_percent = 14.375 | distance_km = 56.30
PUT /api/params/fuel_price_brl {"value":"7.00"}          HTTP 200
route cost_brl = 39.41 | journey_percent = 14.375 | distance_km = 56.30
PUT /api/params/fuel_price_brl {"value":"6.09"}          HTTP 200
route cost_brl = 34.29 | journey_percent = 14.375 | distance_km = 56.30
PUT /api/params/standard_journey_hours {"value":"10"}    HTTP 200
route cost_brl = 34.29 | journey_percent = 11.500 | distance_km = 56.30
PUT /api/params/standard_journey_hours {"value":"8"}     HTTP 200
route cost_brl = 34.29 | journey_percent = 14.375 | distance_km = 56.30
```

(56.3 km / 10 km/l × 7.00 = 39.41; 69 min / 600 min = 11.5 %.)

**Verify (literal, this session):**

```
$ scripts/testdb.sh && cd backend && go build ./... && go vet ./... && go test -count=1 -p 1 ./internal/...
testdb: stoptime_test ready at postgresql:///stoptime_test?host=/home/vitor/terminus/.pg/sock&port=5543
ok  	stoptime/internal/app	9.520s
ok  	stoptime/internal/domain	0.002s
ok  	stoptime/internal/httpapi	3.588s
ok  	stoptime/internal/store	0.003s
exit=0
$ guard greps: httpapi SQL → only routes_test.go:116 jsonCall(… "DELETE", "/api/routes/…") (HTTP method, false positive); domain → empty; templates https:// → empty
$ curl -s http://127.0.0.1:8080/login | grep -o "<title>…</title>\|<strong>…</strong>"
<title>Terminus</title>
<strong>Terminus</strong>
$ cd frontend && pnpm lint      → $ eslint   lint exit=0
$ pnpm build                    → ✓ Compiled successfully … ✓ Generating static pages (15/15)   build exit=0
$ pnpm test:e2e                 → Running 15 tests using 1 worker … 15 passed (15.5s)   e2e exit=0
$ scripts/dev-seed.sh           → dev-seed: 105 closed, 1 active, 1 draft routes
```

**Human review sign-off of the especificação (W10): still PENDING.** It
needs the user and at least one teammate and cannot happen inside an agent
session; it is not recorded as done. Everything else in the plan is in
git.

Stage closes with commit `mvp: T11 branding, campaign, final acceptance
(plans/mvp)` and local tag `plans/mvp/T11`. Not pushed (team-lead
instruction).

**Next:** no card left. Collect the human sign-off and record it verbatim
here; push when the user says so.

**How the session ended:** card finished (sign-off pending outside the
session). No early stop, no compaction.

## Session 12 — T10: especificação reconciled + diagrams re-rendered (2026-09-29)

Context: between T9 and this session, commits d900957..3a1a366 landed
outside a card (CurrentUser `GET /api/auth/me` + PATCH JSON for
UpdateStopTimes; demo seed + `scripts/dev-seed.sh`; README non-nix path;
RN04 per-route aggregate base 11.181%, plain JSON dates, unwrapped period
JSON). The user also decided (2026-09-29) that the client is a Next.js app
in `frontend/`, being built in parallel by another agent — not touched
here — and that the product is named Terminus.

**What landed:**

- `docs/especificacao.md` reconciled with the implementation. Product name
  Terminus (StopTime kept only as identifiers). UC01 gains `CurrentUser` and
  the generic-failure/expired-session alternatives; UC06 moves StartRoute into
  the flow (precondition `draft` or `active`); UC07 blank-keeps-value; UC08
  precondition widened (CloseRoute accepts draft/active), null-cost
  alternative, `ReopenRoute`; UC09 journey percent over `routes_count` ×
  8 h, empty-window alternative, the actual RNF03 test (36 synthetic months,
  12-month window), operations named; UC10 default window; UC12 ordering +
  limit. Section 6 says boundaries are screens.md contracts realized by the
  Next.js client (htmx legacy). ER relationships fixed to the 8 real FKs
  (the fake `ROUTE }o--|| PARAMETER` removed; `location.created_by` and
  `parameter.updated_by` added), comments in pt-BR, 0001/0002 explained.
  Section 9: five seed users, driver B km/l override, placeholder addresses,
  period 11,181%, min_stop 6 → B 36 min, demo seed. Matrix: UC01
  CurrentUser, UC08 ReopenRoute, UC09 RN03, UC11 RF11/RN03; RNF rows
  updated; RN coverage sentence. New section 11 (architecture: Next.js
  client + rewrite proxy, Go layers, legacy htmx, JSON conventions incl.
  YYYY-MM-DD dates and the error body, environment with optional nix,
  Node 24 + pnpm).
- Diagrams: casos-de-uso (system boundary Terminus); robustez UC05
  (CreateRoute writes route_stop/reads location; composition writes
  audit_log); UC06 (StartRoute → route); UC07 (RN02 control); UC09 (RN07
  cost control was wrong — dashboard carries no cost — now RN03
  min_stop_minutes). All six re-rendered with PlantUML 1.2026.8
  (classes-conceituais.svg changed from the version bump alone).
- Spec set: architecture.md (two-process diagram, monorepo layout with
  `frontend/` and `backend/web/`, JSON-first transports, toolchain paths,
  Node 24 + pnpm, demo data, `-p 1` baseline, dependency budget split
  backend/frontend incl. the T4 google/uuid decision, no-Node-outside-frontend
  rule); screens.md intro; product.md out-of-scope bullet superseded;
  data-model.md ER relationships; use-cases.md traceability synced with the
  especificação matrix, UC09 text. AGENTS.md "What we're building" and
  out-of-scope lines. README: name, stack, layout, frontend run line.
- notes.md: decisions (Terminus name, Next.js frontend amending "no
  node/npm", nix optional) and "T10 session" facts.

**What was discovered (must not rediscover):** the ER's fake PARAMETER
relationship and two missing FKs; the dashboard has no cost (RN07 in UC09
was wrong); CloseRoute accepts drafts; PlantUML jar path and that a version
bump changes every SVG; the httpapi SQL guard's `DELETE /api/...` false
positive. All in `notes.md` ("T10 session").

**Verify (literal, this session):**

```
$ java -jar ~/.local/share/plantuml/plantuml.jar -tsvg docs/especificacao/diagrams/*.puml
  + error scan + set diff + traceability walk + ER-vs-migrations check
render exit=0
error-scan matches: 0
svg set == puml set (6/6)
traceability ids missing from section 10: none
UC01 operations: Login, Logout, CurrentUser
UC02 operations: CreateDriver, ListDrivers, UpdateDriver
UC03 operations: CreateManager, ListManagers
UC04 operations: CreateLocation, UpdateLocation, ListLocations
UC05 operations: CreateRoute, AddStop, RemoveStop, ReorderStops
UC06 operations: StartRoute, RecordArrival, RecordDeparture, GetRoute
UC07 operations: UpdateStopTimes, ListAudit
UC08 operations: SetRouteDistance, CloseRoute, ReopenRoute
UC09 operations: GetDashboardByDay, GetDashboardByMonth, GetDashboardByPeriod
UC10 operations: ListRoutes, GetRoute, ExportPeriodCSV
UC11 operations: GetParams, UpdateParam
UC12 operations: ListAudit
UC operations not defined in operations.md: none
UC count: 12
docs/especificacao.md ER matches migrations 0001+0002: True (7 tables, 8 FKs)
docs/spec/data-model.md ER matches migrations 0001+0002: True (7 tables, 8 FKs)
```

Also: `go version` → go1.26.8; `cd backend && go build ./... && go vet
./...` → exit 0 (no Go change this session). The Go test suite was NOT run:
docs-only card, and other agents share `stoptime_test` concurrently on this
machine (risk of cross-suite corruption, notes T8).

**Human review sign-off: PENDING.** The card requires the user and at least
one teammate to review `docs/especificacao.md`. That review has not happened
in this session and is not recorded as done. T11 must collect it and record
it verbatim here.

Stage closes with commit `mvp: T10 especificacao reconciled + diagrams
re-rendered (plans/mvp)`. Not pushed, not tagged (team-lead instruction).

**Next:** T11 (name/campaign/demo/final acceptance) per `handover.md`.

**How the session ended:** card finished except the human sign-off, which
cannot happen inside the session (recorded as pending). No early stop, no
compaction.

## Session 11 — T9: dashboard, history, params, audit, export (2026-09-29)

**What landed:** the client's screens. Dashboard page (three cuts over
one window, preset chips, Chart.js drawing the aggregate series,
per-driver ranking; /api/dashboard/{day,month,period} mirrors).
History page (filters incl. driver select for manager/admin, rows with
links to the detail; /api/routes list mirror) plus the audited
corrections form (UpdateStopTimes transport) on the route detail
fragment. Parameters screen (inline edit + /api/params GET/PUT).
Audit screen + /api/audit (the missing ListAudit operation: store
query, admin-only service, entity/date filters). CSV export
(/history/export + /api/export): one SQL query per window, UTF-8 BOM
first, RFC 4180 CRLF, display-zone timestamps, driver scoping enforced.
loginRedirect complete per screens.md §1. Tests: golden dashboard
series (day 161/month 161/period 161+33.542+3), history golden rows +
driver scoping, params API + forms + 403s, corrections → audit trail
assertions, CSV round-trip parse (BOM bytes, header, 12 golden rows,
address+minutes+total per row, driver-scoped export).

**What was discovered (must not rediscover):**

- UpdateParam echoed its input — services re-read rows after every
  mutating update (the T4 CloseRoute lesson is now an invariant).
- Full-year dashboard assertions see other suites' fixtures — narrow
  windows or presence, never broad exact counts.
- Template eq needs plain strings (pointer structs are service
  inputs, page data renders strings).
- ListAudit was a genuine spec-vs-plan gap (operations.md listed it;
  no card ever built it) — built this session per the contract.

All in `notes.md` ("T9 session").

**Verify (literal, this session):**

- `W9-VERIFY-GREEN`: fresh testdb + `go build ./... && go vet ./... &&
  go test -count=1 -p 1 ./internal/...` → `ok app 3.662s`,
  `ok domain 0.003s`, `ok httpapi 1.564s` (incl. the CSV test).
- Acceptance criteria 2–4 (tp.md §10), live server + golden seed:
  - 2: day `"total_stopped_minutes":161`; month `{"2026-06",161}`;
    period 161 / 33.542% / 3 routes + by_driver ranking.
  - 3: route A detail — every stop with address + arrival/departure
    timestamps, stop 1 0s counted=false, total 75 min at 4500s.
  - 4: route (100 km, driver-B km/l 12.50) cost 48.72 → after
    `POST /params/fuel_price_brl 7.00` → 56.00 — no code change.
  - CSV: BOM bytes `efbbbf`; "Rua Peru, 55" row present.
- Guards: templates clean, httpapi SELECT-free, domain pure, no cdn.

Stage closes with commit `mvp: T9 dashboard/history/params/audit/export
(plans/mvp)`, tag `plans/mvp/T9`, pushed to `origin/main`.

**Next:** T10 (part-1 specification — final review and render) per
`handover.md`.

**How the session ended:** card finished — W9/T9 complete and committed,
no early stop, no compaction. Cluster up, test DB migrated fresh.


## Session 10 — T8: route builder + tracker (2026-09-29)

**What landed:** the two screens as htmx state machines (screens.md
§2/§3). Builder: /routes/new composes the ordered stop list client-side
(hidden inputs + ~20 lines of inline JS), POST /routes → detail page;
the RN05 invalid state re-renders inline with a link to the existing
route. Route detail = one template with state branches + the shared
#route-body fragment: draft (add/remove/move/start), active tracker
(arrive/depart now, manual datetime-local entry, no stopwatch on stop 1
— RN01 visible), completed (distance + close), closed (summary with
totals/percent/cost, admin reopen). /routes/today lands drivers on
their route or the empty state; loginRedirect flipped driver →
/routes/today. JSON mirrors for every involved operation (composition
mutations answer with the fresh GetRoute view; record actions with the
stop). Display-zone helpers (fmtTime/fmtDate/mins, tzdata). Tests:
API full-day flow (create → edit → start → record 15/10/50 → close →
GET shows 75/15.625/60.90; RN05 409; double-record 422; driver scoping
403s), page flows (builder form, fragments, manual entry, RN01 button
absence, conflict state with link), today-page redirect.

**What was discovered (must not rediscover):**

- `go test ./...` runs PACKAGES in parallel on the shared test DB —
  parity is now `go test -count=1 -p 1 ./internal/...` (TestParams
  caught the cross-suite audit row).
- ParseFS names templates by file base — the page root must be
  "layout.html" or every render is an empty template.
- Optional-body JSON handlers need the ContentLength guard (bodiless
  close → 400 malformed JSON).
- AddStop without position appends at the end; negative HTML
  assertions must scope to the stop-list markup (the location picker
  echoes all labels); loginSession vs loginSessionAs.

All in `notes.md` ("T8 session").

**Verify (literal, this session):**

- `W8-VERIFY-GREEN`: fresh testdb + `go build ./... && go vet ./... &&
  go test -count=1 -p 1 ./internal/...` → `ok app 3.666s`,
  `ok domain 0.001s`, `ok httpapi 1.066s`.
- Curl walkthrough (live server, seeded DB, golden fixtures):
  login 303; builder form POST → 303 to the route; start fragment
  "active"; 8 tracker posts (manual times) all 200; distance + close
  200; `GET /api/routes/{id}` → `"total_stopped_minutes":75`,
  `"journey_percent":"15.625"`, `"estimated_cost_brl":"60.90"`,
  `"status":"closed"` — route A end-to-end from the UI contract.
- Guards: templates no URLs; `SELECT` in httpapi empty (route-pattern
  strings contain HTTP verbs, not SQL); domain pure.

Stage closes with commit `mvp: T8 builder + tracker screens (plans/mvp)`,
tag `plans/mvp/T8`, pushed to `origin/main`.

**Next:** T9 (dashboard, history, params, audit, export) per
`handover.md`.

**How the session ended:** card finished — W8/T8 complete and committed,
no early stop, no compaction. Cluster up, test DB migrated fresh.


## Session 9 — T7: HTTP shell + adapters + directories (2026-09-29)

**What landed:** vendored assets (htmx 2.0.6, Chart.js 4.4.9, Pico
2.1.1) embedded via package `stoptime/web`. `internal/httpapi` —
router (method patterns, static from embed FS), session middleware
(decode once → Actor in context; requirePage bounces to /login,
requireAPI 401s), template engine (layout + per-page content blocks,
labels maps per screen), sentinel→HTTP mapping (operations.md table:
400/401/403/404/409/422), signed one-shot flash cookie, inline form
errors for the invalid state, recoverer (log server-side, generic 500).
Screens: Login (entry/invalid/success-by-role — home stand-in until
T8/T9 flip the map), Home, Directories (drivers list+create+edit,
managers list+create, locations list+create+edit) per screens.md §6,
plus JSON mirrors: /api/auth/login+logout, /api/drivers
(GET/POST/PATCH), /api/managers, /api/locations. cmd/server wires
store→app→httpapi with env config (SESSION_KEY dev default + warning).
store.User/Driver/Location gained snake_case json tags (PasswordHash
`json:"-"` — never serialized; tested). Tests: login page flow + bad
creds, unauthenticated bounce, directory pages (golden driver listed,
create→flash→invalid-inline→edit→rename), driver-role 403s on pages
and API, API flows (201 create, 409 duplicate, PATCH partial, no
password leak), static assets, healthz.

**What was discovered (must not rediscover):**

- loginRedirect is the single flip point for screens.md's role
  redirects — home stand-in now, /routes/today + /dashboard in T8/T9.
- Output structs carry json tags; inputs stay untagged with adapter
  DTOs (absent vs empty needs pointers anyway).
- Edit forms source rows from the directory lists — operations.md
  defines no GetDriver/GetLocation op.
- html/template: ParseFS(layout+page) per page set; page's
  {{define "content"}} overrides per set.

All in `notes.md` ("T7 session").

**Verify (literal, this session):**

- `W7-VERIFY-GREEN`: fresh testdb + `go build ./... && go vet ./... &&
  go test -count=1 ./internal/...` → `ok app 3.601s`,
  `ok domain 0.001s`, `ok httpapi 0.588s`.
- Curl walkthrough (live server on :8099, seeded test DB): healthz
  `ok`; POST /login → 303; POST /drivers (Curl Driver) → 303;
  GET /drivers lists Curl Driver + golden driver + flash; /api/drivers
  contains it with NO password_hash; anonymous GET /drivers → 303.
- Guards all empty: templates URLs, httpapi SQL (SELECT/INSERT/UPDATE/
  DELETE), domain purity, https://cdn in backend/.

Stage closes with commit `mvp: T7 HTTP shell + directories (plans/mvp)`,
tag `plans/mvp/T7`, pushed to `origin/main`.

**Next:** T8 (route builder + tracker — htmx state machines, JSON
mirrors, curl walkthrough driving route A to 75 minutes) per
`handover.md`.

**How the session ended:** card finished — W7/T7 complete and committed,
no early stop, no compaction. Cluster up, test DB migrated fresh.


## Session 8 — T6: auth + role matrix (2026-09-28)

**What landed:** `app/session.go` — the HMAC-SHA256 session (cookie
`st_session`, 12h lifetime, pure encode/decode folding every failure
into ErrUnauthenticated) plus context carriers for the T7 middleware.
`app/auth.go` — Login (bcrypt verify, active check, cookie value
issuance; no user-enumeration signal) and Logout. `app/permissions.go`
— the operations.md role matrix with three shapes (plain, own-route,
scoped) and the `allow`/`ownRoute`/`forceDriverScope` helpers.
Enforcement woven through every service (Actor threading completed;
zero-session → ErrUnauthenticated, wrong role → ErrForbidden,
driver-own-route enforced after load, scoped reads force the driver's
id into the query). ErrUnauthenticated/ErrForbidden added to the error
model. `store.UserByEmail` for login. Tests: auth suite (login success/
failures/logout/cookie crypto: tamper, expiry, wrong key, garbage) and
the table-driven matrix test (zmatrix_test.go) walking all 28
operations × admin/manager/driver/anonymous with real calls and fresh
fixtures per cell, printing every cell; TestOwnRouteBoundary for the
not-own denial side; reads tests upgraded to prove enforced scoping
(driver actor, no filter passed).

**What was discovered (must not rediscover):**

- Enforcement immediately caught a T4 test bug: reopening as manager —
  spec says admin-only. Spec wins; test fixed.
- The matrix test's expected table must be transcribed independently
  from operations.md (never read the enforcement's map) — two tables
  agreeing is the pin.
- Fresh driver per fixture means route dates may repeat without RN05
  conflicts; a date generator that ran past day 31 broke date parsing —
  cycle within 1..28.
- zmatrix_test.go sorts LAST on purpose: its 2025-08 fixtures would
  otherwise reach the reads tests' exact-count full-year window.

All in `notes.md` ("T6 session").

**Verify (literal, this session):**

- Fresh `scripts/testdb.sh` + eval db-up, `go build ./... && go vet
  ./... && go test -count=1 ./internal/...` → `ok stoptime/internal/app
  3.671s`, `ok ...domain 0.002s`, exit 0 (`W6-VERIFY-GREEN`).
- Matrix run (`-run TestRoleMatrix -v`) prints every cell, e.g.
  `CreateManager ALLOW FORBID FORBID NOAUTH`,
  `ReopenRoute      ALLOW FORBID FORBID NOAUTH`,
  `StartRoute       ALLOW ALLOW ALLOW  NOAUTH`,
  `GetDashboardByDay ALLOW ALLOW ALLOW NOAUTH` — all 28 rows green,
  anonymous column NOAUTH everywhere except Login (the entry point).
- Guards: domain purity grep empty; `grep net/http internal/app/`
  empty (session encoding stayed pure).

Stage closes with commit `mvp: T6 auth + role matrix (plans/mvp)`, tag
`plans/mvp/T6`, pushed to `origin/main`.

**Next:** T7 (HTTP shell + adapters + directories screens) per
`handover.md`.

**How the session ended:** card finished — W6/T6 complete and committed,
no early stop, no compaction. Cluster up, test DB migrated fresh.


## Session 7 — T5: reads, SQL aggregation, cost, the 3-second rule (2026-09-28)

**What landed:** `backend/internal/store/reads.go` — the read services'
SQL: RouteWithTotals (route + driver name + total seconds/minutes,
journey percent, estimated cost in one query), RouteStopDetails (stop +
location join, counted flag), ListRoutes (window + driver/status filters,
aggregates per row), DashboardByDay/Month (series with data only),
DashboardByPeriod (GROUPING SETS: grand + per-driver rows in one query).
Parameters pivoted per query, min_stop_minutes honored in the SUM CASE,
percent rounded once to 3, cost once to 2, no distance → NULL cost.
`backend/internal/app/reads.go` — GetRoute, ListRoutes (current-month
defaults), GetDashboardByDay/Month/Period with input validation and
query-level driver scoping. Tests: golden pins (GetRoute detail incl.
RN01 stop 1 counted=false/0s + domain oracle agreement; ListRoutes
41/45/75 with percent strings; day/month/period 161; period 33.542%
+ by_driver), driver filter, validation, and the perf test: 36 months
of synthetic data (Go helper, one INSERT…SELECT per table, ~4700 routes),
12-month dashboards timed, EXPLAIN ANALYZE printed and asserted on
`route_route_date_idx` via a one-month window. TestMain now applies
db/seed/golden.sql (comment-stripped, ';' split) after truncation.

**What was discovered (must not rediscover):**

- Journey percent scale = 3 (round once, matches golden 15.625); period
  percent = uniform RN04 formula over one standard day (interpretation
  recorded in notes — golden period can't distinguish alternatives).
- GROUPING() identifies the period grand row; max(u.name) does NOT
  return NULL for it (name-max is a name) — cost one red iteration.
  Pivoted params must be in every grouping set.
- Full-coverage windows legitimately seq-scan; the index assertion runs
  on a selective (1/36) window while the 3s bound covers the full
  12-month aggregation. Measured: day 4.6ms, month 4.7ms, period 7.1ms.
- gen_year as a Go helper (not a seed file): avoids the collation
  ordering trap and keeps perf data out of testdb --seed.
- pgx Exec takes one statement: golden.sql loads via comment-strip +
  ';' split; seed files must keep ';' only as terminators (a comment
  semicolon broke attempt one).
- reads_test.go runs BEFORE service_test.go (alphabetical) — read tests
  see only golden data; suites disjoint by date windows (2026-06 /
  2026-07 / 2022–2024).
- generate_series(date, date, interval) returns timestamptz: cast
  g::date before date arithmetic (interval % int does not exist).

All in `notes.md` ("T5 session").

**Verify (literal, this session):**

- Fresh `scripts/testdb.sh` + eval db-up, then `go build ./... && go vet
  ./... && go test -count=1 ./internal/...` → `ok stoptime/internal/app
  0.988s`, `ok ...domain 0.003s`, exit 0 (`W5-VERIFY-GREEN`).
- Perf test log: `12-month dashboards: day(262 points) 4.560082ms,
  month(12 points) 4.69064ms, period(1572 routes) 7.143578ms`;
  one-month EXPLAIN ANALYZE shows `Bitmap Index Scan on
  route_route_date_idx` (Index Cond on the window) + hash join to
  route_stop; both plans printed in the test log.
- Purity guard on domain → empty.

Stage closes with commit `mvp: T5 reads + aggregation + cost + perf
(plans/mvp)`, tag `plans/mvp/T5`, pushed to `origin/main`.

**Next:** T6 (auth + role matrix) per `handover.md`.

**How the session ended:** card finished — W5/T5 complete and committed,
no early stop, no compaction. Cluster up, test DB migrated fresh.


## Session 6 — T4: store + write operations (2026-09-28)

**What landed:** `db/migrations/0002_audit_entity_id_text.sql` (spec bug:
parameter keys are text; data-model.md + especificação ER corrected in
the same commit). `backend/internal/store` — pgxpool with session
TimeZone=UTC, `DB` (pool-or-tx) interface, `WithTx` (mutation + audit row
commit together), repositories for users/drivers/managers, locations,
routes, stops (two-phase-park renumbering, MaxStops 999), params, audit;
constraint translation (named-unique sentinels + `ConstraintError`).
`backend/internal/app` — the 21 write-path services from the card with
plain input structs, `Actor` for created_by/updated_by/audit actor,
injected clock, domain validation calls, `FieldError` detail under
ErrValidation, sentinels per operations.md's error model. Integration
suite (app/service_test.go): full day for route A (composition edits
with id-stability, recording, RN01-verified generated column, domain
oracle cross-check 4500s/75min/15.625%, correction audit, close,
closed-route rejections, reopen), RN05 conflict, RN02 service + DB
backstop (constraint name pinned), params + text entity_id audit,
directories CRUD, distance rules.

**What was discovered (must not rediscover):**

- `audit_log.entity_id` had to become text — an audited `update_param`
  cannot write a parameter key into a uuid column (T2 spec bug, fixed
  spec-first + migration 0002, same commit).
- Unreferenced bind parameters are illegal: `$2` passed but absent from
  the SQL → 42P18 "could not determine data type".
- pgx returns timestamptz in the PROCESS timezone (time.Local), not the
  session's — assertions compare instants, never strings.
- numeric round-trips are scale-normalized (6.09 → "6.0900", 100 →
  "100.00"); services re-read rows instead of echoing inputs.
- `(NOT $1 OR active)` is the correct activeOnly filter; `$1 OR active`
  is inverted.
- google/uuid added to the dependency budget (pgx ships no uuid type);
  recorded in notes. pgx v5.11.0, x/crypto v0.57.0 (CreateDriver needs
  the hasher, so bcrypt landed here not T6).
- `go test` can report `(cached)` after a DB rebuild — use `-count=1`
  for a literal fresh run.

All in `notes.md` ("T4 session").

**Verify (literal, this session):**

- `scripts/testdb.sh` → `stoptime_test ready`, then
  `go build ./... && go vet ./... && go test -count=1 ./internal/...` →
  `ok 	stoptime/internal/app	0.718s`, `ok ...domain 0.003s`, exit 0
  (`W4-VERIFY-UNCACHED-GREEN`).
- Purity guard `grep -rn "time.Now\|sql\|http" backend/internal/domain/`
  → empty. SQL outside store: only `internal/app/service_test.go` (tests
  assert DB state directly — by design).

Stage closes with commit `mvp: T4 store + write operations (plans/mvp)`,
tag `plans/mvp/T4`, pushed to `origin/main`.

**Next:** T5 (reads, SQL aggregation, cost, 3-second rule) per
`handover.md`.

**How the session ended:** card finished — W4/T4 complete and committed,
no early stop, no compaction. Cluster up, test DB migrated fresh.


## Session 5 — T3: domain package, pure rules (2026-09-28)

**What landed:** `backend/internal/domain` — the business rules of
`business-rules.md` as pure functions over plain values: `StopSeconds`
(RN01+RN02), `WholeMinutes`, `RouteTotalSeconds` (RN03 honoring
min_stop_minutes/RF10), `JourneyPercent` (RN04), `EstimatedCost` +
`RoundMoney` (RN07), and validation (`ValidateStopTimes`,
`ValidateStopOrders`, `ValidateDistance`) with sentinel errors
(`ErrDepartureBeforeArrival`, `ErrInvalidStopOrder`, `ErrInvalidDistance`,
`ErrInvalidKmPerL`, `ErrInvalidParameter`). Types `Stop`/`Route` with
nil-able timestamps mirroring column NULLs. Tests in the external
`domain_test` package: golden pins (A/B/C = 75/41/45 min, day 161,
15.625%, B = 36 at min_stop_minutes 6) plus edge cases (midnight span,
open stop, floor-once vs sum-of-floors, 359s/360s threshold boundary,
cost exactness incl. repeating liters, all error paths). Written test-first:
red confirmed ("no non-test Go files"), then implemented green.

**What was discovered (must not rediscover):**

- big.Rat chosen for money/percent — the domain is the oracle against
  exact numeric columns; T4/T5 stores must feed it with string-scanned
  numerics, never float64.
- Repeating decimals pin as exact fractions (125/6), never truncated
  decimal literals.
- Comma-ok trap: `_, wantErr := m[k]` returns key-presence, not the
  map value — cost one red iteration; case tables carry wantErr now.

All in `notes.md` ("T3 session").

**Verify (literal, this session):**

- `go test ./internal/domain/ -v` → 27 `--- PASS` lines ending
  `ok  	stoptime/internal/domain	0.002s`.
- `go vet ./...` clean.
- Purity guard `grep -rn "time.Now\|sql\|http" backend/internal/domain/`
  → empty.
- Parity `go build ./... && go vet ./... && go test ./...` green
  (`?  stoptime/cmd/server	[no test files]` + `ok ...internal/domain`).

Stage closes with commit `mvp: T3 domain package - pure rules (plans/mvp)`,
tag `plans/mvp/T3`, pushed to `origin/main`.

**Next:** T4 (store + write operations) per `handover.md`.

**How the session ended:** card finished — W3/T3 complete and committed,
no early stop, no compaction. Cluster up; test DB still golden-seeded from T2.


## Session 4 — T2: schema migration + golden seed (2026-09-28)

**What landed:** `db/migrations/0001_init.sql` — every table from
`data-model.md` (app_user + lower(email) unique index, driver_profile,
location, route with RN05 unique, route_stop with the RN01/RN02 generated
`stop_seconds` column, parameter, audit_log) and its indexes. `db/seed/
golden.sql` — 5 demo users (admin, manager, three drivers — RN05), driver
profiles (B overrides km_per_l), 12 locations (route A addresses verbatim
from tp.md §5, placeholders elsewhere), routes A/B/C on 2026-06-15, seeded
stop times with -03:00 offsets, parameters at defaults. `db/seed/
golden_check.sql` — DO-block assertions + printed proof.
Supporting: `scripts/testdb.sh` skips `*_check.sql` when seeding;
`db/seed/README.md` documents the convention; `docs/spec/data-model.md`
schema_migrations corrected to `name` (matches T1's migrate.sh); `plan.md`
T2 card corrected (three drivers, cd backend).

**What was discovered (must not rediscover):**

- The T2 card's "three demo users, one per role" was impossible against
  RN05 (one route per driver per date — routes A/B/C share one date): the
  golden seed needs three drivers. Card corrected; spec unchanged.
- Shell glob order follows the locale collation, not bytes: under
  en_US.UTF-8, `golden_check.sql` sorts BEFORE `golden.sql` (punctuation
  ignored). The first testdb.sh run applied the check to an empty DB and
  aborted before seeding. Fix: `*_check.sql` is a check, never a seed;
  migration zero-padding is what makes order deterministic.
- `\set ON_ERROR_STOP on` inside golden_check.sql makes plain `psql -f`
  exit nonzero on assertion failure (without it, SQL errors exit 0). Both
  directions verified: green run exits 0, empty-DB run exits 3.
- The spec's generated-column expression works verbatim on Postgres 18
  (implicit numeric→int cast, no `::int` needed).
- UUID last groups are 12 hex chars — first seed draft wrote 10-char groups
  and Postgres rejected them.

All in `notes.md` ("T2 session").

**Verify (literal, this session):**

- `scripts/testdb.sh --seed` → `testdb: seeded golden.sql` +
  `stoptime_test ready`.
- `psql "$TEST_DATABASE_URL" -f db/seed/golden_check.sql` → exit 0 (`GOLDEN-CHECK-EXIT=0`) printing
  `Roteiro A | 75, Roteiro B | 41, Roteiro C | 45`, `day_total_minutes 161`,
  `route_a_journey_percent 15.625`; DO block also asserts stop-1 = 0 and
  min_stop_minutes=6 → B = 36.
- `scripts/migrate.sh` applied 0001 to the dev DB too (12 CREATEs,
  `migrate [stoptime]: applied 0001_init.sql`).
- Parity: `cd backend && go build ./... && go vet ./... && go test ./...` →
  `[no test files]`, exit 0 — no regression.

Stage closes with commit `mvp: T2 schema + golden seed (plans/mvp)`, tag
`plans/mvp/T2`, pushed to `origin/main`.

**Next:** T3 (domain package, pure rules) per `handover.md`.

**How the session ended:** card finished — W2/T2 complete and committed, no
early stop, no compaction. Cluster left up, both DBs migrated, test DB seeded.


## Session 3 — T1: devshell, scaffold, database bring-up (2026-09-28)

**What landed:** GitHub remote `origin` linked (`git@github.com:xendak/terminus.git`,
SSH) and pushed — the repo is the graded hand-in, so every session-close
commit pushes from now on. `flake.nix` + `flake.lock` (go 1.26.7,
`postgresql_18` 18.6, gopls, plantuml; nixpkgs pinned at `83199d0d…`). Go
module `stoptime` at `backend/` with `cmd/server` serving `GET /healthz` →
`ok`. `scripts/`: `lib.sh`, `db-init.sh`, `db-up.sh`, `db-down.sh`,
`migrate.sh`, `testdb.sh`. `db/migrations/` and `db/seed/` with READMEs
(empty until T2). `.env.example`. Cluster facts: socket `.pg/sock`, port
5543.

**What was discovered (must not rediscover):**

- Nix flakes only evaluate git-tracked files — `git add` new files before
  `nix develop` (the flake itself was refused until staged).
- nix 2.34 passes `self` to `outputs`; the pattern must accept it.
- All parity commands need `cd backend` — the module lives at `backend/` per
  `architecture.md`; the root-cwd spellings in `AGENTS.md`/`docs/method.md`
  fail with "go.mod not found". Spec layout wins over card text; W1's verify
  line in `plan.md` was updated to the `cd backend` form.
- `db-up.sh` stdout must stay eval-clean: pg_ctl/createdb stdout used to leak
  into it and break `eval "$(scripts/db-up.sh)"`; fixed with the fd-3
  redirect. All tool output now goes to stderr.
- A backgrounded dev server must not inherit the session's output pipes
  (redirect to a logfile, `kill` + `wait`) or the shell hangs on pipe EOF.

All in `notes.md`.

**Verify (literal, this session):**

- `nix develop -c bash -c 'cd backend && go build ./... && go vet ./... &&
  psql --version'` → `psql (PostgreSQL) 18.6`, exit 0.
- `scripts/db-init.sh` → cluster initialized in `.pg/data`; then
  `eval "$(scripts/db-up.sh)"` → `stoptime` + `stoptime_test` created;
  `psql "$DATABASE_URL" -c 'select 1'` → one row, value `1`.
- `scripts/migrate.sh` → `migrate [stoptime]: db/migrations/ has no .sql
  files — nothing to do` (expected until T2); `scripts/testdb.sh` →
  `testdb: stoptime_test ready at postgresql:///stoptime_test?host=…`.
- Restart cycle: `scripts/db-down.sh` → `server stopped`; `db-up.sh` again
  → `select count(*) from schema_migrations` → `0` (tracking table
  persisted across restart).
- `curl -s localhost:8080/healthz` → `ok`; server log
  `stoptime listening on http://127.0.0.1:8080`.
- Full parity: `go test ./...` → `?   stoptime/cmd/server	[no test files]`,
  exit 0. `bash -n scripts/*.sh` clean.

Stage closes with commit `mvp: T1 devshell + scaffold + database bring-up
(plans/mvp)`, tag `plans/mvp/T1`, pushed to `origin/main`.

**Next:** T2 (schema migration + golden seed), per `handover.md`.

**How the session ended:** card finished — W1/T1 complete and committed, no
early stop, no compaction. Cluster left up for the T2 baseline.

## Session 2 — part-1 deliverable: pt-BR specification + diagrams (2026-09-28)

**What landed:** `docs/especificacao.md` — the part-1 specification document,
Portuguese prose with verbatim English identifiers — with its diagrams in
`docs/especificacao/diagrams/`: six `.puml` sources and six rendered `.svg`
(use case diagram, robustness diagrams for UC05/06/07/09, conceptual class
diagram); the crow's foot ER is inline Mermaid mirroring
`docs/spec/data-model.md`. Supporting updates: `product.md` (language rules +
deliverables map), `use-cases.md` (redefined as the English working source),
`plan.md` (W10 verify reworded, T10 card rewritten as the final-review/render
card, T1 devshell gains `plantuml`, risk table), `notes.md` (naming policy
decision, PlantUML facts, open question 1 resolved), `references.md`,
`README.md`, `AGENTS.md`, `plans/README.md`, this handover refresh.

**What was discovered (must not rediscover):**

- User clarification: the graded hand-in is the GitHub repo itself; the
  specification is its own markdown file, strictly Portuguese prose, with four
  diagram types — use case, class, crow's foot ER, robustness — in Mermaid and
  PlantUML.
- User correction, now the naming policy: identifiers in the deliverable are
  verbatim English from the codebase. A translated identifier ("momento",
  "USUARIO", "perfil") describes a system that does not exist. Recorded in
  `notes.md`.
- PlantUML robustness syntax, verified empirically against the rendered SVG
  primitives: plain diagrams with `boundary`/`control`/`entity` keywords
  render the proper robustness icons; a `robustness` directive does not exist
  (errors at line 2). Output names follow the `@startuml <name>` directive, not
  the filename. Derived-attribute notation (`/attr`) works. All in
  `notes.md`.

**Verify:** `nix run nixpkgs#plantuml -- -tsvg docs/especificacao/diagrams/*.puml`
exited 0 and produced six SVGs; error scan of all SVGs clean; expected labels
confirmed present in the renders (`UC12`, `RecordArrival`, `km_per_l`,
`/total_stopped_minutes`). Stage closes with commit
`mvp: part-1 deliverable - especificacao pt-BR + diagrams (plans/mvp)`.

**Next:** T1 (devshell + scaffold + database bring-up), unchanged, per
`handover.md`.

**How the session ended:** stage complete (part-1 deliverable + W10/T10
re-scope), no card started, no early stop, no compaction.

## Session 1, follow-up — spec wording fixup + handover refresh (2026-09-28)

Right after the bootstrap commit, a wording sweep found four occurrences of a
coined term ("journeyday") in the specs and notes, inconsistent with the
`journey`/`standard_journey_hours` naming used everywhere else. Fixed forward
in `a30c387` (per the no-amend rule), touching `docs/spec/product.md`,
`business-rules.md`, `use-cases.md`, and `plans/mvp/notes.md`. No content or
design change.

**Verify:** `grep -rn journeyday docs/ plans/` empty; `git log --oneline` shows
`a30c387` on top of `44fce62`; `git status --porcelain` empty. This commit
follows so that the handover's provenance is HEAD again.

**How the session ended:** bootstrap stage still complete; T1 remains the next
card.

## Session 1 — bootstrap: spec set + plan, git initialized (2026-09-28)

**What landed:** git repo initialized; the full spec set written from
`tp.md` (`docs/spec/`: product, architecture, business-rules, data-model,
operations, screens, use-cases); `docs/method.md` snapshot (one example adapted:
the parity check is `go build`/`go vet`/`go test`, not `zig build`); `AGENTS.md`
session protocol; `plans/README.md` scheduler; this folder (plan with cards
T1–T11, handover, notes, references); root `README.md`; `.gitignore`.

**What was discovered (must not rediscover):**

- This machine has no system psql, no running PostgreSQL, no Docker. Nix with
  flakes is available and nixpkgs provides postgresql 18.6. Consequence: the
  database story is a flake devshell plus repo-local cluster under `.pg/`,
  managed by scripts. Recorded in `notes.md`.
- The stack was decided with the user in this session: Go backend, SSR +
  htmx, vendored Chart.js, pgx, plain SQL migrations via psql. The user added
  five architectural caveats (handlers only parse and format; services return
  plain structs; aggregation in SQL not the UI; operation-first API contract;
  screens defined by use case and states) because a switch to a React
  frontend must stay possible without breaking the backend. All five are
  normative rules in `docs/spec/architecture.md`.
- Language split decided with the user: English docs and code; UI English by
  default with per-screen labels maps for a future translation layer; user
  input accepted in Portuguese; golden seed keeps the Portuguese addresses
  from `tp.md` section 5.
- `tp.md` section 5's routes A/B/C are the golden fixture; totals A=75, B=41,
  C=45 minutes; day total 161. Formalized with assertions in
  `docs/spec/business-rules.md`.

**Verify:** stage closes with the bootstrap commit
`mvp: bootstrap - spec set, plan, protocol (plans/mvp)`. Confirmed after the
ritual: `git status --porcelain` empty, `git log --oneline -1` shows the
message above.

**Next:** T1 (devshell + scaffold + database bring-up), per `handover.md`.

**How the session ended:** bootstrap stage complete, no card started, no early
stop, no compaction.
