package httpapi

import (
	"net/http"

	"stoptime/internal/app"
)

// Parameters (screens.md §7): every tunable number in one screen —
// change a value, the audit row is written, history recalculates on the
// next read (business-rules.md RN07).

var paramsLabels = labelsFor(map[string]string{
	"Title":     "Parameters",
	"Key":       "Key",
	"Value":     "Value",
	"Unit":      "Unit",
	"Updated":   "Last updated",
	"Save":      "Save",
	"Empty":     "No parameters registered.",
})

type paramsPageData struct {
	Params []appParamView
}

type appParamView struct {
	Key       string
	Value     string
	Unit      string
	UpdatedAt string
}

func (s *Server) paramsPage(w http.ResponseWriter, r *http.Request) {
	actor, _ := s.actor(r)
	params, err := s.svc.GetParams(r.Context(), actor)
	if err != nil {
		http.Error(w, "forbidden", statusFor(err))
		return
	}
	views := make([]appParamView, 0, len(params))
	for _, p := range params {
		views = append(views, appParamView{
			Key: p.Key, Value: p.Value, Unit: p.Unit,
			UpdatedAt: fmtTime(p.UpdatedAt),
		})
	}
	s.render(w, r, http.StatusOK, "params", paramsPageData{Params: views}, paramsLabels, nil)
}

// paramUpdate serves the inline edit form (POST /params/{key}).
func (s *Server) paramUpdate(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	_ = r.ParseForm()
	actor, _ := s.actor(r)
	_, err := s.svc.UpdateParam(r.Context(), actor, app.UpdateParamInput{
		Key: key, Value: r.PostFormValue("value"),
	})
	if err != nil {
		if statusFor(err) >= http.StatusInternalServerError {
			logInternal(w, "param update", err)
			return
		}
		params, perr := s.svc.GetParams(r.Context(), actor)
		if perr != nil {
			http.Error(w, "forbidden", statusFor(perr))
			return
		}
		views := make([]appParamView, 0, len(params))
		for _, p := range params {
			views = append(views, appParamView{Key: p.Key, Value: p.Value, Unit: p.Unit, UpdatedAt: fmtTime(p.UpdatedAt)})
		}
		s.render(w, r, http.StatusOK, "params", paramsPageData{Params: views}, paramsLabels, formErrors(err))
		return
	}
	s.setFlash(w, "Parameter updated.")
	http.Redirect(w, r, "/params", http.StatusSeeOther)
}

// --- JSON mirrors -------------------------------------------------------

func (s *Server) apiParamsList(w http.ResponseWriter, r *http.Request) {
	actor, _ := s.actor(r)
	params, err := s.svc.GetParams(r.Context(), actor)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"params": params})
}

type apiParamBody struct {
	Value string `json:"value"`
}

func (s *Server) apiParamUpdate(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	var body apiParamBody
	if err := decodeJSON(r, &body); err != nil {
		writeJSONError(w, err)
		return
	}
	actor, _ := s.actor(r)
	p, err := s.svc.UpdateParam(r.Context(), actor, app.UpdateParamInput{Key: key, Value: body.Value})
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"param": p})
}
