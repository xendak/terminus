package httpapi

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"net/http"

	"stoptime/internal/app"
	"stoptime/web"
)

func mustStaticFS() fs.FS {
	sub, err := fs.Sub(web.FS, "static")
	if err != nil {
		panic("httpapi: static assets missing from embed: " + err.Error())
	}
	return sub
}

type ctxKeyActor struct{}

// withSession authenticates the session cookie once per request through
// the service (Services.Authenticate: signature, expiry, and the user
// still existing and active) and stores the Actor (and the session) in
// the request context. Absent cookie = anonymous; a cookie the service
// rejects is cleared and the request continues anonymous, so pages
// redirect to /login and JSON answers 401.
func (s *Server) withSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(app.SessionCookieName); err == nil {
			sess, err := s.svc.Authenticate(r.Context(), c.Value)
			switch {
			case err == nil:
				ctx := context.WithValue(r.Context(), ctxKeyActor{}, app.ActorFromSession(sess))
				ctx = app.WithSession(ctx, sess)
				r = r.WithContext(ctx)
			case errors.Is(err, app.ErrUnauthenticated):
				clearSessionCookie(w)
			default:
				log.Printf("httpapi: authenticate: %v", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// clearSessionCookie expires the session cookie in the browser.
func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: app.SessionCookieName, Value: "", Path: "/", MaxAge: -1})
}

// actor returns the verified acting user, if any. Handlers NEVER take
// identity from request input — only from the decoded session.
func (s *Server) actor(r *http.Request) (app.Actor, bool) {
	a, ok := r.Context().Value(ctxKeyActor{}).(app.Actor)
	return a, ok
}

// requirePage bounces anonymous users to the login form.
func (s *Server) requirePage(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.actor(r); !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

// requireAPI answers 401 for anonymous JSON callers.
func (s *Server) requireAPI(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.actor(r); !ok {
			writeJSONError(w, app.ErrUnauthenticated)
			return
		}
		next(w, r)
	}
}

// recoverer logs panics server-side and answers a generic 500 — no
// stack traces or error text to the client (architecture.md Security).
func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic %s %s: %v", r.Method, r.URL.Path, rec)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
