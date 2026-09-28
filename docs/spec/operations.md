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
- Money is BRL as a number rounded to 2 places (SQL numeric, rounded once).
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
| ErrDuplicateEmail | email already registered | 409 |

htmx adapters map the same sentinels to inline form errors and flash messages.

## Role matrix

| Operation | Admin | Manager | Driver |
| --- | --- | --- | --- |
| Login / Logout | yes | yes | yes |
| CreateDriver, UpdateDriver, ListDrivers | yes | yes | no |
| CreateManager, ListManagers | yes | no | no |
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
Input: `{email, password}`. Output: `{user}` + sets session cookie.
Errors: ErrUnauthenticated.
Transports: `POST /login` (form, redirect), `POST /api/auth/login` (JSON).

**Logout**
Input: none. Output: clears cookie. Transports: `POST /logout`, `POST /api/auth/logout`.

### Directories

**CreateDriver**
Input: `{name, email, password, phone, document?, vehicle_name?, vehicle_plate?, km_per_l?}`.
Output: `{driver}` (user + profile). Errors: ErrDuplicateEmail, ErrValidation.
Creates the `app_user` (role driver) and `driver_profile` in one transaction.
Transports: `POST /drivers`, `POST /api/drivers`.

**UpdateDriver**
Input: `{driver_id, name?, phone?, document?, vehicle_name?, vehicle_plate?, km_per_l?, active?}`.
Output: `{driver}`. Deactivating (LGPD removal path) sets `active = false`.
Transports: `POST /drivers/{id}/edit`, `PATCH /api/drivers/{id}`.

**ListDrivers**
Input: `{active_only?}`. Output: `{drivers: [{id, name, phone, vehicle, km_per_l, active}]}`.
Transports: `GET /drivers`, `GET /api/drivers`.

**CreateManager**
Input: `{name, email, password, phone}`. Output: `{manager}`.
Transports: `POST /managers`, `POST /api/managers`.

**ListManagers**
Input: none. Output: `{managers}`. Transports: `GET /managers`, `GET /api/managers`.

**CreateLocation**
Input: `{label, address, latitude?, longitude?}`. Output: `{location}`.
Transports: `POST /locations`, `POST /api/locations`.

**UpdateLocation**
Input: `{location_id, label?, address?, latitude?, longitude?}`. Output: `{location}`.
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
Transports: `POST /routes/{id}/stops/{order}/arrive`, `POST /api/routes/{id}/stops/{order}/arrive`.

**RecordDeparture**
Input: `{route_id, stop_order, at? (default: now)}`. Output: `{stop}`.
Same null-only rule. Requires `arrival_at` set.
Transports: `POST /routes/{id}/stops/{order}/depart`, `POST /api/routes/{id}/stops/{order}/depart`.

**UpdateStopTimes** (correction, RNF05-audited)
Input: `{route_id, stop_order, arrival_at?, departure_at?}`. Output: `{stop}`.
Manager/admin only. Any change to an already-set timestamp writes an audit row
(`update_times`) with old and new values. Validated against RN02
(ErrDepartureBeforeArrival).
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
Stop 1 shows `counted: false`; the UI renders no stopwatch for it.
Transports: `GET /routes/{id}` (page + `?partial=1` fragment), `GET /api/routes/{id}`.

**ListRoutes**
Input: `{from?, to?, driver_user_id?, status?}` (defaults: current month).
Output: `{routes: [{id, route_date, driver_name, stop_count,
total_stopped_minutes, journey_percent, estimated_cost_brl, status}]}`.
This is the history view (RF07): rows carry addresses at detail level.
Transports: `GET /history`, `GET /api/routes`.

**GetDashboardByDay**
Input: `{from, to}`. Output: `{series: [{date, total_stopped_minutes}]}` one
point per day with data. Aggregated in SQL.
Transports: `GET /dashboard?from&to` (page), `GET /api/dashboard/day?from&to`.

**GetDashboardByMonth**
Input: `{from, to}`. Output: `{series: [{month, total_stopped_minutes}]}`.
Transports: `GET /api/dashboard/month?from&to` (the page reuses /dashboard with
a tab partial).

**GetDashboardByPeriod**
Input: `{from, to}`. Output: `{total_stopped_minutes, journey_percent,
by_driver: [{driver_name, total_stopped_minutes, journey_percent}],
routes_count}`.
Transports: `GET /api/dashboard/period?from&to` (same page, third tab).

### Parameters and export

**GetParams**
Input: none. Output: `{params: [{key, value, unit, updated_at, updated_by}]}`.
Transports: `GET /params`, `GET /api/params`.

**UpdateParam**
Input: `{key, value}`. Output: `{param}`. Audited (`update_param`).
Errors: ErrValidation (negative values rejected).
Transports: `POST /params/{key}`, `PUT /api/params/{key}`.

**ExportPeriodCSV** (RF12)
Input: `{from, to, driver_user_id?}`. Output: CSV stream, RFC 4180, UTF-8 with
BOM so pt-BR Excel opens it directly; columns: route date, driver, stop order,
address, arrival, departure, stop minutes, route total minutes, route cost.
Transports: `GET /history/export?from&to` (download), `GET /api/export?from&to`.

**ListAudit**
Input: `{entity?, from?, to?}`. Output: `{entries: [{at, actor, entity,
entity_id, action, old_values, new_values}]}`.
Transports: `GET /audit`, `GET /api/audit`.
