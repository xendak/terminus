package app

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"

	"stoptime/internal/store"
)

// Services implements the operations of docs/spec/operations.md. Every
// audited mutation runs in one transaction with its audit row (RNF05);
// services speak plain structs only — no transport types, no SQL.
type Services struct {
	Store *store.Store
	Now   func() time.Time // injected clock: recorded times come from the server clock
}

// Actor is the acting user of an operation. The session provides it
// (T6); tests construct it directly. Role enforcement per the
// operations.md matrix is T6's card — the write path records who acted
// (created_by, updated_by, audit rows).
type Actor struct {
	UserID uuid.UUID
	Role   string
}

func New(st *store.Store) *Services {
	return &Services{Store: st, Now: time.Now}
}

func requireNonEmpty(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return &FieldError{Field: field, Reason: "required"}
	}
	return nil
}

// parseDecimal parses exact decimal text ("75.50", "6.09"); malformed
// input is ErrBadInput. Values stay exact rationals end to end.
func parseDecimal(field, s string) (*big.Rat, error) {
	r, ok := new(big.Rat).SetString(strings.TrimSpace(s))
	if !ok {
		return nil, fmt.Errorf("%w: %s must be a decimal number", ErrBadInput, field)
	}
	return r, nil
}

func fieldErr(field string, err error) error {
	return &FieldError{Field: field, Reason: err.Error()}
}
