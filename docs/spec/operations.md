# Operations (API contract)

The contract is per operation, not per endpoint. An operation is: who may call
it, a plain input struct, a plain output struct, and typed errors. Transports
are thin adapters over it. Today every operation has an htmx transport (form
posts returning fragments or redirects) and most have a JSON transport under
`/api/*` (used by tests, available for a future SPA). Adding a transport adds
an adapter; it never changes the service.

Conventions:

- IDs are UUIDs. JSON keys are `snake_case`. Timestamps are RFC 3339 with
  offset. Durations are whole minutes unless a field says `seconds`.
- Calendar days (`route_date`, the day series `date`) are plain
  `"YYYY-MM-DD"` strings — a SQL `date` has no time or zone. Months are
  `"YYYY-MM"`.
- Exact decimals travel as JSON **strings**, never floats: `journey_percent`
  (3 places, e.g. `"15.625"`), `distance_km` (2 places), `estimated_cost_brl`
  (2 places), parameter `value`, `km_per_l`. They come straight from SQL
  `numeric` rounding; clients parse them for display and never re-round or sum
  them. Nullable ones (`distance_km`, `estimated_cost_brl`) are `null` when
  unknown — never `"0"`. Whole counts and minutes are JSON numbers.
- Money is BRL rounded to 2 places (SQL numeric, rounded once), sent as a
  string per the rule above.
- Inputs are validated in the service; failures return field errors, not
  generic 400s, when the transport can show them (htmx forms).

## Error model

| Sentinel | Meaning | JSON status |
| --- | --- | --- |
| ErrBadInput | malformed or missing field | 400 |
| ErrValidation | field-level validation failure (details attached) | 422 |
| ErrUnauthenticated | no or invalid session | 401 |
| ErrForbidden | role or ownership violation | 403 |
| ErrNotFound | unknown id | 404 |
| ErrDriverDateConflict | route already exists for driver+date (RN05) | 409 |
| ErrRouteClosed | mutation on a closed route | 409 |
| ErrDepartureBeforeArrival | violates RN02 constraint | 422 |
| ErrStopTimesOutOfOrder | stop times break the route sequence (RN06 sequence; carries `field`/`reason`) | 422 |
| ErrDuplicateEmail | email already registered | 409 |

JSON error body (every `/api/*` failure, including the anonymous 401):

```
{"error": "<message>", "field": "<field>", "reason": "<reason>"}
```

`error` is always present. `field` and `reason` are present only for
field-level validation failures (422 from a `FieldError`, e.g.
`{"error": "arrival_at: already recorded; use UpdateStopTimes", "field":
"arrival_at", "reason": "already recorded; use UpdateStopTimes"}`). Clients
branch on the HTTP status, not on the message text. A 500 answers
`{"error": "internal error"}`; the real error is logged server-side only.

htmx adapters map the same sentinels to inline form errors and flash messages.

## Role matrix

| Operation | Admin | Manager | Driver |
| --- | --- | --- | --- |
| Login / Logout | yes | yes | yes |
| CurrentUser | yes | yes | yes |
| CreateDriver, UpdateDriver, ListDrivers | yes | yes | no |
| AnonymizeDriver | yes | no | no |
| CreateManager, UpdateManager, AnonymizeManager | yes | no | no |
| ListManagers | yes | minimized | no |
| CreateLocation, UpdateLocation, ListLocations | yes | yes | no |
| CreateRoute, AddStop, RemoveStop, ReorderStops | yes | yes | no |
| StartRoute, CloseRoute | yes | yes | own route |
| ReopenRoute | yes | no | no |
| RecordArrival, RecordDeparture | yes | yes | own route |
| UpdateStopTimes (correction) | yes | yes | no |
| SetRouteDistance | yes | yes | own route |
| GetRoute, ListRoutes | all | all | own only |
| GetDashboardByDay/Month/Period | yes | yes | own data only |
| GetParams, UpdateParam | yes | yes | no |
| ExportPeriodCSV | all | all | own data only |
| ListAudit | yes | no | no |

Drivers see only their own routes in every read operation; the store filters by
the session's user id, it never post-filters.

## Operations

### Auth

**Login**
Input: `{email, password}`. Output: `{user, expires_at}` + sets session cookie
(`st_session`, HttpOnly, SameSite=Lax). `user` is
`{id, name, email, phone, role, active}` (never the password hash).
Errors: ErrUnauthenticated.
Transports: `POST /login` (form, redirect), `POST /api/auth/login` (JSON).

Every authenticated request re-validates the session (`Services.Authenticate`):
a user deleted, deactivated or anonymized since login gets 401 (JSON) or a
redirect to login (pages) on the next request, and the cookie is cleared.

**Logout**
Input: none. Output: clears cookie. Transports: `POST /logout`,
`POST /api/auth/logout` (204, no body).

**CurrentUser**
Input: none (the session). Output: `{user, expires_at}` — same shape as
Login; `user` is re-read from the database so name and active flag are
fresh, `expires_at` is the session's expiry.
Errors: ErrUnauthenticated (anonymous, or the user was deactivated since login).
Transports: `GET /api/auth/me` (JSON only; the SPA's boot check).

### Directories

**CreateDriver**
Input: `{name, email, password, phone, document?, vehicle_name?, vehicle_plate?, km_per_l?, manager_user_id?}`.
Output: `{driver}` (user + profile). Errors: ErrDuplicateEmail, ErrValidation
(including `password` shorter than 8 characters).
Creates the `app_user` (role driver) and `driver_profile` in one transaction.
Transports: `POST /drivers`, `POST /api/drivers`.

**UpdateDriver**
Input: `{driver_id, name?, phone?, document?, vehicle_name?, vehicle_plate?, km_per_l?, manager_user_id?, active?}`.
Output: `{driver}`. Deactivating (LGPD removal path) sets `active = false`.
JSON body: an absent key keeps the field; for the optional profile fields
(`document`, `vehicle_name`, `vehicle_plate`, `km_per_l`) an explicit `null`
clears it — `km_per_l: null` falls back to the `default_km_per_l` parameter.
`name`, `phone`, `active` cannot be cleared (null = keep). In the `driver`
output, unset optional fields are omitted.
Transports: `POST /drivers/{id}/edit`, `PATCH /api/drivers/{id}`.

**AnonymizeDriver** (RNF06 full erasure)
Input: `{driver_id}`. Output: `{driver}` — the pseudonymized driver:
`name: "Motorista removido <id[:8]>"`, `email: "removido-<id>@anonimo.invalid"`,
`phone: ""`, `active: false`, `document`/`vehicle_name`/`vehicle_plate` absent
(NULL), `km_per_l` kept. One transaction plus an `anonymize` audit row
(entity `app_user`) listing the cleared fields, not their values. Routes and
aggregates are untouched. Idempotent: a second call answers 200 with the same
driver and writes nothing. Errors: ErrNotFound (unknown id or not a driver),
ErrForbidden (non-admin). Details: data-model.md "LGPD approach".
Transports: `POST /api/drivers/{id}/anonymize` (JSON, no body).

**ListDrivers**
Input: `{active_only?}`. Output: `{drivers: [driver]}`.

A `driver` object (all driver operations) is
`{id, name, email, phone, role, active, document?, document_masked,
vehicle_name?, vehicle_plate?, km_per_l?, manager_user_id?, manager_name?}`;
unset optional fields are omitted. `manager_user_id` / `manager_name` name the
driver's responsible manager (tp.md §8 team; data-model.md driver_profile). It
must be an active manager (ErrValidation on `manager_user_id` otherwise); in
UpdateDriver's JSON an explicit `null` clears it, absent keeps. It is an
attribute, not an access rule: every manager still sees every driver.
For the manager role `document` is masked (every digit but the last two →
`*`, e.g. `"***.***.***-11"`) and `document_masked` is `true`; admin gets the
full value and `false` (RNF06, data-model.md "LGPD approach"). Sending the
masked value back in UpdateDriver keeps the stored document.
Transports: `GET /drivers`, `GET /api/drivers`.

**CreateManager**
Input: `{name, email, password, phone}`. Output: `{manager}`.
Errors: ErrDuplicateEmail, ErrValidation (`password` shorter than 8 characters).
Transports: `POST /managers`, `POST /api/managers`.

**ListManagers**
Input: none. Output: `{managers}`. Transports: `GET /managers`, `GET /api/managers`.
Admin gets full manager objects. A manager caller (e.g. choosing a driver's
responsible manager) gets the minimized row `{id, name, active, team_size}` —
email and phone are removed in the service (RNF06 minimization), not merely
hidden. Drivers: 403.

A `manager` object is `{id, name, email, phone, role, active, team_size}` —
`team_size` counts the active drivers whose responsible manager it is.

**UpdateManager**
Input: `{manager_id, name?, phone?, active?}`. Output: `{manager}`. Admin only.
Absent keeps; `name`/`phone` cannot be emptied (ErrValidation); email is
immutable (as for drivers). `active: false` is the LGPD deactivation path —
the manager's open sessions end on their next request. Not audited (like
UpdateDriver). Errors: ErrNotFound (unknown id or not a manager).
Transports: `PATCH /api/managers/{id}`.

**AnonymizeManager** (RNF06 full erasure)
Input: `{manager_id}`. Output: `{manager}` with `name: "Gestor removido
<id[:8]>"`, `email: "removido-<id>@anonimo.invalid"`, `phone: ""`,
`active: false`. Same transaction, audit row and idempotency as
AnonymizeDriver (cleared: name, email, phone, password_hash).
Transports: `POST /api/managers/{id}/anonymize` (JSON, no body).

**CreateLocation**
Input: `{label, address, latitude?, longitude?}`. Output: `{location}`.
Transports: `POST /locations`, `POST /api/locations`.

**UpdateLocation**
Input: `{location_id, label?, address?, latitude?, longitude?}`. Output: `{location}`.
Audited (`update_location`, entity `location`, old/new `{label, address,
latitude, longitude}`). Existing route stops keep their snapshot of the
location (data-model.md, route_stop), so past routes, history and the CSV
never change; routes created afterwards use the new values.
Transports: `POST /locations/{id}/edit`, `PATCH /api/locations/{id}`.

**ListLocations**
Input: none (optionally `{q}` free-text filter). Output: `{locations}`.
Transports: `GET /locations`, `GET /api/locations`.

### Route composition

**CreateRoute**
Input: `{driver_user_id, route_date, location_ids: [uuid, ...] in visit order, note?}`.
Output: `{route}` with stops assigned orders 1..n, status `draft`.
Errors: ErrDriverDateConflict (RN05), ErrValidation (needs at least 2 stops so a
counted stop can exist).
The first location is the departure point (stop 1).
Transports: `POST /routes`, `POST /api/routes`.

**AddStop**
Input: `{route_id, location_id, position? (default: end)}`. Output: `{route}` (renumbered).
Draft/active routes only. Audited (`add_stop`).
Transports: `POST /routes/{id}/stops`, `POST /api/routes/{id}/stops`.

**RemoveStop**
Input: `{route_id, stop_order}`. Output: `{route}` (renumbered). Draft/active
only. Audited (`remove_stop`). Stop removal renumbers so orders stay dense (RN06).
Transports: `POST /routes/{id}/stops/{order}/remove`, `DELETE /api/routes/{id}/stops/{order}`.

**ReorderStops**
Input: `{route_id, stop_order, direction: up|down}`. Output: `{route}`.
Audited (`reorder`). One-position moves only; drag-and-drop is out of scope.
Transports: `POST /routes/{id}/stops/{order}/move`, `POST /api/routes/{id}/stops/{order}/move`.

**StartRoute**
Input: `{route_id}`. Output: `{route}` (status `active`).
Transports: `POST /routes/{id}/start`, `POST /api/routes/{id}/start`.

**CloseRoute**
Input: `{route_id, distance_km?}`. Output: `{route}` (status `closed`; freezes
times and composition). Audited (`close_route`).
Transports: `POST /routes/{id}/close`, `POST /api/routes/{id}/close`.

**ReopenRoute**
Input: `{route_id}`. Output: `{route}`. Admin only. Audited (`reopen_route`).
Transports: `POST /routes/{id}/reopen`, `POST /api/routes/{id}/reopen`.

### Time recording (RF05)

**RecordArrival**
Input: `{route_id, stop_order, at? (default: now)}`. Output: `{stop}`.
Sets `arrival_at` only if currently null; otherwise it is a correction and goes
through UpdateStopTimes. Active routes only. Drivers: own route.
RN06 sequence (business-rules.md): for stop n ≥ 3 the departure from stop n-1
must already be recorded, and the arrival must not precede it; for stop 2 a
recorded stop-1 departure is a lower bound. Violations: ErrStopTimesOutOfOrder,
`field: "arrival_at"`.
Transports: `POST /routes/{id}/stops/{order}/arrive`, `POST /api/routes/{id}/stops/{order}/arrive`.

**RecordDeparture**
Input: `{route_id, stop_order, at? (default: now)}`. Output: `{stop}`.
Same null-only rule. Requires `arrival_at` set. Must not be later than an
arrival already recorded at stop n+1 (ErrStopTimesOutOfOrder,
`field: "departure_at"`).
Transports: `POST /routes/{id}/stops/{order}/depart`, `POST /api/routes/{id}/stops/{order}/depart`.

**UpdateStopTimes** (correction, RNF05-audited)
Input: `{route_id, stop_order, arrival_at?, departure_at?}`. Output: `{stop}`.
Manager/admin only. Any change to an already-set timestamp writes an audit row
(`update_times`) with old and new values. Validated against RN02
(ErrDepartureBeforeArrival) and the RN06 time sequence against the
neighbouring stops (ErrStopTimesOutOfOrder); unlike RecordArrival it may fill a
stop whose predecessor has no departure yet. Closed routes: ErrRouteClosed
(reopen first).
JSON body: `{"arrival_at"?: RFC 3339, "departure_at"?: RFC 3339}`; an absent
or empty field keeps the current value; malformed is 400. Answers
`{"stop": {id, route_id, stop_order, location_id, arrival_at, departure_at,
note}}` like RecordArrival/RecordDeparture.
Transports: `POST /routes/{id}/stops/{order}/times`, `PATCH /api/routes/{id}/stops/{order}/times`.

**SetRouteDistance**
Input: `{route_id, distance_km}`. Output: `{route}`. Active route or at close.
Transports: `POST /routes/{id}/distance`, `PUT /api/routes/{id}/distance`.

### Reads and aggregation

**GetRoute**
Input: `{route_id}`. Output:
```
{route: {id, driver, route_date, status, distance_km,
         stops: [{stop_order, label, address, latitude, longitude,
                  arrival_at, departure_at, stop_seconds, counted}],
         total_stopped_seconds, total_stopped_minutes, journey_percent,
         estimated_cost_brl}}
```
Each stop also carries `below_min`. `counted` means "adds to the route
total": `false` for stop 1 (departure point, RN01 — no stopwatch) and for a
completed stop shorter than `min_stop_minutes` (`below_min: true`, RN03);
such a stop keeps its recorded `stop_seconds`. An open stop (times missing) is
`counted: true, below_min: false`.
Transports: `GET /routes/{id}` (page + `?partial=1` fragment), `GET /api/routes/{id}`.

**ListRoutes**
Input: `{from?, to?, driver_user_id?, status?, manager_user_id?}` (defaults: current month);
on `GET /api/routes` these are query-string parameters (`from`/`to` as
`YYYY-MM-DD`; a driver's own scope is forced whatever they pass).
Output: `{routes: [{id, route_date, driver_name, stop_count,
total_stopped_minutes, journey_percent, estimated_cost_brl, status}]}`.
This is the history view (RF07): rows carry addresses at detail level.
An empty window answers `{"routes": []}`.

**Team filter** (`manager_user_id`, on ListRoutes, the three dashboards and
ExportPeriodCSV — query parameter on every JSON transport): keeps the routes of
drivers whose responsible manager is that id, as assigned **now** (not at the
route's date). Combines with the other filters; a driver's own scope is still
forced. A malformed id is 422 with `field: manager_user_id`.
Transports: `GET /history`, `GET /api/routes`.

**GetDashboardByDay**
Input: `{from, to, manager_user_id?}`. Output: `{series: [{date, total_stopped_minutes,
journey_percent}], standard_journey_hours}` one point per day with data (`date` is `"YYYY-MM-DD"`;
`journey_percent` over that day's worked routes, RN04). Aggregated in SQL.
Transports: `GET /dashboard?from&to` (page), `GET /api/dashboard/day?from&to`.

**GetDashboardByMonth**
Input: `{from, to, manager_user_id?}`. Output: `{series: [{month, total_stopped_minutes,
journey_percent}], standard_journey_hours}` one point per month with data (`month` is `"YYYY-MM"`;
`journey_percent` over that month's worked routes, RN04). Aggregated in SQL.
Transports: `GET /api/dashboard/month?from&to` (the page reuses /dashboard with
a tab partial).

**GetDashboardByPeriod**
Input: `{from, to, manager_user_id?}`. Output: `{standard_journey_hours, total_stopped_minutes,
journey_percent, routes_count, avg_stopped_minutes_per_route,
by_driver: [{driver_user_id, driver_name, total_stopped_minutes,
journey_percent, avg_stopped_minutes_per_route}]}` — the JSON body is this
object itself (no wrapper). `avg_stopped_minutes_per_route` is an integer:
floor(counted stopped seconds / worked routes / 60), computed in SQL over the
same base as `journey_percent` (the window's `routes_count`, or that driver's
own worked routes); 0 when there are no routes. `by_driver` has one row per driver id (two drivers
sharing a name are two rows), ordered by name then id.
`routes_count` is the worked routes in the window; `journey_percent` is over
`routes_count` standard days (RN04 in business-rules.md), each `by_driver`
row over that driver's own routes. An empty window answers
`{"standard_journey_hours": "8.0000", "total_stopped_minutes": 0,
"journey_percent": "0.000", "routes_count": 0,
"avg_stopped_minutes_per_route": 0, "by_driver": []}`.

All three dashboard reads:

- `standard_journey_hours` is the parameter value the percents were computed
  with (read in the same database snapshot), as its exact decimal string
  (same text as GetParams, e.g. `"8.0000"`) — drivers, who cannot call
  GetParams, label the percent with it.
- An empty series is `[]`, never `null`.
- A bucket whose recorded stops are all below `min_stop_minutes` still
  appears, with `total_stopped_minutes: 0` and `journey_percent: "0.000"`;
  its routes stay in the base (they are worked days).
Transports: `GET /api/dashboard/period?from&to` (same page, third tab).

### Parameters and export

**GetParams**
Input: none. Output: `{params: [{key, value, unit, updated_at, updated_by,
updated_by_name}]}` (`updated_by` is the user id, `updated_by_name` that
user's name). UpdateParam's `{param}` has the same shape.
Transports: `GET /params`, `GET /api/params`.

**UpdateParam**
Input: `{key, value}`. Output: `{param}`. Audited (`update_param`).
Errors: ErrValidation (negative values rejected).
Transports: `POST /params/{key}`, `PUT /api/params/{key}`.

**ExportPeriodCSV** (RF12)
Input: `{from, to, driver_user_id?, manager_user_id?}`. Output: CSV stream, RFC 4180, UTF-8 with
BOM so pt-BR Excel opens it directly; columns (pt-BR headers, in order):
`Data, Motorista, Ordem, Endereço, Chegada, Saída, Minutos parados,
Conta no total, Total do roteiro (min), Custo do roteiro (R$)` — route date,
driver, stop order, address, arrival, departure, stop minutes, whether the stop
adds to the route total (`Sim`/`Não` — GetRoute's `counted`: `Não` for stop 1
and for a stop under `min_stop_minutes`, whose minutes still show), route total
minutes, route cost. Dates `DD/MM/YYYY`, times `DD/MM/YYYY HH:MM` in America/Sao_Paulo.
Download name: `terminus-<from>-a-<to>.csv`
(`Content-Disposition: attachment`). Errors: on `/api/export` the JSON error
body (400 bad/missing `from`/`to`, 422 bad `driver_user_id` or inverted window,
401/403); the page download answers plain text.
Transports: `GET /history/export?from&to` (download), `GET /api/export?from&to`.

**ListAudit**
Input: `{entity?, from?, to?}`. Output: `{entries: [{at, actor, entity,
entity_id, action, old_values, new_values}]}`.
Transports: `GET /audit`, `GET /api/audit`.
