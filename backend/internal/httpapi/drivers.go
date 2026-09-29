package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"

	"stoptime/internal/app"
	"stoptime/internal/store"
)

// Directories: drivers (screens.md §6). Handlers parse, call, format.
// The service enforces the matrix; a driver role here yields 403.

var driversLabels = labelsFor(map[string]string{
	"Title":        "Drivers",
	"CreateTitle":  "New driver",
	"ListTitle":    "Registered drivers",
	"Name":         "Name",
	"Email":        "Email",
	"Password":     "Password",
	"Phone":        "Phone",
	"Document":     "Document",
	"VehicleName":  "Vehicle",
	"VehiclePlate": "Plate",
	"KmPerL":       "km/l",
	"Active":       "Active",
	"Save":         "Save",
	"Edit":         "Edit",
	"Cancel":       "Cancel",
	"Empty":        "No drivers yet.",
})

type driversPageData struct {
	Drivers []store.Driver
}

func (s *Server) driversPage(w http.ResponseWriter, r *http.Request) {
	actor, _ := s.actor(r)
	drivers, err := s.svc.ListDrivers(r.Context(), actor, false)
	if err != nil {
		http.Error(w, "forbidden", statusFor(err))
		return
	}
	s.render(w, r, http.StatusOK, "drivers", driversPageData{Drivers: drivers}, driversLabels, nil)
}

func (s *Server) driverCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	actor, _ := s.actor(r)
	in := app.CreateDriverInput{
		Name:         r.PostFormValue("name"),
		Email:        r.PostFormValue("email"),
		Password:     r.PostFormValue("password"),
		Phone:        r.PostFormValue("phone"),
		Document:     formOpt(r, "document"),
		VehicleName:  formOpt(r, "vehicle_name"),
		VehiclePlate: formOpt(r, "vehicle_plate"),
		KmPerL:       formOpt(r, "km_per_l"),
	}
	_, err := s.svc.CreateDriver(r.Context(), actor, in)
	if err != nil {
		s.driverFormError(w, r, err)
		return
	}
	s.setFlash(w, "Driver created.")
	http.Redirect(w, r, "/drivers", http.StatusSeeOther)
}

// driverFormError re-renders the drivers page in the "invalid" state.
func (s *Server) driverFormError(w http.ResponseWriter, r *http.Request, err error) {
	if statusFor(err) >= http.StatusInternalServerError {
		log.Printf("httpapi: driver form: %v", err)
		err = errInternal
	}
	actor, _ := s.actor(r)
	drivers, listErr := s.svc.ListDrivers(r.Context(), actor, false)
	if listErr != nil {
		http.Error(w, "forbidden", statusFor(listErr))
		return
	}
	s.render(w, r, http.StatusOK, "drivers", driversPageData{Drivers: drivers}, driversLabels, formErrors(err))
}

var errInternal = errors.New("internal error")

type driverEditPageData struct {
	Driver store.Driver
}

// findDriver serves the edit form (screens.md has no GetDriver
// operation — the directory list is the source; pick by id).
func (s *Server) findDriver(r *http.Request, id uuid.UUID) (store.Driver, error) {
	actor, _ := s.actor(r)
	drivers, err := s.svc.ListDrivers(r.Context(), actor, false)
	if err != nil {
		return store.Driver{}, err
	}
	for _, d := range drivers {
		if d.ID == id {
			return d, nil
		}
	}
	return store.Driver{}, app.ErrNotFound
}

func (s *Server) driverEditPage(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	d, err := s.findDriver(r, id)
	if err != nil {
		http.Error(w, "not found", statusFor(err))
		return
	}
	s.render(w, r, http.StatusOK, "driver_edit", driverEditPageData{Driver: d}, driversLabels, nil)
}

func (s *Server) driverEditSubmit(w http.ResponseWriter, r *http.Request) {
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
	_, err = s.svc.UpdateDriver(r.Context(), actor, app.UpdateDriverInput{
		DriverID:     id,
		Name:         formOpt(r, "name"),
		Phone:        formOpt(r, "phone"),
		Document:     formOpt(r, "document"),
		VehicleName:  formOpt(r, "vehicle_name"),
		VehiclePlate: formOpt(r, "vehicle_plate"),
		KmPerL:       formOpt(r, "km_per_l"),
		Active:       formBool(r, "active"),
	})
	if err != nil {
		if statusFor(err) >= http.StatusInternalServerError {
			log.Printf("httpapi: driver edit: %v", err)
			err = errInternal
		}
		d, derr := s.findDriver(r, id)
		if derr != nil {
			http.Error(w, "not found", statusFor(derr))
			return
		}
		s.render(w, r, http.StatusOK, "driver_edit", driverEditPageData{Driver: d}, driversLabels, formErrors(err))
		return
	}
	s.setFlash(w, "Driver updated.")
	http.Redirect(w, r, "/drivers", http.StatusSeeOther)
}

// --- JSON mirrors -------------------------------------------------------

type apiCreateDriverBody struct {
	Name         string  `json:"name"`
	Email        string  `json:"email"`
	Password     string  `json:"password"`
	Phone        string  `json:"phone"`
	Document     *string `json:"document"`
	VehicleName  *string `json:"vehicle_name"`
	VehiclePlate *string `json:"vehicle_plate"`
	KmPerL       *string `json:"km_per_l"`
}

func (s *Server) apiDriversList(w http.ResponseWriter, r *http.Request) {
	actor, _ := s.actor(r)
	drivers, err := s.svc.ListDrivers(r.Context(), actor, r.URL.Query().Get("active_only") == "true")
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"drivers": drivers})
}

func (s *Server) apiDriversCreate(w http.ResponseWriter, r *http.Request) {
	var body apiCreateDriverBody
	if err := decodeJSON(r, &body); err != nil {
		writeJSONError(w, err)
		return
	}
	actor, _ := s.actor(r)
	d, err := s.svc.CreateDriver(r.Context(), actor, app.CreateDriverInput{
		Name: body.Name, Email: body.Email, Password: body.Password, Phone: body.Phone,
		Document: body.Document, VehicleName: body.VehicleName,
		VehiclePlate: body.VehiclePlate, KmPerL: body.KmPerL,
	})
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"driver": d})
}

// apiUpdateDriverBody: absent keeps a field; for the optional profile
// fields an explicit null clears it (km_per_l null = default parameter).
type apiUpdateDriverBody struct {
	Name         *string        `json:"name"`
	Phone        *string        `json:"phone"`
	Document     nullableString `json:"document"`
	VehicleName  nullableString `json:"vehicle_name"`
	VehiclePlate nullableString `json:"vehicle_plate"`
	KmPerL       nullableString `json:"km_per_l"`
	Active       *bool          `json:"active"`
}

// nullableString tells a JSON key's three states apart: absent (Set
// false), null (Set true, Value nil), or a string value.
type nullableString struct {
	Set   bool
	Value *string
}

func (n *nullableString) UnmarshalJSON(b []byte) error {
	n.Set = true
	if string(b) == "null" {
		n.Value = nil
		return nil
	}
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	n.Value = &v
	return nil
}

// null reports an explicit JSON null.
func (n nullableString) null() bool { return n.Set && n.Value == nil }

func (s *Server) apiDriverUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSONError(w, fmt.Errorf("%w: invalid id", app.ErrBadInput))
		return
	}
	var body apiUpdateDriverBody
	if err := decodeJSON(r, &body); err != nil {
		writeJSONError(w, err)
		return
	}
	actor, _ := s.actor(r)
	d, err := s.svc.UpdateDriver(r.Context(), actor, app.UpdateDriverInput{
		DriverID: id, Name: body.Name, Phone: body.Phone,
		Document: body.Document.Value, VehicleName: body.VehicleName.Value,
		VehiclePlate: body.VehiclePlate.Value, KmPerL: body.KmPerL.Value, Active: body.Active,
		Clear: app.DriverClear{
			Document: body.Document.null(), VehicleName: body.VehicleName.null(),
			VehiclePlate: body.VehiclePlate.null(), KmPerL: body.KmPerL.null(),
		},
	})
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"driver": d})
}

// apiDriverAnonymize is the JSON transport of AnonymizeDriver (admin,
// RNF06 full erasure); answers the pseudonymized driver.
func (s *Server) apiDriverAnonymize(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSONError(w, fmt.Errorf("%w: invalid id", app.ErrBadInput))
		return
	}
	actor, _ := s.actor(r)
	d, err := s.svc.AnonymizeDriver(r.Context(), actor, id)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"driver": d})
}
