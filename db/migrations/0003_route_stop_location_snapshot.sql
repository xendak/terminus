-- 0003_route_stop_location_snapshot.sql
--
-- A stop keeps the label, address and coordinates its location had when
-- the stop was added (RNF05 + acceptance criterion 3: history is tied to
-- the address where the stop happened). Before this, editing a location
-- rewrote the address shown on past routes, the history and the CSV.
--
-- The snapshot is taken by the database on INSERT (and when a stop's
-- location_id changes), so every insert path — the store, the seeds, test
-- fixtures — gets it without repeating the lookup. Later location edits
-- never touch existing stops. Rationale and alternatives:
-- docs/spec/data-model.md ("route_stop", location snapshot).

ALTER TABLE route_stop
    ADD COLUMN label_snapshot     text,
    ADD COLUMN address_snapshot   text,
    ADD COLUMN latitude_snapshot  numeric(9,6),
    ADD COLUMN longitude_snapshot numeric(9,6);

-- Backfill: existing stops take their location's current values (the
-- best information available; no location was edited before RNF05 audit
-- existed for it).
UPDATE route_stop rs
   SET label_snapshot     = l.label,
       address_snapshot   = l.address,
       latitude_snapshot  = l.latitude,
       longitude_snapshot = l.longitude
  FROM location l
 WHERE l.id = rs.location_id;

ALTER TABLE route_stop
    ALTER COLUMN label_snapshot   SET NOT NULL,
    ALTER COLUMN address_snapshot SET NOT NULL;

CREATE FUNCTION route_stop_location_snapshot() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'INSERT' OR NEW.location_id IS DISTINCT FROM OLD.location_id THEN
        SELECT l.label, l.address, l.latitude, l.longitude
          INTO NEW.label_snapshot, NEW.address_snapshot,
               NEW.latitude_snapshot, NEW.longitude_snapshot
          FROM location l
         WHERE l.id = NEW.location_id;
        IF NOT FOUND THEN
            -- Same report the foreign key would give (the NOT NULL check
            -- on the snapshot would otherwise fire first).
            RAISE foreign_key_violation USING
                MESSAGE = format('location %s does not exist', NEW.location_id),
                CONSTRAINT = 'route_stop_location_id_fkey',
                TABLE = 'route_stop';
        END IF;
    END IF;
    RETURN NEW;
END
$$;

CREATE TRIGGER route_stop_location_snapshot
    BEFORE INSERT OR UPDATE OF location_id ON route_stop
    FOR EACH ROW EXECUTE FUNCTION route_stop_location_snapshot();
