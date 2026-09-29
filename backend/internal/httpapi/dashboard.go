package httpapi

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"stoptime/internal/app"
)

// Dashboard (screens.md §4): three cuts over one window — day series,
// month series, period summary + per-driver ranking. Chart.js renders
// the aggregate series (it never computes); the JSON mirrors carry the
// same series the page embeds.

var dashboardLabels = labelsFor(map[string]string{
	"Title":       "Dashboard",
	"From":        "From",
	"To":          "To",
	"Apply":       "Apply",
	"DayTab":      "Per day",
	"MonthTab":    "Per month",
	"PeriodTab":   "Period",
	"TotalMinutes": "Total stopped (min)",
	"Minutes":     "min",
	"Percent":     "Journey",
	"Routes":      "Routes",
	"ByDriver":    "By driver",
	"Empty":       "No stops recorded in this period — widen the range.",
	"Cost":        "Cost",
})

type dashboardPageData struct {
	From, To   string
	Presets    []preset
	DayJSON    template.JS
	MonthJSON  template.JS
	PeriodJSON template.JS
	Period     app.PeriodSummary
	Empty      bool
}

type preset struct {
	Label, From, To string
}

func (s *Server) dashboardPresets() []preset {
	now := s.svc.Now()
	ym := now.Format("2006-01")
	day := now.Format("2006-01-02")
	return []preset{
		{"today", day, day},
		{"7 days", now.AddDate(0, 0, -6).Format("2006-01-02"), day},
		{"this month", ym + "-01", day},
		{"12 months", now.AddDate(-1, 0, 0).Format("2006-01-02"), day},
	}
}

func (s *Server) dashboardWindow(r *http.Request) (string, string, error) {
	q := r.URL.Query()
	now := s.svc.Now()
	from := now.Format("2006-01") + "-01"
	to := now.Format("2006-01-02")
	if q.Get("from") != "" {
		f, err := parseDateParam("from", q.Get("from"))
		if err != nil {
			return "", "", err
		}
		from = f
	}
	if q.Get("to") != "" {
		t, err := parseDateParam("to", q.Get("to"))
		if err != nil {
			return "", "", err
		}
		to = t
	}
	return from, to, nil
}

// parseDateParam validates a YYYY-MM-DD query value.
func parseDateParam(name, value string) (string, error) {
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return "", fmt.Errorf("%w: %s must be YYYY-MM-DD", app.ErrBadInput, name)
	}
	return value, nil
}

func (s *Server) dashboardPage(w http.ResponseWriter, r *http.Request) {
	actor, _ := s.actor(r)
	from, to, err := s.dashboardWindow(r)
	if err != nil {
		http.Error(w, "bad window", statusFor(err))
		return
	}
	in := app.DashboardInput{From: from, To: to}
	day, err := s.svc.GetDashboardByDay(r.Context(), actor, in)
	if err != nil {
		http.Error(w, "forbidden", statusFor(err))
		return
	}
	month, _ := s.svc.GetDashboardByMonth(r.Context(), actor, in)
	period, _ := s.svc.GetDashboardByPeriod(r.Context(), actor, in)

	dayRaw, _ := json.Marshal(day.Series)
	monthRaw, _ := json.Marshal(month.Series)
	periodRaw, _ := json.Marshal(period)
	data := dashboardPageData{
		From: from, To: to, Presets: s.dashboardPresets(),
		DayJSON:    template.JS(dayRaw),
		MonthJSON:  template.JS(monthRaw),
		PeriodJSON: template.JS(periodRaw),
		Period:     period,
		Empty:      len(day.Series) == 0 && len(month.Series) == 0,
	}
	s.render(w, r, http.StatusOK, "dashboard", data, dashboardLabels, nil)
}

// --- JSON mirrors -------------------------------------------------------

// The JSON mirrors answer the service outputs as-is: day and month are
// {series, standard_journey_hours}, the period is the summary object
// (operations.md).

func (s *Server) apiDashboardDay(w http.ResponseWriter, r *http.Request) {
	s.apiDashboard(w, r, func(in app.DashboardInput, actor app.Actor) (any, error) {
		return s.svc.GetDashboardByDay(r.Context(), actor, in)
	})
}

func (s *Server) apiDashboardMonth(w http.ResponseWriter, r *http.Request) {
	s.apiDashboard(w, r, func(in app.DashboardInput, actor app.Actor) (any, error) {
		return s.svc.GetDashboardByMonth(r.Context(), actor, in)
	})
}

func (s *Server) apiDashboardPeriod(w http.ResponseWriter, r *http.Request) {
	s.apiDashboard(w, r, func(in app.DashboardInput, actor app.Actor) (any, error) {
		return s.svc.GetDashboardByPeriod(r.Context(), actor, in)
	})
}

// apiDashboard runs a dashboard read over the request's window.
func (s *Server) apiDashboard(w http.ResponseWriter, r *http.Request, call func(app.DashboardInput, app.Actor) (any, error)) {
	from, to, err := s.dashboardWindow(r)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	team, err := optUUIDParam(r, "manager_user_id")
	if err != nil {
		writeJSONError(w, err)
		return
	}
	actor, _ := s.actor(r)
	out, err := call(app.DashboardInput{From: from, To: to, ManagerUserID: team}, actor)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
