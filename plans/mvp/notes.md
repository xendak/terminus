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

## Decisions (with the user, bootstrap session)

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
