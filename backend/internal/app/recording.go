package app

import (
	"context"
	"time"

	"github.com/google/uuid"

	"stoptime/internal/domain"
	"stoptime/internal/store"
)

// Time recording (RF05): first-time records fill null fields only;
// corrections go through UpdateStopTimes (audited). Recording on a
// closed route is rejected (business-rules.md, edge cases).

type RecordTimeInput struct {
	RouteID   uuid.UUID
	StopOrder int
	At        *time.Time // default: server clock
}

// RecordArrival sets arrival_at only if currently null; a stop with
// arrival already set is a correction (UpdateStopTimes), not a record.
// Drivers record on their own routes only.
func (s *Services) RecordArrival(ctx context.Context, actor Actor, in RecordTimeInput) (store.RouteStop, error) {
	if err := s.allow(actor, OpRecordArrival); err != nil {
		return store.RouteStop{}, err
	}
	at := in.At
	if at == nil {
		t := s.Now()
		at = &t
	}
	r, err := s.Store.RouteByID(ctx, in.RouteID)
	if err != nil {
		return store.RouteStop{}, mapErr(err)
	}
	if err := s.ownRoute(actor, r.DriverUserID); err != nil {
		return store.RouteStop{}, err
	}
	if err := routeWritableForRecording(r.Status); err != nil {
		return store.RouteStop{}, err
	}
	stop, err := s.Store.StopByOrder(ctx, in.RouteID, in.StopOrder)
	if err != nil {
		return store.RouteStop{}, mapErr(err)
	}
	if stop.ArrivalAt != nil {
		return store.RouteStop{}, &FieldError{
			Field: "arrival_at", Reason: "already recorded; use UpdateStopTimes",
		}
	}
	if err := domain.ValidateStopTimes(at, stop.DepartureAt); err != nil {
		return store.RouteStop{}, err
	}
	if err := s.Store.UpdateStopTimes(ctx, stop.ID, at, stop.DepartureAt); err != nil {
		return store.RouteStop{}, mapErr(err)
	}
	stop.ArrivalAt = at
	return stop, nil
}

// RecordDeparture sets departure_at only if currently null and after
// arrival was recorded. Drivers record on their own routes only.
func (s *Services) RecordDeparture(ctx context.Context, actor Actor, in RecordTimeInput) (store.RouteStop, error) {
	if err := s.allow(actor, OpRecordDeparture); err != nil {
		return store.RouteStop{}, err
	}
	at := in.At
	if at == nil {
		t := s.Now()
		at = &t
	}
	r, err := s.Store.RouteByID(ctx, in.RouteID)
	if err != nil {
		return store.RouteStop{}, mapErr(err)
	}
	if err := s.ownRoute(actor, r.DriverUserID); err != nil {
		return store.RouteStop{}, err
	}
	if err := routeWritableForRecording(r.Status); err != nil {
		return store.RouteStop{}, err
	}
	stop, err := s.Store.StopByOrder(ctx, in.RouteID, in.StopOrder)
	if err != nil {
		return store.RouteStop{}, mapErr(err)
	}
	if stop.ArrivalAt == nil {
		return store.RouteStop{}, &FieldError{
			Field: "arrival_at", Reason: "record arrival before departure",
		}
	}
	if stop.DepartureAt != nil {
		return store.RouteStop{}, &FieldError{
			Field: "departure_at", Reason: "already recorded; use UpdateStopTimes",
		}
	}
	if err := domain.ValidateStopTimes(stop.ArrivalAt, at); err != nil {
		return store.RouteStop{}, err
	}
	if err := s.Store.UpdateStopTimes(ctx, stop.ID, stop.ArrivalAt, at); err != nil {
		return store.RouteStop{}, mapErr(err)
	}
	stop.DepartureAt = at
	return stop, nil
}

// routeWritableForRecording: recording happens on active routes only.
func routeWritableForRecording(status string) error {
	switch status {
	case "closed":
		return ErrRouteClosed
	case "active":
		return nil
	}
	return &FieldError{Field: "route", Reason: "not active"}
}

type UpdateStopTimesInput struct {
	RouteID     uuid.UUID
	StopOrder   int
	ArrivalAt   *time.Time // nil = unchanged
	DepartureAt *time.Time // nil = unchanged
}

// UpdateStopTimes is the audited correction path (RNF05): any effective
// change to a stop's timestamps writes an update_times audit row with
// old and new values, in the same transaction as the write.
func (s *Services) UpdateStopTimes(ctx context.Context, actor Actor, in UpdateStopTimesInput) (store.RouteStop, error) {
	if err := s.allow(actor, OpUpdateStopTimes); err != nil {
		return store.RouteStop{}, err
	}
	r, err := s.Store.RouteByID(ctx, in.RouteID)
	if err != nil {
		return store.RouteStop{}, mapErr(err)
	}
	if r.Status == "closed" {
		return store.RouteStop{}, ErrRouteClosed
	}
	stop, err := s.Store.StopByOrder(ctx, in.RouteID, in.StopOrder)
	if err != nil {
		return store.RouteStop{}, mapErr(err)
	}

	newArrival := stop.ArrivalAt
	if in.ArrivalAt != nil {
		newArrival = in.ArrivalAt
	}
	newDeparture := stop.DepartureAt
	if in.DepartureAt != nil {
		newDeparture = in.DepartureAt
	}
	if sameTime(newArrival, stop.ArrivalAt) && sameTime(newDeparture, stop.DepartureAt) {
		return stop, nil // no effective change, no write, no audit
	}
	if err := domain.ValidateStopTimes(newArrival, newDeparture); err != nil {
		return store.RouteStop{}, err
	}

	err = s.Store.WithTx(ctx, func(tx *store.Store) error {
		if err := tx.UpdateStopTimes(ctx, stop.ID, newArrival, newDeparture); err != nil {
			return err
		}
		return tx.InsertAudit(ctx, store.AuditEntry{
			ActorUserID: actor.UserID,
			Entity:      "route_stop",
			EntityID:    stop.ID.String(),
			Action:      "update_times",
			OldValues: map[string]any{
				"arrival_at":   timePtr(stop.ArrivalAt),
				"departure_at": timePtr(stop.DepartureAt),
			},
			NewValues: map[string]any{
				"arrival_at":   timePtr(newArrival),
				"departure_at": timePtr(newDeparture),
			},
		})
	})
	if err != nil {
		return store.RouteStop{}, mapErr(err)
	}
	stop.ArrivalAt, stop.DepartureAt = newArrival, newDeparture
	return stop, nil
}

// SetRouteDistance records the manually entered distance (RN07).
// Active routes or at close; drivers set their own routes' distance.
func (s *Services) SetRouteDistance(ctx context.Context, actor Actor, routeID uuid.UUID, distanceKm string) (store.Route, error) {
	if err := s.allow(actor, OpSetRouteDistance); err != nil {
		return store.Route{}, err
	}
	d, err := parseDecimal("distance_km", distanceKm)
	if err != nil {
		return store.Route{}, err
	}
	if err := domain.ValidateDistance(d); err != nil {
		return store.Route{}, fieldErr("distance_km", err)
	}
	r, err := s.Store.RouteByID(ctx, routeID)
	if err != nil {
		return store.Route{}, mapErr(err)
	}
	if err := s.ownRoute(actor, r.DriverUserID); err != nil {
		return store.Route{}, err
	}
	switch r.Status {
	case "closed":
		return store.Route{}, ErrRouteClosed
	case "active":
		// allowed
	default:
		return store.Route{}, &FieldError{
			Field: "route", Reason: "distance is set on active routes or at close",
		}
	}
	if err := s.Store.UpdateRouteDistance(ctx, routeID, distanceKm); err != nil {
		return store.Route{}, mapErr(err)
	}
	// Re-read: the row is the truth (normalized decimals).
	return s.Store.RouteByID(ctx, routeID)
}

func sameTime(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(*b)
}
