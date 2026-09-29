# Progress — mvp

Newest entry on top. Append-only.

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
