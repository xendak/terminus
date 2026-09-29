# AGENTS.md — Terminus MVP (es2/tp2)

Operating instructions for AI coding agents in this repo. Read this whole file
at session start, then run the [session protocol](#session-protocol).

This is a graded university deliverable (Engenharia de Software II, 2º trabalho
avaliativo, PUC Minas). The requirements brief is `tp.md`, and it is immutable:
it is the professor's document, not ours to edit.

## Where things live

Several things get called "the plan." Confusing them is the fastest way to
break a resume:

| Path                | Role                                                                                                                    | Discipline                                                                                                    |
| ------------------- | ----------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| `tp.md`             | **The brief** — requirements ground truth (RN01–07, RF01–12, RNF01–6, acceptance criteria). Never edit it.             | Where a spec file disagrees with `tp.md`, `tp.md` wins and the spec is updated in the same commit.             |
| `docs/especificacao.md` | **The part-1 submission document** — Portuguese prose; identifiers verbatim English from the spec/code; diagrams in `docs/especificacao/diagrams/` (PlantUML `.puml` + committed `.svg`; crow's foot ER is inline Mermaid). | The repo is the graded hand-in. Never translate an identifier (naming policy in `plans/mvp/notes.md`); an edited `.puml` re-renders its `.svg` in the same commit. |
| `docs/method.md`    | **The convention** — how multi-session work is planned, executed, verified, handed over. The rulebook.                  | Never record project state here. Read it when bootstrapping or unsure of a ritual.                            |
| `docs/spec/`        | **The spec set** — what to build: product, architecture, business rules, data model, operations, screens, use cases.     | The authority for design. A card that contradicts a spec updates the spec (or obeys it), never guesses past it.|
| `plans/README.md`   | **The scheduler** — ordered task index, one line per task with status. Answers _what to work on next_.                  | Rewritten freely; never appended to as history.                                                              |
| `plans/mvp/`        | **The work** — `plan.md` (goal, task cards T1–T11), `handover.md` (resume card), `progress.md` (append-only log), `notes.md` (verified facts). | All rules per `docs/method.md`. Task state and discovered facts live here and nowhere else.                  |

## Session protocol

### Step 0 — identify (every session, before anything else)

1. `git status` — clean tree: the last session ended on purpose (a completed
   stage, committed). Dirty tree: it stopped mid-stage; the newest `progress.md`
   entry names the open point — that's the resume point.
2. `git log -1` — note the hash.
3. Read `plans/README.md` — the current task is the topmost `[~]`, or the
   topmost `[ ]` if none is active.
4. If that task has a `handover.md`: read it, then check its provenance —
   don't let the file certify itself:
   `git log -1 --format=%H -- plans/mvp/handover.md` vs
   `git log -1 --format=%H`. Equal → current (nothing committed since it was
   written). Different → the repo moved: read `git log <that commit>..HEAD` and
   reconcile before trusting the rest. (A handover can't name its own commit's
   hash — the hash covers the file — so freshness is derived from git, never
   self-declared.)
5. Confirm the things this session will touch exist where the plan claims:
   the spec files the card's Plan (read) line names, `scripts/`, `db/migrations/`,
   the devshell. Check with `ls` / `test -f` — "the card says so" is not
   existence. A missing path is a plan update, never a guess to code around.
6. Run the baseline the handover names — **yourself, in this session, now**,
   whatever the handover or `progress.md` claims about the previous one.
   Before T1 lands the devshell the baseline is `go version` + `nix eval
   --raw nixpkgs#postgresql.version` + clean tree. After T1 it is the parity
   command: `nix develop -c bash -c 'go build ./... && go vet ./... && go test
   ./...'` with the database cluster up (`scripts/db-up.sh`). A prior
   session's "tests passed" is a claim, not a fact. Green baseline: any later
   red is this session's fault. Red baseline: fixing it is this session's first
   task — fix, commit, then continue.

### Routing — what to work on

After Step 0, in precedence order:

1. **Dirty tree** → mid-task stop. Resume at the point the newest
   `progress.md` entry names (after the baseline, per Step 0.6).
2. **Current task has a current handover** → start at the first unchecked item
   of the task card the handover names. Read _only_ the files that card's
   **Plan (read)** line names. If the handover's state says the task is
   complete, mark it `[x]` in `plans/README.md` and move to the next task.
3. **Task folder exists, no handover** → start its T1 card; if it will span
   commits, create its `step_<n>_progress.md` from the card first.
4. **Task listed in `plans/README.md` with no folder** → bootstrap it per
   `docs/method.md`.
5. **No `plans/` at all** → the convention has not started here. `git init`
   first, create `plans/README.md`, then bootstrap per case 4.

One **task card** per session. When the card's Stop-when is met, stop — the
next card is the next session's.

## Go and web work — required reading

The spec set replaces a cookbook: before nontrivial work, read the spec file
that owns the territory, not just the card:

| Work type                        | Read first                                            |
| -------------------------------- | ----------------------------------------------------- |
| Any time math, cost, totals       | `docs/spec/business-rules.md` (all of it)             |
| SQL, schema, migrations, audit   | `docs/spec/data-model.md`                             |
| Any service or endpoint          | `docs/spec/operations.md` (the operation's entry)     |
| Any page or partial              | `docs/spec/screens.md` (the screen's state table)     |
| Package layout, deps, security    | `docs/spec/architecture.md`                          |

Standing rules, all of them normative in `architecture.md`:

- **Layering is not negotiable.** Handlers parse, call a service, format. No
  SQL in `httpapi`, no business rules in handlers, no IO in `domain`.
  Guard greps that must come back empty before a commit:
  `grep -rn "SELECT\|INSERT\|UPDATE\|DELETE" backend/internal/httpapi/`,
  `grep -rn "time.Now\|sql\|http" backend/internal/domain/`.
- **The database is the source of truth for derived values, and SQL does the
  aggregating.** Dashboard series, route totals, cost, and journey percent are
  SQL-computed; Go receives plain structs; charts receive aggregate data and
  never sum rows client-side. The domain package mirrors the formulas as a
  tested oracle for validation.
- **Migrations are append-only.** `db/migrations/*.sql` in filename order,
  applied by `scripts/migrate.sh` (psql, one transaction per file, tracked in
  `schema_migrations`). An applied migration is never edited; fixes are new
  migrations. psql is the oracle for SQL behavior: when unsure what Postgres
  18 does with a construct, try it in psql against the test DB, don't reason
  from memory.
- **Parameters are data, not code.** Fuel price, cost per km, km/l default,
  journey hours, min stop minutes live in the `parameter` table
  (`business-rules.md`). Hardcoding any of them is an acceptance-criteria
  violation, not a style issue.
- **Test-first for the math.** Every formula change lands with a test pinning
  the golden fixture (routes A/B/C, 75/41/45, 161 day total, 15.625%) from
  `business-rules.md`. Integration tests run against `stoptime_test`, created
  fresh by `scripts/testdb.sh` — never against the dev database.
- **Timestamps** are `timestamptz` in UTC; `route_date` is a `date`. "Now"
  comes from the server clock. Never format or compare times as strings.
- **Assets are local.** The legacy htmx pages' htmx, Chart.js, and stylesheet
  are vendored in `backend/web/static/`; templates contain no external URLs
  (`grep -rn "https://" backend/web/templates/` stays empty).
- **The dependency budget is closed** (backend: pgx, x/crypto, google/uuid,
  vendored assets; frontend: what `frontend/package.json` lists — see
  `architecture.md`). Anything else needs a recorded decision in
  `plans/mvp/notes.md` first.

## What we're building

**Terminus** (formerly the working title StopTime, which survives only in
identifiers: Go module `stoptime`, databases, cookie `st_session`): a web MVP
that monitors how long delivery field workers stay stopped at each point of
their daily route. Go + PostgreSQL backend with JSON transports under `/api/*`;
the client is a Next.js app in `frontend/` (pt-BR UI, same-origin rewrite
proxy :3210 → :8080; user decision 2026-09-29). The original server-rendered
htmx pages remain in the Go binary as a legacy transport, still tested.

1. Registers couriers, managers, and points (locations with address and
   coordinates).
2. Builds a daily route per driver: ordered stops, stop 1 is the departure
   point and never accrues stopped time (RN01).
3. Records arrival and departure per stop; stopped time per stop is the
   difference (RN02); route total is the sum over counted stops (RN03).
4. Shows the dashboard the client asked for: stopped time per day, per month,
   and per period, always tied to addresses and timestamps, with the 8h-day
   percentage (RN04).
5. Computes route cost from distance, km per liter, fuel price, and cost per
   km (RN07), all parameterizable without code changes.
6. Enforces roles (driver/manager/admin, RNF04) and audits every change to
   points and times (RNF05), with an LGPD-lite data policy (RNF06).

Out of scope by the brief (`tp.md` 3.2): route optimization, payroll/ERP
integration, live vehicle telemetry, native apps. Out of scope by us: Docker,
an ORM, migration frameworks, Node anywhere outside `frontend/` (the Go binary,
scripts, and database workflow stay Node-free).

## Ground truth — do not guess

- **Requirements**: `tp.md` is the source of truth. Our spec set sharpens it;
  a conflict is resolved in `tp.md`'s favor and the resolution is recorded in
  `plans/mvp/notes.md`. Professor answers (email) are also ground truth —
  record them in `notes.md` when they arrive.
- **SQL behavior**: PostgreSQL 18's actual behavior, checked in psql against
  the test database, beats anything remembered about SQL.
- **Library APIs** (pgx, html/template, htmx, Chart.js): their current docs,
  reached through `plans/mvp/references.md`, beat training-data memories. Go
  stdlib is stable, but check method-pattern routing and template semantics in
  the version the devshell pins.
- If none of these settles something, say so explicitly rather than filling
  the gap with a plausible guess, and flag it as an open item in the plan.

## Task cards open with Step 0: identify

Every task card `T<n>` in `plans/mvp/plan.md` carries a **Step 0 — identify**
item before any **Do**: the existence and sanity checks the card depends on.
A failed Step 0 check updates the card (and `notes.md`) — it never gets
silently worked around in code. Discovering the card is wrong is a result;
guessing past it is the failure mode the whole method exists to prevent.

## Session end — the handover ritual

Every session ends with **one commit**, built in this order:

1. Cross off what landed in `plans/mvp/plan.md` — an item is done when it's
   **in git**, not when the code is written or works locally.
2. Append the `progress.md` entry (newest on top): what landed, what was
   discovered ("must not rediscover" — those facts also go in `notes.md`),
   what is next, the **literal output** of the verify command that proves the
   stage is done — actual text or confirmed exit status, never "tests passed"
   — and how the session ended: card finished, early stop (at which sub-item),
   or compacted.
3. Rewrite `plans/mvp/handover.md` — never append: current state, the next
   card's plan/do/verify, the baseline command, the facts that task needs,
   open risks, out-of-scope list. Mid-card: name the open card and the first
   unchecked item. Mid-red: also carry the **debug state** — literal failing
   output, bisection position, hypotheses ruled out, leading hypothesis, the
   single next check, anything tried and reverted. It carries no commit hash
   of its own — a file can't contain its own commit's hash; its freshness is
   provenance (the commit that last touched it is HEAD).
4. `git add -A && git commit -m "mvp: <stage> (plan T<n>)"` — the work, the
   plan files, and any open checklist in **one** commit.
5. `git log -1` and `git status` — the ritual's commit is HEAD and the tree is
   clean, confirmed not asserted. If either is false, fix forward in a
   follow-up commit; there is no amend exception.
6. Optionally `git tag plans/mvp/T<n>` so the stage can be checked out or
   diffed later.

If this session started a task (bootstrapped its folder or worked its first
card), its `plans/README.md` line goes `[~]` in this commit; if it finished the
last card, `[x]`.

## Commit discipline (from `docs/method.md`, non-negotiable)

- One commit per completed stage — a card's Verify going green. Never squash
  stages into one commit; never split one verified stage across commits a later
  session can't tell apart.
- Test-first work may commit the failing test mid-stage _when the card says so_;
  the stage's closing commit is the green one.
- Don't amend or rebase a commit that anything points at — a hash pasted into a
  `progress.md` entry, or the commit a handover's provenance check anchors to.
  Fix forward in a new commit.
- The log is append-only, like `progress.md`.

## Early stop and compaction — standing instructions

- If the user says **"end session now"** (usually because the context meter is
  low), do not try to finish one more thing. Run the end-of-session ritual
  immediately, mid-card if that's where you are: commit what's green, leave the
  open state exact, rewrite the handover to name the open card and the first
  unchecked item. An early stop leaves the same repository shape as a finished
  card.
- If you notice the context was **compacted** — a summary replaced recent
  memory, or your sense of what's done no longer matches the checklist — stop
  coding and re-run the session-start steps from disk: `git status`, handover,
  baseline. Trust the files over your memory, re-run the current item's verify
  before crossing it off, and record the compaction in the progress entry.
- **Stop on drift, not on a gauge** — you cannot measure your own context
  fullness, so never estimate it; watch the work instead. End the session when
  any of these holds:
  - you've had to understand files beyond the card's Plan (read) list;
  - a red verify is three fix-iterations deep without the bisection moving —
    write the debug state into the handover first;
  - a Step 0 check failed or the spec contradicts the card — record the
    discovery and end; re-cutting the card is a new session. Exception: don't
    stop one mechanical step from green — finish the card. The stop lands on a
    **committable boundary**: commit the failing test with a note or revert to
    green, never leave half-applied edits. Say plainly in your final message:
    "session should end here — start the next one with the opening prompt."

## Explicitly out of scope for now

- No route optimization, payroll/ERP integration, vehicle telemetry, or native
  apps (`tp.md` 3.2 — the professor excluded them).
- The Next.js client (`frontend/`, Node 24 + pnpm) is the only place a
  JS build pipeline exists; it consumes `/api/*` and never reaches the
  database or Go internals directly. No Node in `backend/`, `scripts/`, `db/`.
- No Docker, no ORMs, no migration frameworks beyond the psql scripts.
- No speculative features beyond the brief: if a card has spare capacity, the
  card is mis-sized — split it or stop, don't gold-plate.
