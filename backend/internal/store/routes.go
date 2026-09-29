package store

import (
	"context"

	"github.com/google/uuid"
)

func (s *Store) InsertRoute(ctx context.Context, r Route) error {
	_, err := s.db.Exec(ctx, `
INSERT INTO route (id, driver_user_id, route_date, status, note, created_by)
VALUES ($1, $2, $3, $4, $5, $6)`,
		r.ID, r.DriverUserID, r.RouteDate, r.Status, r.Note, r.CreatedBy)
	return translate(err)
}

const routeByIDSQL = `
SELECT id, driver_user_id, route_date, distance_km::text, status, note, created_by
  FROM route WHERE id = $1`

func (s *Store) RouteByID(ctx context.Context, id uuid.UUID) (Route, error) {
	var r Route
	err := s.db.QueryRow(ctx, routeByIDSQL, id).
		Scan(&r.ID, &r.DriverUserID, &r.RouteDate, &r.DistanceKm, &r.Status, &r.Note, &r.CreatedBy)
	if err != nil {
		return Route{}, scanOne(err)
	}
	return r, nil
}

// UpdateRouteStatus sets the status column; unknown ids are ErrNotFound.
func (s *Store) UpdateRouteStatus(ctx context.Context, id uuid.UUID, status string) error {
	tag, err := s.db.Exec(ctx, `UPDATE route SET status = $2 WHERE id = $1`, id, status)
	if err != nil {
		return translate(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateRouteDistance sets distance_km (exact decimal text).
func (s *Store) UpdateRouteDistance(ctx context.Context, id uuid.UUID, distanceKm string) error {
	tag, err := s.db.Exec(ctx, `UPDATE route SET distance_km = $2::numeric WHERE id = $1`, id, distanceKm)
	if err != nil {
		return translate(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CountStops returns the number of stops of a route.
func (s *Store) CountStops(ctx context.Context, routeID uuid.UUID) (int, error) {
	var n int
	err := s.db.QueryRow(ctx,
		`SELECT count(*) FROM route_stop WHERE route_id = $1`, routeID).Scan(&n)
	if err != nil {
		return 0, translate(err)
	}
	return n, nil
}
