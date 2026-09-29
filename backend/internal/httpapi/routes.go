package httpapi

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"

	"stoptime/internal/app"
	"stoptime/internal/store"
)

// Route builder (screens.md §2) and route tracker (screens.md §3).
// Fragment mutations re-render #route-body — the htmx swap contract
// (hx-target="#route-body" hx-swap="outerHTML"); service errors render
// the fragment's inline error state. Drivers act only on their own
// routes (service-enforced); managers read the tracker (screens.md:
// manager/admin read-only in the active state).

var routeLabels = labelsFor(map[string]string{
	"Title":         "Route",
	"BuilderTitle":  "New route",
	"Driver":        "Driver",
	"Date":          "Date",
	"Note":          "Note",
	"Stops":         "Stops",
	"EmptyList":     "Pick locations in visit order — the first is the departure point.",
	"AddStop":       "Location",
	"AddStopBtn":    "Add stop",
	"Submit":        "Create route",
	"Conflict":      "This driver already has a route on that date:",
	"ConflictLink":  "open it",
	"Status":        "Status",
	"StatusDraft":   "draft",
	"StatusActive":  "active",
	"StatusClosed":  "closed",
	"Start":         "Start route",
	"Close":         "Close route",
	"Reopen":        "Reopen route",
	"Distance":      "Distance (km)",
	"Save":          "Save",
	"Arrive":        "Arrive now",
	"Depart":        "Depart now",
	"ManualArrive":  "Save arrival",
	"ManualDepart":  "Save departure",
	"Departure":     "departure point",
	"TotalMinutes":  "Total stopped",
	"Minutes":       "min",
	"Percent":       "Journey",
	"Cost":          "Cost",
	"NoCost":        "No distance recorded — no cost computed.",
	"NoRoute":       "No route assigned for today.",
})

type builderPageData struct {
	Drivers          []store.Driver
	Locations        []store.Location
	Date             string
	ConflictRouteID  *uuid.UUID // the RN05 invalid state links to the existing route
}

func (s *Server) routeNewPage(w http.ResponseWriter, r *http.Request) {
	actor, _ := s.actor(r)
	drivers, err := s.svc.ListDrivers(r.Context(), actor, false)
	if err != nil {
		http.Error(w, "forbidden", statusFor(err))
		return
	}
	locations, err := s.svc.ListLocations(r.Context(), actor, nil)
	if err != nil {
		http.Error(w, "forbidden", statusFor(err))
		return
	}
	s.render(w, r, http.StatusOK, "route_new", builderPageData{
		Drivers: drivers, Locations: locations,
	}, routeLabels, nil)
}

func (s *Server) routeCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	actor, _ := s.actor(r)
	driverID, err := uuid.Parse(r.PostFormValue("driver_user_id"))
	if err != nil {
		s.builderError(w, r, actor, "", fmt.Errorf("%w: pick a driver", app.ErrBadInput))
		return
	}
	date := r.PostFormValue("route_date")
	locationIDs := make([]uuid.UUID, 0, len(r.PostForm["location_ids"]))
	for _, raw := range r.PostForm["location_ids"] {
		id, err := uuid.Parse(raw)
		if err != nil {
			s.builderError(w, r, actor, date, fmt.Errorf("%w: invalid location", app.ErrBadInput))
			return
		}
		locationIDs = append(locationIDs, id)
	}
	detail, err := s.svc.CreateRoute(r.Context(), actor, app.CreateRouteInput{
		DriverUserID: driverID, RouteDate: date, LocationIDs: locationIDs,
		Note: formOpt(r, "note"),
	})
	if err != nil {
		s.builderError(w, r, actor, date, err)
		return
	}
	s.setFlash(w, "Route created.")
	http.Redirect(w, r, "/routes/"+detail.Route.ID.String(), http.StatusSeeOther)
}

// builderError re-renders the builder in the invalid state; the RN05
// conflict additionally links to the existing route (screens.md §2).
func (s *Server) builderError(w http.ResponseWriter, r *http.Request, actor app.Actor, date string, err error) {
	if statusFor(err) >= http.StatusInternalServerError {
		logInternal(w, "route create", err)
		return
	}
	data := builderPageData{Date: date}
	if drivers, derr := s.svc.ListDrivers(r.Context(), actor, false); derr == nil {
		data.Drivers = drivers
	}
	if locations, lerr := s.svc.ListLocations(r.Context(), actor, nil); lerr == nil {
		data.Locations = locations
	}
	if errors.Is(err, app.ErrDriverDateConflict) {
		if driverID, perr := uuid.Parse(r.PostFormValue("driver_user_id")); perr == nil {
			rows, lerr := s.svc.ListRoutes(r.Context(), actor, app.ListRoutesInput{
				From: &date, To: &date, DriverUserID: &driverID,
			})
			if lerr == nil && len(rows) > 0 {
				id := rows[0].ID
				data.ConflictRouteID = &id
			}
		}
	}
	s.render(w, r, http.StatusOK, "route_new", data, routeLabels, formErrors(err))
}

func logInternal(w http.ResponseWriter, where string, err error) {
	log.Printf("httpapi: %s: %v", where, err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}

// routeToday: the driver's own route for today (screens.md §3 "not
// started" entry point) or the empty state.
func (s *Server) routeToday(w http.ResponseWriter, r *http.Request) {
	actor, _ := s.actor(r)
	if actor.Role != "driver" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	today := s.svc.Now().Format("2006-01-02")
	rows, err := s.svc.ListRoutes(r.Context(), actor, app.ListRoutesInput{
		From: &today, To: &today,
	})
	if err != nil {
		logInternal(w, "route today", err)
		return
	}
	if len(rows) == 0 {
		s.render(w, r, http.StatusOK, "route_none", nil, routeLabels, nil)
		return
	}
	http.Redirect(w, r, "/routes/"+rows[0].ID.String(), http.StatusSeeOther)
}

type routeBodyData struct {
	View        app.RouteView
	Draft       bool
	Active      bool
	Closed      bool
	AllRecorded bool
	OwnRoute    bool // the actor drives this route
	CanCompose  bool // admin/manager
	IsAdmin     bool
	Locations   []store.Location // the draft add-stop picker
}

func (s *Server) routeData(r *http.Request, actor app.Actor, view app.RouteView) routeBodyData {
	d := routeBodyData{
		View: view,
		Draft: view.Status == "draft",
		Active: view.Status == "active",
		Closed: view.Status == "closed",
		OwnRoute: actor.UserID == view.DriverUserID,
		CanCompose: actor.Role == "admin" || actor.Role == "manager",
		IsAdmin: actor.Role == "admin",
	}
	d.AllRecorded = true
	for _, st := range view.Stops {
		if st.Counted && (st.ArrivalAt == nil || st.DepartureAt == nil) {
			d.AllRecorded = false
		}
	}
	if d.Draft && d.CanCompose {
		if locations, err := s.svc.ListLocations(r.Context(), actor, nil); err == nil {
			d.Locations = locations
		}
	}
	return d
}

func (s *Server) routeDetail(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	actor, _ := s.actor(r)
	view, err := s.svc.GetRoute(r.Context(), actor, id)
	if err != nil {
		http.Error(w, "not found or forbidden", statusFor(err))
		return
	}
	data := s.routeData(r, actor, view)
	if r.URL.Query().Get("partial") == "1" {
		s.renderFragment(w, http.StatusOK, "route_body", pageData{
			Labels: routeLabels, Actor: actor, Authed: true, Data: data,
		})
		return
	}
	s.render(w, r, http.StatusOK, "route_detail", data, routeLabels, nil)
}

// fragment responds to a tracker/builder mutation: re-render the body.
func (s *Server) fragment(w http.ResponseWriter, r *http.Request, routeID uuid.UUID, errs map[string]string) {
	actor, _ := s.actor(r)
	view, err := s.svc.GetRoute(r.Context(), actor, routeID)
	if err != nil {
		logInternal(w, "route fragment", err)
		return
	}
	s.renderFragment(w, http.StatusOK, "route_body", pageData{
		Labels: routeLabels, Actor: actor, Authed: true, Errors: errs,
		Data: s.routeData(r, actor, view),
	})
}

// fragmentAfter runs a mutation, then swaps the body (errors render
// inline — screens.md "error" state).
func (s *Server) fragmentAfter(w http.ResponseWriter, r *http.Request, routeID uuid.UUID, run func() error) {
	if err := run(); err != nil {
		if statusFor(err) >= http.StatusInternalServerError {
			logInternal(w, "route action", err)
			return
		}
		s.fragment(w, r, routeID, formErrors(err))
		return
	}
	s.fragment(w, r, routeID, nil)
}

func routeIDFrom(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(r.PathValue("id"))
}

func orderFrom(r *http.Request) (int, error) {
	var order int
	if _, err := fmt.Sscanf(r.PathValue("order"), "%d", &order); err != nil {
		return 0, err
	}
	return order, nil
}

func (s *Server) routeStart(w http.ResponseWriter, r *http.Request) {
	id, err := routeIDFrom(r)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	actor, _ := s.actor(r)
	s.fragmentAfter(w, r, id, func() error {
		_, err := s.svc.StartRoute(r.Context(), actor, id)
		return err
	})
}

func (s *Server) routeClose(w http.ResponseWriter, r *http.Request) {
	id, err := routeIDFrom(r)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	_ = r.ParseForm()
	actor, _ := s.actor(r)
	s.fragmentAfter(w, r, id, func() error {
		_, err := s.svc.CloseRoute(r.Context(), actor, app.CloseRouteInput{
			RouteID: id, DistanceKm: formOpt(r, "distance_km"),
		})
		return err
	})
}

func (s *Server) routeReopen(w http.ResponseWriter, r *http.Request) {
	id, err := routeIDFrom(r)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	actor, _ := s.actor(r)
	s.fragmentAfter(w, r, id, func() error {
		_, err := s.svc.ReopenRoute(r.Context(), actor, id)
		return err
	})
}

func (s *Server) routeDistance(w http.ResponseWriter, r *http.Request) {
	id, err := routeIDFrom(r)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	_ = r.ParseForm()
	actor, _ := s.actor(r)
	s.fragmentAfter(w, r, id, func() error {
		_, err := s.svc.SetRouteDistance(r.Context(), actor, id, r.PostFormValue("distance_km"))
		return err
	})
}

func (s *Server) routeAddStop(w http.ResponseWriter, r *http.Request) {
	id, err := routeIDFrom(r)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	_ = r.ParseForm()
	actor, _ := s.actor(r)
	locationID, err := uuid.Parse(r.PostFormValue("location_id"))
	if err != nil {
		s.fragment(w, r, id, map[string]string{"location_id": "pick a location"})
		return
	}
	in := app.AddStopInput{RouteID: id, LocationID: locationID}
	if pos := formOpt(r, "position"); pos != nil {
		var p int
		if _, err := fmt.Sscanf(*pos, "%d", &p); err == nil {
			in.Position = &p
		}
	}
	s.fragmentAfter(w, r, id, func() error {
		_, err := s.svc.AddStop(r.Context(), actor, in)
		return err
	})
}

func (s *Server) routeRemoveStop(w http.ResponseWriter, r *http.Request) {
	id, err := routeIDFrom(r)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	order, err := orderFrom(r)
	if err != nil {
		http.Error(w, "bad order", http.StatusBadRequest)
		return
	}
	actor, _ := s.actor(r)
	s.fragmentAfter(w, r, id, func() error {
		_, err := s.svc.RemoveStop(r.Context(), actor, app.RemoveStopInput{RouteID: id, StopOrder: order})
		return err
	})
}

func (s *Server) routeMoveStop(w http.ResponseWriter, r *http.Request) {
	id, err := routeIDFrom(r)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	order, err := orderFrom(r)
	if err != nil {
		http.Error(w, "bad order", http.StatusBadRequest)
		return
	}
	_ = r.ParseForm()
	actor, _ := s.actor(r)
	s.fragmentAfter(w, r, id, func() error {
		_, err := s.svc.ReorderStops(r.Context(), actor, app.ReorderStopsInput{
			RouteID: id, StopOrder: order, Direction: r.PostFormValue("direction"),
		})
		return err
	})
}

func (s *Server) routeArrive(w http.ResponseWriter, r *http.Request) {
	s.recordTime(w, r, true)
}

func (s *Server) routeDepart(w http.ResponseWriter, r *http.Request) {
	s.recordTime(w, r, false)
}

// recordTime handles both tracker actions: "now" buttons post without
// `at`; manual entry posts a datetime-local wall clock in the display
// zone (screens.md: caught-up data entry for null fields only).
func (s *Server) recordTime(w http.ResponseWriter, r *http.Request, arrival bool) {
	id, err := routeIDFrom(r)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	order, err := orderFrom(r)
	if err != nil {
		http.Error(w, "bad order", http.StatusBadRequest)
		return
	}
	_ = r.ParseForm()
	at, err := parseManualTime(r)
	if err != nil {
		s.fragment(w, r, id, formErrors(err))
		return
	}
	actor, _ := s.actor(r)
	s.fragmentAfter(w, r, id, func() error {
		in := app.RecordTimeInput{RouteID: id, StopOrder: order, At: at}
		if arrival {
			_, err := s.svc.RecordArrival(r.Context(), actor, in)
			return err
		}
		_, err := s.svc.RecordDeparture(r.Context(), actor, in)
		return err
	})
}

// parseManualTime reads the optional manual timestamp (display-zone
// wall clock).
func parseManualTime(r *http.Request) (*time.Time, error) {
	v := r.PostFormValue("at")
	if v == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02T15:04", v, saoPaulo)
	if err != nil {
		return nil, fmt.Errorf("%w: time must be date + HH:MM", app.ErrBadInput)
	}
	return &t, nil
}
