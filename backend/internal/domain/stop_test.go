package domain_test

import (
	"errors"
	"testing"

	"stoptime/internal/domain"
)

// StopSeconds mirrors the generated stop_seconds column (RN01, RN02):
// order 1 is always 0; open stops are 0; otherwise full seconds, never
// floored minutes.
func TestStopSeconds(t *testing.T) {
	assertEq(t, "order 1 with timestamps",
		domain.StopSeconds(domain.Stop{Order: 1, Arrival: at(15, 8, 0, 0), Departure: at(15, 8, 5, 0)}), 0)
	assertEq(t, "90-second stop",
		domain.StopSeconds(domain.Stop{Order: 2, Arrival: at(15, 9, 0, 0), Departure: at(15, 9, 1, 30)}), 90)
	assertEq(t, "59-second stop",
		domain.StopSeconds(domain.Stop{Order: 2, Arrival: at(15, 9, 0, 0), Departure: at(15, 9, 0, 59)}), 59)
	assertEq(t, "open stop, arrival only",
		domain.StopSeconds(domain.Stop{Order: 2, Arrival: at(15, 9, 0, 0)}), 0)
	assertEq(t, "untouched stop",
		domain.StopSeconds(domain.Stop{Order: 2}), 0)
}

// Edge case (business-rules.md): a stop spanning midnight belongs to its
// route date and counts in full — no day splitting.
func TestStopSecondsMidnightSpan(t *testing.T) {
	got := domain.StopSeconds(domain.Stop{
		Order:     3,
		Arrival:   at(15, 23, 50, 0),
		Departure: at(16, 0, 20, 0),
	})
	assertEq(t, "midnight span", got, 30*60)
}

// RN02 ordering rule: departure at or after arrival; nil timestamps are
// valid (untouched or open stop — filling them is a flow concern).
func TestValidateStopTimes(t *testing.T) {
	if err := domain.ValidateStopTimes(nil, nil); err != nil {
		t.Errorf("untouched stop: %v", err)
	}
	if err := domain.ValidateStopTimes(at(15, 9, 0, 0), nil); err != nil {
		t.Errorf("open stop: %v", err)
	}
	if err := domain.ValidateStopTimes(at(15, 9, 0, 0), at(15, 9, 0, 0)); err != nil {
		t.Errorf("zero-length stop: %v", err)
	}
	err := domain.ValidateStopTimes(at(15, 9, 1, 0), at(15, 9, 0, 0))
	assertErr(t, "departure before arrival", err, domain.ErrDepartureBeforeArrival)
}

// RN06: stop orders are the dense sequence 1..n — positive, no gaps, no
// duplicates. An empty route (no stops yet) is valid.
func TestValidateStopOrders(t *testing.T) {
	cases := []struct {
		name    string
		orders  []int
		wantErr bool
	}{
		{"empty", nil, false},
		{"single", []int{1}, false},
		{"dense", []int{1, 2, 3}, false},
		{"zero", []int{0, 1}, true},
		{"negative", []int{-1}, true},
		{"gap", []int{1, 3}, true},
		{"duplicate", []int{1, 1}, true},
		{"not started", []int{2, 3}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := domain.ValidateStopOrders(tc.orders)
			if tc.wantErr != (err != nil) {
				t.Errorf("ValidateStopOrders(%v) = %v, wantErr %v", tc.orders, err, tc.wantErr)
			}
			if err != nil && !errors.Is(err, domain.ErrInvalidStopOrder) {
				t.Errorf("ValidateStopOrders(%v) = %v, want ErrInvalidStopOrder", tc.orders, err)
			}
		})
	}
}
