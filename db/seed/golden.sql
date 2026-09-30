-- golden.sql — the tp.md section 5 fixture, verbatim
-- (docs/spec/business-rules.md, "Golden fixture").
--
-- Applied by scripts/testdb.sh --seed to stoptime_test after a fresh migrate.
-- Deterministic ids make rows debuggable; timestamps use -03:00 offsets to
-- catch conversion bugs (business-rules.md, Edge cases).
--
-- Route date: all three routes share DATE '2026-06-15' so the by-day series
-- reads 161 minutes for that day, the by-month 161, the period 161.
--
-- Demo password: every demo user logs in as "stoptime-dev"
-- (bcrypt cost 10, verified with golang.org/x/crypto/bcrypt). Dev seed only.

-- Demo users: one admin, one manager, THREE drivers. RN05 (one route per
-- driver per date) forbids one driver owning routes A/B/C on the same date,
-- so the golden fixture needs three distinct drivers.

INSERT INTO app_user (id, name, email, phone, password_hash, role) VALUES
  ('aa000000-0000-4000-8000-000000000001', 'Ana Administradora', 'admin@stoptime.dev',    '31 99999-0001', '$2a$10$Vvg8W2vXGpgIF6Qd/ofIi.X0N1Tuw1uOLWJN3rkRpnpwup/uZu6NG', 'admin'),
  ('aa000000-0000-4000-8000-000000000002', 'Gustavo Gerente',    'manager@stoptime.dev', '31 99999-0002', '$2a$10$Vvg8W2vXGpgIF6Qd/ofIi.X0N1Tuw1uOLWJN3rkRpnpwup/uZu6NG', 'manager'),
  ('aa000000-0000-4000-8000-000000000003', 'Marcos Motorista',  'driver-a@stoptime.dev','31 99999-0003', '$2a$10$Vvg8W2vXGpgIF6Qd/ofIi.X0N1Tuw1uOLWJN3rkRpnpwup/uZu6NG', 'driver'),
  ('aa000000-0000-4000-8000-000000000004', 'Bianca Batista',    'driver-b@stoptime.dev','31 99999-0004', '$2a$10$Vvg8W2vXGpgIF6Qd/ofIi.X0N1Tuw1uOLWJN3rkRpnpwup/uZu6NG', 'driver'),
  ('aa000000-0000-4000-8000-000000000005', 'Carla Camargo',     'driver-c@stoptime.dev','31 99999-0005', '$2a$10$Vvg8W2vXGpgIF6Qd/ofIi.X0N1Tuw1uOLWJN3rkRpnpwup/uZu6NG', 'driver');

-- Driver profiles: A and C fall back to the default_km_per_l parameter
-- (NULL); B overrides it, pinning both RN07 branches for later cost tests.
-- All three are in Gustavo Gerente's team (responsible manager, 0004).
INSERT INTO driver_profile (user_id, document, vehicle_name, vehicle_plate, km_per_l, manager_user_id) VALUES
  ('aa000000-0000-4000-8000-000000000003', '123.456.789-00', 'Fiorino', 'ABC1D23', NULL,  'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-000000000004', '234.567.890-11', 'Saveiro', 'DEF2E34', 12.50, 'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-000000000005', '345.678.901-22', 'Strada',  'GHI3F45', NULL,  'aa000000-0000-4000-8000-000000000002');

-- Locations. Route A addresses are verbatim from tp.md section 5; tp.md
-- names the other points without addresses, so they get documented
-- placeholders (per the T2 card). Registered by the manager.
INSERT INTO location (id, label, address, created_by) VALUES
  ('aa000000-0000-4000-8000-000000000101', 'Seg. Família', 'Av. Partida, 100', 'aa000000-0000-4000-8000-000000000002'), -- placeholder address (tp.md gives the point name only)
  ('aa000000-0000-4000-8000-000000000102', 'Ponto A2',    'Rua Peru, 55',     'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-000000000103', 'Ponto A3',    'Rua X, 5',         'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-000000000104', 'Ponto A4',    'Av. João César',   'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-000000000105', 'Partida B',    'Endereço B1',      'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-000000000106', 'Ponto B2',     'Endereço B2',      'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-000000000107', 'Ponto B3',     'Endereço B3',      'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-000000000108', 'Ponto B4',     'Endereço B4',      'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-000000000109', 'Partida C',    'Endereço C1',      'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-00000000010a', 'Ponto C2',     'Endereço C2',      'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-00000000010b', 'Ponto C3',     'Endereço C3',      'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-00000000010c', 'Ponto C4',     'Endereço C4',      'aa000000-0000-4000-8000-000000000002');

-- Routes A/B/C, one per driver, all on 2026-06-15 (RN05). Status closed:
-- fully recorded days. distance_km stays NULL (no invention; RN07 cost
-- legitimately shows nothing until a distance is entered).
INSERT INTO route (id, driver_user_id, route_date, status, note, created_by) VALUES
  ('aa000000-0000-4000-8000-000000000201', 'aa000000-0000-4000-8000-000000000003', DATE '2026-06-15', 'closed', 'Roteiro A (tp.md §5)', 'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-000000000202', 'aa000000-0000-4000-8000-000000000004', DATE '2026-06-15', 'closed', 'Roteiro B (tp.md §5)', 'aa000000-0000-4000-8000-000000000002'),
  ('aa000000-0000-4000-8000-000000000203', 'aa000000-0000-4000-8000-000000000005', DATE '2026-06-15', 'closed', 'Roteiro C (tp.md §5)', 'aa000000-0000-4000-8000-000000000002');

-- Stops. stop_seconds is a generated column — never inserted. Stop 1 carries
-- timestamps on purpose: it must still contribute 0 (RN01).
-- Route A: 0 + 15 + 10 + 50 = 75 min (4500 s).
INSERT INTO route_stop (id, route_id, stop_order, location_id, arrival_at, departure_at) VALUES
  ('aa000000-0000-4000-8000-000000000301', 'aa000000-0000-4000-8000-000000000201', 1, 'aa000000-0000-4000-8000-000000000101', TIMESTAMPTZ '2026-06-15 08:00-03', TIMESTAMPTZ '2026-06-15 08:05-03'), -- departure point (RN01)
  ('aa000000-0000-4000-8000-000000000302', 'aa000000-0000-4000-8000-000000000201', 2, 'aa000000-0000-4000-8000-000000000102', TIMESTAMPTZ '2026-06-15 09:00-03', TIMESTAMPTZ '2026-06-15 09:15-03'), -- 15 min
  ('aa000000-0000-4000-8000-000000000303', 'aa000000-0000-4000-8000-000000000201', 3, 'aa000000-0000-4000-8000-000000000103', TIMESTAMPTZ '2026-06-15 10:00-03', TIMESTAMPTZ '2026-06-15 10:10-03'), -- 10 min
  ('aa000000-0000-4000-8000-000000000304', 'aa000000-0000-4000-8000-000000000201', 4, 'aa000000-0000-4000-8000-000000000104', TIMESTAMPTZ '2026-06-15 11:00-03', TIMESTAMPTZ '2026-06-15 11:50-03'), -- 50 min

-- Route B: 0 + 10 + 5 + 26 = 41 min (2460 s).
  ('aa000000-0000-4000-8000-000000000305', 'aa000000-0000-4000-8000-000000000202', 1, 'aa000000-0000-4000-8000-000000000105', TIMESTAMPTZ '2026-06-15 08:00-03', TIMESTAMPTZ '2026-06-15 08:02-03'), -- departure point (RN01)
  ('aa000000-0000-4000-8000-000000000306', 'aa000000-0000-4000-8000-000000000202', 2, 'aa000000-0000-4000-8000-000000000106', TIMESTAMPTZ '2026-06-15 09:00-03', TIMESTAMPTZ '2026-06-15 09:10-03'), -- 10 min
  ('aa000000-0000-4000-8000-000000000307', 'aa000000-0000-4000-8000-000000000202', 3, 'aa000000-0000-4000-8000-000000000107', TIMESTAMPTZ '2026-06-15 09:40-03', TIMESTAMPTZ '2026-06-15 09:45-03'), -- 5 min
  ('aa000000-0000-4000-8000-000000000308', 'aa000000-0000-4000-8000-000000000202', 4, 'aa000000-0000-4000-8000-000000000108', TIMESTAMPTZ '2026-06-15 10:30-03', TIMESTAMPTZ '2026-06-15 10:56-03'), -- 26 min

-- Route C: 0 + 5 + 10 + 30 = 45 min (2700 s).
  ('aa000000-0000-4000-8000-000000000309', 'aa000000-0000-4000-8000-000000000203', 1, 'aa000000-0000-4000-8000-000000000109', TIMESTAMPTZ '2026-06-15 08:00-03', TIMESTAMPTZ '2026-06-15 08:01-03'), -- departure point (RN01)
  ('aa000000-0000-4000-8000-00000000030a', 'aa000000-0000-4000-8000-000000000203', 2, 'aa000000-0000-4000-8000-00000000010a', TIMESTAMPTZ '2026-06-15 09:10-03', TIMESTAMPTZ '2026-06-15 09:15-03'), -- 5 min
  ('aa000000-0000-4000-8000-00000000030b', 'aa000000-0000-4000-8000-000000000203', 3, 'aa000000-0000-4000-8000-00000000010b', TIMESTAMPTZ '2026-06-15 10:00-03', TIMESTAMPTZ '2026-06-15 10:10-03'), -- 10 min
  ('aa000000-0000-4000-8000-00000000030c', 'aa000000-0000-4000-8000-000000000203', 4, 'aa000000-0000-4000-8000-00000000010c', TIMESTAMPTZ '2026-06-15 11:20-03', TIMESTAMPTZ '2026-06-15 11:50-03'); -- 30 min

-- Parameters at business-rules.md defaults.
INSERT INTO parameter (key, value, unit, updated_by) VALUES
  ('fuel_price_brl',         6.09, 'BRL',     'aa000000-0000-4000-8000-000000000001'),
  ('cost_per_km_brl',        0.00, 'BRL',     'aa000000-0000-4000-8000-000000000001'),
  ('standard_journey_hours', 8.00, 'hours',   'aa000000-0000-4000-8000-000000000001'),
  ('min_stop_minutes',       0,    'minutes', 'aa000000-0000-4000-8000-000000000001'),
  ('default_km_per_l',       10.00, 'km/l',   'aa000000-0000-4000-8000-000000000001'),
  ('stop_warn_minutes',      15,    'minutes', 'aa000000-0000-4000-8000-000000000001'), -- 0005: dashboard stop colours
  ('stop_alert_minutes',     45,    'minutes', 'aa000000-0000-4000-8000-000000000001');
