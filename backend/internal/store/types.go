package store

import (
	"time"

	"github.com/google/uuid"
)

// Plain row structs shared by the store and the service layer. Decimal
// columns (numeric) surface as exact text; nil-able columns as pointers.

type User struct {
	ID           uuid.UUID `db:"id"`
	Name         string    `db:"name"`
	Email        string    `db:"email"`
	Phone        string    `db:"phone"`
	PasswordHash string    `db:"password_hash"`
	Role         string    `db:"role"`
	Active       bool      `db:"active"`
}

// Driver is a user joined with its driver_profile (lists never select
// the password hash).
type Driver struct {
	User
	Document     *string
	VehicleName  *string
	VehiclePlate *string
	KmPerL       *string
}

type DriverProfile struct {
	UserID       uuid.UUID
	Document     *string
	VehicleName  *string
	VehiclePlate *string
	KmPerL       *string
}

type Location struct {
	ID        uuid.UUID `db:"id"`
	Label     string    `db:"label"`
	Address   string    `db:"address"`
	Latitude  *string   `db:"latitude"`
	Longitude *string   `db:"longitude"`
	CreatedBy uuid.UUID `db:"created_by"`
}

type Route struct {
	ID           uuid.UUID `db:"id"`
	DriverUserID uuid.UUID `db:"driver_user_id"`
	RouteDate    time.Time `db:"route_date"`
	DistanceKm   *string   `db:"distance_km"`
	Status       string    `db:"status"`
	Note         *string   `db:"note"`
	CreatedBy    uuid.UUID `db:"created_by"`
}

type RouteStop struct {
	ID          uuid.UUID  `db:"id"`
	RouteID     uuid.UUID  `db:"route_id"`
	StopOrder   int        `db:"stop_order"`
	LocationID  uuid.UUID  `db:"location_id"`
	ArrivalAt   *time.Time `db:"arrival_at"`
	DepartureAt *time.Time `db:"departure_at"`
	Note        *string    `db:"note"`
}

type Param struct {
	Key       string    `db:"key"`
	Value     string    `db:"value"`
	Unit      string    `db:"unit"`
	UpdatedBy uuid.UUID `db:"updated_by"`
	UpdatedAt time.Time `db:"updated_at"`
}

// AuditEntry is one audit_log write (RNF05). OldValues/NewValues are
// Go values the store marshals to jsonb; create-type actions pass an
// empty map as OldValues.
type AuditEntry struct {
	ActorUserID uuid.UUID
	Entity      string // route_stop | route | parameter
	EntityID    string // uuid string, or the parameter key
	Action      string // update_times | add_stop | remove_stop | reorder
	// | close_route | reopen_route | update_param
	OldValues any
	NewValues any
}
