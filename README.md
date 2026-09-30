# Terminus

> **Resumo (pt-BR).** Terminus é o MVP do 2º Trabalho Avaliativo de
> Engenharia de Software II (PUC Minas): um sistema web que mede quanto tempo
> cada entregador fica parado em cada ponto do roteiro diário (o ponto de
> partida nunca conta) e mostra isso num painel por dia, por mês e por
> período, com endereço, horário, percentual da jornada de 8 h e custo
> estimado da rota. Backend em Go + PostgreSQL 18, cliente web em Next.js com
> interface em português. A especificação (1ª parte) está em
> [`docs/especificacao.md`](docs/especificacao.md) e a campanha de divulgação
> em [`docs/campanha.md`](docs/campanha.md).

MVP for the 2º Trabalho Avaliativo of Engenharia de Software II (PUC Minas):
a web system that measures how long delivery field workers stay stopped at
each point of their daily route, and shows it per day, per month, and per
period on a dashboard, always tied to the address and time of each stop,
with the share of the 8-hour workday and the estimated route cost.
Managers build routes, drivers record arrival and departure on their phone,
and cost and workday parameters change on a screen, not in code.

The requirements brief is [`tp.md`](tp.md) (Portuguese, professor's document,
immutable). The engineering specs derived from it live in
[`docs/spec/`](docs/spec/). The part-1 deliverable — the specification
document, in Portuguese — is
[`docs/especificacao.md`](docs/especificacao.md). The campaign material for
the graded extras (name rationale, pitch, tagline, screenshots, social posts)
is [`docs/campanha.md`](docs/campanha.md); its in-app version is the public
`/sobre` page.

(StopTime was the working title; it survives only in identifiers — the Go
module `stoptime`, the `stoptime`/`stoptime_test` databases, the `st_session`
cookie, the `@stoptime.dev` demo logins.)

## Stack

- **Backend**: Go 1.26 (stdlib `net/http`), pgx for PostgreSQL; JSON API under
  `/api/*`. The first server-rendered htmx pages (Chart.js, vendored) are
  still served as a legacy UI.
- **Frontend**: Next.js 16 (App Router, TypeScript, Tailwind CSS) in
  `frontend/`, pt-BR UI with a light/dark Nord palette (text at WCAG AA
  contrast); charts are plain React/SVG components fed by the API's
  aggregates, no chart library. It proxies `/api/*` to the Go server (Node 24
  + pnpm). Playwright for end-to-end tests.
- **Database**: PostgreSQL 18, repo-local cluster (Nix devshell or apt)

## The dashboard (Painel)

The Painel (`/painel`) was redesigned by Rafael Grossi (dashboard v2, merged
from the `dashboardv2` branch): KPI cards for the chosen window (total stopped
time, mean per route, share of the 8-hour workday, routes), a stopped-time bar
chart and an aggregate table by day (windows up to 30 days) or by month, a
per-driver ranking, and — for a single day — a per-driver timeline of the
day's stops plus a "Pontos do dia" list with each stop's number ("Ponto N" =
its order in the route), driver and address. Every bar, row and ranking entry
drills down to the pre-filtered history. Every number is computed by the API
in SQL; the page never sums rows itself.

Parameters (screen `/parametros`, stored in the `parameter` table, audited,
never hardcoded): fuel price, cost per km, default km/l, standard workday
hours (8), `min_stop_minutes` (stops shorter than this do not count toward
totals), and the Painel's stop highlight thresholds `stop_warn_minutes` (15)
and `stop_alert_minutes` (45, must be ≥ the warn value), which only colour
long stops and never change totals, percentages or cost.

## Repository layout

| Path | What it is |
| --- | --- |
| `tp.md` | Requirements brief (professor's, immutable) |
| `docs/especificacao.md` | Part-1 deliverable: the specification, in Portuguese, with its diagrams in `docs/especificacao/diagrams/` (PlantUML + rendered SVGs, and the crow's foot ER in Mermaid) |
| `docs/campanha.md`, `docs/campanha/` | Campaign material (pt-BR) and its screenshots |
| `docs/spec/` | The spec set: product, architecture, business rules, data model, operations, screens, use cases |
| `backend/` | Go module `stoptime`: services, JSON API, legacy htmx pages |
| `frontend/` | Next.js client: `app/` (pages, pt-BR routes such as `/painel`, `/hoje`, `/historico`, `/sobre`), `components/`, `lib/`, `e2e/` (Playwright); see `frontend/README.md` |
| `db/migrations/`, `db/seed/` | Plain SQL migrations, the golden seed, and the demo seed (`db/seed/demo/`) |
| `scripts/` | Database cluster and migration helpers (psql wrappers) |
| `plans/` | Session planning for agents and humans alike (`docs/method.md` is the rulebook) |
| `AGENTS.md` | Operating instructions for AI coding agents |

## Running it

Two ways to get the toolchain (Go 1.26 + PostgreSQL 18, plus Node 24 + pnpm
for the web client); the scripts are the same afterwards.

**Nix** (the pinned devshell):

```
nix develop        # Go, PostgreSQL, PlantUML, Node 24 and pnpm on PATH
```

**Ubuntu without Nix**: install PostgreSQL 18 from the PGDG apt repository
(`apt install postgresql-18`; the scripts only need its binaries, not the
system service) and Go 1.26 under `/usr/local/go`, then put both on PATH
(the web client also needs Node 24 and pnpm):

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

The web client, in a second terminal (Node 24 + pnpm):

```
cd frontend && pnpm install && pnpm dev   # http://localhost:3210, proxies /api/* to the Go server
```

Open <http://localhost:3210> and log in (below). The public campaign page is
<http://localhost:3210/sobre>. The Go server also still serves the first
server-rendered htmx pages on <http://127.0.0.1:8080> as a legacy UI.

`scripts/dev-seed.sh` truncates the dev database and loads the golden
fixture plus ~8 weeks of demo routes relative to today (Belo Horizonte
addresses, active routes today for drivers A, B and C with their first
deliveries done — so the Painel's timeline has several rows — and a draft for
tomorrow for driver B). Demo logins, all with password `stoptime-dev`:

| Email | Role |
| --- | --- |
| `admin@stoptime.dev` | admin |
| `manager@stoptime.dev` | manager |
| `driver-a@stoptime.dev` | driver (Marcos Motorista) |
| `driver-b@stoptime.dev` | driver (Bianca Batista) |
| `driver-c@stoptime.dev` | driver (Carla Camargo) |

Set `SESSION_KEY` (32+ bytes) before serving beyond your machine; without it
the server warns and uses a fixed dev key.

## Tests

Backend: unit and integration tests run against a separate `stoptime_test`
database, recreated from scratch by `scripts/testdb.sh`, that the demo data
never touches (`-p 1`: the packages share that database):

```
scripts/testdb.sh && cd backend && go build ./... && go vet ./... && go test -count=1 -p 1 ./internal/...
```

Frontend: lint, production build, and the Playwright end-to-end suite
(desktop and a 375 px phone). The e2e suite drives the real stack — both
servers up, dev database seeded — and writes routes into the dev database,
so reseed afterwards for a clean demo:

```
cd frontend && pnpm lint && pnpm build && pnpm test:e2e
scripts/dev-seed.sh
```

## How this repo is worked on

Every work session (agent or human) follows the protocol in
[`AGENTS.md`](AGENTS.md) and the rulebook in
[`docs/method.md`](docs/method.md): resume from `plans/README.md` → the
active task's `handover.md` → work one card → verify → commit with the plan
files → rewrite the handover. The plan folder is the source of truth for where
the work stands, not anyone's memory.
