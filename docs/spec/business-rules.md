# Business rules

This file is the authority for every stopped-time and cost calculation in the
system. Any session touching time math reads this first. The worked examples in
"Golden fixture" come from `tp.md` section 5 and are the golden test data for
the SQL layer (`db/seed/golden.sql`), the domain package, and the dashboard.

## Terms

- **Route** (`roteiro`): one driver's ordered set of stops for one calendar
  date.
- **Stop** (`ponto` in a route): one location visited in a route, at a given
  order. Order 1 is the **departure point**, where the day starts.
- **Counted stop**: a stop with `stop_order > 1`. Only counted stops accrue
  stopped time.
- **Arrival / departure**: the timestamps recorded at a stop.

## Rules (from `tp.md` section 4, made testable)

### RN01 — departure point never counts

Stop order 1 accrues no stopped time, ever. It may carry timestamps (for
example, when the driver leaves the base), and the system stores them, but they
add zero minutes to every total. The stopwatch starts at the second point.

Enforcement: the `route_stop` generated column returns 0 seconds for order 1
(see `data-model.md`), so no aggregation can accidentally include it. The
tracker UI shows no stopwatch on stop 1.

### RN02 — stopped time per stop

```
stop_seconds(stop) = departure_at - arrival_at          (full seconds)
stop_minutes(stop) = floor(stop_seconds / 60)           (display only)
```

Constraint: `departure_at >= arrival_at`, enforced by a check constraint in the
database and validated in the service before the write. A violation is rejected
with error `ErrDepartureBeforeArrival` (HTTP 422).

If either timestamp is null, the stop is open or untouched and contributes 0 to
totals until both are set.

### RN03 — total stopped time per route

```
route_total_seconds = SUM(stop_seconds) over counted stops, honoring
                      min_stop_minutes (see Parameters)
route_total_minutes = floor(route_total_seconds / 60)
```

Totals are computed from seconds, then floored once for display. Never sum
already-floored per-stop minutes; the golden examples pin this: three stops of
59, 59, and 59 seconds are 2 minutes total (floor(177/60)), not 0.

Totals are computed on read (SQL aggregation), never stored on the route row.
No stored total can go stale.

### RN04 — standard workday as percentage base

```
journey_percent = route_total_seconds / (standard_journey_hours * 3600) * 100
```

`standard_journey_hours` is a parameter, default 8 (480 minutes = 100%).
Example: 75 minutes stopped on an 8h day = 15.6%. The dashboard shows this
percentage per route and per day.

**Aggregates (day, period, per driver): one standard day per route.** The 8h
base is a workday of ONE driver, and a route is exactly one driver-day (RN05).
So any bucket that sums several routes divides by as many standard days as it
has routes:

```
journey_percent = total_stopped_seconds
                  / (routes_count * standard_journey_hours * 3600) * 100
```

rounded once, in SQL, to 3 places. `routes_count` counts the bucket's worked
routes — routes with at least one recorded stop interval; a planned route with
nothing recorded is not a worked day and stays out of the base. With one route
the formula reduces to the per-route one. It applies to the period grand total,
each `by_driver` row (that driver's own routes), each day point of the by-day
series (that day's routes), and each month point of the by-month series (that
month's routes) — every dashboard bucket carries its percent.

This replaces an earlier reading ("period total over ONE standard day"), which
produced percentages above 100% for any multi-day window (e.g. 30 days of
three drivers read 644%) and so did not measure what RN04 describes: the share
of the workday spent stopped.

### RN05 — one route per driver per date

A route belongs to exactly one driver and exactly one date. The database
enforces `UNIQUE (driver_user_id, route_date)`. Creating a second route for the
same driver and date fails with `ErrDriverDateConflict` (HTTP 409), and the UI
pre-checks and links to the existing route.

### RN06 — stops have sequential order

Stops are numbered 1..n dense (1, 2, 3, 4...). The database enforces
`UNIQUE (route_id, stop_order)` and `stop_order >= 1`. Reordering (move up/down,
remove) renumbers to close gaps in the same transaction. Order defines the
day's path and identifies stop 1 as the departure point.

**Time sequence.** The order is also the order of the day, so times follow it:

- An arrival at stop n must not be earlier than the departure recorded at
  stop n-1; a departure at stop n must not be later than an arrival already
  recorded at stop n+1. This holds for every path — driver records and manager
  corrections (UpdateStopTimes).
- Recording (the driver's flow): arriving at stop n (n ≥ 3) requires the
  departure from stop n-1 to be recorded first — the driver leaves one stop
  before reaching the next. Stop 1 is the exception: it is the departure point
  with no stopwatch (RN01), so its departure is optional; when recorded it
  bounds the arrival at stop 2.
- Corrections (manager/admin) may fill a stop whose predecessor has no
  departure yet (reconstructing a day after the fact), but never out of order.

Violations are `ErrStopTimesOutOfOrder` (422) with the offending field
(`arrival_at` / `departure_at`) and a reason. The pure check is
`domain.ValidateStopSequence`.

### RN07 — route cost

```
km_per_l        = driver's vehicle km/l if set, else the default parameter
liters          = distance_km / km_per_l
fuel_cost       = liters * fuel_price
overhead        = distance_km * cost_per_km
estimated_cost  = fuel_cost + overhead
```

All money math happens in SQL `numeric` (exact decimal), rounded once, to 2
places, at the end. Cost uses the parameter values current at read time. Editing
a parameter therefore recalculates history on the next read; that is the
accepted MVP semantics and it satisfies the acceptance criterion that parameters
change without code changes.

`distance_km` is entered manually per route (odometer or estimate), because
route optimization and GPS telemetry are out of scope (`tp.md` section 3.2). A
route with no distance yet shows no cost, not zero cost.

## Parameters (`tp.md` sections 3.1, 6 RF09/RF10)

All live in the `parameter` table, editable in the UI, never hardcoded:

| Key | Type | Default | Used by |
| --- | --- | --- | --- |
| `fuel_price_brl` | numeric(10,2) | 6.09 | RN07 |
| `cost_per_km_brl` | numeric(10,2) | 0.00 | RN07 |
| `standard_journey_hours` | numeric(4,2) | 8.00 | RN04 |
| `min_stop_minutes` | int | 0 | RN03 rule below |
| `default_km_per_l` | numeric(6,2) | 10.00 | RN07 fallback |

`min_stop_minutes` implements RF10's "calculation rules" parameterization: when
greater than 0, a stop whose `stop_minutes` is below the threshold still
records its timestamps (and `stop_seconds`) but contributes 0 to route totals.
The route detail reports it as `counted: false, below_min: true` — `counted`
means "adds to the total" (false for stop 1 and for below-minimum stops).
Default 0 keeps RN03 pure. The domain unit tests pin both behaviors; with
`min_stop_minutes = 12`, route A's 10-minute stop is not counted and route A
totals 65 minutes.

## Golden fixture (`tp.md` section 5)

The canonical test data. Addresses stay in Portuguese verbatim.

| Route | Stop / address | Stop time | Note |
| --- | --- | --- | --- |
| A | 1 — Seg. Família (departure) | 0 | departure point, never counted |
| A | 2 — Rua Peru, 55 | 15 min | intermediate |
| A | 3 — Rua X, 5 | 10 min | intermediate |
| A | 4 — Av. João César | 50 min | final point |
| B | 1 — Partida | 0 | never counted |
| B | 2 | 10 min | |
| B | 3 | 5 min | |
| B | 4 | 26 min | |
| C | 1 — Partida | 0 | never counted |
| C | 2 | 5 min | |
| C | 3 | 10 min | |
| C | 4 | 30 min | |

Assertions every layer must reproduce (T2 seeds these and asserts via SQL; T3
pins them in domain unit tests; T5 reuses them for dashboard series):

- Route A total: 75 minutes. Route B: 41. Route C: 45.
- Stop 1 of each route contributes exactly 0 even when seeded with timestamps.
- Journey percent of route A at default 8h: 75/480 = 15.625%.
- Period / golden day journey percent: 161 / (3 routes × 480) = 11.181%
  (exact 805/72); each driver's row equals its single route's percent
  (A 15.625, B 8.542, C 9.375).
- With `min_stop_minutes = 6`, route B's 5-minute stop contributes 0, so route B
  total becomes 36 minutes.

Seed convention: all three routes on the same date makes the by-day series show
(75 + 41 + 45) = 161 minutes for that day; the by-month series shows 161 for
that month; the period total shows 161 across the range.

## Edge cases

- **Stop spanning midnight**: timestamps are `timestamptz`; a stop that crosses
  midnight belongs to its route's `route_date` and counts in full. No day
  splitting.
- **Correction after the fact**: `UpdateStopTimes` (see `operations.md`) changes
  an already-set timestamp, writes an audit row, and totals recompute on next
  read by construction.
- **Closed route**: `status = closed` freezes stop times and composition. Writes
  to a closed route fail with `ErrRouteClosed` (HTTP 409). Reopening is an
  admin action that itself audits.
- **Duplicate times**: setting arrival when arrival is already set is a
  correction (manager/admin, audited), not a fresh record. The driver's
  "mark arrival" button only ever fills a null field.
- **Timezone**: storage is UTC `timestamptz`; `route_date` is a `date`. "Now"
  comes from the server clock. The UI formats in America/Sao_Paulo. Golden seed
  uses -03:00 offsets to catch conversion bugs.

## Performance criterion (RNF03)

Dashboard aggregation over a 12-month window must answer in under 3 seconds.
The queries aggregate in SQL over indexed columns (`route(route_date)`,
`route_stop(route_id)`); a synthetic 12-month seed (T5) measures wall-clock and
runs `EXPLAIN ANALYZE` to confirm index usage. At course scale this lands in
milliseconds; the test exists so a regression (accidental per-row queries from
Go, missing index) fails loudly.
