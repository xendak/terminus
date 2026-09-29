package domain_test

import (
	"testing"

	"stoptime/internal/domain"
)

// RN03 floor semantics: totals are summed in exact seconds and floored
// once for display — never sum per-stop floored minutes. Three stops of
// 59 seconds are 2 whole minutes, not 0.
func TestSumOfSecondsFlooredOnce(t *testing.T) {
	r := domain.Route{Stops: []domain.Stop{
		{Order: 1, Arrival: at(15, 8, 0, 0), Departure: at(15, 8, 1, 0)},     // departure point
		{Order: 2, Arrival: at(15, 9, 0, 0), Departure: at(15, 9, 0, 59)},   // 59 s
		{Order: 3, Arrival: at(15, 9, 30, 0), Departure: at(15, 9, 30, 59)}, // 59 s
		{Order: 4, Arrival: at(15, 10, 0, 0), Departure: at(15, 10, 0, 59)},  // 59 s
	}}
	total := domain.RouteTotalSeconds(r, 0)
	assertEq(t, "RouteTotalSeconds", total, 3*59) // 177 seconds, not 0
	assertEq(t, "WholeMinutes", domain.WholeMinutes(total), 2)
}

// The min_stop_minutes threshold compares WHOLE minutes (RN03/RF10):
// 359 seconds is 5 whole minutes and drops at threshold 6; 360 seconds
// is 6 whole minutes and stays.
func TestMinStopThresholdBoundary(t *testing.T) {
	under := domain.Route{Stops: []domain.Stop{
		{Order: 2, Arrival: at(15, 9, 0, 0), Departure: at(15, 9, 5, 59)}, // 359 s
	}}
	assertEq(t, "359s at threshold 6", domain.RouteTotalSeconds(under, 6), 0)

	over := domain.Route{Stops: []domain.Stop{
		{Order: 2, Arrival: at(15, 9, 0, 0), Departure: at(15, 9, 6, 0)}, // 360 s
	}}
	assertEq(t, "360s at threshold 6", domain.RouteTotalSeconds(over, 6), 360)

	// Default threshold 0 keeps RN03 pure: everything counted, including
	// sub-minute stops.
	assertEq(t, "59s at threshold 0", domain.RouteTotalSeconds(under, 0), 359)
}
