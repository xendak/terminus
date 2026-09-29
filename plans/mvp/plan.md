# Plan — StopTime MVP

Goal (one line): deliver the `tp.md` MVP — a stopped-time monitoring web app on
Go + PostgreSQL with route composition, time recording, day/month/period
dashboard, history, CSV export, cost parameters, roles, and audit — with all
`tp.md` section 10 acceptance criteria green, the part-1 specification document
produced, and the extra-point name and campaign done.

Ground truth: `tp.md` (immutable). Spec set: `docs/spec/`. Where a card and a
spec disagree, the spec wins and the card is updated; where a spec and `tp.md`
disagree, `tp.md` wins and the spec is updated. Both are discoveries, recorded.

Bootstrap (this session, already landed): spec set written, plan written, git
initialized, first commit made. No code exists. Cards start at T1.

## Work items

Items are crossed off when they land in git, verified green in the session that
closed them.

- [x] W1. Devshell + repo scaffold + database bring-up (T1)
  *verify:* `nix develop -c bash -c 'cd backend && go build ./... && go vet ./... && psql --version'` green (the module lives at `backend/` per `architecture.md`, so parity commands `cd backend` — recorded in `notes.md`); `scripts/db-init.sh && scripts/db-up.sh` then `psql "$DATABASE_URL" -c 'select 1'` green; `curl -s localhost:8080/healthz` returns ok.
- [x] W2. Schema migration + golden seed (T2)
  *verify:* `scripts/testdb.sh --seed` then `psql "$TEST_DATABASE_URL" -f db/seed/golden_check.sql` exits 0 printing A=75, B=41, C=45.
- [x] W3. Domain package, pure rules (T3)
  *verify:* `go test ./internal/domain/` green on the golden fixture and edge cases.
- [x] W4. Store + write operations with audit (T4)
  *verify:* `go test ./internal/...` green against a fresh test DB (RN05 conflict test, audit row assertions).
- [x] W5. Read + aggregation + cost + performance (T5)
  *verify:* `go test ./internal/...` green; 12-month seed aggregation test under 3s with EXPLAIN showing index scans.
- [x] W6. Auth + role matrix (T6)
  *verify:* `go test ./internal/...` green incl. role matrix table test.
- [x] W7. HTTP shell + adapters + directories screens (T7)
  *verify:* build/vet/test green; scripted curl walkthrough of login + driver create; `grep -rn "https://" backend/web/templates/` empty.
- [x] W8. Route builder + tracker screens (T8)
  *verify:* handler tests green; curl walkthrough drives route A end-to-end and reads total 75 minutes from the API.
- [x] W9. Dashboard + history + params + export screens (T9)
  *verify:* build/vet/test green; golden dashboard series asserted; CSV parses in a Go test; acceptance criteria 2–4 demonstrated.
- [ ] W10. Part-1 specification document (T10)
  *verify:* `docs/especificacao.md` reconciled with the finished implementation, every RF/RN/UC cross-referenced, SVGs re-rendered from the current .puml sources and committed; human review sign-off recorded in progress.
- [ ] W11. Name + campaign + demo + final acceptance (T11)
  *verify:* acceptance checklist from `tp.md` section 10 all green in-session; tag `mvp/T11`; clean tree.

## Task cards

One card per session. All cards assume the session protocol in `AGENTS.md` was
run first (baseline green before any Do).

### T1. Devshell, scaffold, database bring-up — the repo can build and psql answers

- **Step 0 (identify):** confirm `nix flake` support (`nix --version`, flakes
  enabled — verified at bootstrap, recorded in `notes.md`); confirm ports the
  scripts claim are free; confirm `docs/spec/architecture.md` environment
  section exists and matches what you plan to build.
- **Plan (read):** `docs/spec/architecture.md`, `plans/mvp/notes.md`
  (environment facts).
- **Do:** `flake.nix` devshell (go 1.26 toolchain, `postgresql_18`, gopls,
  `plantuml` so diagram re-renders are reproducible). Go
  module `stoptime` with `cmd/server/main.go` serving `GET /healthz` → `ok`.
  `scripts/`: `db-init.sh` (initdb into `.pg/`, unix socket + chosen port),
  `db-up.sh` (start, create `stoptime` + `stoptime_test`), `db-down.sh`,
  `migrate.sh` (psql, filename order, one tx per file, `schema_migrations`
  tracking), `testdb.sh` (drop/create/migrate/seed the test DB). Empty
  `db/migrations/` and `db/seed/` with a README each. `.env.example`.
- **Verify:** all W1 commands, run in this session, from a clean tree.
- **Bisection:** devshell fails to evaluate → check nixpkgs attr name
  (`postgresql_18`) and go version pin; psql connect fails → read the socket
  and port the script printed, check `pg_ctl status` output; build fails →
  module path vs package path mismatch.
- **Escape hatch:** if `postgresql_18` is missing from the pinned nixpkgs, use
  the newest `postgresql_1X` present and record the substitution in
  `notes.md`; the spec only needs psql ≥ 14 (generated columns).
- **Stop-when:** W1 verify green in this session, committed, `git log -1`
  confirmed, handover rewritten.

### T2. Schema + golden seed — the tp.md example is in the database and checks out

- **Step 0:** T1 verify commands green (baseline); `docs/spec/data-model.md`
  and `business-rules.md` exist and their field lists agree with each other.
- **Plan (read):** `docs/spec/data-model.md`, `docs/spec/business-rules.md`
  (Parameters + Golden fixture sections).
- **Do:** `db/migrations/0001_init.sql` — every table, constraint, generated
  column, and index from `data-model.md`. `db/seed/golden.sql` — demo users
  (admin, manager, **three drivers** — RN05: one route per driver per date,
  so routes A/B/C on one date need distinct drivers; bcrypt hashes of the
  documented dev password), locations with the Portuguese addresses from
  `tp.md` section 5, routes A/B/C on one date with the seeded stop times,
  parameters at defaults. Routes B and C use placeholder addresses per the
  brief. `db/seed/golden_check.sql` — assertion queries (with
  `\set ON_ERROR_STOP on` so plain `psql -f` exits nonzero on failure):
  totals 75/41/45, stop-1 rows contribute 0, day total 161.
- **Verify:** W2 command sequence, run fresh (drop DB first), green in this
  session. `nix develop -c bash -c 'cd backend && go test ./...'` still
  green (nothing in Go reads the DB yet, this proves no regression).
- **Bisection:** seed violates a constraint → diff seed column list against
  `data-model.md`, fix the seed (spec wins); assertion fails → dump the
  computed rows with psql, compare against the golden table in
  `business-rules.md` before touching DDL; generated column misbehaves →
  check it against the RN01/RN02 expression in the spec, not against intuition.
- **Escape hatch:** a spec bug discovered here (wrong type, missed constraint)
  is fixed in the spec first, then in the migration, in the same commit, and
  recorded in progress.
- **Stop-when:** W2 green in this session, committed, handover rewritten.

### T3. Domain package — the rules as pure, unit-tested functions

- **Step 0:** baseline green; `business-rules.md` golden table present;
  `internal/domain` does not exist yet.
- **Plan (read):** `docs/spec/business-rules.md` (whole file).
- **Do:** `internal/domain` — plain types (`Route`, `Stop`, timestamps) and
  pure functions: `StopSeconds` (RN01+RN02), `RouteTotalSeconds` (RN03 with
  `min_stop_minutes`), `JourneyPercent` (RN04), `EstimatedCost` (RN07), and
  validation (`departure >= arrival`, stop order rules, distance > 0).
  No IO, no clock, no imports beyond stdlib. Unit tests pin the golden
  numbers (75/41/45; 15.625% for A; min_stop_minutes=6 → B totals 36) and the
  edge cases (midnight span, open stop, floor-of-sum vs sum-of-floors).
- **Verify:** `go test ./internal/domain/ -v` green; `go vet ./...` clean;
  `grep -rn "time.Now\|sql\|http" backend/internal/domain/` empty (purity
  guard).
- **Bisection:** a golden number fails → recompute by hand from the table in
  `business-rules.md`; if the spec's arithmetic is wrong, fix the spec and the
  test together and record it. Domain vs SQL divergence is what these tests
  guard; SQL remains the persistence truth, this package the validation oracle.
- **Stop-when:** W3 green in this session, committed, handover rewritten.

### T4. Store + write operations — the service layer can run a whole day

- **Step 0:** baseline green; T2's migrations apply to a fresh DB (re-run
  `scripts/testdb.sh`); pgx v5 added per the dependency budget in
  `architecture.md`.
- **Plan (read):** `docs/spec/operations.md` (directories, route composition,
  time recording sections), `docs/spec/data-model.md` (audit section).
- **Do:** `internal/store` — pgxpool setup, tx helper, repositories, audit
  writes inside the same tx. `internal/app` — services for: CreateDriver,
  UpdateDriver, ListDrivers, CreateManager, ListManagers, CreateLocation,
  UpdateLocation, ListLocations, CreateRoute, AddStop, RemoveStop,
  ReorderStops, StartRoute, CloseRoute, ReopenRoute, RecordArrival,
  RecordDeparture, UpdateStopTimes, SetRouteDistance, GetParams, UpdateParam.
  Sentinel errors per the error model. No HTTP anywhere in these packages.
  Integration tests against `stoptime_test` driving a full day for route A
  and asserting: RN05 conflict raised on duplicate, RN02 rejection, audit
  rows on corrections/params, closed-route rejection.
- **Verify:** `go build ./... && go vet ./... && go test ./internal/...` green
  from a fresh `scripts/testdb.sh`.
- **Bisection (write path):** request struct → service validation → SQL error
  → test DB state, cheapest first. A SQL error whose text does not match the
  constraint name in `data-model.md` means code and spec drifted; fix the
  spec first if the spec is wrong, else the code.
- **Escape hatch:** if a transaction pattern fights pgx, keep audit-in-tx
  non-negotiable and simplify the repository shape, not the guarantee.
- **Stop-when:** W4 green in this session, committed, handover rewritten.

### T5. Reads, SQL aggregation, cost, and the 3-second rule

- **Step 0:** baseline green; T4 services callable from a test.
- **Plan (read):** `docs/spec/operations.md` (reads and aggregation section),
  `docs/spec/business-rules.md` (RN04, RN07, performance section),
  `docs/spec/data-model.md` (indexes).
- **Do:** read services: GetRoute, ListRoutes, GetDashboardByDay/Month/Period,
  each backed by one SQL aggregation query returning plain structs. Cost and
  journey percent computed in SQL numeric, rounded once. Driver scoping
  (own routes only) as a query-level filter. `db/seed/gen_year.sql` (or a Go
  test helper) generating 12 months of synthetic routes. Tests: golden series
  from the golden seed (day 161, month 161, period 161); a timed test asserting
  the 12-month dashboard answers under 3s and `EXPLAIN ANALYZE` output shows
  index scans (assert on the plan, printed in the test log).
- **Verify:** `go test ./internal/...` green including the perf test, run
  fresh in this session; W5's conditions hold.
- **Bisection:** wrong series → run the same aggregation by hand in psql
  against the golden seed and diff; slow → read the printed EXPLAIN, missing
  index means `data-model.md` was not followed in 0001_init.sql (fix forward
  in a new migration).
- **Escape hatch:** if the synthetic generator makes the perf test flaky on
  hardware variance, lower the row count and record the new number; the
  pattern guard (index scan, no per-row loops) matters more than the absolute
  number, which sits in milliseconds at this scale anyway.
- **Stop-when:** W5 green in this session, committed, handover rewritten.

### T6. Auth + roles — the matrix is enforced in the service layer

- **Step 0:** baseline green; role matrix in `operations.md` read.
- **Plan (read):** `docs/spec/operations.md` (error model + role matrix),
  `docs/spec/architecture.md` (security section).
- **Do:** Login/Logout services (bcrypt verify, HMAC session cookie per
  `architecture.md`), a session type carried through a context, role checks
  in services (not only middleware), driver scoping enforced in the queries.
  A table-driven test walking the whole matrix: every operation × every role
  → allowed/denied as the spec says.
- **Verify:** `go test ./internal/...` green; the matrix test output shows
  every cell.
- **Bisection:** an allowed/denied cell disagrees with the spec → the spec is
  the target, not the test, unless the spec matches `tp.md` and the code does
  not (then the code is wrong).
- **Stop-when:** W6 green in this session, committed, handover rewritten.

### T7. HTTP shell + adapters + directories — pages exist, guardrails hold

- **Step 0:** baseline green; `backend/internal/httpapi` does not exist; the
  vendored assets (htmx, chart.js, css) need a one-time download, so confirm
  network access or bring the files.
- **Plan (read):** `docs/spec/architecture.md` (layering rules + dependency
  budget), `docs/spec/screens.md` (Login + Directories), `docs/spec/operations.md`
  (transports).
- **Do:** `internal/httpapi` — router (net/http method patterns), middleware
  (session load, role gate), template engine (html/template layouts +
  partials), sentinel→HTTP error mapping, flash messages. Vendor htmx,
  chart.js, and one classless CSS into `backend/web/static/`. Screens: Login,
  Directories (drivers, managers, locations) per the state tables in
  `screens.md`, plus the JSON `/api/*` mirrors for their operations.
- **Verify:** build/vet/test green; a scripted curl walkthrough (login as
  admin → create driver → list drivers) greps the expected markers;
  `grep -rn "https://" backend/web/templates/` and
  `grep -rn "SELECT" backend/internal/httpapi/` both empty.
- **Bisection:** page 500s → check the sentinel mapping table first (mapping
  is the usual suspect), then the template name, then the service; a test that
  only fails under the full test suite but not alone → test DB pollution,
  check `testdb.sh` isolation.
- **Stop-when:** W7 green in this session, committed, handover rewritten.

### T8. Route builder + tracker — a full day runs from the UI contract

- **Step 0:** baseline green; T7 shell compiles and login works.
- **Plan (read):** `docs/spec/screens.md` (Route builder, Route tracker),
  `docs/spec/operations.md` (route composition + time recording).
- **Do:** htmx flows for the two screens' state tables: builder (empty →
  adding → reordering → invalid → saved), tracker (not started → active →
  arrived → departed → completed → closed → error). "Mark arrival/departure
  now" buttons; manual timestamp entry for null fields; no stopwatch UI on
  stop 1 (RN01 visible in the product). JSON mirrors for every operation
  involved.
- **Verify:** build/vet/test green; curl walkthrough creates route A via the
  builder endpoints, records every arrival/departure via the tracker
  endpoints, closes, and `GET /api/routes/{id}` shows total 75 minutes.
- **Bisection:** htmx swap lands wrong → check the partial/id contract named
  in the screen's state table before suspecting the service; totals wrong →
  psql the same route and compare (T2's golden_check pattern).
- **Stop-when:** W8 green in this session, committed, handover rewritten.

### T9. Dashboard, history, params, audit, export — the client's screens

- **Step 0:** baseline green; aggregation services exist (T5).
- **Plan (read):** `docs/spec/screens.md` (Dashboard, History, Parameters,
  Audit), `docs/spec/business-rules.md` (parameters).
- **Do:** dashboard page with three tabs fed by the aggregate series (Chart.js
  renders, never computes); history list + route detail + corrections form
  (audited, manager/admin only); parameters screen; audit list (admin);
  CSV export endpoint per `operations.md` (UTF-8 BOM, RFC 4180). A Go test
  parses the exported CSV back and asserts rows.
- **Verify:** build/vet/test green; a scripted end-to-end run demonstrates
  acceptance criteria 2–4 of `tp.md` section 10; the CSV test passes.
- **Bisection:** chart shows wrong numbers → fetch the JSON the chart
  consumes and compare with `psql` on the same seed (aggregation truth lives
  in SQL); CSV garbled in Excel → check the BOM is the first three bytes.
- **Stop-when:** W9 green in this session, committed, handover rewritten.

### T10. Part-1 specification document — final review and render

The deliverable already exists: `docs/especificacao.md` (Portuguese prose,
verbatim English identifiers) with diagrams in `docs/especificacao/diagrams/`
(PlantUML use case / robustness / class; Mermaid crow's foot ER inline). This
card reconciles it with the finished implementation and renders the final
SVGs. The repo itself is the graded hand-in.

- **Step 0:** cards T1–T9 crossed off in git (this card reviews the finished
  system); the devshell provides `plantuml` (from T1), or
  `nix run nixpkgs#plantuml` as fallback.
- **Plan (read):** `docs/especificacao.md`, `docs/spec/use-cases.md`, `tp.md`
  sections 4, 6, 10, `plans/mvp/notes.md` (naming policy + PlantUML facts).
- **Do:** walk every UC description and diagram label against the real
  operations, screens, and tables; fix drift on both sides in one commit.
  Identifiers are verbatim English — the naming policy in `notes.md`; never
  translate one, even when a translation looks obvious. Re-render and commit:
  `nix develop -c plantuml -tsvg docs/especificacao/diagrams/*.puml`. If the
  professor later asks for a PDF export, render one into `docs/deliverables/`;
  otherwise the markdown file is the document.
- **Verify:** traceability walk — every RF01–RF12 and RNF01–RNF06 appears in the
  matrix or a documented non-UC decision; every UC names its operations; a
  fresh render exits 0 with an empty error scan; the SVG set matches the .puml
  set. Human review sign-off (user + at least one teammate) recorded in
  progress.
- **Bisection:** a diagram fails to render → check the .puml against the syntax
  facts in `notes.md` (boundary/control/entity render robustness icons; there
  is no `robustness` directive; output name follows the `@startuml <name>`
  directive, not the file name). An identifier that disagrees with the schema
  → the schema is the truth; fix the diagram.
- **Stop-when:** W10 green, committed, handover rewritten.

### T11. Name, campaign, demo, final acceptance — the finish line

- **Step 0:** baseline green; all W1–W10 crossed off in git.
- **Plan (read):** `tp.md` (whole file, one last time), `docs/spec/product.md`
  (deliverables map).
- **Do:** pick the product name (graded extra: "melhor nome do produto"),
  apply it in one branding commit (page titles, wordmark, README). Write the
  campaign material (graded extra: one page — pitch, three screenshots, a
  short post text). Build the demo scenario seed (a believable week for two
  drivers). Final acceptance run of `tp.md` section 10, item by item, output
  recorded in progress. README final (how to run: `nix develop`, scripts,
  login). Tag `plans/mvp/T11`.
- **Verify:** the four acceptance criteria demonstrated in-session with
  literal outputs; `git status` clean; `git log -1` is the finish commit.
- **Stop-when:** the plan's items are all crossed off, the final progress
  entry carries real verify output, tag exists. The folder becomes a record.

## Open risks

| Risk | Check that catches it | Sanctioned response |
| --- | --- | --- |
| Diagrams drift from spec/code (edited .puml without re-render, translated identifier) | T10 review walk; a stale .svg shows in `git status` | Fix the diagram, never translate an identifier (naming policy in `notes.md`); re-render all .puml |
| Teammates unfamiliar with Go | README quickstart + scripted DB workflow keeps their diff surface small | If it becomes a blocker, re-scope cards to pair on them |
| htmx reordering UX grows beyond up/down buttons | T8 card keeps drag-and-drop out of scope | Stay with one-position moves; record if rejected in review |
| Postgres version drift (devshell pins 18; another machine has other plans) | `scripts/testdb.sh` runs wherever the devshell runs | The devshell is the only supported environment |
| Timezone bugs (-03:00 offsets) | Golden seed uses -03:00 timestamps; T3/T5 tests | Fix at the store boundary, never by string manipulation |
| RN05 duplicate-date UX frustration | T8 builder pre-checks and links the existing route | Keep the 409 path as the backstop |
| Chart.js + htmx interplay (fragment swaps vs chart init) | T9 scripted walkthrough asserts the rendered series data | Feed charts from stable JSON endpoints, not from swapped fragments |

## How to resume

Follow `AGENTS.md`. The handover names the next card; this file holds the full
card set and the risk list. A card discovered wrong is a discovery: record it,
fix the plan, never guess past it.
