package store

import (
	"time"

	"github.com/google/uuid"
)

// Plain row structs shared by the store and the service layer. Decimal
// columns (numeric) surface as exact text; nil-able columns as pointers.

type User struct {
	ID           uuid.UUID `db:"id" json:"id"`
	Name         string    `db:"name" json:"name"`
	Email        string    `db:"email" json:"email"`
	Phone        string    `db:"phone" json:"phone"`
	PasswordHash string    `db:"password_hash" json:"-"` // never serialized
	Role         string    `db:"role" json:"role"`
	Active       bool      `db:"active" json:"active"`
}

// Driver is a user joined with its driver_profile (lists never select
// the password hash).
type Driver struct {
	User
	Document     *string `json:"document,omitempty"`
	VehicleName  *string `json:"vehicle_name,omitempty"`
	VehiclePlate *string `json:"vehicle_plate,omitempty"`
	KmPerL       *string `json:"km_per_l,omitempty"`
}

type DriverProfile struct {
	UserID       uuid.UUID
	Document     *string
	VehicleName  *string
	VehiclePlate *string
	KmPerL       *string
}

type Location struct {
	ID        uuid.UUID `db:"id" json:"id"`
	Label     string    `db:"label" json:"label"`
	Address   string    `db:"address" json:"address"`
	Latitude  *string   `db:"latitude" json:"latitude"`
	Longitude *string   `db:"longitude" json:"longitude"`
	CreatedBy uuid.UUID `db:"created_by" json:"created_by"`
}

type Route struct {
	ID           uuid.UUID `db:"id" json:"id"`
	DriverUserID uuid.UUID `db:"driver_user_id" json:"driver_user_id"`
	RouteDate    time.Time `db:"route_date" json:"route_date"`
	DistanceKm   *string   `db:"distance_km" json:"distance_km"`
	Status       string    `db:"status" json:"status"`
	Note         *string   `db:"note" json:"note"`
	CreatedBy    uuid.UUID `db:"created_by" json:"created_by"`
}

type RouteStop struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	RouteID     uuid.UUID  `db:"route_id" json:"route_id"`
	StopOrder   int        `db:"stop_order" json:"stop_order"`
	LocationID  uuid.UUID  `db:"location_id" json:"location_id"`
	ArrivalAt   *time.Time `db:"arrival_at" json:"arrival_at"`
	DepartureAt *time.Time `db:"departure_at" json:"departure_at"`
	Note        *string    `db:"note" json:"note"`
}

type Param struct {
	Key       string    `db:"key" json:"key"`
	Value     string    `db:"value" json:"value"`
	Unit      string    `db:"unit" json:"unit"`
	UpdatedBy uuid.UUID `db:"updated_by" json:"updated_by"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
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
