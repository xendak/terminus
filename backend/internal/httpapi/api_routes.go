package httpapi

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"stoptime/internal/app"
	"stoptime/internal/store"
)

// JSON mirrors for the builder + tracker operations (operations.md
// transports). Composition mutations answer with the fresh route view
// (totals included); record actions answer with the stop per the
// contract.

// apiRoute is the GetRoute wire shape: the route row with its SQL
// aggregates plus the ordered stops, under one "route" key.
type apiRoute struct {
	store.RouteWithTotals
	Stops []store.StopDetail `json:"stops"`
}

func routeJSONBody(view app.RouteView) apiRoute {
	return apiRoute{RouteWithTotals: view.RouteWithTotals, Stops: view.Stops}
}

type apiCreateRouteBody struct {
	DriverUserID uuid.UUID `json:"driver_user_id"`
	RouteDate    string    `json:"route_date"`
	LocationIDs  []uuid.UUID `json:"location_ids"`
	Note         *string   `json:"note"`
}

func (s *Server) apiCreateRoute(w http.ResponseWriter, r *http.Request) {
	var body apiCreateRouteBody
	if err := decodeJSON(r, &body); err != nil {
		writeJSONError(w, err)
		return
	}
	actor, _ := s.actor(r)
	detail, err := s.svc.CreateRoute(r.Context(), actor, app.CreateRouteInput{
		DriverUserID: body.DriverUserID, RouteDate: body.RouteDate,
		LocationIDs: body.LocationIDs, Note: body.Note,
	})
	if err != nil {
		writeJSONError(w, err)
		return
	}
	// Answer with the full view (totals included).
	view, err := s.svc.GetRoute(r.Context(), actor, detail.Route.ID)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"route": routeJSONBody(view)})
}

func (s *Server) apiGetRoute(w http.ResponseWriter, r *http.Request) {
	id, err := routeIDFrom(r)
	if err != nil {
		writeJSONError(w, fmt.Errorf("%w: invalid id", app.ErrBadInput))
		return
	}
	actor, _ := s.actor(r)
	view, err := s.svc.GetRoute(r.Context(), actor, id)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"route": routeJSONBody(view)})
}

// viewAfter runs a composition mutation and answers with the fresh view.
func (s *Server) viewAfter(w http.ResponseWriter, r *http.Request, routeID uuid.UUID, run func() error) {
	actor, _ := s.actor(r)
	if err := run(); err != nil {
		writeJSONError(w, err)
		return
	}
	view, err := s.svc.GetRoute(r.Context(), actor, routeID)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"route": routeJSONBody(view)})
}

type apiAddStopBody struct {
	LocationID uuid.UUID `json:"location_id"`
	Position   *int      `json:"position"`
}

func (s *Server) apiAddStop(w http.ResponseWriter, r *http.Request) {
	id, err := routeIDFrom(r)
	if err != nil {
		writeJSONError(w, fmt.Errorf("%w: invalid id", app.ErrBadInput))
		return
	}
	var body apiAddStopBody
	if err := decodeJSON(r, &body); err != nil {
		writeJSONError(w, err)
		return
	}
	actor, _ := s.actor(r)
	s.viewAfter(w, r, id, func() error {
		_, err := s.svc.AddStop(r.Context(), actor, app.AddStopInput{
			RouteID: id, LocationID: body.LocationID, Position: body.Position,
		})
		return err
	})
}

func (s *Server) apiRemoveStop(w http.ResponseWriter, r *http.Request) {
	id, err := routeIDFrom(r)
	if err != nil {
		writeJSONError(w, fmt.Errorf("%w: invalid id", app.ErrBadInput))
		return
	}
	order, err := orderFrom(r)
	if err != nil {
		writeJSONError(w, fmt.Errorf("%w: invalid order", app.ErrBadInput))
		return
	}
	actor, _ := s.actor(r)
	s.viewAfter(w, r, id, func() error {
		_, err := s.svc.RemoveStop(r.Context(), actor, app.RemoveStopInput{RouteID: id, StopOrder: order})
		return err
	})
}

type apiMoveStopBody struct {
	Direction string `json:"direction"`
}

func (s *Server) apiMoveStop(w http.ResponseWriter, r *http.Request) {
	id, err := routeIDFrom(r)
	if err != nil {
		writeJSONError(w, fmt.Errorf("%w: invalid id", app.ErrBadInput))
		return
	}
	order, err := orderFrom(r)
	if err != nil {
		writeJSONError(w, fmt.Errorf("%w: invalid order", app.ErrBadInput))
		return
	}
	var body apiMoveStopBody
	if err := decodeJSON(r, &body); err != nil {
		writeJSONError(w, err)
		return
	}
	actor, _ := s.actor(r)
	s.viewAfter(w, r, id, func() error {
		_, err := s.svc.ReorderStops(r.Context(), actor, app.ReorderStopsInput{
			RouteID: id, StopOrder: order, Direction: body.Direction,
		})
		return err
	})
}

func (s *Server) apiStartRoute(w http.ResponseWriter, r *http.Request) {
	id, err := routeIDFrom(r)
	if err != nil {
		writeJSONError(w, fmt.Errorf("%w: invalid id", app.ErrBadInput))
		return
	}
	actor, _ := s.actor(r)
	s.viewAfter(w, r, id, func() error {
		_, err := s.svc.StartRoute(r.Context(), actor, id)
		return err
	})
}

type apiCloseRouteBody struct {
	DistanceKm *string `json:"distance_km"`
}

func (s *Server) apiCloseRoute(w http.ResponseWriter, r *http.Request) {
	id, err := routeIDFrom(r)
	if err != nil {
		writeJSONError(w, fmt.Errorf("%w: invalid id", app.ErrBadInput))
		return
	}
	var body apiCloseRouteBody
	if r.ContentLength > 0 {
		if err := decodeJSON(r, &body); err != nil {
			writeJSONError(w, err)
			return
		}
	}
	actor, _ := s.actor(r)
	s.viewAfter(w, r, id, func() error {
		_, err := s.svc.CloseRoute(r.Context(), actor, app.CloseRouteInput{
			RouteID: id, DistanceKm: body.DistanceKm,
		})
		return err
	})
}

func (s *Server) apiReopenRoute(w http.ResponseWriter, r *http.Request) {
	id, err := routeIDFrom(r)
	if err != nil {
		writeJSONError(w, fmt.Errorf("%w: invalid id", app.ErrBadInput))
		return
	}
	actor, _ := s.actor(r)
	s.viewAfter(w, r, id, func() error {
		_, err := s.svc.ReopenRoute(r.Context(), actor, id)
		return err
	})
}

type apiDistanceBody struct {
	DistanceKm string `json:"distance_km"`
}

func (s *Server) apiSetDistance(w http.ResponseWriter, r *http.Request) {
	id, err := routeIDFrom(r)
	if err != nil {
		writeJSONError(w, fmt.Errorf("%w: invalid id", app.ErrBadInput))
		return
	}
	var body apiDistanceBody
	if err := decodeJSON(r, &body); err != nil {
		writeJSONError(w, err)
		return
	}
	actor, _ := s.actor(r)
	s.viewAfter(w, r, id, func() error {
		_, err := s.svc.SetRouteDistance(r.Context(), actor, id, body.DistanceKm)
		return err
	})
}

type apiRecordTimeBody struct {
	At *string `json:"at"` // RFC 3339
}

func (s *Server) apiArrive(w http.ResponseWriter, r *http.Request) {
	s.apiRecordTime(w, r, true)
}

func (s *Server) apiDepart(w http.ResponseWriter, r *http.Request) {
	s.apiRecordTime(w, r, false)
}

func (s *Server) apiRecordTime(w http.ResponseWriter, r *http.Request, arrival bool) {
	id, err := routeIDFrom(r)
	if err != nil {
		writeJSONError(w, fmt.Errorf("%w: invalid id", app.ErrBadInput))
		return
	}
	order, err := orderFrom(r)
	if err != nil {
		writeJSONError(w, fmt.Errorf("%w: invalid order", app.ErrBadInput))
		return
	}
	var body apiRecordTimeBody
	if r.ContentLength > 0 {
		if err := decodeJSON(r, &body); err != nil {
			writeJSONError(w, err)
			return
		}
	}
	var at *time.Time
	if body.At != nil && *body.At != "" {
		t, err := time.Parse(time.RFC3339, *body.At)
		if err != nil {
			writeJSONError(w, fmt.Errorf("%w: at must be RFC 3339", app.ErrBadInput))
			return
		}
		at = &t
	}
	actor, _ := s.actor(r)
	in := app.RecordTimeInput{RouteID: id, StopOrder: order, At: at}
	var (
		stop store.RouteStop
		serr error
	)
	if arrival {
		stop, serr = s.svc.RecordArrival(r.Context(), actor, in)
	} else {
		stop, serr = s.svc.RecordDeparture(r.Context(), actor, in)
	}
	if serr != nil {
		writeJSONError(w, serr)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stop": stop})
}

type apiStopTimesBody struct {
	ArrivalAt   *string `json:"arrival_at"`   // RFC 3339; absent/empty keeps the current value
	DepartureAt *string `json:"departure_at"` // RFC 3339; absent/empty keeps the current value
}

// apiCorrectTimes is the JSON transport of UpdateStopTimes (manager/
// admin correction, audited); answers with the stop like the record
// actions.
func (s *Server) apiCorrectTimes(w http.ResponseWriter, r *http.Request) {
	id, err := routeIDFrom(r)
	if err != nil {
		writeJSONError(w, fmt.Errorf("%w: invalid id", app.ErrBadInput))
		return
	}
	order, err := orderFrom(r)
	if err != nil {
		writeJSONError(w, fmt.Errorf("%w: invalid order", app.ErrBadInput))
		return
	}
	var body apiStopTimesBody
	if err := decodeJSON(r, &body); err != nil {
		writeJSONError(w, err)
		return
	}
	arrival, err := parseOptionalRFC3339("arrival_at", body.ArrivalAt)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	departure, err := parseOptionalRFC3339("departure_at", body.DepartureAt)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	actor, _ := s.actor(r)
	stop, err := s.svc.UpdateStopTimes(r.Context(), actor, app.UpdateStopTimesInput{
		RouteID: id, StopOrder: order, ArrivalAt: arrival, DepartureAt: departure,
	})
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stop": stop})
}

// parseOptionalRFC3339 reads an optional JSON timestamp: absent or
// empty is nil (keep), malformed is ErrBadInput.
func parseOptionalRFC3339(field string, v *string) (*time.Time, error) {
	if v == nil || *v == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *v)
	if err != nil {
		return nil, fmt.Errorf("%w: %s must be RFC 3339", app.ErrBadInput, field)
	}
	return &t, nil
}
