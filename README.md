# StopTime (working title)

MVP for the 2º Trabalho Avaliativo of Engenharia de Software II (PUC Minas):
a web system that measures how long delivery field workers stay stopped at
each point of their daily route, and shows it per day, per month, and per
period on a dashboard.

The requirements brief is [`tp.md`](tp.md) (Portuguese, professor's document,
immutable). The engineering specs derived from it live in
[`docs/spec/`](docs/spec/). The part-1 deliverable — the specification
document, in Portuguese — is
[`docs/especificacao.md`](docs/especificacao.md).

## Stack

- **Backend**: Go (stdlib `net/http`, `html/template`), pgx for PostgreSQL
- **Frontend**: server-rendered pages with htmx; Chart.js for the dashboard
  graphs; both vendored, no build pipeline
- **Database**: PostgreSQL 18 via a Nix devshell, repo-local cluster

## Repository layout

| Path | What it is |
| --- | --- |
| `tp.md` | Requirements brief (professor's, immutable) |
| `docs/especificacao.md` | Part-1 deliverable: the specification, in Portuguese, with its diagrams in `docs/especificacao/diagrams/` (PlantUML + rendered SVGs, and the crow's foot ER in Mermaid) |
| `docs/spec/` | The spec set: product, architecture, business rules, data model, operations, screens, use cases |
| `backend/` | Go module (created in plan card T1) |
| `db/migrations/`, `db/seed/` | Plain SQL migrations and the golden seed |
| `scripts/` | Database cluster and migration helpers (psql wrappers) |
| `plans/` | Session planning for agents and humans alike (`docs/method.md` is the rulebook) |
| `AGENTS.md` | Operating instructions for AI coding agents |

## Running it

Two ways to get the toolchain (Go 1.26 + PostgreSQL 18); the scripts are the
same afterwards.

**Nix** (the pinned devshell):

```
nix develop        # Go + PostgreSQL toolchain on PATH
```

**Ubuntu without Nix**: install PostgreSQL 18 from the PGDG apt repository
(`apt install postgresql-18`; the scripts only need its binaries, not the
system service) and Go 1.26 under `/usr/local/go`, then put both on PATH:

```
export PATH=$PATH:/usr/local/go/bin:/usr/lib/postgresql/18/bin
```

Then, from the repo root:

```
scripts/db-init.sh             # first time only: initdb into .pg/ (port 5543, unix socket .pg/sock)
eval "$(scripts/db-up.sh)"     # start the cluster, create the databases, export DATABASE_URL / TEST_DATABASE_URL
scripts/migrate.sh             # apply db/migrations/*.sql to the dev database
scripts/dev-seed.sh            # optional: reset the dev database to the demo dataset
cd backend && go run ./cmd/server   # http://127.0.0.1:8080 (LISTEN_ADDR to change)
```

`scripts/dev-seed.sh` truncates the dev database and loads the golden
fixture plus ~8 weeks of demo routes relative to today (Belo Horizonte
addresses, an active route today for driver A, a draft for tomorrow for
driver B). Demo logins, all with password `stoptime-dev`:

| Email | Role |
| --- | --- |
| `admin@stoptime.dev` | admin |
| `manager@stoptime.dev` | manager |
| `driver-a@stoptime.dev` | driver (Marcos Motorista) |
| `driver-b@stoptime.dev` | driver (Bianca Batista) |
| `driver-c@stoptime.dev` | driver (Carla Camargo) |

Set `SESSION_KEY` (32+ bytes) before serving beyond your machine; without it
the server warns and uses a fixed dev key.

Tests run against a separate `stoptime_test` database that the demo data
never touches:

```
scripts/testdb.sh && cd backend && go build ./... && go vet ./... && go test -count=1 -p 1 ./internal/...
```

## How this repo is worked on

Every work session (agent or human) follows the protocol in
[`AGENTS.md`](AGENTS.md) and the rulebook in
[`docs/method.md`](docs/method.md): resume from `plans/README.md` → the
active task's `handover.md` → work one card → verify → commit with the plan
files → rewrite the handover. The plan folder is the source of truth for where
the work stands, not anyone's memory.
