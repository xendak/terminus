package domain

import "math/big"

// EstimatedCost (RN07): fuel plus overhead from distance and consumption.
//
//	liters     = distance_km / km_per_l      (driver override or default)
//	fuel_cost  = liters * fuel_price
//	overhead   = distance_km * cost_per_km
//	cost       = fuel_cost + overhead
//
// Exact rationals throughout; RoundMoney applies the single rounding the
// rule allows — once, to 2 places, at the end.
func EstimatedCost(distanceKm, kmPerL, fuelPriceBRL, costPerKmBRL *big.Rat) (*big.Rat, error) {
	if distanceKm == nil || kmPerL == nil || fuelPriceBRL == nil || costPerKmBRL == nil {
		return nil, ErrInvalidParameter
	}
	if distanceKm.Sign() <= 0 {
		return nil, ErrInvalidDistance
	}
	if kmPerL.Sign() <= 0 {
		return nil, ErrInvalidKmPerL
	}
	if fuelPriceBRL.Sign() < 0 || costPerKmBRL.Sign() < 0 {
		return nil, ErrInvalidParameter
	}
	liters := new(big.Rat).Quo(distanceKm, kmPerL)
	fuel := liters.Mul(liters, fuelPriceBRL)
	overhead := new(big.Rat).Mul(distanceKm, costPerKmBRL)
	return fuel.Add(fuel, overhead), nil
}

// RoundMoney rounds an exact amount to 2 decimal places, half away from
// zero — the same semantics as the persistence layer's numeric rounding
// (RN07: "rounded once, to 2 places, at the end").
func RoundMoney(v *big.Rat) *big.Rat {
	cents := new(big.Rat).Mul(v, big.NewRat(100, 1))
	num := new(big.Int).Mul(cents.Num(), big.NewInt(2))
	denom := cents.Denom()
	if num.Sign() >= 0 {
		num.Add(num, denom) // (2n + d) / 2d rounds half up
	} else {
		num.Sub(num, denom) // and half down for negatives: away from zero
	}
	rounded := new(big.Int).Quo(num, new(big.Int).Mul(denom, big.NewInt(2)))
	return new(big.Rat).SetFrac(rounded, big.NewInt(100))
}

// ValidateDistance (RN07): a route distance is optional, but a set one
// must be positive. A route without a distance shows no cost at all —
// that decision belongs to the caller, not this validator.
func ValidateDistance(distanceKm *big.Rat) error {
	if distanceKm == nil {
		return nil
	}
	if distanceKm.Sign() <= 0 {
		return ErrInvalidDistance
	}
	return nil
}
