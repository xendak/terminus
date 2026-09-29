// Package domain holds the stopped-time and cost rules of
// docs/spec/business-rules.md as pure functions over plain values: no
// database, no clock reads, no transport. The database remains the
// persistence truth — this package is the tested oracle that mirrors the
// same formulas for validation (docs/spec/architecture.md, layering
// rule 4).
//
// Rule numbers (RN01–RN07) refer to tp.md section 4. Quantities the
// persistence layer keeps as exact decimals (money, percentages) use
// math/big.Rat here too, so the oracle can never diverge from the
// numeric columns through binary floating point.
package domain

import (
	"errors"
	"time"
)

// Sentinel errors. Adapters map them to transport statuses; services
// return them unwrapped (docs/spec/architecture.md, error contract).
var (
	// ErrDepartureBeforeArrival: both timestamps set but departure
	// precedes arrival (RN02; mapped to status 422).
	ErrDepartureBeforeArrival = errors.New("departure before arrival")

	// ErrInvalidStopOrder: stop orders are not the dense sequence 1..n
	// (RN06).
	ErrInvalidStopOrder = errors.New("invalid stop order")

	// ErrInvalidDistance: a set route distance is zero or negative (RN07).
	ErrInvalidDistance = errors.New("invalid distance")

	// ErrInvalidKmPerL: zero or negative consumption in a cost computation
	// (RN07).
	ErrInvalidKmPerL = errors.New("invalid km per liter")

	// ErrInvalidParameter: a non-positive standard journey day or a
	// negative price in a computation (RN04, RN07).
	ErrInvalidParameter = errors.New("invalid parameter")
)

// Stop mirrors one route_stop row as the domain sees it. Order 1 is the
// departure point; nil timestamps are not yet recorded.
type Stop struct {
	Order     int
	Arrival   *time.Time
	Departure *time.Time
}

// Route mirrors a route: its ordered stops. Identity, driver, date, and
// status ride on the service structs; the rules need only the stops.
type Route struct {
	Stops []Stop
}
