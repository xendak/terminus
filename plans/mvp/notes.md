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
