# References — mvp

## In this repo

| Pointer | What it is |
| --- | --- |
| `tp.md` | The professor's requirements brief. Immutable ground truth. |
| `AGENTS.md` | Session protocol for agents working this repo. |
| `docs/method.md` | The multi-session rulebook this plan runs on. |
| `docs/spec/product.md` | Scope, actors, deliverables map, traceability. |
| `docs/spec/architecture.md` | Layers, monorepo layout, environment, security. |
| `docs/spec/business-rules.md` | RN01–RN07 formalized; golden fixture; parameters. |
| `docs/spec/data-model.md` | ER, tables, constraints, indexes, audit, LGPD. |
| `docs/spec/operations.md` | The operation-first API contract and role matrix. |
| `docs/spec/screens.md` | Screens as use cases with states. |
| `docs/spec/use-cases.md` | Part-1 deliverable draft (UC, robustness, classes). |

## External (to consult, never to reconstruct from memory)

- pgx v5: https://github.com/jackc/pgx — pool, tx, scanning rules.
- PlantUML: https://plantuml.com/ — use case and class diagram syntax.
  Robustness has no dedicated docs page; the working syntax
  (boundary/control/entity keywords) is verified empirically and recorded in
  `plans/mvp/notes.md`.
- Mermaid `erDiagram` (crow's foot):
  https://mermaid.js.org/syntax/entityRelationshipDiagram.html
- Go stdlib `net/http` routing patterns (Go 1.22+ method+path mux) and
  `html/template` — auto-escaping behavior.
- htmx: https://htmx.org/docs/ — attribute reference for partial swaps.
- Chart.js: https://www.chartjs.org/docs/ — bar charts for the day/month series.
- PostgreSQL 18 docs — generated columns, `timestamptz`, `date`,
  `EXPLAIN ANALYZE`, numeric arithmetic. psql is the oracle for SQL behavior.
- RFC 4180 — CSV format for the export endpoint.
- bcrypt (Provos & Mazières) and `golang.org/x/crypto/bcrypt` — cost parameter.
- LGPD (Lei 13.709/2018) — the compliance points we implement are data
  minimization, role-based access, and a documented removal path
  (`docs/spec/data-model.md`, LGPD section).
- The method's master copy: `/home/xendak/Programming/xendak/zorai/wzcli/docs/method.md`
  (this repo's `docs/method.md` is its snapshot; one example adapted to Go).

## Vendored assets (downloaded once in T7, committed into backend/web/static/)

- htmx minified single file.
- Chart.js UMD build.
- One classless CSS (Pico or equivalent) for responsive defaults.
