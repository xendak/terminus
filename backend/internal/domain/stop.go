package domain

import "time"

// StopSeconds mirrors the route_stop.stop_seconds generated column
// (RN01, RN02): order 1 is always 0; a stop missing either timestamp is
// open and contributes 0; otherwise the full seconds between departure
// and arrival. It never floors — minutes are a display concern
// (WholeMinutes).
func StopSeconds(s Stop) int {
	if s.Order == 1 {
		return 0 // RN01: the departure point never counts.
	}
	if s.Arrival == nil || s.Departure == nil {
		return 0 // Open or untouched stop.
	}
	return int(s.Departure.Sub(*s.Arrival) / time.Second)
}

// ValidateStopTimes applies the RN02 ordering rule: a stop with both
// timestamps must not depart before it arrived. Nil timestamps (untouched
// or open stops) are valid; the flow that fills them is a service concern.
func ValidateStopTimes(arrival, departure *time.Time) error {
	if arrival == nil || departure == nil {
		return nil
	}
	if departure.Before(*arrival) {
		return ErrDepartureBeforeArrival
	}
	return nil
}

// ValidateStopOrders checks that stop orders form the dense sequence
// 1..n (RN06): every stop numbered from 1, no gaps, no duplicates. An
// empty route (no stops yet) is valid.
func ValidateStopOrders(orders []int) error {
	max := 0
	seen := make(map[int]struct{}, len(orders))
	for _, order := range orders {
		if order < 1 {
			return ErrInvalidStopOrder
		}
		if _, dup := seen[order]; dup {
			return ErrInvalidStopOrder
		}
		seen[order] = struct{}{}
		if order > max {
			max = order
		}
	}
	// Distinct positive integers are 1..n exactly when the max is n.
	if max != len(orders) {
		return ErrInvalidStopOrder
	}
	return nil
}
