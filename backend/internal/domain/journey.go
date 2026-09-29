package domain

import "math/big"

// JourneyPercent (RN04): a total of stopped seconds as a percentage of
// the standard journey day.
//
//	percent = total_seconds / (standard_journey_hours * 3600) * 100
//
// Exact rational arithmetic, matching the persistence layer's numeric
// computation; rounding for display happens at the edge, not here.
func JourneyPercent(totalSeconds int, standardJourneyHours *big.Rat) (*big.Rat, error) {
	if standardJourneyHours == nil || standardJourneyHours.Sign() <= 0 {
		return nil, ErrInvalidParameter
	}
	seconds := new(big.Rat).Mul(standardJourneyHours, big.NewRat(3600, 1))
	pct := new(big.Rat).Quo(new(big.Rat).SetInt64(int64(totalSeconds)), seconds)
	return pct.Mul(pct, big.NewRat(100, 1)), nil
}

// PeriodJourneyPercent (RN04 over a period): the base is one standard
// journey day per route, since a route is one driver-day (RN05).
//
//	percent = total_seconds / (routes * standard_journey_hours * 3600) * 100
//
// With one route it equals JourneyPercent. Zero routes has no base.
func PeriodJourneyPercent(totalSeconds, routes int, standardJourneyHours *big.Rat) (*big.Rat, error) {
	if routes <= 0 {
		return nil, ErrInvalidParameter
	}
	pct, err := JourneyPercent(totalSeconds, standardJourneyHours)
	if err != nil {
		return nil, err
	}
	return pct.Quo(pct, big.NewRat(int64(routes), 1)), nil
}
