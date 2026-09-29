package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"
	_ "time/tzdata" // America/Sao_Paulo rendering without a system zone db

	"stoptime/internal/app"
	"stoptime/internal/store"
	"stoptime/web"
)

// The template engine: one template set per page (layout + page file),
// rendered with labels, the actor, flash, and per-page data. Labels are
// one map per screen (English defaults — screens.md), so a translation
// layer can be added without touching template logic.

// pages: name -> extra template files parsed WITH the layout.
var pages = map[string][]string{
	"login":        {"templates/login.html"},
	"home":         {"templates/home.html"},
	"drivers":      {"templates/drivers.html"},
	"driver_edit":  {"templates/driver_edit.html"},
	"managers":     {"templates/managers.html"},
	"locations":    {"templates/locations.html"},
	"location_edit": {"templates/location_edit.html"},
	"route_new":    {"templates/route_new.html"},
	"route_detail": {"templates/route_detail.html", "templates/route_body.html"},
	"route_none":   {"templates/route_none.html"},
	"dashboard":    {"templates/dashboard.html"},
	"history":      {"templates/history.html"},
	"params":       {"templates/params.html"},
	"audit":        {"templates/audit.html"},
}

// fragments: rendered WITHOUT the layout (htmx partial swaps).
var fragments = map[string][]string{
	"route_body": {"templates/route_body.html"},
}

// saoPaulo: the display zone (architecture.md timezone policy: UTC
// storage, America/Sao_Paulo rendering).
var saoPaulo = mustLoadLocation()

func mustLoadLocation() *time.Location {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		panic("httpapi: tzdata: " + err.Error())
	}
	return loc
}

// Display-only formatting helpers (template FuncMap + handlers).
// Business math stays in SQL/domain; these format what the services
// already computed (RN02/RN03 call per-stop minutes "display only").
func fmtTime(t time.Time) string { return t.In(saoPaulo).Format("02/01 15:04") }
// fmtDate shows a calendar day as-is: a route_date has no zone to
// convert (converting UTC midnight to -03:00 would show the day before).
func fmtDate(d store.Date) string { return d.Format("02/01/2006") }

var funcs = template.FuncMap{
	"fmtTime": fmtTime,
	"fmtDate": fmtDate,
	"dtLocal": func(t *time.Time) string { // datetime-local input value
		if t == nil {
			return ""
		}
		return t.In(saoPaulo).Format("2006-01-02T15:04")
	},
	"mins": func(secs *int) int {
		if secs == nil {
			return 0
		}
		return *secs / 60
	},
}

func parseTemplates() (map[string]*template.Template, error) {
	out := make(map[string]*template.Template, len(pages)+len(fragments))
	for name, files := range pages {
		t, err := template.New("layout.html").Funcs(funcs).
			ParseFS(web.FS, append([]string{"templates/layout.html"}, files...)...)
		if err != nil {
			return nil, err
		}
		out[name] = t
	}
	for name, files := range fragments {
		t, err := template.New(name).Funcs(funcs).ParseFS(web.FS, files...)
		if err != nil {
			return nil, err
		}
		out[name] = t
	}
	return out, nil
}

// navLabels are shared by every screen's map (the layout renders them).
var navLabels = map[string]string{
	"NavDrivers":    "Drivers",
	"NavLocations":  "Locations",
	"NavManagers":   "Managers",
	"NavLogout":     "Log out",
}

func labelsFor(screen map[string]string) map[string]string {
	m := make(map[string]string, len(navLabels)+len(screen))
	for k, v := range navLabels {
		m[k] = v
	}
	for k, v := range screen {
		m[k] = v
	}
	return m
}

type pageData struct {
	Labels map[string]string
	Actor  app.Actor
	Authed bool
	Flash  string
	Errors map[string]string
	Data   any
}

// render writes a page with session, flash, and labels in place.
func (s *Server) render(w http.ResponseWriter, r *http.Request, status int, page string, data any, labels map[string]string, errs map[string]string) {
	actor, ok := s.actor(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	err := s.tpl[page].Execute(w, pageData{
		Labels: labels, Actor: actor, Authed: ok,
		Flash: s.readFlash(w, r), Errors: errs, Data: data,
	})
	if err != nil {
		log.Printf("httpapi: render %s: %v", page, err)
	}
}

// renderFragment writes an htmx partial (no layout) with the same
// pageData the full page gets.
func (s *Server) renderFragment(w http.ResponseWriter, status int, fragment string, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := s.tpl[fragment].ExecuteTemplate(w, fragment, data); err != nil {
		log.Printf("httpapi: render fragment %s: %v", fragment, err)
	}
}

// Flash messages: one signed cookie (same key as the session), set on
// a redirect, read and cleared by the next page render.
const flashCookie = "st_flash"

func (s *Server) setFlash(w http.ResponseWriter, msg string) {
	http.SetCookie(w, &http.Cookie{
		Name: flashCookie, Value: signFlash(s.key, msg), Path: "/",
		HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
}

// readFlash returns and clears the flash message, if present and valid.
func (s *Server) readFlash(w http.ResponseWriter, r *http.Request) string {
	c, err := r.Cookie(flashCookie)
	if err != nil {
		return ""
	}
	http.SetCookie(w, &http.Cookie{Name: flashCookie, Value: "", Path: "/", MaxAge: -1})
	body, sig, ok := strings.Cut(c.Value, ".")
	if !ok {
		return ""
	}
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(body))
	got, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil || !hmac.Equal(mac.Sum(nil), got) {
		return ""
	}
	msg, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return ""
	}
	return string(msg)
}

func signFlash(key []byte, msg string) string {
	body := base64.RawURLEncoding.EncodeToString([]byte(msg))
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(body))
	return body + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// form helpers: form values into service inputs; empty optional fields
// stay nil (absent, not blank).
func formOpt(r *http.Request, name string) *string {
	v := r.PostFormValue(name)
	if v == "" {
		return nil
	}
	return &v
}

func formBool(r *http.Request, name string) *bool {
	v := r.PostFormValue(name) == "on"
	return &v
}
