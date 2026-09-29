package httpapi

import (
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"

	"stoptime/internal/app"
	"stoptime/internal/store"
)

// Directories: locations / points registry (screens.md §6, RF03).

var locationsLabels = labelsFor(map[string]string{
	"Title":       "Locations",
	"CreateTitle": "New location",
	"ListTitle":   "Registered locations",
	"Label":       "Label",
	"Address":     "Address",
	"Latitude":    "Latitude",
	"Longitude":   "Longitude",
	"Save":        "Save",
	"Edit":        "Edit",
	"Cancel":      "Cancel",
	"Empty":       "No locations yet.",
})

type locationsPageData struct {
	Locations []store.Location
}

func (s *Server) locationsPage(w http.ResponseWriter, r *http.Request) {
	actor, _ := s.actor(r)
	locations, err := s.svc.ListLocations(r.Context(), actor, nil)
	if err != nil {
		http.Error(w, "forbidden", statusFor(err))
		return
	}
	s.render(w, r, http.StatusOK, "locations", locationsPageData{Locations: locations}, locationsLabels, nil)
}

func (s *Server) locationCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	actor, _ := s.actor(r)
	_, err := s.svc.CreateLocation(r.Context(), actor, app.CreateLocationInput{
		Label:     r.PostFormValue("label"),
		Address:   r.PostFormValue("address"),
		Latitude:  formOpt(r, "latitude"),
		Longitude: formOpt(r, "longitude"),
	})
	if err != nil {
		s.locationsFormError(w, r, err)
		return
	}
	s.setFlash(w, "Location created.")
	http.Redirect(w, r, "/locations", http.StatusSeeOther)
}

func (s *Server) locationsFormError(w http.ResponseWriter, r *http.Request, err error) {
	if statusFor(err) >= http.StatusInternalServerError {
		log.Printf("httpapi: location form: %v", err)
		err = errInternal
	}
	actor, _ := s.actor(r)
	locations, listErr := s.svc.ListLocations(r.Context(), actor, nil)
	if listErr != nil {
		http.Error(w, "forbidden", statusFor(listErr))
		return
	}
	s.render(w, r, http.StatusOK, "locations", locationsPageData{Locations: locations}, locationsLabels, formErrors(err))
}

type locationEditPageData struct {
	Location store.Location
}

// findLocation serves the edit form (the list is the source — pick by
// id; screens.md defines no GetLocation operation).
func (s *Server) findLocation(r *http.Request, id uuid.UUID) (store.Location, error) {
	actor, _ := s.actor(r)
	locations, err := s.svc.ListLocations(r.Context(), actor, nil)
	if err != nil {
		return store.Location{}, err
	}
	for _, l := range locations {
		if l.ID == id {
			return l, nil
		}
	}
	return store.Location{}, app.ErrNotFound
}

func (s *Server) locationEditPage(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	l, err := s.findLocation(r, id)
	if err != nil {
		http.Error(w, "not found", statusFor(err))
		return
	}
	s.render(w, r, http.StatusOK, "location_edit", locationEditPageData{Location: l}, locationsLabels, nil)
}

func (s *Server) locationEditSubmit(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	actor, _ := s.actor(r)
	_, err = s.svc.UpdateLocation(r.Context(), actor, app.UpdateLocationInput{
		LocationID: id,
		Label:      formOpt(r, "label"),
		Address:    formOpt(r, "address"),
		Latitude:   formOpt(r, "latitude"),
		Longitude:  formOpt(r, "longitude"),
	})
	if err != nil {
		if statusFor(err) >= http.StatusInternalServerError {
			log.Printf("httpapi: location edit: %v", err)
			err = errInternal
		}
		l, lerr := s.findLocation(r, id)
		if lerr != nil {
			http.Error(w, "not found", statusFor(lerr))
			return
		}
		s.render(w, r, http.StatusOK, "location_edit", locationEditPageData{Location: l}, locationsLabels, formErrors(err))
		return
	}
	s.setFlash(w, "Location updated.")
	http.Redirect(w, r, "/locations", http.StatusSeeOther)
}

// --- JSON mirrors -------------------------------------------------------

type apiCreateLocationBody struct {
	Label     string  `json:"label"`
	Address   string  `json:"address"`
	Latitude  *string `json:"latitude"`
	Longitude *string `json:"longitude"`
}

func (s *Server) apiLocationsList(w http.ResponseWriter, r *http.Request) {
	actor, _ := s.actor(r)
	locations, err := s.svc.ListLocations(r.Context(), actor, nil)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"locations": locations})
}

func (s *Server) apiLocationsCreate(w http.ResponseWriter, r *http.Request) {
	var body apiCreateLocationBody
	if err := decodeJSON(r, &body); err != nil {
		writeJSONError(w, err)
		return
	}
	actor, _ := s.actor(r)
	l, err := s.svc.CreateLocation(r.Context(), actor, app.CreateLocationInput{
		Label: body.Label, Address: body.Address,
		Latitude: body.Latitude, Longitude: body.Longitude,
	})
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"location": l})
}

type apiUpdateLocationBody struct {
	Label     *string `json:"label"`
	Address   *string `json:"address"`
	Latitude  *string `json:"latitude"`
	Longitude *string `json:"longitude"`
}

func (s *Server) apiLocationUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSONError(w, fmt.Errorf("%w: invalid id", app.ErrBadInput))
		return
	}
	var body apiUpdateLocationBody
	if err := decodeJSON(r, &body); err != nil {
		writeJSONError(w, err)
		return
	}
	actor, _ := s.actor(r)
	l, err := s.svc.UpdateLocation(r.Context(), actor, app.UpdateLocationInput{
		LocationID: id, Label: body.Label, Address: body.Address,
		Latitude: body.Latitude, Longitude: body.Longitude,
	})
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"location": l})
}
