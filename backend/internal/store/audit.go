package store

import (
	"context"
	"encoding/json"
	"fmt"
)

// InsertAudit writes one audit_log row (RNF05). Callers invoke it on the
// transaction-bound Store so the row commits with the mutation it
// records. Values marshal to jsonb; a nil side becomes {} per the spec
// ("empty object on create-type actions").
func (s *Store) InsertAudit(ctx context.Context, e AuditEntry) error {
	oldValues, err := marshalSide(e.OldValues)
	if err != nil {
		return fmt.Errorf("store: audit old_values: %w", err)
	}
	newValues, err := marshalSide(e.NewValues)
	if err != nil {
		return fmt.Errorf("store: audit new_values: %w", err)
	}
	_, err = s.db.Exec(ctx, `
INSERT INTO audit_log (actor_user_id, entity, entity_id, action, old_values, new_values)
VALUES ($1, $2, $3, $4, $5, $6)`,
		e.ActorUserID, e.Entity, e.EntityID, e.Action, oldValues, newValues)
	return translate(err)
}

func marshalSide(v any) (json.RawMessage, error) {
	if v == nil {
		return json.RawMessage(`{}`), nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}
