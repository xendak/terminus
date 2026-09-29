package httpapi

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"stoptime/internal/app"
	"stoptime/internal/store"
)

// History (screens.md §5): filtered route list with a detail link, and
// the audited corrections form on the route detail (manager/admin).

var historyLabels = labelsFor(map[string]string{
	"Title":      "History",
	"From":       "From",
	"To":         "To",
	"Driver":     "Driver",
	"Status":     "Status",
	"Apply":      "Apply",
	"Date":       "Date",
	"Stops":      "Stops",
	"TotalMinutes": "Total stopped (min)",
	"Percent":    "Journey",
	"Cost":       "Cost",
	"StatusCol":  "Status",
	"Open":       "Open",
	"Export":     "Export CSV",
	"AllDrivers":  "All drivers",
	"AllStatuses": "All",
	"Empty":      "No routes in this range.",
})

type historyPageData struct {
	From, To     string
	DriverFilter string
	StatusFilter string
	Drivers      []store.Driver // for the manager/admin filter
	Routes       []store.RouteListRow
}

func (s *Server) historyPage(w http.ResponseWriter, r *http.Request) {
	actor, _ := s.actor(r)
	q := r.URL.Query()

	in := app.ListRoutesInput{}
	if q.Get("from") != "" {
		in.From = ptrStr(q.Get("from"))
	}
	if q.Get("to") != "" {
		in.To = ptrStr(q.Get("to"))
	}
	if q.Get("status") != "" {
		in.Status = ptrStr(q.Get("status"))
	}
	if q.Get("driver_user_id") != "" {
		id, err := uuid.Parse(q.Get("driver_user_id"))
		if err != nil {
			http.Error(w, "bad driver", http.StatusBadRequest)
			return
		}
		in.DriverUserID = &id
	}

	data := historyPageData{
		DriverFilter: q.Get("driver_user_id"),
		StatusFilter: q.Get("status"),
		From:         s.svc.Now().Format("2006-01") + "-01",
		To:           s.svc.Now().Format("2006-01-02"),
	}
	// The driver filter needs the directory (manager/admin only).
	if actor.Role == "admin" || actor.Role == "manager" {
		if drivers, err := s.svc.ListDrivers(r.Context(), actor, false); err == nil {
			data.Drivers = drivers
		}
	}
	routes, err := s.svc.ListRoutes(r.Context(), actor, in)
	if err != nil {
		http.Error(w, "forbidden", statusFor(err))
		return
	}
	data.Routes = routes
	if in.From != nil {
		data.From = *in.From
	}
	if in.To != nil {
		data.To = *in.To
	}
	s.render(w, r, http.StatusOK, "history", data, historyLabels, nil)
}

func ptrStr(v string) *string { return &v }
func ptrOrNil(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

// apiRoutesList: the ListRoutes JSON mirror. Drivers are scoped by the
// service; the driver filter is a query-level input.
func (s *Server) apiRoutesList(w http.ResponseWriter, r *http.Request) {
	actor, _ := s.actor(r)
	q := r.URL.Query()
	in := app.ListRoutesInput{}
	if q.Get("from") != "" {
		in.From = ptrStr(q.Get("from"))
	}
	if q.Get("to") != "" {
		in.To = ptrStr(q.Get("to"))
	}
	if q.Get("status") != "" {
		in.Status = ptrStr(q.Get("status"))
	}
	if q.Get("driver_user_id") != "" {
		id, err := uuid.Parse(q.Get("driver_user_id"))
		if err != nil {
			writeJSONError(w, errBadDriver)
			return
		}
		in.DriverUserID = &id
	}
	routes, err := s.svc.ListRoutes(r.Context(), actor, in)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"routes": routes})
}

var errBadDriver = errBadInput("invalid driver id")

func errBadInput(msg string) error {
	return &app.FieldError{Field: "driver_user_id", Reason: msg}
}

// routeCorrectTimes: the corrections form (UpdateStopTimes,
// manager/admin — audited). Empty fields keep the current timestamps.
func (s *Server) routeCorrectTimes(w http.ResponseWriter, r *http.Request) {
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
	arrival, err := parseOptionalDTLocal(r, "arrival_at")
	if err != nil {
		s.fragment(w, r, id, formErrors(err))
		return
	}
	departure, err := parseOptionalDTLocal(r, "departure_at")
	if err != nil {
		s.fragment(w, r, id, formErrors(err))
		return
	}
	actor, _ := s.actor(r)
	s.fragmentAfter(w, r, id, func() error {
		_, err := s.svc.UpdateStopTimes(r.Context(), actor, app.UpdateStopTimesInput{
			RouteID: id, StopOrder: order, ArrivalAt: arrival, DepartureAt: departure,
		})
		return err
	})
}

// parseOptionalDTLocal reads an optional datetime-local field; empty
// keeps the current value (nil), malformed is ErrBadInput.
func parseOptionalDTLocal(r *http.Request, name string) (*time.Time, error) {
	v := r.PostFormValue(name)
	if v == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02T15:04", v, saoPaulo)
	if err != nil {
		return nil, &app.FieldError{Field: name, Reason: "must be date + HH:MM"}
	}
	return &t, nil
}
