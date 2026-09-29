package domain_test

import (
	"testing"

	"stoptime/internal/domain"
)

// RN07: estimated cost = liters*fuel_price + distance*cost_per_km, exact
// rationals throughout; RoundMoney applies the single allowed rounding
// (once, to 2 places, at the end).
func TestEstimatedCost(t *testing.T) {
	// 100 km at 10 km/l, fuel 6.09, overhead 0.00:
	// 10 liters * 6.09 = 60.90 exactly.
	cost, err := domain.EstimatedCost(rat("100"), rat("10"), rat("6.09"), rat("0"))
	if err != nil {
		t.Fatalf("EstimatedCost: %v", err)
	}
	assertRat(t, "cost", cost, rat("60.9"))
	assertRat(t, "RoundMoney", domain.RoundMoney(cost), rat("60.9"))

	// 120 km, driver override 12.5 km/l, fuel 6.09, 0.45/km:
	// liters 9.6, fuel 58.464, overhead 54 → exact 112.464, rounded 112.46.
	cost, err = domain.EstimatedCost(rat("120"), rat("12.5"), rat("6.09"), rat("0.45"))
	if err != nil {
		t.Fatalf("EstimatedCost: %v", err)
	}
	assertRat(t, "exact cost", cost, rat("112.464"))
	assertRat(t, "RoundMoney", domain.RoundMoney(cost), rat("112.46"))

	// Repeating liters are exact in rationals: 100 km at 3 km/l is
	// 100/3 liters * 6.09 = 203 exactly.
	cost, err = domain.EstimatedCost(rat("100"), rat("3"), rat("6.09"), rat("0"))
	if err != nil {
		t.Fatalf("EstimatedCost: %v", err)
	}
	assertRat(t, "repeating liters, exact cost", cost, rat("203"))
}

// RoundMoney: half away from zero at the cent boundary, matching the
// persistence layer's numeric rounding.
func TestRoundMoney(t *testing.T) {
	assertRat(t, "58.464", domain.RoundMoney(rat("58.464")), rat("58.46"))
	assertRat(t, "58.465", domain.RoundMoney(rat("58.465")), rat("58.47")) // half rounds away
	assertRat(t, "integer", domain.RoundMoney(rat("60")), rat("60"))
}

// Cost validation (RN07 + card): distance and consumption must be
// positive; prices must not be negative.
func TestEstimatedCostValidation(t *testing.T) {
	_, err := domain.EstimatedCost(rat("0"), rat("10"), rat("6.09"), rat("0"))
	assertErr(t, "zero distance", err, domain.ErrInvalidDistance)

	_, err = domain.EstimatedCost(rat("-5"), rat("10"), rat("6.09"), rat("0"))
	assertErr(t, "negative distance", err, domain.ErrInvalidDistance)

	_, err = domain.EstimatedCost(rat("100"), rat("0"), rat("6.09"), rat("0"))
	assertErr(t, "zero km/l", err, domain.ErrInvalidKmPerL)

	_, err = domain.EstimatedCost(rat("100"), rat("10"), rat("-6.09"), rat("0"))
	assertErr(t, "negative fuel price", err, domain.ErrInvalidParameter)

	_, err = domain.EstimatedCost(rat("100"), rat("10"), rat("6.09"), rat("-0.45"))
	assertErr(t, "negative cost per km", err, domain.ErrInvalidParameter)
}

// A route distance is optional, but a set one must be positive (RN07).
func TestValidateDistance(t *testing.T) {
	if err := domain.ValidateDistance(nil); err != nil {
		t.Errorf("no distance: %v", err)
	}
	if err := domain.ValidateDistance(rat("75.50")); err != nil {
		t.Errorf("valid distance: %v", err)
	}
	assertErr(t, "zero distance", domain.ValidateDistance(rat("0")), domain.ErrInvalidDistance)
	assertErr(t, "negative distance", domain.ValidateDistance(rat("-1")), domain.ErrInvalidDistance)
}
