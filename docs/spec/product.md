# Product spec — StopTime (working title)

Working title for all internal naming (module, database, docs) is **StopTime**.
The user-facing product name is a graded extra ("melhor nome do produto") and is
chosen deliberately in card T11. Nothing in code or schema may hardcode a
product name that T11 cannot rename in one commit.

## Problem

Urban logistics and delivery companies cannot see where and how long their field
workers stay stopped during the daily route. There is no reliable record of how
long a courier, driver, or transporter stays at each point of the route. That
gap hides bottlenecks, blocks renegotiating deadlines with clients, and makes the
real cost of each route unknowable. The client asked for a dashboard with charts
of stopped time per day, per month, and per period, always tied to the points of
the route. (Source: `tp.md` section 1.)

## MVP goal

Build a minimum viable product that can:

1. Measure how long the courier stays stopped at each point of a daily route.
2. Persist points, routes, and the collected times.
3. Show a dashboard with stopped-time charts per day, per month, and per period.
4. Compute route cost indicators from distance, fuel price, and the vehicle's
   km per liter.

## Actors

| Actor | Who they are | What they need |
| --- | --- | --- |
| Admin | Transport company owner / system administrator | Full control: users, parameters, audit, all routes |
| Manager | Coordinator / dispatcher | Build routes, correct times, see dashboards and history, set cost parameters |
| Driver | Courier / motoboy, on a phone | See today's own route, mark arrival and departure at each stop, set distance |

## Scope

### In scope (from `tp.md` section 3.1)

- Register drivers/couriers, managers/coordinators, points (locations), and
  routes.
- Collect route point data: address, arrival date/time, departure date/time.
- Compute stopped time per point and total stopped time per route.
- History of points and stopped times per period, with addresses.
- Dashboard with charts per day, per month, per period.
- Cost parameters and the standard 8h workday parameterization.

### Out of scope (from `tp.md` section 3.2 — do not build)

- Automatic route planning or route optimization.
- Payroll or ERP integration.
- Real-time vehicle telemetry tracking.
- A native app published in app stores.

### Additional out-of-scope decisions (ours)

- ~~No SPA build pipeline in the MVP.~~ Superseded 2026-09-29 (user decision,
  `plans/mvp/notes.md`): the client is a Next.js app in `frontend/` over the
  `/api/*` JSON transports. The operation contract (`operations.md`) was the
  seam that made the switch possible without touching services or schema; the
  first htmx pages stay in the Go binary as a legacy transport.
- No Docker. The Nix devshell provides PostgreSQL; scripts manage the cluster.
- No ORM. Plain SQL through pgx; migrations are plain `.sql` files applied by
  psql.
- No i18n framework now. UI labels are English by default and centralized per
  screen so a translation layer can be added later (see "Language rules").

## Language rules

- All engineering artifacts (specs, plans, code, comments, commit messages) are
  in English.
- The UI is English by default. Every screen's visible strings live in one
  labels map per screen (see `screens.md`) so a pt-BR translation layer can be
  added after the MVP runs. Not a priority now.
- User-entered data is accepted in Portuguese (names, addresses, notes) with no
  validation that breaks on accented characters.
- Seed data keeps the Portuguese addresses from `tp.md` section 5 verbatim: they
  are the golden test fixture, not styling.
- The part-1 submission document, `docs/especificacao.md`, is the one Portuguese
  exception: its prose is strictly pt-BR. Its identifiers (tables, columns,
  classes, operations, screens, role values) stay verbatim English, copied from
  this spec set. A translated identifier describes a system that does not
  exist; the naming policy is recorded in `plans/mvp/notes.md`.

## Deliverables map (`tp.md` section 9)

| Deliverable | Where it lives | Delivered by card |
| --- | --- | --- |
| Dashboard with day/month/period charts | `backend/` dashboard screen | T9 |
| History of points and stopped times per period, with addresses | `backend/` history + route detail | T9 |
| Route data collection module (arrival/departure entry, order addresses) | `backend/` builder + tracker screens | T7, T8 |
| Cost parameters (fuel price, km/l, cost per km) | `parameter` table + params screen | T2, T9 |
| Stopped-time calculation parameters, 8h/day standard | `parameter` table + params screen | T2, T9 |
| Persistence layer: points, routes, drivers, managers | `db/migrations/` | T2 |
| Specification document (use cases, robustness, conceptual classes, crow's foot ER) | `docs/especificacao.md` (pt-BR) + `docs/especificacao/diagrams/` | drafted; T10 final review |
| Product name + campaign (extra points) | branding + campaign material | T11 |

## Requirements traceability

Functional requirements (`tp.md` section 6), the spec section that pins them, and
the card that implements them:

| ID | Requirement (short) | Spec | Card |
| --- | --- | --- | --- |
| RF01 | Register driver data (name, phone, document, vehicle) | `operations.md`, `data-model.md` | T4, T7 |
| RF02 | Register manager data (name, phone, email) | `operations.md`, `data-model.md` | T4, T7 |
| RF03 | Register points with address and coordinates | `operations.md`, `data-model.md` | T4, T7 |
| RF04 | Build the daily route: ordered points, driver, date | `operations.md`, `business-rules.md` | T4, T8 |
| RF05 | Record arrival and departure per point | `operations.md`, `business-rules.md` | T4, T8 |
| RF06 | Auto-compute stopped time per point and per route | `business-rules.md`, `data-model.md` | T2, T3 |
| RF07 | History of points and stopped times per period, with addresses | `operations.md` | T5, T9 |
| RF08 | Dashboard charts by day, month, period | `operations.md`, `screens.md` | T5, T9 |
| RF09 | Cost parameters: fuel price, km/l, cost per km | `business-rules.md`, `data-model.md` | T2, T9 |
| RF10 | Stopped-time calculation rules and 8h standard day | `business-rules.md` | T2, T3 |
| RF11 | Estimated route cost from parameters and distance | `business-rules.md` | T3, T5 |
| RF12 | Export period reports | `operations.md` | T9 |

Non-functional requirements (`tp.md` section 7):

| ID | Requirement (short) | Spec | Card |
| --- | --- | --- | --- |
| RNF01 | Database persistence, full route history | `data-model.md` | T2 |
| RNF02 | Responsive web UI, usable on the driver's phone | `architecture.md`, `screens.md` | T7–T9 |
| RNF03 | Dashboard under 3s for up to 12 months of data | `business-rules.md` (perf section), `architecture.md` | T5 |
| RNF04 | Access control by profile: driver, manager, admin | `operations.md` (role matrix) | T6 |
| RNF05 | Audit log of changes to points and times | `data-model.md`, `operations.md` | T2, T4 |
| RNF06 | LGPD compliance for field workers' personal data | `data-model.md` (LGPD section) | T2, T6 |

## Acceptance criteria (`tp.md` section 10)

Each criterion names the check that proves it:

1. The system does not count stopped time at the route's departure point.
   Check: golden tests for routes A/B/C assert stop order 1 contributes zero
   (T2, T3); the tracker UI shows no stopwatch on stop 1 (T8).
2. The dashboard shows the three requested cuts: day, month, period. Check:
   handler test asserts the three series payloads from seeded data (T9).
3. Every displayed stopped time is tied to a recorded address and date/time.
   Check: route detail and history rows join through `route_stop` → `location`
   and timestamps; integration test asserts no row renders without them (T5).
4. Cost and journey parameters change without code changes. Check: update
   `parameter` rows via the params screen, recompute, and totals move (T9);
   unit test pins that no business constant is hardcoded (T3).
