package httpapi

import (
	"log"
	"net/http"

	"stoptime/internal/app"
	"stoptime/internal/store"
)

// Directories: managers (screens.md §6). Admin-only per the matrix —
// the service rejects everyone else with 403.

var managersLabels = labelsFor(map[string]string{
	"Title":       "Managers",
	"CreateTitle": "New manager",
	"ListTitle":   "Registered managers",
	"Name":        "Name",
	"Email":       "Email",
	"Password":    "Password",
	"Phone":       "Phone",
	"Active":      "Active",
	"Save":        "Save",
	"Empty":       "No managers yet.",
})

type managersPageData struct {
	Managers []store.User
}

func (s *Server) managersPage(w http.ResponseWriter, r *http.Request) {
	actor, _ := s.actor(r)
	managers, err := s.svc.ListManagers(r.Context(), actor)
	if err != nil {
		http.Error(w, "forbidden", statusFor(err))
		return
	}
	s.render(w, r, http.StatusOK, "managers", managersPageData{Managers: managers}, managersLabels, nil)
}

func (s *Server) managerCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	actor, _ := s.actor(r)
	_, err := s.svc.CreateManager(r.Context(), actor, app.CreateManagerInput{
		Name:     r.PostFormValue("name"),
		Email:    r.PostFormValue("email"),
		Password: r.PostFormValue("password"),
		Phone:    r.PostFormValue("phone"),
	})
	if err != nil {
		if statusFor(err) >= http.StatusInternalServerError {
			log.Printf("httpapi: manager create: %v", err)
			err = errInternal
		}
		s.managersFormError(w, r, err)
		return
	}
	s.setFlash(w, "Manager created.")
	http.Redirect(w, r, "/managers", http.StatusSeeOther)
}

func (s *Server) managersFormError(w http.ResponseWriter, r *http.Request, err error) {
	actor, _ := s.actor(r)
	managers, listErr := s.svc.ListManagers(r.Context(), actor)
	if listErr != nil {
		http.Error(w, "forbidden", statusFor(listErr))
		return
	}
	s.render(w, r, http.StatusOK, "managers", managersPageData{Managers: managers}, managersLabels, formErrors(err))
}

// --- JSON mirrors -------------------------------------------------------

type apiCreateManagerBody struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Phone    string `json:"phone"`
}

func (s *Server) apiManagersList(w http.ResponseWriter, r *http.Request) {
	actor, _ := s.actor(r)
	managers, err := s.svc.ListManagers(r.Context(), actor)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"managers": managers})
}

func (s *Server) apiManagersCreate(w http.ResponseWriter, r *http.Request) {
	var body apiCreateManagerBody
	if err := decodeJSON(r, &body); err != nil {
		writeJSONError(w, err)
		return
	}
	actor, _ := s.actor(r)
	u, err := s.svc.CreateManager(r.Context(), actor, app.CreateManagerInput{
		Name: body.Name, Email: body.Email, Password: body.Password, Phone: body.Phone,
	})
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"manager": u})
}
