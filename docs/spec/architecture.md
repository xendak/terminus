# Architecture

One Go binary serving server-rendered HTML (htmx partial swaps) plus a JSON
mirror under `/api/*`, one PostgreSQL 18 database, no build pipeline. The
transport (htmx today, a React client if ever needed) is an adapter over an
operation-first service contract, so replacing it touches only the presentation
layer.

```mermaid
flowchart LR
    subgraph Browser
        P[Pages + htmx partials]
        C[Chart.js, fed aggregates only]
    end
    subgraph Go binary
        H[httpapi adapters\nparse, call service, format]
        S[app services\noperations, plain structs in/out]
        D[domain\npure rules, unit-tested]
        T[store\npgx, SQL + aggregation]
    end
    subgraph PostgreSQL
        DB[(stoptime / stoptime_test)]
    end
    P -->|GET/POST + htmx| H
    C <P
    H --> S --> T
    S --> D
    T <--> DB
```

## Monorepo layout

```
tp2/
├── tp.md                     professor's brief, immutable ground truth
├── AGENTS.md                 agent session protocol (read at session start)
├── README.md                 human entry point: what this is, how to run
├── flake.nix                 devshell: go 1.26, postgresql_18 (psql), tools
├── docs/
│   ├── method.md             multi-session rulebook
│   └── spec/                 this spec set
├── plans/                    session planning (method.md governs it)
├── scripts/                  db-init.sh, db-up.sh, db-down.sh, migrate.sh,
│                             testdb.sh — all thin psql/pg_ctl wrappers
├── db/
│   ├── migrations/           ordered plain .sql, applied via psql
│   └── seed/                 golden.sql (tp.md section 5), gen_year.sql
└── backend/                  Go module "stoptime"
    ├── cmd/server/           main.go: config, pool, router, listen
    └── internal/
        ├── domain/           pure rules from business-rules.md, no IO
        ├── store/            pgx repositories, SQL aggregation, audit writes
        ├── app/              operation services, plain structs, transactions
        ├── httpapi/          handlers (htmx + json adapters), middleware,
        │                     templates, error mapping
        └── web/              templates + static assets (htmx, chart.js, css)
```

There is no `frontend/` directory in the MVP: templates are part of the Go
binary. A future SPA becomes a `frontend/` package consuming the `/api/*`
adapters; nothing else moves.

## Layering rules

These are requirements, not preferences. A review or test that finds a
violation wins.

1. **Handlers only parse, call, and format.** A handler parses the request into
   an operation input, invokes a service function, and renders its output
   (template partial or JSON). Business rules in a handler are a defect.
   Grep guard: `grep -rn "SELECT\|INSERT\|UPDATE\|DELETE" backend/internal/httpapi/`
   must come back empty.
2. **Services speak plain structs.** Every operation has a Go input struct and
   output struct with no HTTP, SQL, or template types. `DashboardByMonth` must
   be callable today from an HTTP handler and tomorrow from a CLI or gRPC
   without changing a line of it.
3. **SQL owns queries and aggregation.** The dashboard series, route totals,
   and cost are computed by SQL (see `business-rules.md`), and the store
   returns them as plain structs. The chart receives aggregate data; it never
   receives raw rows to sum client-side.
4. **Domain is pure.** `internal/domain` holds the RN formulas and validation as
   functions over plain values: no database, no clock reads, no HTTP. Unit
   tests pin them to the golden fixture.
5. **Screens are use cases with states.** `screens.md` defines each screen by
   its states and the operations that feed them, not by markup. Swapping htmx
   for React replaces markup only; the state and operation contracts stay.
6. **Templates render operation outputs.** A template may branch on fields of a
   service struct and nothing else. No template calls the store or recomputes
   business math.

Transactions open in the service (via a store helper), so a composition change
plus its audit row commit or roll back together.

## Operation-first contract (the transport-switch seam)

The API contract in `operations.md` is written per operation: who may call it,
input struct, output struct, typed errors. Each operation then lists its
transports:

- an htmx path returning an HTML fragment or a redirect (current UI), and
- a JSON path under `/api/*` (used by tests today, by a SPA client if ever).

Both adapters are thin and call the same service function. Errors are sentinel
values in the service (`ErrDriverDateConflict`, `ErrRouteClosed`,
`ErrDepartureBeforeArrival`, `ErrForbidden`, ...); adapters map them to HTTP
status codes and to inline form errors respectively. Adding a transport means
adding an adapter, not changing the service.

## Environment and tooling

- **Nix devshell** (`flake.nix`, T1): Go 1.26, `postgresql_18` (which provides
  `psql`, `initdb`, `pg_ctl`), `gopls`. Enter with `nix develop`. This machine
  has no system psql; the devshell is the only supported way to get one
  (verified at bootstrap; see `plans/mvp/notes.md`).
- **Repo-local cluster**: `scripts/db-init.sh` runs `initdb` into `.pg/`
  (gitignored), `db-up.sh` starts it and creates the `stoptime` and
  `stoptime_test` databases, `db-down.sh` stops it. The cluster uses a
  unix-socket dir or a non-default port owned by the scripts; nothing global is
  touched.
- **Migrations**: `db/migrations/000N_name.sql`, applied in filename order by
  `scripts/migrate.sh` (psql, one transaction per file, recorded in
  `schema_migrations`). Applied migrations are immutable; fixes are new files.
- **Test database**: `scripts/testdb.sh` drops, recreates, migrates, and
  optionally seeds `stoptime_test`. `go test` integration tests connect to it.
- **Config**: environment variables only, `.env.example` documents them:
  `DATABASE_URL`, `LISTEN_ADDR` (default `127.0.0.1:8080`), `SESSION_KEY`
  (32+ bytes, required in production, a fixed dev default otherwise).
- **Baseline / parity commands** (every session, per `docs/method.md`):
  `go build ./... && go vet ./... && go test ./...` after the cluster is up.

## Dependency budget

Allowed, and the complete list:

| Dependency | Why |
| --- | --- |
| `github.com/jackc/pgx/v5` | Postgres driver + pool. The one real dependency. |
| `golang.org/x/crypto` | bcrypt password hashing. Quasi-stdlib. |
| vendored `htmx.min.js` | UI partial swaps, one file in `backend/web/static/`. |
| vendored `chart.umd.js` | the day/month/period charts. |
| vendored classless CSS (e.g. Pico) | responsive forms and tables without a CSS pipeline. |

Anything else needs a recorded decision in `plans/mvp/notes.md` first. No ORM,
no migration tool beyond psql, no node/npm at any point. Static assets are
served from the binary's `web/` directory; templates and pages contain no
external URLs (`grep -rn "https://cdn" backend/` stays empty).

## Security

- Passwords: bcrypt, cost 10.
- Sessions: HMAC-SHA256 signed cookie `st_session` carrying user id, role, and
  expiry; 12h lifetime; HttpOnly; SameSite=Lax; Secure when serving over TLS.
  Logout clears the cookie. Accepted tradeoff (no server-side revocation)
  recorded here; upgrade path is a `session` table behind the same middleware.
- Authorization: role checks live in services, not just middleware, so every
  caller (htmx, JSON, future CLI) enforces the same matrix. Driver scoping
  (own routes only) is a query-level filter, not a post-filter.
- Input: forms and JSON are decoded into structs and validated in the service;
  the store parameterizes every query (pgx); templates escape by default.
- Errors: adapters map sentinel errors to status codes; internal errors log
  server-side and return a generic body. No stack traces or SQL text to the
  client.

## Performance (RNF03)

Aggregation happens in one SQL query per dashboard cut, scanning indexed
`route(route_date)` and joining `route_stop`. No per-row loops in Go. T5
generates 12 months of synthetic routes and asserts wall-clock under 3s and
`EXPLAIN ANALYZE` using an index scan. At MVP data volume this is trivial; the
test guards the pattern, not just the number.

## Timezone policy

`timestamptz` in UTC everywhere; `route_date` is a `date` (no time component).
The server renders timestamps in America/Sao_Paulo; JSON returns RFC 3339 with
offset. "Now" for a driver's mark-arrival tap is server time, recorded once.
The golden seed uses -03:00 offsets so conversion bugs surface in tests.
