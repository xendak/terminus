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

	var h http.Handler = mux
	h = s.withSession(h)
	h = recoverer(h)
	return h
}
