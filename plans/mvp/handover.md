# Handover — mvp

## State

Bootstrap complete (session 1) and part-1 deliverable drafted (session 2).
The repository holds: the spec set under `docs/spec/` (product, architecture,
business-rules, data-model, operations, screens, use-cases), the session
rulebook (`docs/method.md`, `AGENTS.md`), this plan folder, and the part-1
specification `docs/especificacao.md` (Portuguese prose, verbatim English
identifiers) with its diagrams in `docs/especificacao/diagrams/` (PlantUML
sources + rendered SVGs; the crow's foot ER is inline Mermaid). No code
exists: no flake, no scripts, no Go module, no migrations. All eleven cards
(T1–T11) in `plan.md` are unchecked. The working tree is clean at the
session-2 commit.

## Next

**T1. Devshell, scaffold, database bring-up** (`plans/mvp/plan.md`).

- Plan (read): `docs/spec/architecture.md` (environment/tooling section),
  `plans/mvp/notes.md` (environment facts).
- Do: `flake.nix` devshell (go 1.26, `postgresql_18`, gopls); Go module
  `stoptime` with `/healthz`; `scripts/db-init.sh`, `db-up.sh`, `db-down.sh`,
  `migrate.sh`, `testdb.sh`; empty `db/migrations/` and `db/seed/`;
  `.env.example`.
- Verify: `nix develop -c bash -c 'go build ./... && go vet ./... && psql --version'`;
  `scripts/db-init.sh && scripts/db-up.sh` then
  `psql "$DATABASE_URL" -c 'select 1'`; `curl -s localhost:8080/healthz`.
- Stop-when: verify green in-session, committed, handover rewritten.

## Baseline commands

Before T1 lands the devshell, healthy means:

```
git status                       # clean tree
go version                       # go1.26.7 (system Go, for the scaffold until the devshell exists)
nix eval --raw nixpkgs#postgresql.version   # 18.6 (proves nix + flakes reach postgresql)
```

After T1, the baseline becomes the parity command from `docs/method.md`
(`go build ./... && go vet ./... && go test ./...` with the cluster up); each
card's Verify names its own.

## Facts this task needs

- This machine has **no system psql** and no running PostgreSQL. The devshell
  (`postgresql_18` from nixpkgs, version 18.6 confirmed reachable) is the only
  supported source. The cluster is repo-local under `.pg/` (gitignored),
  created and started by the scripts T1 writes.
- Flakes are enabled on this machine (`nix-command flakes` in both
  `/etc/nix/nix.conf` and `~/.config/nix/nix.conf`); nix 2.34.8.
- Stack decision (user, bootstrap session): Go backend, SSR + htmx frontend,
  Chart.js vendored for charts, pgx for SQL, plain-SQL migrations applied by
  psql. The layering rules that make a later React switch cheap are normative
  in `docs/spec/architecture.md`; the operation-first contract is
  `docs/spec/operations.md`; screens are state machines in
  `docs/spec/screens.md`.
- Language split (user, bootstrap session): engineering artifacts English; UI
  labels English by default with one labels map per screen (translation layer
  later, not now); user-entered data accepted in Portuguese; golden seed keeps
  the Portuguese addresses from `tp.md` section 5 verbatim.
- Working title StopTime for internal naming; the product name is chosen in
  T11 (graded extra) and nothing may hardcode it in a way T11 cannot rename.
- `docs/especificacao.md` is the part-1 deliverable: Portuguese prose, verbatim
  English identifiers. The naming policy and the PlantUML facts live in
  `notes.md`; any session that renames a public identifier updates the
  document (and re-renders diagrams) in the same commit.

## Open risks (subset relevant to T1)

- `postgresql_18` attr name must exist in the pinned nixpkgs (escape hatch in
  the card: fall back to newest available, record in notes).
- Docker does not exist on this machine and is out of scope; do not reach for
  containers to "fix" the database story.

## Out of scope

No code beyond what T1's card names. No schema work (T2), no domain logic (T3).
Do not start the flake before reading the architecture doc's environment
section; it pins the layout the scripts must produce.
