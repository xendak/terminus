package httpapi

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// CSV export (RF12): UTF-8 with BOM so pt-BR Excel opens it directly;
// RFC 4180 (CRLF, quoted fields); timestamps in the display zone.
// Columns per operations.md, headed in pt-BR: Data, Motorista, Ordem,
// Endereço, Chegada, Saída, Minutos parados, Total do roteiro (min),
// Custo do roteiro (R$).
// Totals come from SQL — the writer only formats.

// exportCSV is the page transport (download link): errors are plain text.
func (s *Server) exportCSV(w http.ResponseWriter, r *http.Request) {
	s.writeExport(w, r, func(err error) { http.Error(w, "bad request", statusFor(err)) })
}

// apiExportCSV is the JSON transport: same CSV on success, the JSON
// error body on failure (operations.md error model).
func (s *Server) apiExportCSV(w http.ResponseWriter, r *http.Request) {
	s.writeExport(w, r, func(err error) { writeJSONError(w, err) })
}

func (s *Server) writeExport(w http.ResponseWriter, r *http.Request, fail func(error)) {
	actor, _ := s.actor(r)
	q := r.URL.Query()
	var driver *uuid.UUID
	if id := q.Get("driver_user_id"); id != "" {
		parsed, err := uuid.Parse(id)
		if err != nil {
			fail(errBadDriver)
			return
		}
		driver = &parsed
	}
	team, err := optUUIDParam(r, "manager_user_id")
	if err != nil {
		fail(err)
		return
	}
	from, to := q.Get("from"), q.Get("to")
	rows, err := s.svc.ExportPeriodCSV(r.Context(), actor, from, to, driver, team)
	if err != nil {
		fail(err)
		return
	}

	// from/to passed the service's YYYY-MM-DD validation: safe in a filename.
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="terminus-`+from+`-a-`+to+`.csv"`)
	// The BOM is the first three bytes — Excel's pt-BR friend.
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	cw := csv.NewWriter(w)
	cw.UseCRLF = true // RFC 4180
	_ = cw.Write([]string{
		"Data", "Motorista", "Ordem", "Endereço",
		"Chegada", "Saída", "Minutos parados",
		"Total do roteiro (min)", "Custo do roteiro (R$)",
	})
	for _, row := range rows {
		_ = cw.Write([]string{
			fmtDate(row.RouteDate),
			row.DriverName,
			strconv.Itoa(row.StopOrder),
			row.Address,
			fmtDT(row.ArrivalAt),
			fmtDT(row.DepartureAt),
			optInt(row.StopMinutes),
			strconv.Itoa(row.RouteTotalMinutes),
			optStr(row.RouteCost),
		})
	}
	cw.Flush()
}

func fmtDT(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.In(saoPaulo).Format("02/01/2006 15:04")
}

func optStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func optInt(i *int) string {
	if i == nil {
		return ""
	}
	return strconv.Itoa(*i)
}
