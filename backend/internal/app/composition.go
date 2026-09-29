package app

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"stoptime/internal/domain"
	"stoptime/internal/store"
)

// Route composition (operations.md): create, edit, start, close, reopen.
// Every audited action (add_stop, remove_stop, reorder, close_route,
// reopen_route) commits with its audit row in one transaction.

// RouteDetail is a route with its ordered stops.
type RouteDetail struct {
	Route store.Route
	Stops []store.RouteStop
}

type CreateRouteInput struct {
	DriverUserID uuid.UUID
	RouteDate    string // YYYY-MM-DD
	LocationIDs  []uuid.UUID
	Note         *string
}

// CreateRoute creates a draft route with stops ordered 1..n in visit
// order; the first location is the departure point (RN01).
func (s *Services) CreateRoute(ctx context.Context, actor Actor, in CreateRouteInput) (RouteDetail, error) {
	if err := s.allow(actor, OpCreateRoute); err != nil {
		return RouteDetail{}, err
	}
	date, err := time.Parse("2006-01-02", in.RouteDate)
	if err != nil {
		return RouteDetail{}, fmt.Errorf("%w: route_date must be YYYY-MM-DD", ErrBadInput)
	}
	if len(in.LocationIDs) < 2 {
		return RouteDetail{}, &FieldError{Field: "location_ids", Reason: "a route needs at least 2 stops"}
	}
	driver, err := s.Store.UserByID(ctx, in.DriverUserID)
	if err != nil {
		return RouteDetail{}, mapErr(err)
	}
	if driver.Role != "driver" {
		return RouteDetail{}, &FieldError{Field: "driver_user_id", Reason: "not a driver"}
	}

	route := store.Route{
		ID: uuid.New(), DriverUserID: in.DriverUserID, RouteDate: store.DateOf(date),
		Status: "draft", Note: in.Note, CreatedBy: actor.UserID,
	}
	err = s.Store.WithTx(ctx, func(tx *store.Store) error {
		if err := tx.InsertRoute(ctx, route); err != nil {
			return err
		}
		return tx.InsertStops(ctx, route.ID, in.LocationIDs)
	})
	if err != nil {
		return RouteDetail{}, mapErr(err)
	}
	return s.routeDetail(ctx, route.ID)
}

type AddStopInput struct {
	RouteID    uuid.UUID
	LocationID uuid.UUID
	Position   *int // default: end
}

// AddStop inserts a stop (renumbering below the end) and audits it.
func (s *Services) AddStop(ctx context.Context, actor Actor, in AddStopInput) (RouteDetail, error) {
	if err := s.allow(actor, OpAddStop); err != nil {
		return RouteDetail{}, err
	}
	err := s.Store.WithTx(ctx, func(tx *store.Store) error {
		r, err := tx.RouteByID(ctx, in.RouteID)
		if err != nil {
			return err
		}
		if r.Status == "closed" {
			return ErrRouteClosed
		}
		n, err := tx.CountStops(ctx, in.RouteID)
		if err != nil {
			return err
		}
		if n >= store.MaxStops {
			return &FieldError{Field: "position", Reason: "route is full"}
		}
		pos := n + 1
		if in.Position != nil {
			pos = *in.Position
			if pos < 1 || pos > n+1 {
				return &FieldError{Field: "position", Reason: fmt.Sprintf("must be between 1 and %d", n+1)}
			}
		}
		if pos <= n {
			if err := tx.ShiftStopOrders(ctx, in.RouteID, pos, +1); err != nil {
				return err
			}
		}
		stop := store.RouteStop{
			ID: uuid.New(), RouteID: in.RouteID, StopOrder: pos, LocationID: in.LocationID,
		}
		if err := tx.InsertStop(ctx, stop); err != nil {
			return err
		}
		return tx.InsertAudit(ctx, store.AuditEntry{
			ActorUserID: actor.UserID,
			Entity:      "route_stop",
			EntityID:    stop.ID.String(),
			Action:      "add_stop",
			OldValues:   map[string]any{},
			NewValues:   map[string]any{"stop_order": pos, "location_id": in.LocationID.String()},
		})
	})
	if err != nil {
		return RouteDetail{}, mapErr(err)
	}
	return s.routeDetail(ctx, in.RouteID)
}

type RemoveStopInput struct {
	RouteID   uuid.UUID
	StopOrder int
}

// RemoveStop deletes a stop and renumbers so orders stay dense (RN06);
// audited. Stop ids of shifted rows are preserved.
func (s *Services) RemoveStop(ctx context.Context, actor Actor, in RemoveStopInput) (RouteDetail, error) {
	if err := s.allow(actor, OpRemoveStop); err != nil {
		return RouteDetail{}, err
	}
	err := s.Store.WithTx(ctx, func(tx *store.Store) error {
		r, err := tx.RouteByID(ctx, in.RouteID)
		if err != nil {
			return err
		}
		if r.Status == "closed" {
			return ErrRouteClosed
		}
		stop, err := tx.StopByOrder(ctx, in.RouteID, in.StopOrder)
		if err != nil {
			return err
		}
		if err := tx.DeleteStop(ctx, in.RouteID, in.StopOrder); err != nil {
			return err
		}
		if err := tx.ShiftStopOrders(ctx, in.RouteID, in.StopOrder+1, -1); err != nil {
			return err
		}
		return tx.InsertAudit(ctx, store.AuditEntry{
			ActorUserID: actor.UserID,
			Entity:      "route_stop",
			EntityID:    stop.ID.String(),
			Action:      "remove_stop",
			OldValues: map[string]any{
				"stop_order":   stop.StopOrder,
				"location_id":  stop.LocationID.String(),
				"arrival_at":   timePtr(stop.ArrivalAt),
				"departure_at": timePtr(stop.DepartureAt),
			},
			NewValues: map[string]any{},
		})
	})
	if err != nil {
		return RouteDetail{}, mapErr(err)
	}
	return s.routeDetail(ctx, in.RouteID)
}

type ReorderStopsInput struct {
	RouteID   uuid.UUID
	StopOrder int
	Direction string // up | down (one position)
}

// ReorderStops moves a stop one position up or down; audited.
func (s *Services) ReorderStops(ctx context.Context, actor Actor, in ReorderStopsInput) (RouteDetail, error) {
	if err := s.allow(actor, OpReorderStops); err != nil {
		return RouteDetail{}, err
	}
	if in.Direction != "up" && in.Direction != "down" {
		return RouteDetail{}, fmt.Errorf("%w: direction must be up or down", ErrBadInput)
	}
	err := s.Store.WithTx(ctx, func(tx *store.Store) error {
		r, err := tx.RouteByID(ctx, in.RouteID)
		if err != nil {
			return err
		}
		if r.Status == "closed" {
			return ErrRouteClosed
		}
		n, err := tx.CountStops(ctx, in.RouteID)
		if err != nil {
			return err
		}
		if in.Direction == "up" && (in.StopOrder < 2 || in.StopOrder > n) {
			return &FieldError{Field: "stop_order", Reason: "cannot move up"}
		}
		if in.Direction == "down" && (in.StopOrder < 1 || in.StopOrder > n-1) {
			return &FieldError{Field: "stop_order", Reason: "cannot move down"}
		}
		moved, err := tx.StopByOrder(ctx, in.RouteID, in.StopOrder)
		if err != nil {
			return err
		}
		targetOrder := in.StopOrder - 1
		if in.Direction == "down" {
			targetOrder = in.StopOrder + 1
		}
		target, err := tx.StopByOrder(ctx, in.RouteID, targetOrder)
		if err != nil {
			return err
		}
		// The store helper swaps order and order-1.
		high := in.StopOrder
		if in.Direction == "down" {
			high = in.StopOrder + 1
		}
		if err := tx.SwapAdjacentStops(ctx, in.RouteID, high); err != nil {
			return err
		}
		return tx.InsertAudit(ctx, store.AuditEntry{
			ActorUserID: actor.UserID,
			Entity:      "route",
			EntityID:    in.RouteID.String(),
			Action:      "reorder",
			OldValues: map[string]any{
				moved.ID.String():  moved.StopOrder,
				target.ID.String(): target.StopOrder,
			},
			NewValues: map[string]any{
				moved.ID.String():  target.StopOrder,
				target.ID.String(): moved.StopOrder,
			},
		})
	})
	if err != nil {
		return RouteDetail{}, mapErr(err)
	}
	return s.routeDetail(ctx, in.RouteID)
}

// StartRoute moves a draft route to active (recording becomes
// possible). Drivers may start only their own routes.
func (s *Services) StartRoute(ctx context.Context, actor Actor, routeID uuid.UUID) (store.Route, error) {
	if err := s.allow(actor, OpStartRoute); err != nil {
		return store.Route{}, err
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
		return r, nil // idempotent
	}
	if err := s.Store.UpdateRouteStatus(ctx, routeID, "active"); err != nil {
		return store.Route{}, mapErr(err)
	}
	r.Status = "active"
	return r, nil
}

type CloseRouteInput struct {
	RouteID    uuid.UUID
	DistanceKm *string // optional: recorded at close
}

// CloseRoute freezes times and composition; audited. Drivers may close
// only their own routes.
func (s *Services) CloseRoute(ctx context.Context, actor Actor, in CloseRouteInput) (store.Route, error) {
	if err := s.allow(actor, OpCloseRoute); err != nil {
		return store.Route{}, err
	}
	r, err := s.Store.RouteByID(ctx, in.RouteID)
	if err != nil {
		return store.Route{}, mapErr(err)
	}
	if err := s.ownRoute(actor, r.DriverUserID); err != nil {
		return store.Route{}, err
	}
	if r.Status == "closed" {
		return store.Route{}, ErrRouteClosed
	}
	if in.DistanceKm != nil {
		d, err := parseDecimal("distance_km", *in.DistanceKm)
		if err != nil {
			return store.Route{}, err
		}
		if err := domain.ValidateDistance(d); err != nil {
			return store.Route{}, fieldErr("distance_km", err)
		}
	}
	err = s.Store.WithTx(ctx, func(tx *store.Store) error {
		if in.DistanceKm != nil {
			if err := tx.UpdateRouteDistance(ctx, in.RouteID, *in.DistanceKm); err != nil {
				return err
			}
		}
		if err := tx.UpdateRouteStatus(ctx, in.RouteID, "closed"); err != nil {
			return err
		}
		return tx.InsertAudit(ctx, store.AuditEntry{
			ActorUserID: actor.UserID,
			Entity:      "route",
			EntityID:    in.RouteID.String(),
			Action:      "close_route",
			OldValues:   map[string]any{"status": r.Status},
			NewValues:   map[string]any{"status": "closed"},
		})
	})
	if err != nil {
		return store.Route{}, mapErr(err)
	}
	// Re-read: the row is the truth (normalized decimals, set distance).
	closed, err := s.Store.RouteByID(ctx, in.RouteID)
	return closed, mapErr(err)
}

// ReopenRoute un-freezes a closed route (admin only per the role
// matrix); audited.
func (s *Services) ReopenRoute(ctx context.Context, actor Actor, routeID uuid.UUID) (store.Route, error) {
	if err := s.allow(actor, OpReopenRoute); err != nil {
		return store.Route{}, err
	}
	r, err := s.Store.RouteByID(ctx, routeID)
	if err != nil {
		return store.Route{}, mapErr(err)
	}
	if r.Status != "closed" {
		return store.Route{}, &FieldError{Field: "route", Reason: "not closed"}
	}
	err = s.Store.WithTx(ctx, func(tx *store.Store) error {
		if err := tx.UpdateRouteStatus(ctx, routeID, "active"); err != nil {
			return err
		}
		return tx.InsertAudit(ctx, store.AuditEntry{
			ActorUserID: actor.UserID,
			Entity:      "route",
			EntityID:    routeID.String(),
			Action:      "reopen_route",
			OldValues:   map[string]any{"status": "closed"},
			NewValues:   map[string]any{"status": "active"},
		})
	})
	if err != nil {
		return store.Route{}, mapErr(err)
	}
	r.Status = "active"
	return r, nil
}

func (s *Services) routeDetail(ctx context.Context, routeID uuid.UUID) (RouteDetail, error) {
	r, err := s.Store.RouteByID(ctx, routeID)
	if err != nil {
		return RouteDetail{}, mapErr(err)
	}
	stops, err := s.Store.StopsByRoute(ctx, routeID)
	if err != nil {
		return RouteDetail{}, mapErr(err)
	}
	return RouteDetail{Route: r, Stops: stops}, nil
}

func timePtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}
