-- 0001_init.sql — StopTime schema.
--
-- Field-level authority: docs/spec/data-model.md. Rule numbers (RN0x) refer
-- to tp.md section 4, operationalized in docs/spec/business-rules.md.
-- Applied by scripts/migrate.sh (psql, one transaction, filename order).
--
-- schema_migrations is NOT created here: it is migrate.sh infrastructure and
-- already exists when this file runs.

-- app_user — unified account for all three profiles (RNF04, LGPD RNF06:
-- only name, phone, email; the driver document lives in driver_profile).
CREATE TABLE app_user (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name          text        NOT NULL,
    email         text        NOT NULL,
    phone         text        NOT NULL,
    password_hash text        NOT NULL,              -- bcrypt, cost 10
    role          text        NOT NULL
                  CHECK (role IN ('admin', 'manager', 'driver')),
    active        boolean     NOT NULL DEFAULT true, -- LGPD soft-deactivate
    created_at    timestamptz NOT NULL DEFAULT now()
);

-- Case-insensitive unique email without a citext dependency
-- (data-model.md: "citext or lower-case unique index").
CREATE UNIQUE INDEX app_user_email_uniq ON app_user (lower(email));

-- driver_profile — 1:1 with app_user where role = 'driver' (tp.md
-- "Motorista"). role gating is a service-level rule (T4+).
CREATE TABLE driver_profile (
    user_id       uuid        PRIMARY KEY
                  REFERENCES app_user (id) ON DELETE CASCADE,
    document      text,                          -- CPF; masked in non-admin views
    vehicle_name  text,
    vehicle_plate text,
    km_per_l      numeric(6,2)                   -- NULL = default_km_per_l param (RN07)
);

-- location — the point registry (RF03). No geocoding in the MVP.
CREATE TABLE location (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    label       text        NOT NULL,
    address     text        NOT NULL,
    latitude    numeric(9,6),
    longitude   numeric(9,6),
    created_by  uuid        NOT NULL REFERENCES app_user (id),
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- route — one driver, one calendar date (RN05). Totals and cost are never
-- stored here; they are computed on read (business-rules.md RN03/RN07).
CREATE TABLE route (
    id             uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_user_id uuid        NOT NULL REFERENCES app_user (id),
    route_date     date        NOT NULL,
    distance_km    numeric(7,2),                    -- manual entry; no telemetry
    status         text        NOT NULL DEFAULT 'draft'
                   CHECK (status IN ('draft', 'active', 'closed')),
    note           text,
    created_by     uuid        NOT NULL REFERENCES app_user (id),
    created_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT route_driver_date_uniq UNIQUE (driver_user_id, route_date) -- RN05
);

CREATE INDEX route_route_date_idx ON route (route_date); -- dashboard scans

-- route_stop — ordered stops of a route; order 1 is the departure point.
-- stop_seconds is the schema-level enforcement of RN01 + RN02.
CREATE TABLE route_stop (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    route_id     uuid        NOT NULL
                 REFERENCES route (id) ON DELETE CASCADE,
    stop_order   int         NOT NULL CHECK (stop_order >= 1), -- RN06
    location_id  uuid        NOT NULL REFERENCES location (id),
    arrival_at   timestamptz,
    departure_at timestamptz,
    stop_seconds int GENERATED ALWAYS AS (
        CASE
            WHEN stop_order = 1 THEN 0                                -- RN01
            WHEN arrival_at IS NULL OR departure_at IS NULL THEN NULL -- open stop
            ELSE floor(extract(epoch FROM departure_at - arrival_at)) -- RN02
        END
    ) STORED,
    note         text,
    CONSTRAINT route_stop_order_uniq UNIQUE (route_id, stop_order), -- RN06
    CONSTRAINT route_stop_times_order
        CHECK (departure_at IS NULL OR arrival_at IS NULL
               OR departure_at >= arrival_at)
);
-- route_stop_order_uniq also serves route detail and composition lookups
-- (data-model.md, Indexes).

-- parameter — every tunable number lives here (RF09/RF10; never hardcoded).
-- Keys and defaults are fixed in business-rules.md "Parameters".
CREATE TABLE parameter (
    key        text          PRIMARY KEY,
    value      numeric(12,4) NOT NULL,
    unit       text          NOT NULL,  -- display hint: BRL, hours, minutes, km/l
    updated_by uuid          NOT NULL REFERENCES app_user (id),
    updated_at timestamptz   NOT NULL DEFAULT now()
);

-- audit_log — append-only, written by the store inside the mutation's
-- transaction (RNF05). Entity/action value domains per data-model.md.
CREATE TABLE audit_log (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    at            timestamptz NOT NULL DEFAULT now(),
    actor_user_id uuid        NOT NULL REFERENCES app_user (id),
    entity        text        NOT NULL, -- route_stop | route | parameter
    entity_id     uuid        NOT NULL,
    action        text        NOT NULL, -- update_times | add_stop | remove_stop
                                       -- | reorder | close_route | reopen_route
                                       -- | update_param
    old_values    jsonb       NOT NULL, -- {} on create-type actions
    new_values    jsonb       NOT NULL
);

CREATE INDEX audit_log_at_idx ON audit_log (at); -- admin audit browsing
