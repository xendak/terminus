package app

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"stoptime/internal/store"
)

// Read services (operations.md, "Reads and aggregation"): every output
// comes from one SQL aggregation query; the service layer only parses
// input, applies window defaults, and assembles plain structs. Driver
// scoping is a query-level filter (never post-filtering); the role
// matrix that forces it is T6's card.

// RouteView is the GetRoute output: route row with driver name, the
// ordered stops (with locations), and the SQL-computed totals.
type RouteView struct {
	store.RouteWithTotals
	Stops []store.StopDetail
}

func (s *Services) GetRoute(ctx context.Context, actor Actor, routeID uuid.UUID) (RouteView, error) {
	if err := s.allow(actor, OpGetRoute); err != nil {
		return RouteView{}, err
	}
	rt, err := s.Store.RouteWithTotals(ctx, routeID)
	if err != nil {
		return RouteView{}, mapErr(err)
	}
	// "Own only" (operations.md): drivers may read their own routes.
	if err := s.ownRoute(actor, rt.DriverUserID); err != nil {
		return RouteView{}, err
	}
	stops, err := s.Store.RouteStopDetails(ctx, routeID)
	if err != nil {
		return RouteView{}, mapErr(err)
	}
	return RouteView{RouteWithTotals: rt, Stops: stops}, nil
}

// ListRoutesInput: optional window (defaults to the current month),
// optional driver and status filters.
type ListRoutesInput struct {
	From        *string
	To          *string
	DriverUserID *uuid.UUID
	Status      *string
}

func (s *Services) ListRoutes(ctx context.Context, actor Actor, in ListRoutesInput) ([]store.RouteListRow, error) {
	if err := s.allow(actor, OpListRoutes); err != nil {
		return nil, err
	}
	// "Own only": drivers' queries are scoped to their own routes — a
	// filter, never a post-filter.
	if actor.Role == "driver" {
		id := actor.UserID
		in.DriverUserID = &id
	}
	now := s.Now()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	to := now.Format("2006-01-02")
	if in.From != nil {
		f, err := parseDate("from", *in.From)
		if err != nil {
			return nil, err
		}
		from = f
	}
	if in.To != nil {
		t, err := parseDate("to", *in.To)
		if err != nil {
			return nil, err
		}
		to = t
	}
	if from > to {
		return nil, &FieldError{Field: "from", Reason: "window start after end"}
	}
	routes, err := s.Store.ListRoutes(ctx, from, to, in.DriverUserID, in.Status)
	return routes, mapErr(err)
}

// DashboardInput: a required window plus the optional driver scope.
type DashboardInput struct {
	From        string
	To          string
	DriverUserID *uuid.UUID
}

func (s *Services) dashboardWindow(in DashboardInput) (string, string, error) {
	from, err := parseDate("from", in.From)
	if err != nil {
		return "", "", err
	}
	to, err := parseDate("to", in.To)
	if err != nil {
		return "", "", err
	}
	if from > to {
		return "", "", &FieldError{Field: "from", Reason: "window start after end"}
	}
	return from, to, nil
}

func (s *Services) GetDashboardByDay(ctx context.Context, actor Actor, in DashboardInput) ([]store.DayPoint, error) {
	if err := s.allow(actor, OpGetDashboardByDay); err != nil {
		return nil, err
	}
	forceDriverScope(actor, &in)
	from, to, err := s.dashboardWindow(in)
	if err != nil {
		return nil, err
	}
	points, err := s.Store.DashboardByDay(ctx, from, to, in.DriverUserID)
	return points, mapErr(err)
}

func (s *Services) GetDashboardByMonth(ctx context.Context, actor Actor, in DashboardInput) ([]store.MonthPoint, error) {
	if err := s.allow(actor, OpGetDashboardByMonth); err != nil {
		return nil, err
	}
	forceDriverScope(actor, &in)
	from, to, err := s.dashboardWindow(in)
	if err != nil {
		return nil, err
	}
	points, err := s.Store.DashboardByMonth(ctx, from, to, in.DriverUserID)
	return points, mapErr(err)
}

// DriverSummary is one by_driver row of the period summary.
type DriverSummary struct {
	DriverName        string `json:"driver_name"`
	TotalStoppedMinut int    `json:"total_stopped_minutes"`
	JourneyPercent    string `json:"journey_percent"`
}

// PeriodSummary is the GetDashboardByPeriod output. The journey percent
// is the uniform RN04 formula (total seconds over one standard journey
// day) — the interpretation is recorded in plans/mvp/notes.md.
type PeriodSummary struct {
	TotalStoppedMinut int             `json:"total_stopped_minutes"`
	JourneyPercent    string          `json:"journey_percent"`
	RoutesCount       int             `json:"routes_count"`
	ByDriver          []DriverSummary `json:"by_driver"`
}

func (s *Services) GetDashboardByPeriod(ctx context.Context, actor Actor, in DashboardInput) (PeriodSummary, error) {
	if err := s.allow(actor, OpGetDashboardByPeriod); err != nil {
		return PeriodSummary{}, err
	}
	forceDriverScope(actor, &in)
	from, to, err := s.dashboardWindow(in)
	if err != nil {
		return PeriodSummary{}, err
	}
	rows, err := s.Store.DashboardByPeriod(ctx, from, to, in.DriverUserID)
	if err != nil {
		return PeriodSummary{}, mapErr(err)
	}
	var summary PeriodSummary
	for _, r := range rows {
		if r.IsTotal == 1 {
			summary.TotalStoppedMinut = r.TotalStoppedMinut
			summary.JourneyPercent = r.JourneyPercent
			summary.RoutesCount = r.RoutesCount
			continue
		}
		summary.ByDriver = append(summary.ByDriver, DriverSummary{
			DriverName:        *r.DriverName,
			TotalStoppedMinut: r.TotalStoppedMinut,
			JourneyPercent:    r.JourneyPercent,
		})
	}
	return summary, nil
}

// forceDriverScope implements the "own data only" cells: a driver's
// dashboard queries always filter to their own routes, regardless of
// what the input asked for.
func forceDriverScope(actor Actor, in *DashboardInput) {
	if actor.Role == "driver" {
		id := actor.UserID
		in.DriverUserID = &id
	}
}

// parseDate validates and normalizes a YYYY-MM-DD input field;
// missing or malformed is ErrBadInput (operations.md error model).
func parseDate(field, value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("%w: %s is required", ErrBadInput, field)
	}
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return "", fmt.Errorf("%w: %s must be YYYY-MM-DD", ErrBadInput, field)
	}
	return value, nil
}
