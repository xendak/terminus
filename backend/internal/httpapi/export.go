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
// Columns per operations.md: route date, driver, stop order, address,
// arrival, departure, stop minutes, route total minutes, route cost.
// Totals come from SQL — the writer only formats.

func (s *Server) exportCSV(w http.ResponseWriter, r *http.Request) {
	actor, _ := s.actor(r)
	q := r.URL.Query()
	var driver *uuid.UUID
	if id := q.Get("driver_user_id"); id != "" {
		parsed, err := uuid.Parse(id)
		if err != nil {
			http.Error(w, "bad driver", http.StatusBadRequest)
			return
		}
		driver = &parsed
	}
	rows, err := s.svc.ExportPeriodCSV(r.Context(), actor, q.Get("from"), q.Get("to"), driver)
	if err != nil {
		http.Error(w, "bad request", statusFor(err))
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="stoptime-export.csv"`)
	// The BOM is the first three bytes — Excel's pt-BR friend.
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	cw := csv.NewWriter(w)
	cw.UseCRLF = true // RFC 4180
	_ = cw.Write([]string{
		"route date", "driver", "stop order", "address",
		"arrival", "departure", "stop minutes",
		"route total minutes", "route cost",
	})
	for _, row := range rows {
		_ = cw.Write([]string{
			row.RouteDate.In(saoPaulo).Format("02/01/2006"),
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
