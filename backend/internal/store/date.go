package store

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// Date is a SQL `date` (route_date): a calendar day, not an instant. It
// wraps time.Time at UTC midnight so Go code keeps the time API, but it
// serializes as plain "YYYY-MM-DD" (operations.md conventions) and must
// be formatted as-is — converting it to another zone shifts the day.
type Date struct {
	time.Time
}

const dateLayout = "2006-01-02"

// DateOf truncates t to its calendar day (in t's own location).
func DateOf(t time.Time) Date {
	return Date{time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)}
}

// String is the ISO calendar day.
func (d Date) String() string { return d.Format(dateLayout) }

// MarshalJSON writes "YYYY-MM-DD".
func (d Date) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.Format(dateLayout) + `"`), nil
}

// Scan implements sql.Scanner (pgx decodes `date` to time.Time first).
func (d *Date) Scan(src any) error {
	t, ok := src.(time.Time)
	if !ok {
		return fmt.Errorf("store: cannot scan %T into Date", src)
	}
	*d = DateOf(t)
	return nil
}

// Value implements driver.Valuer: the day as ISO text, cast by Postgres.
func (d Date) Value() (driver.Value, error) {
	return d.Format(dateLayout), nil
}
