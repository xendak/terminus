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
	ErrStopTimesOutOfOrder    = domain.ErrStopTimesOutOfOrder

	ErrRouteClosed = errors.New("route is closed")

	ErrUnauthenticated = errors.New("no or invalid session")
	ErrForbidden       = errors.New("role or ownership violation")
)

// FieldError attaches field-level detail to a 422 sentinel —
// ErrValidation unless Err names another (ErrStopTimesOutOfOrder);
// transports show it inline (htmx forms) or as a 422 with details (JSON).
type FieldError struct {
	Field  string
	Reason string
	Err    error // nil = ErrValidation
}

func (e *FieldError) Error() string { return e.Field + ": " + e.Reason }
func (e *FieldError) Unwrap() error {
	if e.Err != nil {
		return e.Err
	}
	return ErrValidation
}

// sequenceFieldError lifts a domain sequence violation into the
// operation error model (field + reason, ErrStopTimesOutOfOrder).
func sequenceFieldError(err error) error {
	var se *domain.SequenceError
	if errors.As(err, &se) {
		return &FieldError{Field: se.Field, Reason: se.Reason, Err: ErrStopTimesOutOfOrder}
	}
	return err
}

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
