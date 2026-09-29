# Handover — mvp

## State

T1 landed (session 3): devshell (`flake.nix` + `flake.lock` — go 1.26.7,
`postgresql_18` 18.6, gopls, plantuml, nixpkgs pinned at `83199d0d…`), Go
module `stoptime` at `backend/` with `cmd/server` serving `GET /healthz` →
`ok`, cluster scripts under `scripts/` (`lib.sh`, db-init/db-up/db-down/
migrate/testdb; socket `.pg/sock`, port 5543), `db/migrations/` and
`db/seed/` (empty, READMEs only), `.env.example`. The repo-local cluster is
**up**; `stoptime` and `stoptime_test` exist; `schema_migrations` exists with
zero rows. Remote `origin` = `git@github.com:xendak/terminus.git` (SSH) —
push every session-close commit. Working tree clean at the session-3 commit.

## Next

**T2. Schema + golden seed** (`plans/mvp/plan.md`).

- Step 0: baseline green (below); `docs/spec/data-model.md` and
  `docs/spec/business-rules.md` exist and their field lists agree.
- Plan (read): `docs/spec/data-model.md` (all of it),
  `docs/spec/business-rules.md` (Parameters + Golden fixture sections).
- Do: `db/migrations/0001_init.sql` — every table, constraint, generated
  column, and index from `data-model.md`. `db/seed/golden.sql` — three demo
  users (one per role, bcrypt hashes of a documented dev password),
  locations with the Portuguese addresses from `tp.md` section 5, routes
  A/B/C on one date with the seeded stop times, parameters at defaults.
  Routes B and C use placeholder addresses per the brief.
  `db/seed/golden_check.sql` — assertion queries: totals 75/41/45, stop-1
  rows contribute 0, day total 161.
- Verify: W2 command sequence fresh: `scripts/testdb.sh --seed` then
  `psql "$TEST_DATABASE_URL" -f db/seed/golden_check.sql` exits 0 printing
  A=75, B=41, C=45; `nix develop -c bash -c 'cd backend && go test ./...'`
  still green (proves no regression).
- Stop-when: W2 green in this session, committed, pushed, handover rewritten.

## Baseline commands

```
git status                                        # clean tree
nix develop -c bash -c 'scripts/db-up.sh'         # only if the cluster is down
nix develop -c bash -c 'cd backend && go build ./... && go vet ./... && go test ./...'
```

`go test` with no test files prints `[no test files]` — that is green.
Cluster check: `nix develop -c bash -c 'pg_ctl status -D .pg/data'`.
`git add -A` before `nix develop` whenever a flake-referenced file was just
created — flakes only see tracked files (notes.md).

## Facts this task needs

- Parity commands **cd into `backend/`** first — the module lives there
  (monorepo layout, `architecture.md`); root-cwd spellings fail with
  "go.mod not found".
- Golden fixture numbers (do not re-derive): route A=75, B=41, C=45; stop 1
  of every route contributes 0 (RN01); day/month/period total 161; route A
  journey percent 15.625%; with `min_stop_minutes = 6` route B → 36.
  Full table in `notes.md` and `business-rules.md`.
- Migrations: `000N_name.sql`, filename order, one transaction per file,
  tracked in `schema_migrations` (created by `migrate.sh`; applied files are
  immutable — fixes are new files).
- psql is the oracle: verify SQL behavior against `stoptime_test` (rebuilt
  by `scripts/testdb.sh`), never against the dev database, never from
  memory.

## Open risks (subset relevant to T2)

- A spec bug discovered while writing `0001_init.sql` (wrong type, missed
  constraint) is fixed in the spec first, then the migration, in the same
  commit (card escape hatch) and recorded in progress.

## Out of scope

No Go product code (T3+); no store, no handlers. Do not design `0001` before
reading `data-model.md` in full. No golden_check numbers beyond the
assertions the card names.
