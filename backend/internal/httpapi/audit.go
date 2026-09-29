package httpapi

import (
	"net/http"

	"stoptime/internal/app"
	"stoptime/internal/store"
)

// Audit (screens.md §8): who changed what and when, from the
// append-only audit_log (RNF05). Admin only.

var auditLabels = labelsFor(map[string]string{
	"Title":  "Audit",
	"Entity": "Entity",
	"From":   "From",
	"To":     "To",
	"Apply":  "Apply",
	"When":   "When",
	"Actor":  "Who",
	"Action": "Action",
	"EntityID": "Entity",
	"Changes": "Changes",
	"Empty":  "No audit entries in this range.",
	"All":    "All",
})

type auditPageData struct {
	Entity  string
	From    string
	To      string
	Entries []auditView
}

type auditView struct {
	At        string
	Actor     string
	Entity    string
	EntityID  string
	Action    string
	OldValues string
	NewValues string
}

func (s *Server) auditPage(w http.ResponseWriter, r *http.Request) {
	actor, _ := s.actor(r)
	q := r.URL.Query()
	in := app.ListAuditInput{}
	if q.Get("entity") != "" {
		e := q.Get("entity")
		in.Entity = &e
	}
	if q.Get("from") != "" {
		f := q.Get("from")
		in.From = &f
	}
	if q.Get("to") != "" {
		t := q.Get("to")
		in.To = &t
	}
	entries, err := s.svc.ListAudit(r.Context(), actor, in)
	if err != nil {
		http.Error(w, "forbidden", statusFor(err))
		return
	}
	data := auditPageData{Entity: q.Get("entity"), From: q.Get("from"), To: q.Get("to")}
	for _, e := range entries {
		data.Entries = append(data.Entries, auditView{
			At: fmtTime(e.At), Actor: e.Actor, Entity: e.Entity,
			EntityID: shortID(e.EntityID), Action: e.Action,
			OldValues: string(e.OldValues), NewValues: string(e.NewValues),
		})
	}
	s.render(w, r, http.StatusOK, "audit", data, auditLabels, nil)
}

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8] + "…"
}

// --- JSON mirror --------------------------------------------------------

func (s *Server) apiAuditList(w http.ResponseWriter, r *http.Request) {
	actor, _ := s.actor(r)
	q := r.URL.Query()
	in := app.ListAuditInput{}
	if q.Get("entity") != "" {
		e := q.Get("entity")
		in.Entity = &e
	}
	if q.Get("from") != "" {
		f := q.Get("from")
		in.From = &f
	}
	if q.Get("to") != "" {
		t := q.Get("to")
		in.To = &t
	}
	entries, err := s.svc.ListAudit(r.Context(), actor, in)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	if entries == nil {
		entries = []store.AuditEntryView{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries})
}
