package app

import (
	"context"

	"stoptime/internal/store"
)

// Parameters (RF09/RF10): every tunable number lives in the parameter
// table; nothing is hardcoded. UpdateParam is audited (update_param).

func (s *Services) GetParams(ctx context.Context) ([]store.Param, error) {
	params, err := s.Store.ListParams(ctx)
	return params, mapErr(err)
}

type UpdateParamInput struct {
	Key   string
	Value string
}

// UpdateParam sets a parameter's value. Negative values are rejected;
// the two values that act as divisors (standard_journey_hours,
// default_km_per_l) must stay positive.
func (s *Services) UpdateParam(ctx context.Context, actor Actor, in UpdateParamInput) (store.Param, error) {
	p, err := s.Store.ParamByKey(ctx, in.Key)
	if err != nil {
		return store.Param{}, mapErr(err)
	}
	v, err := parseDecimal("value", in.Value)
	if err != nil {
		return store.Param{}, err
	}
	if v.Sign() < 0 {
		return store.Param{}, &FieldError{Field: "value", Reason: "must not be negative"}
	}
	if v.Sign() == 0 && (in.Key == "standard_journey_hours" || in.Key == "default_km_per_l") {
		return store.Param{}, &FieldError{Field: "value", Reason: "must be positive"}
	}

	now := s.Now()
	err = s.Store.WithTx(ctx, func(tx *store.Store) error {
		if err := tx.UpdateParamValue(ctx, in.Key, in.Value, actor.UserID, now); err != nil {
			return err
		}
		return tx.InsertAudit(ctx, store.AuditEntry{
			ActorUserID: actor.UserID,
			Entity:      "parameter",
			EntityID:    in.Key,
			Action:      "update_param",
			OldValues:   map[string]any{"value": p.Value},
			NewValues:   map[string]any{"value": in.Value},
		})
	})
	if err != nil {
		return store.Param{}, mapErr(err)
	}
	p.Value = in.Value
	p.UpdatedBy = actor.UserID
	p.UpdatedAt = now
	return p, nil
}
