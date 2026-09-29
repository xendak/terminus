package store_test

import (
	"encoding/json"
	"testing"
	"time"

	"stoptime/internal/store"
)

// store.Date is a calendar day in every encoding: JSON and text both use
// "YYYY-MM-DD" (never time.Time's RFC 3339 instant), and decoding
// rejects anything else.
func TestDateEncodings(t *testing.T) {
	d := store.DateOf(time.Date(2026, time.June, 15, 23, 30, 0, 0, time.FixedZone("-03", -3*3600)))

	raw, err := json.Marshal(d)
	if err != nil || string(raw) != `"2026-06-15"` {
		t.Errorf("MarshalJSON = %s, %v", raw, err)
	}
	text, err := d.MarshalText()
	if err != nil || string(text) != "2026-06-15" {
		t.Errorf("MarshalText = %s, %v", text, err)
	}
	// As a map key, encoding/json uses MarshalText.
	keyed, err := json.Marshal(map[store.Date]int{d: 1})
	if err != nil || string(keyed) != `{"2026-06-15":1}` {
		t.Errorf("map key = %s, %v", keyed, err)
	}

	var back store.Date
	if err := json.Unmarshal([]byte(`"2026-06-15"`), &back); err != nil || !back.Equal(d.Time) {
		t.Errorf("UnmarshalJSON = %v, %v", back, err)
	}
	if err := back.UnmarshalText([]byte("2026-07-01")); err != nil || back.String() != "2026-07-01" {
		t.Errorf("UnmarshalText = %v, %v", back, err)
	}
	for _, bad := range []string{`"2026-06-15T00:00:00Z"`, `"15/06/2026"`, `20260615`} {
		if err := json.Unmarshal([]byte(bad), &back); err == nil {
			t.Errorf("UnmarshalJSON(%s) accepted", bad)
		}
	}
	var null store.Date
	if err := json.Unmarshal([]byte(`null`), &null); err != nil || !null.IsZero() {
		t.Errorf("UnmarshalJSON(null) = %v, %v; want zero, no error", null, err)
	}
}
