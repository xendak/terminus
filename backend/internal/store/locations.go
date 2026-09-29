package store

import (
	"context"

	"github.com/google/uuid"
)

func (s *Store) InsertLocation(ctx context.Context, l Location) error {
	_, err := s.db.Exec(ctx, `
INSERT INTO location (id, label, address, latitude, longitude, created_by)
VALUES ($1, $2, $3, $4::numeric, $5::numeric, $6)`,
		l.ID, l.Label, l.Address, l.Latitude, l.Longitude, l.CreatedBy)
	return translate(err)
}

func (s *Store) LocationByID(ctx context.Context, id uuid.UUID) (Location, error) {
	var l Location
	err := s.db.QueryRow(ctx, `
SELECT id, label, address, latitude::text, longitude::text, created_by
  FROM location WHERE id = $1`, id).
		Scan(&l.ID, &l.Label, &l.Address, &l.Latitude, &l.Longitude, &l.CreatedBy)
	if err != nil {
		return Location{}, scanOne(err)
	}
	return l, nil
}

// UpdateLocationFields applies a partial update; nil arguments leave
// the column unchanged. Unknown ids are ErrNotFound.
func (s *Store) UpdateLocationFields(ctx context.Context, id uuid.UUID, label, address, latitude, longitude *string) error {
	tag, err := s.db.Exec(ctx, `
UPDATE location
   SET label = COALESCE($2, label),
       address = COALESCE($3, address),
       latitude = COALESCE($4::numeric, latitude),
       longitude = COALESCE($5::numeric, longitude)
 WHERE id = $1`, id, label, address, latitude, longitude)
	if err != nil {
		return translate(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListLocations(ctx context.Context, q *string) ([]Location, error) {
	rows, err := s.db.Query(ctx, `
SELECT id, label, address, latitude::text, longitude::text, created_by
  FROM location
 WHERE ($1::text IS NULL OR label ILIKE '%' || $1 || '%' OR address ILIKE '%' || $1 || '%')
 ORDER BY label`, q)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	var locations []Location
	for rows.Next() {
		var l Location
		if err := rows.Scan(&l.ID, &l.Label, &l.Address, &l.Latitude, &l.Longitude, &l.CreatedBy); err != nil {
			return nil, translate(err)
		}
		locations = append(locations, l)
	}
	return locations, translate(rows.Err())
}
