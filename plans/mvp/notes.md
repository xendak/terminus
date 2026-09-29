# Notes — mvp

Verified facts. A fact is recorded here the moment it is discovered, so no
session re-derives it. Claims of completion are not facts; they are re-verified
every session per `docs/method.md`.

## Environment (verified 2026-09-28, bootstrap session)

- OS: NixOS. `nix` 2.34.8 with flakes enabled (`nix-command flakes` in both
  `/etc/nix/nix.conf` and `~/.config/nix/nix.conf`).
- **No system `psql`, no `pg_isready`, no PostgreSQL service, no Docker.**
  Only a `postgresql-14.7-lib` exists in the store (some other package's
  dependency). Postgres comes from the devshell: `nix eval --raw
  nixpkgs#postgresql.version` → 18.6.
- Go 1.26.7 (system). Node v24.19.0 (unused by this project). Python 3.14.7
  (unused). Zig 0.16.0 (unused). git 2.55.0.
- Nothing listens on 5432 at bootstrap.

## Environment (T1 session, verified 2026-09-28)

- Devshell (`flake.nix`, committed `flake.lock`) pins nixpkgs rev
  `83199d0d373dd3ac2b9a1996b1d0263f76ab7a4c`: go 1.26.7, `postgresql_18`
  18.6, gopls, plantuml. The `postgresql_18` attr exists — no escape hatch
  was needed.
- **Nix flakes only see git-tracked files.** `nix develop` refused to
  evaluate `flake.nix` until it was `git add`ed. Rule for every future
  session: after creating any file a flake-step needs, `git add` it (or
  `git add -A`) before `nix develop`.
- nix 2.34 calls `outputs` with a `self` argument; the pattern must accept
  it (`outputs = { self, nixpkgs }:`), or evaluation fails with "unexpected
  argument 'self'".
- **Parity commands run from `backend/`.** The Go module lives at
  `backend/` (monorepo layout in `architecture.md`), so `go build ./...`
  from the repo root fails with "go.mod not found". The baseline is
  `nix develop -c bash -c 'cd backend && go build ./... && go vet ./... &&
  go test ./...'`.
- Cluster facts, owned by `scripts/lib.sh`: unix socket `.pg/sock`, TCP
  port **5543** (verified free at T1), loopback listen only, trust auth,
  superuser = OS user — no password in any URL. URL shape:
  `postgresql:///stoptime?host=<abs path to repo>/.pg/sock&port=5543`; the
  socket path in a URI must be absolute. `scripts/db-up.sh` stdout is
  eval-able: `eval "$(scripts/db-up.sh)"` exports `DATABASE_URL` and
  `TEST_DATABASE_URL`; every tool's output is routed to stderr (fd-3 trick)
  because pg_ctl/createdb stdout used to leak into it and break `eval`.
- `schema_migrations` is created by `scripts/migrate.sh` itself (it is
  infrastructure, not schema): `name text primary key, applied_at timestamptz
  not null default now()`. Files apply one transaction each with
  `ON_ERROR_STOP`.
- A backgrounded dev server must not inherit the session's stdout/stderr
  pipes — redirect to a logfile, then `kill` + `wait` — or the calling
  shell hangs on pipe EOF (hit in session 3).

## T2 session (verified 2026-09-28)

- **RN05 shapes the golden seed**: three routes on one date need three
  distinct drivers (one route per driver per date). `golden.sql` seeds 5
  demo users: admin, manager, drivers A/B/C. The T2 card originally said
  "three demo users, one per role" — impossible against its own spec; card
  corrected.
- **Demo login (dev seed only):** every demo user's password is
  `stoptime-dev` (bcrypt cost 10, hash generated and verified with
  golang.org/x/crypto/bcrypt outside the repo, in /tmp — the module stays
  dependency-free until T4).
- **Shell glob order follows the locale collation, not byte order.** Under
  `en_US.UTF-8`, punctuation is ignored at the primary level, so
  `golden_check.sql` sorts BEFORE `golden.sql` (`goldenchecksql` <
  `goldensql`). Hit it as a real bug: testdb.sh applied the check to an
  empty DB and died before seeding. Consequences: (1) `*_check.sql` files
  are assertion scripts, skipped by `testdb.sh` seeding (convention now in
  `db/seed/README.md`); (2) migration `000N_` zero-padding is what keeps
  apply order deterministic — never drop it; (3) if a future seed must run
  before another, prefix it numerically too.
- `db/seed/golden_check.sql` starts with `\set ON_ERROR_STOP on` so plain
  `psql -f` exits nonzero (3) on assertion failure — without it, SQL errors
  don't fail the run. Verified both ways (green run exits 0 printing
  75/41/45/161/15.625; the empty-DB run exited 3).
- Golden fixture ids are deterministic UUIDs:
  `aa000000-0000-4000-8000-` + 12-hex group — users `…0001`–`…0005`,
  locations `…0101`–`…010c`, routes `…0201`–`…0203`, stops `…0301`–`…030c`.
  A UUID's last group must be exactly 12 hex chars (first seed attempt
  wrote 10-char groups — invalid input syntax). Golden date: 2026-06-15.
- The `stop_seconds` generated column accepted `floor(extract(epoch FROM
  departure_at - arrival_at))` verbatim from the spec (implicit numeric→int
  cast in the generated-column context, Postgres 18) — no `::int` needed.
- `data-model.md` documented `schema_migrations(version)`; T1's landed
  migrate.sh creates `schema_migrations(name)`. Spec corrected to `name`
  (implementation is what lives in the cluster; infra table, invisible
  elsewhere).
- Route A's departure location has no street in tp.md (it names the point
  only): seeded with placeholder address "Av. Partida, 100"; routes B/C use
  "Endereço Bx/Cx" placeholders per the card. Driver B's profile sets
  km_per_l 12.50 (RN07 override branch); A and C are NULL (default
  parameter fallback branch).

## T3 session (verified 2026-09-28)

- **Domain decimals are exact rationals** (`math/big.Rat`), not float64:
  business-rules.md RN07 demands exact-decimal money and this package is
  the oracle against the numeric columns — a float mirror can diverge at
  the cent. API: `EstimatedCost(distance, kmPerL, fuelPrice, costPerKm
  *big.Rat) (*big.Rat, error)` (exact), `RoundMoney` (half away from
  zero, 2 places — the single allowed rounding), `JourneyPercent(seconds
  int, hours *big.Rat) (*big.Rat, error)`. Consequence for T4/T5: stores
  feeding the oracle scan numerics as strings (`big.Rat.SetString`
  parses "6.09", "8", and fractions like "125/6"); never scan money
  through float64.
- Repeating-decimal pins must be fractions, not truncated literals:
  4500/(6*3600)*100 = 125/6 exactly — `rat("125/6")` in tests.
- The min_stop_minutes comparison is in WHOLE minutes (RN03/RF10):
  floor(stop_seconds/60) < threshold drops the stop — 359 s drops at
  threshold 6, 360 s stays. Pinned in TestMinStopThresholdBoundary.
- Go trap (cost one red iteration): `_, wantErr := m[k]` is the comma-ok
  map lookup — wantErr received "key present", not the value. Test case
  tables carry wantErr in the struct; never re-look it up by name.
- Domain types: `Stop{Order, Arrival, Departure *time.Time}` (nil mirrors
  column NULL; open stop contributes 0) and `Route{Stops []Stop}`.
  Route identity/driver/date/status are service-struct concerns (T4+).
- The purity guard stays empty because domain comments say "persistence
  layer" / "the database" — never the lowercase strings the guard
  greps for. Keep that style.

## T4 session (verified 2026-09-28)

- **Dependencies pinned:** pgx **v5.11.0**, x/crypto **v0.57.0** (bcrypt —
  CreateDriver needs password_hash, so the hasher landed in T4, not T6).
  **Recorded decision:** google/uuid **v1.6.0** joins the budget for plain
  UUID values in structs — pgx ships no uuid type; google/uuid implements
  driver.Valuer/sql.Scanner and works with pgx natively (verified by the
  passing suite). Transitive: pgpassfile, pgservicefile, puddle, x/sync,
  x/text.
- **Migration 0002:** `audit_log.entity_id` uuid→**text**. Spec bug from T2:
  `parameter` rows have text PKs, so an audited `update_param` cannot put
  `fuel_price_brl` in a uuid column. data-model.md (ERD + table row) and
  the `docs/especificacao.md` ER fixed in the same commit; 0001 untouched.
  entity_id now holds the uuid string for route/route_stop and the key for
  parameter.
- **42P18 trap:** every bind parameter must APPEAR in the SQL. A `$2`
  passed but never referenced → `could not determine data type of parameter
  $2`. Hit in ShiftStopOrders' restore phase (was `$1,$3,$4`). Parameter
  numbers must stay contiguous.
- **pgx timestamptz scans come back in the PROCESS timezone** (Go
  constructs time.Time from UTC micros via time.Unix → time.Local), not
  the session TimeZone. The pool sets `RuntimeParams["TimeZone"]="UTC"`
  (server-side renders stay UTC per the timezone policy), but test
  assertions must compare INSTANTS — assertAuditInstants parses RFC 3339
  and uses .Equal. Never compare timestamp strings.
- `ListDrivers(activeOnly)`: the filter is `(NOT $1::boolean OR u.active)` —
  `$1 OR u.active` is inverted (it hides inactive rows on false).
- **numeric→text round-trips are scale-normalized:** value `6.09` stored in
  numeric(12,4) reads back `"6.0900"`; distance `100` in numeric(7,2)
  reads `"100.00"`; km_per_l `12.50`. Services re-read rows after mutations
  (CloseRoute, SetRouteDistance) instead of echoing inputs; tests assert
  the DB-normalized strings.
- **Stop renumbering** uses a two-phase park (order+1000, restore shifted)
  so the `(route_id, stop_order)` unique constraint never sees a
  mid-flight collision; CHECK `stop_order >= 1` forbids negative parking.
  Stop ids stay stable → audit entity ids stay meaningful. `store.MaxStops`
  = 999 caps composition below the park zone.
- `app.Actor{UserID, Role}` exists for created_by/updated_by/audit
  actor_user_id; role ENFORCEMENT (operations.md matrix) is T6's card.
- Integration tests (app/service_test.go): TestMain truncates and
  bootstraps the 5 default params + a bootstrap admin — the suite never
  depends on the golden seed. Tests assert DB state directly (raw SQL in
  the test file only — production app/store split holds: SQL lives in
  internal/store, none in internal/app).
- **Go's test cache can lie about DB state:** after `scripts/testdb.sh`
  rebuilds, `go test` may report `(cached)` without re-running. Use
  `go test -count=1 ./internal/...` for a literal fresh run.
- Error model wiring: store sentinels (ErrDriverDateConflict,
  ErrDuplicateEmail, ErrNotFound) aliased in app; ConstraintError (23503 →
  ErrNotFound, 23514 route_stop_times_order → domain.
  ErrDepartureBeforeArrival) mapped in app.mapErr; FieldError carries
  field detail under ErrValidation.

## T5 session (verified 2026-09-28)

- **Journey percent scale is 3** (`round(x, 3)` in SQL, rounded once —
  RN04): matches the golden 15.625 exactly; T2's golden_check already
  rounded to 3. Percent texts come back scale-normalized: "15.625",
  "8.542", "0.000". Cost stays round(x, 2) (RN07). No distance → cost
  NULL, never zero.
- **Period journey percent interpretation (recorded):** the uniform RN04
  formula — period total seconds over ONE standard journey day — for the
  grand total and each by_driver row (golden: 161 min → 33.542%). A
  "percent of worked days" reading would need a working-days definition
  the specs do not give; the golden period (one day) cannot distinguish
  the two anyway.
- **GROUPING SETS + GROUPING() for the period summary:** one query returns
  the grand row + per-driver rows; the grand row is identified by
  `GROUPING(b.driver_user_id) = 1`. Trap: the pivoted parameter columns
  (p.m, p.h) must appear in EVERY grouping set (they are one row, so this
  adds nothing). And do NOT identify the grand row by `max(u.name) IS
  NULL` — max over names is a name, not NULL (cost one red iteration).
- **Full-coverage windows legitimately seq-scan.** A 12-month query over
  12-month data selects ~100% of rows and the planner is right to
  seq-scan. The index guarantee is therefore asserted on a selective
  window: synthetic data spans 36 months (2022–2024, ~4700 routes / 28k
  stops, 6 drivers), the 3s bound covers the 12-month aggregation
  (measured: day 4.6ms, month 4.7ms, period 7.1ms), and EXPLAIN ANALYZE
  on a one-month window shows `Bitmap Index Scan on route_route_date_idx`
  with route_stop hash-joined — no per-row loops. The plans print in the
  test log (assert: contains route_route_date_idx; not "Seq Scan on
  route " — trailing space, else route_stop matches).
- **gen_year chose the Go test helper, not db/seed/gen_year.sql** (the card
  allowed either): a seed file would hit the en_US collation-order trap
  (gen_year sorts before golden.sql — punctuation ignored) and would
  leak perf data into every `testdb.sh --seed` consumer. The helper
  generates via one INSERT…SELECT per table with a data-modifying CTE
  (WITH ins AS (INSERT … RETURNING) INSERT …) and fixed uuid literals
  (dd00…/ee00… prefixes).
- **TestMain now applies the golden seed** (db/seed/golden.sql) after
  truncation — the read tests' fixture (161/75/41/45) is the same file
  W2's check runs, no duplication. pgx takes ONE statement per Exec, so
  the loader strips `--` comments line-wise and splits on ';' — seed
  files must keep ';' ONLY as a statement terminator (a comment
  semicolon broke the first attempt).
- **Test file order matters: reads_test.go sorts before service_test.go**, so
  the read tests see only the golden seed (+ the perf data generated at
  the end of reads_test.go). Windows keep the suites disjoint: golden
  2026-06, service tests 2026-07, synthetic 2022–2024. The perf data
  persists while service tests run — their assertions are
  presence-based, never exact counts (ListDrivers finds "found" flags,
  GetParams still 5 because perf adds no parameters).
- The test's own pgxpool (raw fixture/EXPLAIN queries) uses the cluster's
  default session TZ (America/Sao_Paulo); only the SERVICE pool pins
  TimeZone=UTC. Timestamps are only ever compared as instants.
- `generate_series(date, date, interval)` returns TIMESTAMPTZ — cast
  `g::date` before date arithmetic (`date - date` = int days; `interval %
  int` does not exist — cost one red iteration).
- Read-query shape: parameters pivoted once per query
  (`max(value) FILTER (WHERE key = …) FROM parameter`), min_stop_minutes
  applied inside the SUM CASE via integer division
  (`stop_seconds / 60 >= p.m` — whole minutes, RN03/RF10), totals summed
  in seconds and floored once. GetRoute's aggregate CTE has no GROUP BY
  → always one row → CROSS JOIN it (no route_id needed).
- `store.DashboardByDaySQL` is exported so the perf test EXPLAINs the
  exact shipped query — no drift between tested and running SQL.

## T6 session (verified 2026-09-28)

- **Session format (architecture.md Security, implemented):** cookie
  `st_session` (const `app.SessionCookieName`) = base64url(json of
  `{uid, role, exp}`) + "." + base64url(HMAC-SHA256 over the body);
  lifetime 12h (`app.SessionLifetime`); `DecodeSession(key, value, now)`
  is pure and folds every failure — bad format, bad signature, expired,
  empty identity — into ErrUnauthenticated. Login fails fast when
  `Services.SessionKey` is < 32 bytes (the key comes from SESSION_KEY;
  tests use a fixed literal). No server-side state — revocation is the
  recorded tradeoff.
- **Enforcement shape:** `app/permissions.go` holds the matrix
  (op → roles) with three shapes — plain, own-route, scoped. Every
  service starts with `allow(actor, op)`; own-route ops add `ownRoute`
  after loading the route; scoped reads call `forceDriverScope` (the
  driver's id replaces the filter — never post-filtered). Order:
  gate → own-route → status/business, so Forbidden leaks nothing.
- **Actor threading is now complete:** every service takes Actor first
  (T4's partial threading finished). Session-in-context helpers
  (WithSession / SessionFromContext / ActorFromSession) exist for the
  T7 middleware; enforcement reads the Actor, the context copy serves
  transport rendering.
- **Login semantics:** all failures — unknown email, wrong password,
  deactivated account — are ErrUnauthenticated (no user enumeration);
  bcrypt compare against `app_user.password_hash`; inactive users
  cannot log in. The golden demo password `stoptime-dev` drives the
  login tests.
- **Matrix test design (zmatrix_test.go):** the expected table is
  transcribed INDEPENDENTLY from operations.md — it must not read the
  enforcement's own map, so the two tables agreeing is the pin. Every
  cell is a REAL service call; allowed cells succeed on fresh fixtures
  (unique driver per fixture, so dates may repeat without RN05
  conflicts — the first date generator ran past 2025-08-31 and broke
  parsing; it now cycles 1..28). Driver cells of own-route ops act as
  the fixture owner; TestOwnRouteBoundary proves the not-own denial.
  The file sorts LAST (z…) so its 2025-08 fixtures never reach the
  other suites' exact-count windows.
- Enforcement caught a T4 test bug immediately: the full-day test
  reopened a route as MANAGER — operations.md says ReopenRoute is
  admin-only. Spec wins; test fixed (the bisection rule working as
  designed).
- `app` still imports no `net/http` — the cookie is a plain string
  value; T7 sets the actual cookie attributes.

## T7 session (verified 2026-09-29)

- **Vendored assets (recorded versions):** htmx 2.0.6, Chart.js 4.4.9
  (chart.umd.js), Pico CSS 2.1.1 — all in `backend/web/static/`, embedded
  via `//go:embed` in package `stoptime/web`. No CDN URLs anywhere in
  `backend/`.
- **Login redirect stand-in (decision):** screens.md §1 says success
  redirects by role (driver → tracker, manager/admin → dashboard).
  Those screens land in T8/T9, so `loginRedirect(role)` currently maps
  every role to `/` (the role-aware home); **T8/T9 flip this one
  function** to /routes/today and /dashboard.
- **JSON contract decision:** OUTPUT plain structs carry `json` tags
  (snake_case, recorded once); `User.PasswordHash` is `json:"-"` and
  must NEVER gain a tag — the API must not leak hashes (tested).
  INPUTS stay untagged: adapters own request DTOs per the transport-seam
  principle ("adding a transport adds an adapter, not a service
  change") — pointer DTO fields distinguish absent from empty on
  PATCH.
- **Templates:** one template set per page — `template.ParseFS(web.FS,
  "templates/layout.html", "templates/<page>.html")` — the page file
  `{{define "content"}}` overrides per set; execute the root. Labels:
  one map per screen merged with shared nav labels (screens.md's
  translation-seam rule).
- **Flash messages:** signed one-shot cookie `st_flash` (same session
  key, HMAC over the base64 body), set on POST redirects, read AND
  cleared by the next page render. Inline `Errors` map renders the
  screens.md "invalid" state (re-render, no redirect).
- **No GetDriver/GetLocation operation exists** (operations.md is the
  authority), so edit forms source their row from the directory list
  (ListDrivers/ListLocations + pick by id in the handler) —
  presentation-side, not a service gap.
- **Middleware vs services:** withSession decodes the cookie once and
  stores Actor + Session in the context; requirePage bounces anonymous
  to /login, requireAPI 401s. The role matrix stays service-side — a
  driver hitting /drivers gets the page shell then 403 from the service
  (tested: page 403, JSON 403).
- main.go now wires store → app → httpapi with env config
  (DATABASE_URL required, LISTEN_ADDR default 127.0.0.1:8080, SESSION_KEY
  dev default + warning log).
- Guard state: templates have no URLs; httpapi has zero SQL; domain
  purity holds; `https://cdn` absent from all of backend/.

## Decisions (with the user, bootstrap session)

- Remote (user, session 3): `origin` = `git@github.com:xendak/terminus.git`
  (SSH), `main` tracks `origin/main`; linked and pushed at the start of
  session 3. Every session-close commit pushes from now on — the repo is
  the graded hand-in (session-2 decision).
- Stack: **Go backend** (stdlib `net/http` + `html/template`, pgx v5, x/crypto
  bcrypt), **SSR + htmx** frontend, **Chart.js vendored** for charts, plain SQL
  migrations applied by psql, PostgreSQL from the Nix devshell, repo-local
  cluster in `.pg/`. No Docker, no npm, no ORM, no build pipeline.
- The user's stated motive: the backend must survive a later frontend switch
  (htmx → React). Their five caveats are normative layering rules in
  `docs/spec/architecture.md`: (1) handlers only parse, call a service, and
  format; (2) services take and return plain structs so `DashboardByMonth` can
  feed HTML or JSON; (3) queries and aggregations live in SQL, so charts receive
  aggregate data only; (4) the API contract is written per operation
  (create route, register arrival/departure, get dashboard by period), not per
  endpoint; (5) screens are defined by use case and states (route builder:
  empty, adding points, reordering, error), not by page fragments.
- Language: engineering artifacts in English. UI English by default, one
  labels map per screen for a possible translation layer later (explicitly not
  a priority until the MVP runs). User-entered data (names, addresses, notes)
  accepted in Portuguese. Golden seed keeps the Portuguese addresses of
  `tp.md` section 5 verbatim.
- Working title **StopTime** for module/DB/internal naming. The graded product
  name is chosen in T11; branding must be one rename commit away.
- Part-1 deliverable format (user, session 2): the graded hand-in is the
  GitHub repo itself; the specification is its own file,
  `docs/especificacao.md`, with prose strictly in Portuguese. Four diagram
  types: use case, robustness, conceptual class (PlantUML sources `.puml` +
  committed rendered `.svg`), and crow's foot ER (Mermaid inline, whose
  `erDiagram` is crow's foot and renders natively on GitHub; PlantUML does
  not render on GitHub, hence committed SVGs).
- **Naming policy for the deliverable (user, session 2, corrects a drift the
  user caught mid-work):** identifiers in the especificação are verbatim from
  the codebase and spec set, in English. A translated identifier ("momento",
  "USUARIO", "perfil") describes a system that does not exist — a false
  diagram. Portuguese is for prose and personas; actors carry the real `role`
  value ("Motorista (role: driver)"); UC titles are Portuguese and map to the
  English operations in the traceability matrix. Even obvious translations
  fail: the translation of "perfil" would be `profile`, but the column is
  `app_user.role` with values `admin`/`manager`/`driver`. Copy the schema,
  never translate it.

## Golden fixture (from `tp.md` section 5 — do not re-derive)

- Route A: stops at Rua Peru 55 (15 min), Rua X 5 (10 min), Av. João César
  (50 min) → total **75 minutes**.
- Route B: 10 + 5 + 26 → **41 minutes**. Route C: 5 + 10 + 30 → **45 minutes**.
- Stop 1 of every route contributes **0** (RN01), even if seeded with
  timestamps.
- Journey percent route A at 8h standard: 75/480 = **15.625%** (RN04).
- All three routes on one date → day/month/period totals all read **161**.
- With `min_stop_minutes = 6`: route B's 5-minute stop drops out → B totals
  **36 minutes**.

## PlantUML and diagrams (verified 2026-09-28, session 2)

- nixpkgs provides `plantuml` (1.2026.6). `nix run nixpkgs#plantuml -- -tsvg
  <files>` works and brings graphviz along; first run fetches ~30 MiB, later
  runs are local. The devshell gets `plantuml` in T1 so teammates render
  without nix incantations.
- **Robustness syntax, verified empirically** (SVG primitive inspection, not
  memory): a plain `@startuml` diagram using the `actor`, `boundary`,
  `control`, `entity` keywords renders the proper robustness icons (boundary
  = circle with vertical bar, control = circle with arrow, entity = circle
  with underline). A `robustness` directive does **not** exist and errors at
  line 2 — do not retry it.
- PlantUML derived-attribute notation works: `/total_stopped_minutes`
  renders as-is in class diagrams.
- The output filename comes from the `@startuml <name>` directive, **not**
  from the `.puml` filename — keep them identical or the `.svg` and `.puml`
  names diverge (this session hit it and fixed it).
- Mermaid renders natively on GitHub and Obsidian; PlantUML does not. That is
  why the ER is inline Mermaid and the PlantUML diagrams are committed SVGs
  referenced from `docs/especificacao.md`.

## tp.md reading notes

- `tp.md` is the professor's brief and is **immutable** in this repo. It is
  the requirements ground truth; our spec set translates and sharpens it.
  Conflicts: `tp.md` wins, spec updated, recorded here.
- Graded extras (up to 2 points): best product name, best publicity campaign.
  Both are deliberately last-mile cards (T11), after function.
- The brief's "Gerente" entity includes "equipe sob responsabilidade"
  (team under responsibility). MVP simplification: single company, all
  managers coordinate all drivers. Recorded as accepted simplification in
  `data-model.md`; extension point if the professor pushes.

## Open questions

1. ~~Professor's expected format for the part-1 document~~ **RESOLVED** (user,
   session 2): the hand-in is the GitHub repo; the specification is
   `docs/especificacao.md` (pt-BR prose, verbatim English identifiers), with
   use case, robustness, and class diagrams in PlantUML and the crow's foot
   ER in Mermaid. Residual: if the professor later asks for a PDF export, T10
   renders one into `docs/deliverables/`.
2. Team composition (dupla ou trio): who the teammates are and whether they
   will work through this plan folder too. The README tells them to.
3. Whether the professor wants the 8h journey percent interpreted per route
   or per driver-day. `tp.md` RN04 says "indicadores de tempo parado" based on
   the 8h day; we compute per route (a driver has one route per day, RN05, so
   the two readings coincide in practice). Recorded in `business-rules.md`.
