package domain_test

import (
	"testing"

	"stoptime/internal/domain"
)

// Golden routes A, B, C from tp.md section 5, exactly as the golden seed
// stores them (business-rules.md, "Golden fixture"). Stop 1 carries
// timestamps on purpose: it must still contribute 0 (RN01).

func goldenRouteA() domain.Route {
	return domain.Route{Stops: []domain.Stop{
		{Order: 1, Arrival: at(15, 8, 0, 0), Departure: at(15, 8, 5, 0)},    // departure point
		{Order: 2, Arrival: at(15, 9, 0, 0), Departure: at(15, 9, 15, 0)},   // 15 min
		{Order: 3, Arrival: at(15, 10, 0, 0), Departure: at(15, 10, 10, 0)},  // 10 min
		{Order: 4, Arrival: at(15, 11, 0, 0), Departure: at(15, 11, 50, 0)},  // 50 min
	}}
}

func goldenRouteB() domain.Route {
	return domain.Route{Stops: []domain.Stop{
		{Order: 1, Arrival: at(15, 8, 0, 0), Departure: at(15, 8, 2, 0)},    // departure point
		{Order: 2, Arrival: at(15, 9, 0, 0), Departure: at(15, 9, 10, 0)},   // 10 min
		{Order: 3, Arrival: at(15, 9, 40, 0), Departure: at(15, 9, 45, 0)},  // 5 min
		{Order: 4, Arrival: at(15, 10, 30, 0), Departure: at(15, 10, 56, 0)}, // 26 min
	}}
}

func goldenRouteC() domain.Route {
	return domain.Route{Stops: []domain.Stop{
		{Order: 1, Arrival: at(15, 8, 0, 0), Departure: at(15, 8, 1, 0)},     // departure point
		{Order: 2, Arrival: at(15, 9, 10, 0), Departure: at(15, 9, 15, 0)},  // 5 min
		{Order: 3, Arrival: at(15, 10, 0, 0), Departure: at(15, 10, 10, 0)},  // 10 min
		{Order: 4, Arrival: at(15, 11, 20, 0), Departure: at(15, 11, 50, 0)}, // 30 min
	}}
}

// The three pinned route totals (RN03): 75, 41, 45 minutes.
func TestGoldenTotals(t *testing.T) {
	cases := []struct {
		name        string
		route       domain.Route
		wantSeconds int
		wantMinutes int
	}{
		{"route A", goldenRouteA(), 4500, 75},
		{"route B", goldenRouteB(), 2460, 41},
		{"route C", goldenRouteC(), 2700, 45},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := domain.RouteTotalSeconds(tc.route, 0)
			assertEq(t, "RouteTotalSeconds", got, tc.wantSeconds)
			assertEq(t, "WholeMinutes", domain.WholeMinutes(got), tc.wantMinutes)
		})
	}
}

// Seed convention: all three routes on one date — the day series reads
// 161 minutes (A + B + C, stop 1 contributing nothing).
func TestGoldenDayTotal(t *testing.T) {
	total := 0
	for _, r := range []domain.Route{goldenRouteA(), goldenRouteB(), goldenRouteC()} {
		total += domain.RouteTotalSeconds(r, 0)
	}
	assertEq(t, "day seconds", total, 9660)
	assertEq(t, "day minutes", domain.WholeMinutes(total), 161)
}

// RN04 golden: 75 stopped minutes on the default 8h standard day = 15.625%.
func TestGoldenJourneyPercent(t *testing.T) {
	pct, err := domain.JourneyPercent(4500, rat("8"))
	if err != nil {
		t.Fatalf("JourneyPercent: %v", err)
	}
	assertRat(t, "JourneyPercent", pct, rat("15.625"))
}

// RF10 golden: with min_stop_minutes = 6, route B's 5-minute stop still
// records its timestamps but contributes 0, so B totals 36 minutes.
func TestGoldenRouteBWithMinStopSix(t *testing.T) {
	got := domain.RouteTotalSeconds(goldenRouteB(), 6)
	assertEq(t, "RouteTotalSeconds with threshold 6", got, 2160)
	assertEq(t, "WholeMinutes", domain.WholeMinutes(got), 36)
}
