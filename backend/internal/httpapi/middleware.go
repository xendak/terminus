package httpapi

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"time"

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

// withSession decodes the session cookie once per request and stores
// the Actor (and the session) in the request context. Absent or
// invalid cookie = anonymous; the services remain the authority.
func (s *Server) withSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(app.SessionCookieName); err == nil {
			if sess, err := app.DecodeSession(s.key, c.Value, time.Now()); err == nil {
				ctx := context.WithValue(r.Context(), ctxKeyActor{}, app.ActorFromSession(sess))
				ctx = app.WithSession(ctx, sess)
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
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
