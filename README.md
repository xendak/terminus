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

The dev environment lands with plan card T1: a Nix devshell providing Go and
PostgreSQL, plus scripts that initialize and start a repo-local database
cluster. Until then there is nothing to run; after T1 the quickstart will be:

```
nix develop        # Go + PostgreSQL toolchain
scripts/db-init.sh # first time only: initdb into .pg/
scripts/db-up.sh   # start the cluster, create the databases
scripts/migrate.sh # apply db/migrations/*.sql
go run ./cmd/server
```

## How this repo is worked on

Every work session (agent or human) follows the protocol in
[`AGENTS.md`](AGENTS.md) and the rulebook in
[`docs/method.md`](docs/method.md): resume from `plans/README.md` → the
active task's `handover.md` → work one card → verify → commit with the plan
files → rewrite the handover. The plan folder is the source of truth for where
the work stands, not anyone's memory.
