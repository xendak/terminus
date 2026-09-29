package httpapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"stoptime/internal/app"
)

// Builder + tracker flows (screens.md §2/§3) over both transports. The
// pages tests run against a live server; fixtures are created per test
// with unique ids.

func resetFuelPrice(t *testing.T) {
	t.Helper()
	// The app suite (which runs first) ends with fuel_price_brl = 6.19;
	// the golden cost assertions here need the default.
	_, err := svc.UpdateParam(context.Background(), app.Actor{UserID: adminID, Role: "admin"},
		app.UpdateParamInput{Key: "fuel_price_brl", Value: "6.09"})
	if err != nil {
		t.Fatal(err)
	}
}

// jsonCall POSTs/PUTs a JSON body and decodes the response.
func jsonCall(t *testing.T, client *http.Client, method, path, body string) (map[string]any, error) {
	t.Helper()
	status, raw, _ := do(t, client, method, path, "application/json", body)
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("status %d, non-JSON body: %s", status, raw)
	}
	if status >= 300 {
		return out, fmt.Errorf("%s %s: status %d, body %s", method, path, status, raw)
	}
	return out, nil
}

func routeView(t *testing.T, client *http.Client, id string) map[string]any {
	t.Helper()
	out, err := jsonCall(t, client, "GET", "/api/routes/"+id, "")
	if err != nil {
		t.Fatal(err)
	}
	route, _ := out["route"].(map[string]any)
	if route == nil {
		t.Fatal("no route object in response")
	}
	return route
}

func stopsOf(route map[string]any) []any {
	stops, _ := route["stops"].([]any)
	return stops
}

func TestBuilderAndTrackerAPI(t *testing.T) {
	resetFuelPrice(t)
	admin := loginSession(t, adminEmail)

	// Fixtures: a driver and four locations.
	driverEmail := "t8-driver-" + uuid.NewString()[:8] + "@test.dev"
	out, err := jsonCall(t, admin, "POST", "/api/drivers", fmt.Sprintf(
		`{"name": "T8 Driver", "email": %q, "password": "pw-driver", "phone": "0"}`, driverEmail))
	must(err)
	driverID := out["driver"].(map[string]any)["id"].(string)

	locationIDs := make([]string, 0, 4)
	for i := 0; i < 4; i++ {
		out, err = jsonCall(t, admin, "POST", "/api/locations", fmt.Sprintf(
			`{"label": "T8 P%d", "address": "Rua T8, %d"}`, i, i))
		must(err)
		locationIDs = append(locationIDs, out["location"].(map[string]any)["id"].(string))
	}

	// Builder: create with the four locations in visit order.
	idsJSON := fmt.Sprintf(`["%s", "%s", "%s", "%s"]`,
		locationIDs[0], locationIDs[1], locationIDs[2], locationIDs[3])
	out, err = jsonCall(t, admin, "POST", "/api/routes", fmt.Sprintf(
		`{"driver_user_id": %q, "route_date": "2026-07-15", "location_ids": %s}`, driverID, idsJSON))
	must(err)
	route := out["route"].(map[string]any)
	routeID := route["id"].(string)
	if len(stopsOf(route)) != 4 || route["status"] != "draft" {
		t.Fatalf("created route = %v", route)
	}
	if stopsOf(route)[0].(map[string]any)["counted"] != false {
		t.Error("stop 1 must be the departure point (counted=false)")
	}

	// One location too few is the builder's invalid state.
	_, err = jsonCall(t, admin, "POST", "/api/routes", fmt.Sprintf(
		`{"driver_user_id": %q, "route_date": "2026-07-16", "location_ids": ["%s"]}`, driverID, locationIDs[0]))
	if err == nil || !strings.Contains(err.Error(), "422") {
		t.Errorf("one-stop route: err = %v, want 422", err)
	}

	// Composition edits over the API: add in the middle, remove it, move.
	extra, _ := jsonCall(t, admin, "POST", "/api/locations", `{"label": "T8 Extra", "address": "Rua T8, 99"}`)
	extraID := extra["location"].(map[string]any)["id"].(string)
	_, err = jsonCall(t, admin, "POST", "/api/routes/"+routeID+"/stops",
		fmt.Sprintf(`{"location_id": %q, "position": 2}`, extraID))
	must(err)
	route = routeView(t, admin, routeID)
	if len(stopsOf(route)) != 5 {
		t.Fatalf("after add: %d stops", len(stopsOf(route)))
	}
	_, err = jsonCall(t, admin, "DELETE", "/api/routes/"+routeID+"/stops/2", "")
	must(err)
	route = routeView(t, admin, routeID)
	if len(stopsOf(route)) != 4 {
		t.Fatalf("after remove: %d stops", len(stopsOf(route)))
	}
	_, err = jsonCall(t, admin, "POST", "/api/routes/"+routeID+"/stops/3/move", `{"direction": "up"}`)
	must(err)
	_, err = jsonCall(t, admin, "POST", "/api/routes/"+routeID+"/stops/2/move", `{"direction": "down"}`)
	must(err)

	// RN05 over the API: same driver, same date.
	_, err = jsonCall(t, admin, "POST", "/api/routes", fmt.Sprintf(
		`{"driver_user_id": %q, "route_date": "2026-07-15", "location_ids": %s}`, driverID, idsJSON))
	if err == nil || !strings.Contains(err.Error(), "409") {
		t.Errorf("duplicate driver+date: err = %v, want 409", err)
	}

	// Tracker: start, then record the route A day (15/10/50 + stop 1).
	_, err = jsonCall(t, admin, "POST", "/api/routes/"+routeID+"/start", "")
	must(err)
	times := [][2]string{
		{"08:00", "08:05"}, // stop 1: departure point (RN01)
		{"09:00", "09:15"}, // 15 min
		{"10:00", "10:10"}, // 10 min
		{"11:00", "11:50"}, // 50 min
	}
	for i, ts := range times {
		order := i + 1
		_, err = jsonCall(t, admin, "POST",
			fmt.Sprintf("/api/routes/%s/stops/%d/arrive", routeID, order),
			fmt.Sprintf(`{"at": "2026-07-15T%s:00-03:00"}`, ts[0]))
		must(err)
		_, err = jsonCall(t, admin, "POST",
			fmt.Sprintf("/api/routes/%s/stops/%d/depart", routeID, order),
			fmt.Sprintf(`{"at": "2026-07-15T%s:00-03:00"}`, ts[1]))
		must(err)
	}

	// Double record is the invalid state (corrections are manager-only).
	_, err = jsonCall(t, admin, "POST", "/api/routes/"+routeID+"/stops/2/arrive", `{"at": "2026-07-15T12:00:00-03:00"}`)
	if err == nil || !strings.Contains(err.Error(), "422") {
		t.Errorf("double arrival: err = %v, want 422", err)
	}

	// Distance + close; the read shows the golden numbers.
	_, err = jsonCall(t, admin, "PUT", "/api/routes/"+routeID+"/distance", `{"distance_km": "100"}`)
	must(err)
	_, err = jsonCall(t, admin, "POST", "/api/routes/"+routeID+"/close", "")
	must(err)

	route = routeView(t, admin, routeID)
	if route["total_stopped_minutes"] != float64(75) {
		t.Errorf("total_stopped_minutes = %v, want 75", route["total_stopped_minutes"])
	}
	if route["journey_percent"] != "15.625" {
		t.Errorf("journey_percent = %v, want 15.625", route["journey_percent"])
	}
	if route["estimated_cost_brl"] != "60.90" {
		t.Errorf("estimated_cost_brl = %v, want 60.90 (100 km / 10 km/l * 6.09)", route["estimated_cost_brl"])
	}
	if route["status"] != "closed" {
		t.Errorf("status = %v", route["status"])
	}

	// Driver scoping: the route's own driver reads it; others get 403.
	driver := loginSessionAs(t, driverEmail, "pw-driver")
	own, err := jsonCall(t, driver, "GET", "/api/routes/"+routeID, "")
	if err != nil {
		t.Fatalf("own route read: %v", err)
	}
	_ = own
	// A different driver (golden driver-a) is forbidden.
	stranger := loginSession(t, driverEmail2())
	status, _, _ := do(t, stranger, "GET", "/api/routes/"+routeID, "", "")
	if status != http.StatusForbidden {
		t.Errorf("stranger read = %d, want 403", status)
	}
	// Drivers cannot create routes.
	status, _, _ = do(t, driver, "POST", "/api/routes", "application/json", fmt.Sprintf(
		`{"driver_user_id": %q, "route_date": "2026-07-17", "location_ids": ["%s", "%s"]}`, driverID, locationIDs[0], locationIDs[1]))
	if status != http.StatusForbidden {
		t.Errorf("driver create route = %d, want 403", status)
	}
}

func driverEmail2() string { return "driver-b@stoptime.dev" }

func TestBuilderAndTrackerPages(t *testing.T) {
	admin := loginSession(t, adminEmail)

	// The builder page composes client-side; the form posts the ordered
	// hidden inputs as repeated location_ids fields.
	status, body, _ := do(t, admin, "GET", "/routes/new", "", "")
	if status != http.StatusOK || !strings.Contains(body, "loc-select") || !strings.Contains(body, "stop-list") {
		t.Fatalf("builder page = %d", status)
	}

	out, err := jsonCall(t, admin, "POST", "/api/drivers", fmt.Sprintf(
		`{"name": "T8 Page Driver", "email": "t8-page-%s@test.dev", "password": "pw-page", "phone": "0"}`, uuid.NewString()[:8]))
	must(err)
	driverID := out["driver"].(map[string]any)["id"].(string)
	driverEmail := out["driver"].(map[string]any)["email"].(string)
	locs := make([]string, 0, 2)
	for i := 0; i < 2; i++ {
		out, err = jsonCall(t, admin, "POST", "/api/locations",
			fmt.Sprintf(`{"label": "T8 Page P%d", "address": "Rua T8P, %d"}`, i, i))
		must(err)
		locs = append(locs, out["location"].(map[string]any)["id"].(string))
	}

	form := url.Values{
		"driver_user_id": {driverID},
		"route_date":     {"2026-07-18"},
		"location_ids":   locs,
	}
	status, _, location := postForm(t, admin, "/routes", form)
	if status != http.StatusSeeOther || !strings.HasPrefix(location, "/routes/") {
		t.Fatalf("builder submit = %d %q", status, location)
	}
	routeID := strings.TrimPrefix(location, "/routes/")

	// Draft detail: stop list, add-stop picker, start control.
	status, body, _ = do(t, admin, "GET", "/routes/"+routeID, "", "")
	if status != http.StatusOK || !strings.Contains(body, "Start route") ||
		!strings.Contains(body, "T8 Page P0") || !strings.Contains(body, "departure point") {
		t.Fatalf("draft detail = %d", status)
	}

	// Fragment mutations return the #route-body partial.
	extra, _ := jsonCall(t, admin, "POST", "/api/locations", `{"label": "T8 Page Extra", "address": "Rua T8PX, 9"}`)
	extraID := extra["location"].(map[string]any)["id"].(string)
	status, body, _ = postForm(t, admin, "/routes/"+routeID+"/stops", url.Values{"location_id": {extraID}})
	if status != http.StatusOK || !strings.Contains(body, `id="route-body"`) || !strings.Contains(body, "T8 Page Extra") {
		t.Fatalf("add stop fragment = %d", status)
	}
	status, body, _ = postForm(t, admin, "/routes/"+routeID+"/stops/3/remove", url.Values{})
	// The removed stop must be gone from the stop LIST — assert on the
	// ordinal markup, not the whole fragment: the draft's location
	// picker legitimately lists every directory location.
	if status != http.StatusOK || strings.Contains(body, "<b>3.</b>") {
		t.Fatalf("remove stop fragment = %d, stop list still has order 3", status)
	}

	// The driver's own draft: start via the tracker.
	driver := loginSessionAs(t, driverEmail, "pw-page")
	status, body, _ = postForm(t, driver, "/routes/"+routeID+"/start", url.Values{})
	if status != http.StatusOK || !strings.Contains(body, "active") {
		t.Fatalf("start fragment = %d", status)
	}

	// Tracker: manual time entry (display-zone wall clock), stop by stop.
	status, body, _ = postForm(t, driver, "/routes/"+routeID+"/stops/1/arrive",
		url.Values{"at": {"2026-07-18T08:00"}})
	if status != http.StatusOK || !strings.Contains(body, "08:00") {
		t.Fatalf("manual arrival stop 1 = %d", status)
	}
	status, body, _ = postForm(t, driver, "/routes/"+routeID+"/stops/1/depart",
		url.Values{"at": {"2026-07-18T08:05"}})
	if status != http.StatusOK || !strings.Contains(body, "08:00") {
		t.Fatalf("manual departure stop 1 = %d", status)
	}
	// No stopwatch on stop 1 (RN01): the rendered stop 1 row has no
	// arrive/depart buttons left.
	if strings.Contains(body, "/stops/1/arrive") {
		t.Error("stop 1 shows recording controls — RN01 violated in the UI")
	}
	// (This route has two stops: order 1 departure + order 2.)
	status, body, _ = postForm(t, driver, "/routes/"+routeID+"/stops/2/arrive",
		url.Values{"at": {"2026-07-18T09:00"}})
	if status != http.StatusOK {
		t.Fatalf("manual arrival stop 2 = %d", status)
	}
	status, body, _ = postForm(t, driver, "/routes/"+routeID+"/stops/2/depart",
		url.Values{"at": {"2026-07-18T09:15"}})
	if status != http.StatusOK || !strings.Contains(body, "15") {
		t.Fatalf("manual departure stop 2 = %d (want stop minutes shown)", status)
	}

	// Completed: distance + close; the summary shows the total.
	if !strings.Contains(body, "Close route") {
		t.Fatal("completed state does not show the close control")
	}
	status, body, _ = postForm(t, driver, "/routes/"+routeID+"/distance", url.Values{"distance_km": {"12.5"}})
	if status != http.StatusOK {
		t.Fatalf("distance fragment = %d", status)
	}
	status, body, _ = postForm(t, driver, "/routes/"+routeID+"/close", url.Values{})
	if status != http.StatusOK || !strings.Contains(body, "15") || !strings.Contains(body, "closed") {
		t.Fatalf("close fragment = %d", status)
	}

	// The builder's RN05 invalid state re-renders with a link to the
	// existing route.
	status, body, _ = postForm(t, admin, "/routes", url.Values{
		"driver_user_id": {driverID},
		"route_date":     {"2026-07-18"},
		"location_ids":    locs,
	})
	if status != http.StatusOK || !strings.Contains(body, "already has a route") ||
		!strings.Contains(body, "/routes/"+routeID) {
		t.Fatalf("conflict state = %d", status)
	}
}

func TestRouteToday(t *testing.T) {
	admin := loginSession(t, adminEmail)
	out, err := jsonCall(t, admin, "POST", "/api/drivers", fmt.Sprintf(
		`{"name": "T8 Today Driver", "email": "t8-today-%s@test.dev", "password": "pw-today", "phone": "0"}`, uuid.NewString()[:8]))
	must(err)
	driverID := out["driver"].(map[string]any)["id"].(string)
	driverEmail := out["driver"].(map[string]any)["email"].(string)

	locs := make([]string, 0, 2)
	for i := 0; i < 2; i++ {
		out, err = jsonCall(t, admin, "POST", "/api/locations",
			fmt.Sprintf(`{"label": "T8 Today P%d", "address": "Rua T8T, %d"}`, i, i))
		must(err)
		locs = append(locs, out["location"].(map[string]any)["id"].(string))
	}

	today := time.Now().Format("2006-01-02")
	out, err = jsonCall(t, admin, "POST", "/api/routes", fmt.Sprintf(
		`{"driver_user_id": %q, "route_date": %q, "location_ids": ["%s", "%s"]}`, driverID, today, locs[0], locs[1]))
	must(err)
	routeID := out["route"].(map[string]any)["id"].(string)

	// The driver's today page lands on their route.
	driver := loginSessionAs(t, driverEmail, "pw-today")
	status, _, h := do(t, driver, "GET", "/routes/today", "", "")
	if status != http.StatusSeeOther || h.Get("Location") != "/routes/"+routeID {
		t.Errorf("driver today = %d %q, want 303 %s", status, h.Get("Location"), "/routes/"+routeID)
	}

	// Manager/admin today is not a tracker concern: home for now.
	status, _, h = do(t, admin, "GET", "/routes/today", "", "")
	if status != http.StatusSeeOther || h.Get("Location") != "/" {
		t.Errorf("admin today = %d %q, want 303 /", status, h.Get("Location"))
	}
}
