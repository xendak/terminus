package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Stop order renumbering uses a two-phase park: affected rows move above
// the dense range (order + parkZone) and drop back shifted, so the
// (route_id, stop_order) unique constraint never sees two rows at the
// same order mid-flight. Services cap routes below parkZone stops.
const parkZone = 1000

// MaxStops caps route composition size below the park zone.
const MaxStops = parkZone - 1

const stopColumns = `id, route_id, stop_order, location_id, arrival_at, departure_at, note`

const stopsByRouteSQL = `
SELECT ` + stopColumns + `
  FROM route_stop WHERE route_id = $1 ORDER BY stop_order`

func (s *Store) StopsByRoute(ctx context.Context, routeID uuid.UUID) ([]RouteStop, error) {
	rows, err := s.db.Query(ctx, stopsByRouteSQL, routeID)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	var stops []RouteStop
	for rows.Next() {
		var st RouteStop
		if err := rows.Scan(&st.ID, &st.RouteID, &st.StopOrder, &st.LocationID,
			&st.ArrivalAt, &st.DepartureAt, &st.Note); err != nil {
			return nil, translate(err)
		}
		stops = append(stops, st)
	}
	return stops, translate(rows.Err())
}

// StopByOrder returns one stop of a route or ErrNotFound.
func (s *Store) StopByOrder(ctx context.Context, routeID uuid.UUID, stopOrder int) (RouteStop, error) {
	var st RouteStop
	err := s.db.QueryRow(ctx,
		`SELECT `+stopColumns+` FROM route_stop WHERE route_id = $1 AND stop_order = $2`,
		routeID, stopOrder).
		Scan(&st.ID, &st.RouteID, &st.StopOrder, &st.LocationID,
			&st.ArrivalAt, &st.DepartureAt, &st.Note)
	if err != nil {
		return RouteStop{}, scanOne(err)
	}
	return st, nil
}

// InsertStop adds one stop at the given order. Callers keep orders dense
// (RN06): use ShiftStopOrders first when inserting below the end.
func (s *Store) InsertStop(ctx context.Context, st RouteStop) error {
	_, err := s.db.Exec(ctx, `
INSERT INTO route_stop (id, route_id, stop_order, location_id, note)
VALUES ($1, $2, $3, $4, $5)`,
		st.ID, st.RouteID, st.StopOrder, st.LocationID, st.Note)
	return translate(err)
}

// InsertStops creates a whole composition at once (CreateRoute): orders
// 1..n in visit order; the first location is the departure point.
func (s *Store) InsertStops(ctx context.Context, routeID uuid.UUID, locationIDs []uuid.UUID) error {
	for i, locID := range locationIDs {
		_, err := s.db.Exec(ctx, `
INSERT INTO route_stop (id, route_id, stop_order, location_id)
VALUES ($1, $2, $3, $4)`,
			uuid.New(), routeID, i+1, locID)
		if err != nil {
			return translate(err)
		}
	}
	return nil
}

// UpdateStopTimes sets both timestamps of a stop (nil clears nothing —
// callers pass the effective pair for partial updates).
func (s *Store) UpdateStopTimes(ctx context.Context, stopID uuid.UUID, arrivalAt, departureAt *time.Time) error {
	tag, err := s.db.Exec(ctx, `
UPDATE route_stop
   SET arrival_at = $2, departure_at = $3
 WHERE id = $1`, stopID, arrivalAt, departureAt)
	if err != nil {
		return translate(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteStop removes a stop by order; ErrNotFound when absent.
func (s *Store) DeleteStop(ctx context.Context, routeID uuid.UUID, stopOrder int) error {
	tag, err := s.db.Exec(ctx,
		`DELETE FROM route_stop WHERE route_id = $1 AND stop_order = $2`,
		routeID, stopOrder)
	if err != nil {
		return translate(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ShiftStopOrders moves stops with order >= from by delta (keeping ids —
// audit entity ids stay stable), in two phases so the unique constraint
// never sees a collision.
func (s *Store) ShiftStopOrders(ctx context.Context, routeID uuid.UUID, from, delta int) error {
	if _, err := s.db.Exec(ctx, `
UPDATE route_stop SET stop_order = stop_order + $3
 WHERE route_id = $1 AND stop_order >= $2`,
		routeID, from, parkZone); err != nil {
		return translate(err)
	}
	if _, err := s.db.Exec(ctx, `
UPDATE route_stop SET stop_order = stop_order - $2 + $3
 WHERE route_id = $1 AND stop_order >= $2`,
		routeID, parkZone, delta); err != nil {
		return translate(err)
	}
	return nil
}

// SwapAdjacentStops exchanges the stops at order and order-1 (ReorderStops
// "up"), two-phase for the same reason.
func (s *Store) SwapAdjacentStops(ctx context.Context, routeID uuid.UUID, order int) error {
	if _, err := s.db.Exec(ctx, `
UPDATE route_stop SET stop_order = stop_order + $3
 WHERE route_id = $1 AND stop_order IN ($2, $2 - 1)`,
		routeID, order, parkZone); err != nil {
		return translate(err)
	}
	_, err := s.db.Exec(ctx, `
UPDATE route_stop
   SET stop_order = CASE stop_order WHEN $2 + $3 THEN $2 - 1 ELSE $2 END
 WHERE route_id = $1 AND stop_order >= $3`,
		routeID, order, parkZone)
	return translate(err)
}
