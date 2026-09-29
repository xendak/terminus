// Package httpapi is the HTTP shell: adapters for the operation
// services, session middleware, the template engine, and the sentinel
// error mapping (docs/spec/operations.md transports). Handlers parse,
// call a service, and format — no SQL, no business rules
// (docs/spec/architecture.md layering rules).
package httpapi

import (
	"html/template"
	"net/http"

	"stoptime/internal/app"
)

// Server wires the services to the transports.
type Server struct {
	svc *app.Services
	key []byte // session/flash signing key (same key the services sign with)
	tpl map[string]*template.Template
}

// New builds the server: parses the embedded templates once.
func New(svc *app.Services, sessionKey []byte) (*Server, error) {
	tpl, err := parseTemplates()
	if err != nil {
		return nil, err
	}
	return &Server{svc: svc, key: sessionKey, tpl: tpl}, nil
}

// Router assembles every transport with middleware. Middleware loads
// the session; the SERVICES enforce the role matrix (architecture.md:
// middleware is a convenience, never the authority).
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// Static assets (vendored; embedded in the binary).
	mux.Handle("GET /static/", http.StripPrefix("/static/",
		http.FileServerFS(mustStaticFS())))

	// Health probe (kept from T1).
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	// Pages (session required; anonymous → login form).
	mux.HandleFunc("GET /login", s.loginPage)
	mux.HandleFunc("POST /login", s.loginSubmit)
	mux.HandleFunc("POST /logout", s.requirePage(s.logout))
	mux.HandleFunc("GET /{$}", s.requirePage(s.home))
	mux.HandleFunc("GET /drivers", s.requirePage(s.driversPage))
	mux.HandleFunc("POST /drivers", s.requirePage(s.driverCreate))
	mux.HandleFunc("GET /drivers/{id}/edit", s.requirePage(s.driverEditPage))
	mux.HandleFunc("POST /drivers/{id}/edit", s.requirePage(s.driverEditSubmit))
	mux.HandleFunc("GET /managers", s.requirePage(s.managersPage))
	mux.HandleFunc("POST /managers", s.requirePage(s.managerCreate))
	mux.HandleFunc("GET /locations", s.requirePage(s.locationsPage))
	mux.HandleFunc("POST /locations", s.requirePage(s.locationCreate))
	mux.HandleFunc("GET /locations/{id}/edit", s.requirePage(s.locationEditPage))
	mux.HandleFunc("POST /locations/{id}/edit", s.requirePage(s.locationEditSubmit))

	// Route builder + tracker pages and htmx fragments.
	mux.HandleFunc("GET /routes/new", s.requirePage(s.routeNewPage))
	mux.HandleFunc("POST /routes", s.requirePage(s.routeCreate))
	mux.HandleFunc("GET /routes/today", s.requirePage(s.routeToday))
	mux.HandleFunc("GET /routes/{id}", s.requirePage(s.routeDetail))
	mux.HandleFunc("POST /routes/{id}/start", s.requirePage(s.routeStart))
	mux.HandleFunc("POST /routes/{id}/close", s.requirePage(s.routeClose))
	mux.HandleFunc("POST /routes/{id}/reopen", s.requirePage(s.routeReopen))
	mux.HandleFunc("POST /routes/{id}/distance", s.requirePage(s.routeDistance))
	mux.HandleFunc("POST /routes/{id}/stops", s.requirePage(s.routeAddStop))
	mux.HandleFunc("POST /routes/{id}/stops/{order}/remove", s.requirePage(s.routeRemoveStop))
	mux.HandleFunc("POST /routes/{id}/stops/{order}/move", s.requirePage(s.routeMoveStop))
	mux.HandleFunc("POST /routes/{id}/stops/{order}/arrive", s.requirePage(s.routeArrive))
	mux.HandleFunc("POST /routes/{id}/stops/{order}/depart", s.requirePage(s.routeDepart))
	mux.HandleFunc("POST /routes/{id}/stops/{order}/times", s.requirePage(s.routeCorrectTimes))

	// Dashboard, history, params, audit, export.
	mux.HandleFunc("GET /dashboard", s.requirePage(s.dashboardPage))
	mux.HandleFunc("GET /history", s.requirePage(s.historyPage))
	mux.HandleFunc("GET /history/export", s.requirePage(s.exportCSV))
	mux.HandleFunc("GET /params", s.requirePage(s.paramsPage))
	mux.HandleFunc("POST /params/{key}", s.requirePage(s.paramUpdate))
	mux.HandleFunc("GET /audit", s.requirePage(s.auditPage))

	// JSON mirrors (anonymous → 401).
	mux.HandleFunc("POST /api/auth/login", s.apiLogin)
	mux.HandleFunc("POST /api/auth/logout", s.apiLogout)
	mux.HandleFunc("GET /api/drivers", s.requireAPI(s.apiDriversList))
	mux.HandleFunc("POST /api/drivers", s.requireAPI(s.apiDriversCreate))
	mux.HandleFunc("PATCH /api/drivers/{id}", s.requireAPI(s.apiDriverUpdate))
	mux.HandleFunc("GET /api/managers", s.requireAPI(s.apiManagersList))
	mux.HandleFunc("POST /api/managers", s.requireAPI(s.apiManagersCreate))
	mux.HandleFunc("GET /api/locations", s.requireAPI(s.apiLocationsList))
	mux.HandleFunc("POST /api/locations", s.requireAPI(s.apiLocationsCreate))
	mux.HandleFunc("PATCH /api/locations/{id}", s.requireAPI(s.apiLocationUpdate))
	mux.HandleFunc("POST /api/routes", s.requireAPI(s.apiCreateRoute))
	mux.HandleFunc("GET /api/routes/{id}", s.requireAPI(s.apiGetRoute))
	mux.HandleFunc("POST /api/routes/{id}/stops", s.requireAPI(s.apiAddStop))
	mux.HandleFunc("DELETE /api/routes/{id}/stops/{order}", s.requireAPI(s.apiRemoveStop))
	mux.HandleFunc("POST /api/routes/{id}/stops/{order}/move", s.requireAPI(s.apiMoveStop))
	mux.HandleFunc("POST /api/routes/{id}/start", s.requireAPI(s.apiStartRoute))
	mux.HandleFunc("POST /api/routes/{id}/close", s.requireAPI(s.apiCloseRoute))
	mux.HandleFunc("POST /api/routes/{id}/reopen", s.requireAPI(s.apiReopenRoute))
	mux.HandleFunc("PUT /api/routes/{id}/distance", s.requireAPI(s.apiSetDistance))
	mux.HandleFunc("POST /api/routes/{id}/stops/{order}/arrive", s.requireAPI(s.apiArrive))
	mux.HandleFunc("POST /api/routes/{id}/stops/{order}/depart", s.requireAPI(s.apiDepart))
	mux.HandleFunc("GET /api/routes", s.requireAPI(s.apiRoutesList))
	mux.HandleFunc("GET /api/dashboard/day", s.requireAPI(s.apiDashboardDay))
	mux.HandleFunc("GET /api/dashboard/month", s.requireAPI(s.apiDashboardMonth))
	mux.HandleFunc("GET /api/dashboard/period", s.requireAPI(s.apiDashboardPeriod))
	mux.HandleFunc("GET /api/params", s.requireAPI(s.apiParamsList))
	mux.HandleFunc("PUT /api/params/{key}", s.requireAPI(s.apiParamUpdate))
	mux.HandleFunc("GET /api/audit", s.requireAPI(s.apiAuditList))
	mux.HandleFunc("GET /api/export", s.requireAPI(s.exportCSV))

	var h http.Handler = mux
	h = s.withSession(h)
	h = recoverer(h)
	return h
}
