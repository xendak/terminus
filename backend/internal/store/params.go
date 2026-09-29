package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const listParamsSQL = `
SELECT key, value::text, unit, updated_by, updated_at
  FROM parameter ORDER BY key`

func (s *Store) ListParams(ctx context.Context) ([]Param, error) {
	rows, err := s.db.Query(ctx, listParamsSQL)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	var params []Param
	for rows.Next() {
		var p Param
		if err := rows.Scan(&p.Key, &p.Value, &p.Unit, &p.UpdatedBy, &p.UpdatedAt); err != nil {
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
SELECT key, value::text, unit, updated_by, updated_at
  FROM parameter WHERE key = $1`, key).
		Scan(&p.Key, &p.Value, &p.Unit, &p.UpdatedBy, &p.UpdatedAt)
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
