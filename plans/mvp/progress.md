# Progress — mvp

Newest entry on top. Append-only.

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
