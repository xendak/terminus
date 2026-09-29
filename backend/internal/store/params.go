package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const listParamsSQL = `
SELECT p.key, p.value::text, p.unit, p.updated_by, u.name, p.updated_at
  FROM parameter p
  JOIN app_user u ON u.id = p.updated_by
 ORDER BY p.key`

func (s *Store) ListParams(ctx context.Context) ([]Param, error) {
	rows, err := s.db.Query(ctx, listParamsSQL)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	var params []Param
	for rows.Next() {
		var p Param
		if err := rows.Scan(&p.Key, &p.Value, &p.Unit, &p.UpdatedBy, &p.UpdatedByName, &p.UpdatedAt); err != nil {
			return nil, translate(err)
		}
		params = append(params, p)
	}
	return params, translate(rows.Err())
}

// ParamByKey returns one parameter row or ErrNotFound.
func (s *Store) ParamByKey(ctx context.Context, key string) (Param, error) {
	var p Param
	err := s.db.QueryRow(ctx, `
SELECT p.key, p.value::text, p.unit, p.updated_by, u.name, p.updated_at
  FROM parameter p
  JOIN app_user u ON u.id = p.updated_by
 WHERE p.key = $1`, key).
		Scan(&p.Key, &p.Value, &p.Unit, &p.UpdatedBy, &p.UpdatedByName, &p.UpdatedAt)
	if err != nil {
		return Param{}, scanOne(err)
	}
	return p, nil
}

// UpdateParamValue sets a parameter's value, updater, and update time
// (server clock, not the database clock).
func (s *Store) UpdateParamValue(ctx context.Context, key, value string, updatedBy uuid.UUID, updatedAt time.Time) error {
	tag, err := s.db.Exec(ctx, `
UPDATE parameter
   SET value = $2::numeric, updated_by = $3, updated_at = $4
 WHERE key = $1`, key, value, updatedBy, updatedAt)
	if err != nil {
		return translate(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// JourneyHours returns standard_journey_hours as exact text (the value
// the dashboard SQL divides by).
func (s *Store) JourneyHours(ctx context.Context) (string, error) {
	var h string
	err := s.db.QueryRow(ctx, `SELECT value::text FROM parameter WHERE key = 'standard_journey_hours'`).Scan(&h)
	if err != nil {
		return "", scanOne(err)
	}
	return h, nil
}
