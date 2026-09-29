package app

import (
	"errors"
	"fmt"

	"stoptime/internal/domain"
	"stoptime/internal/store"
)

// Operation error model (docs/spec/operations.md). Database-condition
// sentinels are the store's (aliased — same values, one truth);
// ErrDepartureBeforeArrival is the domain's. Role and session errors
// (ErrForbidden, ErrUnauthenticated) join with T6.
var (
	ErrBadInput   = errors.New("malformed or missing field")
	ErrValidation = errors.New("validation failed")
	ErrNotFound   = store.ErrNotFound

	ErrDriverDateConflict     = store.ErrDriverDateConflict
	ErrDuplicateEmail         = store.ErrDuplicateEmail
	ErrDepartureBeforeArrival = domain.ErrDepartureBeforeArrival

	ErrRouteClosed = errors.New("route is closed")
)

// FieldError attaches field-level detail to ErrValidation; transports
// show it inline (htmx forms) or as a 422 with details (JSON).
type FieldError struct {
	Field  string
	Reason string
}

func (e *FieldError) Error() string { return e.Field + ": " + e.Reason }
func (e *FieldError) Unwrap() error { return ErrValidation }

// mapErr translates the store's constraint reports onto the error model.
// Unmatched errors pass through unchanged.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	var ce *store.ConstraintError
	if errors.As(err, &ce) {
		switch {
		case ce.Code == "23503":
			return fmt.Errorf("%w: %s", ErrNotFound, ce.Constraint)
		case ce.Code == "23514" && ce.Constraint == "route_stop_times_order":
			return domain.ErrDepartureBeforeArrival
		}
	}
	return err
}
