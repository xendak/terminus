# Data model

PostgreSQL 18 (devshell-pinned). All DDL lives in `db/migrations/` as plain SQL
applied by psql; this file is the field-level authority the migration is written
from. Storage timezone is UTC; `route_date` is a `date`.

## Entity relationship diagram

```mermaid
erDiagram
    APP_USER ||--o| DRIVER_PROFILE : "user_id"
    APP_USER ||--o{ ROUTE : "driver_user_id"
    APP_USER ||--o{ ROUTE : "created_by"
    APP_USER ||--o{ LOCATION : "created_by"
    APP_USER ||--o{ PARAMETER : "updated_by"
    APP_USER ||--o{ AUDIT_LOG : "actor_user_id"
    ROUTE ||--|{ ROUTE_STOP : "route_id"
    LOCATION ||--o{ ROUTE_STOP : "location_id"

    APP_USER {
        uuid id PK
        text name
        text email UK
        text phone
        text password_hash
        text role "admin|manager|driver"
        bool active
        timestamptz created_at
    }
    DRIVER_PROFILE {
        uuid user_id PK_FK
        text document
        text vehicle_name
        text vehicle_plate
        numeric km_per_l "nullable, overrides default param"
    }
    LOCATION {
        uuid id PK
        text label
        text address
        numeric latitude "nullable"
        numeric longitude "nullable"
        uuid created_by FK
        timestamptz created_at
    }
    ROUTE {
        uuid id PK
        uuid driver_user_id FK
        date route_date "unique with driver"
        numeric distance_km "nullable, manual entry"
        text status "draft|active|closed"
        text note "nullable"
        uuid created_by FK
        timestamptz created_at
    }
    ROUTE_STOP {
        uuid id PK
        uuid route_id FK
        int stop_order "1..n, 1 = departure"
        uuid location_id FK
        timestamptz arrival_at "nullable"
        timestamptz departure_at "nullable"
        int stop_seconds "generated, RN01+RN02"
        text note "nullable"
    }
    PARAMETER {
        text key PK
        numeric value
        text unit
        uuid updated_by FK
        timestamptz updated_at
    }
    AUDIT_LOG {
        uuid id PK
        timestamptz at
        uuid actor_user_id FK
        text entity
        text entity_id "uuid string or parameter key"
        text action
        jsonb old_values
        jsonb new_values
    }
```

## Tables

### app_user

Unified account for all three profiles (the `tp.md` "Motorista" and "Gerente"
entities are profiles on top of this account). LGPD: only name, phone, email;
the driver's document lives in `driver_profile`.

| Field | Type | Constraints |
| --- | --- | --- |
| id | uuid | PK, generated client-side or by `gen_random_uuid()` |
| name | text | not null |
| email | text | not null, unique (citext or lower-case unique index) |
| phone | text | not null |
| password_hash | text | not null, bcrypt |
| role | text | not null, check in (`admin`,`manager`,`driver`) |
| active | bool | not null default true (soft deactivate for LGPD removal) |
| created_at | timestamptz | not null default now() |

### driver_profile

1:1 with `app_user` where `role = 'driver'`. Holds the `tp.md` "Motorista"
fields: document, vehicle, km per liter.

| Field | Type | Constraints |
| --- | --- | --- |
| user_id | uuid | PK, FK to app_user, unique |
| document | text | nullable (CPF; masked in non-admin views) |
| vehicle_name | text | nullable |
| vehicle_plate | text | nullable |
| km_per_l | numeric(6,2) | nullable; null means use `default_km_per_l` parameter |

### location

The point registry (RF03). Addresses and coordinates are free text/numbers; no
geocoding in MVP.

| Field | Type | Constraints |
| --- | --- | --- |
| id | uuid | PK |
| label | text | not null (short name, e.g. "Seg. Família") |
| address | text | not null |
| latitude | numeric(9,6) | nullable |
| longitude | numeric(9,6) | nullable |
| created_by | uuid | FK app_user, not null |
| created_at | timestamptz | not null default now() |

### route

| Field | Type | Constraints |
| --- | --- | --- |
| id | uuid | PK |
| driver_user_id | uuid | FK app_user(role driver), not null |
| route_date | date | not null; `UNIQUE (driver_user_id, route_date)` (RN05) |
| distance_km | numeric(7,2) | nullable, manual entry (no telemetry) |
| status | text | not null default `draft`, check in (`draft`,`active`,`closed`) |
| note | text | nullable |
| created_by | uuid | FK app_user, not null |
| created_at | timestamptz | not null default now() |

Totals and cost are never stored here. They are computed on read per
`business-rules.md`.

### route_stop

| Field | Type | Constraints |
| --- | --- | --- |
| id | uuid | PK |
| route_id | uuid | FK route on delete cascade, not null |
| stop_order | int | not null; `UNIQUE (route_id, stop_order)`; check `stop_order >= 1` (RN06) |
| location_id | uuid | FK location, not null |
| arrival_at | timestamptz | nullable |
| departure_at | timestamptz | nullable |
| stop_seconds | int | generated stored, see below (RN01, RN02) |
| note | text | nullable |
| label_snapshot | text | not null — the location's label when the stop was added (0003) |
| address_snapshot | text | not null — the location's address when the stop was added (0003) |
| latitude_snapshot, longitude_snapshot | numeric(9,6) | nullable — coordinates when the stop was added (0003) |

**Location snapshot (migration 0003).** A stop records where the driver was
sent, so route detail, history and the CSV read the stop's snapshot columns,
never the location's current values: editing a location (a rename, a corrected
address) cannot rewrite a past route (acceptance criterion 3; the edit itself
is audited, RNF05). The snapshot is taken **when the stop is added** (CreateRoute
or AddStop) by a `BEFORE INSERT OR UPDATE OF location_id` trigger that copies
the location's label/address/coordinates, so every insert path — store, seeds,
fixtures — fills it identically; an unknown `location_id` raises the same
`route_stop_location_id_fkey` violation the foreign key would. Existing stops
were backfilled from their locations' values at migration time.

Why add time and not close time: add time is one rule with no state machine
(a closed-then-reopened route cannot re-snapshot differently), and the stop
shows the address the manager actually assigned. A location correction that
must reach a draft or active route is applied by removing and re-adding the
stop (both audited). Future routes always use the location's current values.

Generated column (the schema-level enforcement of RN01 + RN02):

```sql
stop_seconds int GENERATED ALWAYS AS (
  CASE
    WHEN stop_order = 1 THEN 0                      -- RN01
    WHEN arrival_at IS NULL OR departure_at IS NULL THEN NULL
    ELSE floor(extract(epoch FROM departure_at - arrival_at))
  END
) STORED
```

Check constraint: `departure_at IS NULL OR arrival_at IS NULL OR departure_at >= arrival_at`.

### parameter

| Field | Type | Constraints |
| --- | --- | --- |
| key | text | PK |
| value | numeric(12,4) | not null |
| unit | text | not null (display hint: 'BRL', 'hours', 'minutes', 'km/l') |
| updated_by | uuid | FK app_user, not null |
| updated_at | timestamptz | not null default now() |

Keys and defaults are fixed in `business-rules.md` ("Parameters").

### audit_log

Append-only. Written by the store layer inside the same transaction as the
mutation it records (RNF05).

| Field | Type | Constraints |
| --- | --- | --- |
| id | uuid | PK |
| at | timestamptz | not null default now() |
| actor_user_id | uuid | FK app_user, not null |
| entity | text | not null (`route_stop`, `route`, `parameter`, `app_user`, `location`) |
| entity_id | text | not null (uuid string for route_stop/route/app_user/location rows; the parameter key for parameter) |
| action | text | not null (`update_times`, `add_stop`, `remove_stop`, `reorder`, `close_route`, `reopen_route`, `update_param`, `anonymize`, `update_location`) |
| old_values | jsonb | not null (empty object on create-type actions) |
| new_values | jsonb | not null |

### schema_migrations

| Field | Type | Constraints |
| --- | --- | --- |
| name | text | PK (migration filename) |
| applied_at | timestamptz | not null default now() |

Created by `scripts/migrate.sh` itself (infrastructure, not schema). Managed
by the same script. Applied migrations are never edited; a fix is a new
migration.

## Indexes

| Index | Serves |
| --- | --- |
| `route (driver_user_id, route_date)` unique | RN05, driver's own-route lookup |
| `route (route_date)` | dashboard by-day/month/period scans |
| `route_stop (route_id, stop_order)` | route detail, composition edits |
| `audit_log (at)` | admin audit browsing |

## Audited actions

Per RNF05 ("changes to points and times"): every `UpdateStopTimes`, stop
add/remove/reorder, route close/reopen, and `UpdateParam` writes one audit row
in the same transaction. `AnonymizeDriver` (RNF06 erasure) writes one
`anonymize` row on `app_user` (drivers and managers). `UpdateLocation` (RNF05,
"changes to points") writes an `update_location` row on `location` with the
old and new `{label, address, latitude, longitude}`. Reads are not audited.

## LGPD approach (RNF06)

- **Minimization**: the system collects only name, phone, email, and (drivers)
  document and vehicle info. No other personal data exists in the schema.
- **Access**: drivers see only their own routes. Managers see operational data
  of all drivers; in every driver read a manager gets (ListDrivers, and the
  CreateDriver/UpdateDriver responses) the `document` is masked in the service
  layer — every digit except the last two becomes `*`
  (`123.456.789-11` → `***.***.***-11`) and `document_masked` is `true`. Admin
  sees the document in full (`document_masked: false`). A manager may still set
  a new document; re-submitting the exact masked value keeps the stored one
  (the edit form round trip), any other value containing `*` is rejected
  (ErrValidation on `document`). Drivers have no directory read of profiles.
- **Retention / removal**: deactivating a user (`active = false`, UpdateDriver
  or UpdateManager) hides them from active directories, blocks login, and ends
  any open session on its next request (the session is re-validated per
  request, architecture.md "Security"), while preserving route history
  (operational records).
- **Full erasure = pseudonymization** (`AnonymizeDriver` and
  `AnonymizeManager`, admin only; for a manager the name placeholder is
  `Gestor removido <first 8 chars of id>` and there is no profile to clear).
  For a driver, in one
  transaction: `name` → `Motorista removido <first 8 chars of id>`, `email` →
  `removido-<id>@anonimo.invalid`, `phone` → `''`, `password_hash` → a random
  non-bcrypt value (no password can match), `active` → false; `driver_profile`
  `document`, `vehicle_name`, `vehicle_plate` → NULL. `km_per_l` stays (it is
  operational: route cost). Routes, stops, totals and costs are untouched, so
  history and dashboards keep counting the (now pseudonymous) driver. The audit
  row (`app_user`, `anonymize`) records which fields were cleared and the
  placeholders — never the erased values. Calling it again on an anonymized
  driver is a no-op answering the same driver (200, no second audit row).
- **Audit**: who changed what and when is answerable from `audit_log`.

## Simplifications, accepted and documented

- `tp.md` lists "equipe sob responsabilidade" (team under the manager's
  responsibility). MVP is single-company: every manager coordinates all drivers.
  No team-assignment table. If the professor pushes, this is the extension
  point.
- Server-side sessions are not stored; the session is an HMAC-signed cookie
  (see `architecture.md`). If revocation becomes a requirement, add a
  `session` table behind the same middleware.
