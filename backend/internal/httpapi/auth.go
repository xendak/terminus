package httpapi

import (
	"errors"
	"log"
	"net/http"
	"time"

	"stoptime/internal/app"
)

// Login (screens.md §1): entry, invalid, success (redirect by role).
// T7 ships the shell: success goes to the home page until T8 (tracker)
// and T9 (dashboard) land their routes — this map is the single flip
// point for the screens.md redirect targets.

var loginLabels = labelsFor(map[string]string{
	"Title":    "Sign in",
	"Email":    "Email",
	"Password": "Password",
	"Submit":   "Sign in",
	"Invalid":  "Invalid email or password.",
})

func loginRedirect(role string) string {
	// screens.md §1: driver → tracker, manager/admin → dashboard —
	// both landed (T8 tracker, T9 dashboard).
	if role == "driver" {
		return "/routes/today"
	}
	return "/dashboard"
}

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, http.StatusOK, "login", nil, loginLabels, nil)
}

func (s *Server) loginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	u, sess, cookie, err := s.svc.Login(r.Context(), app.LoginInput{
		Email:    r.PostFormValue("email"),
		Password: r.PostFormValue("password"),
	})
	if err != nil {
		if errors.Is(err, app.ErrUnauthenticated) {
			s.render(w, r, http.StatusOK, "login", nil, loginLabels,
				map[string]string{"form": loginLabels["Invalid"]})
			return
		}
		log.Printf("httpapi: login: %v", err)
		s.render(w, r, http.StatusOK, "login", nil, loginLabels,
			map[string]string{"form": "login unavailable"})
		return
	}
	setSessionCookie(w, r, cookie, sess.ExpiresAt)
	http.Redirect(w, r, loginRedirect(u.Role), http.StatusSeeOther)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if actor, ok := s.actor(r); ok {
		if err := s.svc.Logout(r.Context(), actor); err != nil {
			writeJSONError(w, err)
			return
		}
	}
	http.SetCookie(w, &http.Cookie{Name: app.SessionCookieName, Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, value string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     app.SessionCookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		Expires:  expiresAt,
	})
}

// home: the role-aware stand-in for the post-login landing pages.
var homeLabels = labelsFor(map[string]string{
	"Title":        "Terminus",
	"Welcome":      "Choose a screen to start.",
})

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, http.StatusOK, "home", nil, homeLabels, nil)
}

// --- JSON mirrors -------------------------------------------------------

type apiLoginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) apiLogin(w http.ResponseWriter, r *http.Request) {
	var body apiLoginBody
	if err := decodeJSON(r, &body); err != nil {
		writeJSONError(w, err)
		return
	}
	u, sess, cookie, err := s.svc.Login(r.Context(), app.LoginInput{
		Email: body.Email, Password: body.Password,
	})
	if err != nil {
		writeJSONError(w, err)
		return
	}
	setSessionCookie(w, r, cookie, sess.ExpiresAt)
	writeJSON(w, http.StatusOK, map[string]any{
		"user":       u,
		"expires_at": sess.ExpiresAt,
	})
}

func (s *Server) apiLogout(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.actor(r)
	if !ok {
		writeJSONError(w, app.ErrUnauthenticated)
		return
	}
	if err := s.svc.Logout(r.Context(), actor); err != nil {
		writeJSONError(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: app.SessionCookieName, Value: "", Path: "/", MaxAge: -1})
	w.WriteHeader(http.StatusNoContent)
}

// apiMe answers who the session belongs to (the SPA's boot check): the
// user re-read from the database plus the session's expiry.
func (s *Server) apiMe(w http.ResponseWriter, r *http.Request) {
	actor, _ := s.actor(r)
	u, err := s.svc.CurrentUser(r.Context(), actor)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	sess, _ := app.SessionFromContext(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"user":       u,
		"expires_at": sess.ExpiresAt,
	})
}
