package httpapi_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"

	"stoptime/internal/app"
)

// T9 screens: dashboard, history, params, audit, export — against the
// golden fixture (June 2026: day/month/period 161, routes 75/41/45).

func TestDashboardsGolden(t *testing.T) {
	admin := loginSession(t, adminEmail)

	// Day cut: one point, 161 minutes.
	out, err := jsonCall(t, admin, "GET", "/api/dashboard/day?from=2026-06-01&to=2026-06-30", "")
	must(err)
	var day struct {
		Series []struct {
			Date              string `json:"date"`
			TotalStoppedMinut int    `json:"total_stopped_minutes"`
			JourneyPercent    string `json:"journey_percent"`
		} `json:"series"`
	}
	must(json.Unmarshal([]byte(jsonString(out)), &day))
	if len(day.Series) != 1 || day.Series[0].TotalStoppedMinut != 161 ||
		day.Series[0].Date != "2026-06-15" || day.Series[0].JourneyPercent != "11.181" {
		t.Errorf("day series = %+v, want one {2026-06-15, 161, 11.181} point", day.Series)
	}
	if out["standard_journey_hours"] != "8.0000" {
		t.Errorf("day standard_journey_hours = %v, want 8.0000", out["standard_journey_hours"])
	}

	// Month cut (narrow window — later suites add July fixtures).
	out, err = jsonCall(t, admin, "GET", "/api/dashboard/month?from=2026-06-01&to=2026-06-30", "")
	must(err)
	var month struct {
		Series []struct {
			Month             string `json:"month"`
			TotalStoppedMinut int    `json:"total_stopped_minutes"`
			JourneyPercent    string `json:"journey_percent"`
		} `json:"series"`
	}
	must(json.Unmarshal([]byte(jsonString(out)), &month))
	if out["standard_journey_hours"] != "8.0000" {
		t.Errorf("month standard_journey_hours = %v, want 8.0000", out["standard_journey_hours"])
	}
	if len(month.Series) != 1 || month.Series[0].Month != "2026-06" || month.Series[0].TotalStoppedMinut != 161 ||
		month.Series[0].JourneyPercent != "11.181" {
		t.Errorf("month series = %+v, want {2026-06, 161, 11.181}", month.Series)
	}

	// Period cut with the per-driver ranking.
	out, err = jsonCall(t, admin, "GET", "/api/dashboard/period?from=2026-06-01&to=2026-06-30", "")
	must(err)
	// operations.md shape: the summary object itself, no wrapper.
	if _, wrapped := out["series"]; wrapped {
		t.Errorf("period answer is wrapped in series: %v", out)
	}
	var period struct {
		StandardJourneyHours string `json:"standard_journey_hours"`
		TotalStoppedMinut int    `json:"total_stopped_minutes"`
		JourneyPercent    string `json:"journey_percent"`
		RoutesCount       int    `json:"routes_count"`
		ByDriver          []struct {
			DriverUserID   string `json:"driver_user_id"`
			DriverName     string `json:"driver_name"`
			JourneyPercent string `json:"journey_percent"`
		} `json:"by_driver"`
	}
	must(json.Unmarshal([]byte(jsonString(out)), &period))
	if period.TotalStoppedMinut != 161 || period.JourneyPercent != "11.181" ||
		period.RoutesCount != 3 || len(period.ByDriver) != 3 || period.ByDriver[2].JourneyPercent != "15.625" {
		t.Errorf("period = %+v, want 161 / 11.181 / 3 routes / 3 drivers", period)
	}
	if len(period.ByDriver) == 3 && period.ByDriver[2].DriverUserID != "aa000000-0000-4000-8000-000000000003" {
		t.Errorf("by_driver[2].driver_user_id = %q, want Marcos's id", period.ByDriver[2].DriverUserID)
	}
	if period.StandardJourneyHours != "8.0000" {
		t.Errorf("period standard_journey_hours = %q, want 8.0000", period.StandardJourneyHours)
	}

	// The page embeds the same series and the chart canvases.
	status, body, _ := do(t, admin, "GET", "/dashboard?from=2026-06-01&to=2026-06-30", "", "")
	if status != http.StatusOK || !strings.Contains(body, "161") ||
		!strings.Contains(body, `id="dayChart"`) || !strings.Contains(body, `id="monthChart"`) {
		t.Errorf("dashboard page = %d, want the series and charts", status)
	}

	// Driver scoping: Bianca sees only her day (41 minutes).
	driverB := loginSession(t, "driver-b@stoptime.dev")
	out, err = jsonCall(t, driverB, "GET", "/api/dashboard/day?from=2026-06-01&to=2026-06-30", "")
	must(err)
	must(json.Unmarshal([]byte(jsonString(out)), &day))
	if len(day.Series) != 1 || day.Series[0].TotalStoppedMinut != 41 {
		t.Errorf("driver's day series = %+v, want one 41-minute point", day.Series)
	}
}

// jsonString re-encodes a decoded map (out is map[string]any already).
func jsonString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func TestHistoryGolden(t *testing.T) {
	admin := loginSession(t, adminEmail)

	status, body, _ := do(t, admin, "GET", "/api/routes?from=2026-06-01&to=2026-06-30", "", "")
	if status != http.StatusOK {
		t.Fatalf("history api = %d", status)
	}
	var list struct {
		Routes []struct {
			RouteDate         string `json:"route_date"`
			DriverName        string `json:"driver_name"`
			TotalStoppedMinut int    `json:"total_stopped_minutes"`
			StopCount         int    `json:"stop_count"`
		} `json:"routes"`
	}
	must(json.Unmarshal([]byte(body), &list))
	if len(list.Routes) != 3 {
		t.Fatalf("history rows = %d, want 3 (golden window)", len(list.Routes))
	}
	for i, want := range []struct {
		name string
		min  int
	}{{"Bianca Batista", 41}, {"Carla Camargo", 45}, {"Marcos Motorista", 75}} {
		if list.Routes[i].DriverName != want.name || list.Routes[i].TotalStoppedMinut != want.min ||
			list.Routes[i].RouteDate != "2026-06-15" {
			t.Errorf("row %d = %+v, want %s %d", i, list.Routes[i], want.name, want.min)
		}
	}

	// Drivers see their own rows only (service scoping through the API).
	driverA := loginSession(t, "driver-a@stoptime.dev")
	status, body, _ = do(t, driverA, "GET", "/api/routes?from=2026-06-01&to=2026-06-30", "", "")
	must(json.Unmarshal([]byte(body), &list))
	if len(list.Routes) != 1 || list.Routes[0].DriverName != "Marcos Motorista" {
		t.Errorf("driver's history = %+v, want only Marcos", list.Routes)
	}

	// The history page lists the golden rows.
	status, body, _ = do(t, admin, "GET", "/history?from=2026-06-01&to=2026-06-30", "", "")
	if status != http.StatusOK || !strings.Contains(body, "Marcos Motorista") || !strings.Contains(body, "75") {
		t.Errorf("history page = %d", status)
	}
	// A route_date is a calendar day: shown as-is, never shifted by a
	// time zone conversion.
	if !strings.Contains(body, "15/06/2026") || strings.Contains(body, "14/06/2026") {
		t.Errorf("history page does not show the golden date 15/06/2026")
	}
}

func TestParamsScreensAndAPI(t *testing.T) {
	admin := loginSession(t, adminEmail)

	out, err := jsonCall(t, admin, "GET", "/api/params", "")
	must(err)
	var params struct {
		Params []struct {
			Key           string `json:"key"`
			Value         string `json:"value"`
			UpdatedByName string `json:"updated_by_name"`
		} `json:"params"`
	}
	must(json.Unmarshal([]byte(jsonString(out)), &params))
	if len(params.Params) != 5 {
		t.Fatalf("params = %d, want 5", len(params.Params))
	}
	for _, p := range params.Params {
		if p.UpdatedByName == "" {
			t.Errorf("param %s JSON lacks updated_by_name", p.Key)
		}
	}

	// The PUT mirror updates; the value round-trips scale-normalized.
	out, err = jsonCall(t, admin, "PUT", "/api/params/fuel_price_brl", `{"value": "6.19"}`)
	must(err)
	if !strings.Contains(jsonString(out), "6.1900") {
		t.Errorf("param update = %v", out)
	}

	// The form path redirects with a flash; invalid re-renders inline.
	status, _, _ := postForm(t, admin, "/params/min_stop_minutes", url.Values{"value": {"6"}})
	if status != http.StatusSeeOther {
		t.Errorf("param form update = %d, want 303", status)
	}
	status, body, _ := postForm(t, admin, "/params/fuel_price_brl", url.Values{"value": {"-1"}})
	if status != http.StatusOK || !strings.Contains(body, "must not be negative") {
		t.Errorf("invalid param = %d, want inline error", status)
	}
	// Restore the default (later suites pin 6.09-dependent costs).
	if _, err := svc.UpdateParam(ctx(), app.Actor{UserID: adminID, Role: "admin"},
		app.UpdateParamInput{Key: "fuel_price_brl", Value: "6.09"}); err != nil {
		t.Fatal(err)
	}

	// Drivers never see parameters (matrix).
	driver := loginSession(t, "driver-a@stoptime.dev")
	status, _, _ = do(t, driver, "GET", "/api/params", "", "")
	if status != http.StatusForbidden {
		t.Errorf("driver params = %d, want 403", status)
	}
}

func TestCorrectionsAndAudit(t *testing.T) {
	admin := loginSession(t, adminEmail)

	// A route with one recorded arrival to correct.
	driverEmail := "t9-corr-" + uuid.NewString()[:8] + "@test.dev"
	out, err := jsonCall(t, admin, "POST", "/api/drivers", fmt.Sprintf(
		`{"name": "T9 Corr Driver", "email": %q, "password": "pw-corr-1", "phone": "0"}`, driverEmail))
	must(err)
	driverID := out["driver"].(map[string]any)["id"].(string)
	locations := make([]string, 0, 2)
	for i := 0; i < 2; i++ {
		out, err = jsonCall(t, admin, "POST", "/api/locations",
			fmt.Sprintf(`{"label": "T9 Corr P%d", "address": "Rua T9C, %d"}`, i, i))
		must(err)
		locations = append(locations, out["location"].(map[string]any)["id"].(string))
	}
	out, err = jsonCall(t, admin, "POST", "/api/routes", fmt.Sprintf(
		`{"driver_user_id": %q, "route_date": "2026-09-01", "location_ids": ["%s", "%s"]}`,
		driverID, locations[0], locations[1]))
	must(err)
	routeID := out["route"].(map[string]any)["id"].(string)
	_, err = jsonCall(t, admin, "POST", "/api/routes/"+routeID+"/start", "")
	must(err)
	_, err = jsonCall(t, admin, "POST", "/api/routes/"+routeID+"/stops/2/arrive", `{"at": "2026-09-01T09:00:00-03:00"}`)
	must(err)

	// The corrections form (manager/admin) updates both fields.
	manager := loginSession(t, "manager@stoptime.dev")
	status, body, _ := postForm(t, manager, "/routes/"+routeID+"/stops/2/times", url.Values{
		"arrival_at":   {"2026-09-01T09:05"},
		"departure_at": {"2026-09-01T09:20"},
	})
	if status != http.StatusOK || !strings.Contains(body, "09:05") {
		t.Fatalf("corrections fragment = %d", status)
	}

	// RN02 through the corrections path.
	status, body, _ = postForm(t, manager, "/routes/"+routeID+"/stops/2/times", url.Values{
		"arrival_at": {"2026-09-01T09:30"},
	})
	_ = body
	if status != http.StatusOK {
		t.Logf("corrections RN02 = %d (rendered inline)", status)
	}

	// The audit trail answers from the API: the correction is there.
	out, err = jsonCall(t, admin, "GET", "/api/audit?entity=route_stop", "")
	must(err)
	raw := jsonString(out)
	if !strings.Contains(raw, "update_times") || !strings.Contains(raw, "Gustavo Gerente") {
		t.Errorf("audit entries missing the correction: %s", raw[:min(len(raw), 400)])
	}

	// Audit is admin-only (matrix): the manager who corrected cannot list.
	status, _, _ = do(t, manager, "GET", "/api/audit", "", "")
	if status != http.StatusForbidden {
		t.Errorf("manager audit = %d, want 403", status)
	}

	// The audit page renders the same trail.
	status, body, _ = do(t, admin, "GET", "/audit", "", "")
	if status != http.StatusOK || !strings.Contains(body, "update_times") {
		t.Errorf("audit page = %d", status)
	}
}

func TestCSVExport(t *testing.T) {
	admin := loginSession(t, adminEmail)

	req, _ := http.NewRequest("GET", tsURL+"/api/export?from=2026-06-01&to=2026-06-30", nil)
	resp := doWithClient(t, admin, req)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("export = %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	must(err)

	// The BOM is the first three bytes — pt-BR Excel opens it directly.
	if !bytes.HasPrefix(raw, []byte{0xEF, 0xBB, 0xBF}) {
		t.Fatal("export is missing the UTF-8 BOM")
	}
	r := csv.NewReader(bytes.NewReader(raw[3:]))
	rows, err := r.ReadAll()
	must(err)
	if len(rows) != 13 { // header + 3 routes × 4 stops
		t.Fatalf("csv rows = %d, want 13", len(rows))
	}
	if cd := resp.Header.Get("Content-Disposition"); cd != `attachment; filename="terminus-2026-06-01-a-2026-06-30.csv"` {
		t.Errorf("Content-Disposition = %q", cd)
	}
	wantHeader := []string{"Data", "Motorista", "Ordem", "Endereço", "Chegada", "Saída",
		"Minutos parados", "Conta no total", "Total do roteiro (min)", "Custo do roteiro (R$)"}
	if len(rows[0]) != len(wantHeader) {
		t.Fatalf("header = %v, want %d columns", rows[0], len(wantHeader))
	}
	for i, col := range wantHeader {
		if rows[0][i] != col {
			t.Fatalf("header[%d] = %q, want %q", i, rows[0][i], col)
		}
	}
	// Route A rows: address + registered timestamps + stop minutes, and
	// the route total (75) on every row — criterion 3 in CSV form.
	marcos := [][]string{}
	for _, row := range rows[1:] {
		if row[1] == "Marcos Motorista" {
			marcos = append(marcos, row)
		}
	}
	if len(marcos) != 4 {
		t.Fatalf("Marcos rows = %d, want 4", len(marcos))
	}
	for i, want := range []struct {
		address string
		minutes string
		arrival string
		counted string
	}{
		{"Av. Partida, 100", "0", "15/06/2026 08:00", "Não"}, // departure point (RN01)
		{"Rua Peru, 55", "15", "15/06/2026 09:00", "Sim"},
		{"Rua X, 5", "10", "15/06/2026 10:00", "Sim"},
		{"Av. João César", "50", "15/06/2026 11:00", "Sim"},
	} {
		row := marcos[i]
		if row[2] != fmt.Sprint(i+1) || row[3] != want.address || row[6] != want.minutes ||
			row[7] != want.counted || row[8] != "75" || row[4] != want.arrival || row[9] != "" {
			t.Errorf("Marcos row %d = %v, want order %d %s %smin total 75", i+1, row, i+1, want.address, want.minutes)
		}
	}

	// With min_stop_minutes = 12, route A's 10-minute stop keeps its
	// minutes but is not counted: "Não", and the route total reads 65.
	setMin := func(v string) {
		t.Helper()
		if _, err := svc.UpdateParam(ctx(), app.Actor{UserID: adminID, Role: "admin"},
			app.UpdateParamInput{Key: "min_stop_minutes", Value: v}); err != nil {
			t.Fatal(err)
		}
	}
	setMin("12")
	t.Cleanup(func() { setMin("0") })
	_, minBody, _ := do(t, admin, "GET", "/api/export?from=2026-06-01&to=2026-06-30", "", "")
	minRows, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(minBody, "\ufeff"))).ReadAll()
	must(err)
	var gotCounted, gotTotals []string
	for _, row := range minRows[1:] {
		if row[1] == "Marcos Motorista" {
			gotCounted = append(gotCounted, row[6]+":"+row[7])
			gotTotals = append(gotTotals, row[8])
		}
	}
	if strings.Join(gotCounted, ",") != "0:Não,15:Sim,10:Não,50:Sim" || strings.Join(gotTotals, ",") != "65,65,65,65" {
		t.Errorf("min 12 export = %v totals %v, want the 10-minute stop not counted and 65", gotCounted, gotTotals)
	}
	setMin("0")

	// Drivers export their own data only.
	client := loginSession(t, "driver-a@stoptime.dev")
	req, _ = http.NewRequest("GET", tsURL+"/api/export?from=2026-06-01&to=2026-06-30", nil)
	resp2 := doWithClient(t, client, req)
	raw2, err := io.ReadAll(resp2.Body)
	must(err)
	r2 := csv.NewReader(bytes.NewReader(raw2[3:]))
	rows2, err2 := r2.ReadAll()
	must(err2)
	if len(rows2) != 5 { // header + 4 own stops
		t.Errorf("driver export rows = %d, want 5", len(rows2))
	}
	for _, row := range rows2[1:] {
		if row[1] != "Marcos Motorista" {
			t.Errorf("driver export leaked another driver: %v", row)
		}
	}

	// The JSON transport answers errors with the JSON error body.
	status, body, h := do(t, admin, "GET", "/api/export?to=2026-06-30", "", "")
	if status != http.StatusBadRequest || !strings.HasPrefix(h.Get("Content-Type"), "application/json") ||
		!strings.Contains(body, `"error"`) {
		t.Errorf("export without from = %d %q %s, want 400 JSON error", status, h.Get("Content-Type"), body)
	}
	status, body, _ = do(t, admin, "GET", "/api/export?from=2026-06-01&to=2026-06-30&driver_user_id=junk", "", "")
	// Same mapping as GET /api/routes: a field error on driver_user_id.
	if status != http.StatusUnprocessableEntity || !strings.Contains(body, `"field":"driver_user_id"`) {
		t.Errorf("export bad driver = %d %s, want 422 JSON field error", status, body)
	}
}

func doWithClient(t *testing.T, client *http.Client, req *http.Request) *http.Response {
	t.Helper()
	resp, err := client.Do(req)
	must(err)
	return resp
}

func ctx() context.Context { return context.Background() }
