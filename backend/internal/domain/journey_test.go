package domain_test

import (
	"testing"

	"stoptime/internal/domain"
)

// RN04: journey percent is exact rational arithmetic against the
// standard_journey_hours parameter.
func TestJourneyPercent(t *testing.T) {
	// Zero stopped time is 0%.
	pct, err := domain.JourneyPercent(0, rat("8"))
	if err != nil {
		t.Fatalf("JourneyPercent(0, 8): %v", err)
	}
	assertRat(t, "JourneyPercent(0, 8)", pct, rat("0"))

	// Route B: 41 minutes on an 8h day — 2460/28800*100 = 205/24 exactly.
	pct, err = domain.JourneyPercent(2460, rat("8"))
	if err != nil {
		t.Fatalf("JourneyPercent(2460, 8): %v", err)
	}
	assertRat(t, "JourneyPercent(2460, 8)", pct, rat("205/24"))

	// A non-integer percent from a 6h standard day: 4500/21600*100 = 125/6
	// exactly (a repeating decimal — pinned as the exact fraction).
	pct, err = domain.JourneyPercent(4500, rat("6"))
	if err != nil {
		t.Fatalf("JourneyPercent(4500, 6): %v", err)
	}
	assertRat(t, "JourneyPercent(4500, 6)", pct, rat("125/6"))
}

// A zero or negative standard journey day cannot be a percentage base.
func TestJourneyPercentInvalidHours(t *testing.T) {
	if _, err := domain.JourneyPercent(4500, rat("0")); err == nil {
		t.Error("JourneyPercent with 0 hours: want error")
	} else {
		assertErr(t, "zero hours", err, domain.ErrInvalidParameter)
	}
	if _, err := domain.JourneyPercent(4500, rat("-8")); err == nil {
		t.Error("JourneyPercent with negative hours: want error")
	} else {
		assertErr(t, "negative hours", err, domain.ErrInvalidParameter)
	}
}
