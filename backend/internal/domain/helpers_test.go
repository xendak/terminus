package domain_test

import (
	"errors"
	"math/big"
	"testing"
	"time"
)

// goldenZone: the golden fixture stores -03:00 offsets on purpose, to
// catch conversion bugs (business-rules.md, Edge cases). All helpers
// build timestamps in that zone; differences are zone-independent.
var goldenZone = time.FixedZone("UTC-3", -3*60*60)

// at returns a pointer to a timestamp in the golden month at the given
// wall clock, so tests read like the fixture tables.
func at(day, hour, min, sec int) *time.Time {
	t := time.Date(2026, time.June, day, hour, min, sec, 0, goldenZone)
	return &t
}

// rat parses an exact decimal ("6.09", "8", "205/24"); panics on bad
// input — a broken literal is a test bug, not a runtime case.
func rat(s string) *big.Rat {
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		panic("bad decimal literal: " + s)
	}
	return r
}

func assertEq[T comparable](t *testing.T, name string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func assertRat(t *testing.T, name string, got, want *big.Rat) {
	t.Helper()
	if got.Cmp(want) != 0 {
		t.Errorf("%s = %s, want %s", name, got.RatString(), want.RatString())
	}
}

func assertErr(t *testing.T, name string, err, want error) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Errorf("%s = %v, want %v", name, err, want)
	}
}
