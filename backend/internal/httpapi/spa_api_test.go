package httpapi_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// JSON transports the SPA relies on beyond the T7–T9 mirrors:
// GET /api/auth/me and PATCH …/stops/{order}/times (UpdateStopTimes).

func TestAPIMe(t *testing.T) {
	// Anonymous: 401 with the JSON error body.
	status, body, _ := do(t, newClient(), "GET", "/api/auth/me", "", "")
	if status != http.StatusUnauthorized || !strings.Contains(body, `"error"`) {
		t.Fatalf("anonymous me = %d %s, want 401 JSON", status, body)
	}

	admin := loginSession(t, adminEmail)
	out, err := jsonCall(t, admin, "GET", "/api/auth/me", "")
	if err != nil {
		t.Fatal(err)
	}
	user, _ := out["user"].(map[string]any)
	if user["email"] != adminEmail || user["role"] != "admin" || user["active"] != true {
		t.Errorf("me user = %v", user)
	}
	if _, leaked := user["password_hash"]; leaked {
		t.Error("me leaks the password hash")
	}
	if out["expires_at"] == nil || out["expires_at"] == "" {
		t.Errorf("me expires_at missing: %v", out)
	}

	// A session outlives deactivation; /me re-reads and answers 401.
	email := "me-" + uuid.NewString()[:8] + "@test.dev"
	out, err = jsonCall(t, admin, "POST", "/api/drivers", fmt.Sprintf(
		`{"name": "Me Driver", "email": %q, "password": "pw-me-123", "phone": "0"}`, email))
	if err != nil {
		t.Fatal(err)
	}
	driverID := out["driver"].(map[string]any)["id"].(string)
	driver := loginSessionAs(t, email, "pw-me-123")
	out, err = jsonCall(t, driver, "GET", "/api/auth/me", "")
	if err != nil || out["user"].(map[string]any)["role"] != "driver" {
		t.Fatalf("driver me = %v, %v", out, err)
	}
	if _, err := jsonCall(t, admin, "PATCH", "/api/drivers/"+driverID, `{"active": false}`); err != nil {
		t.Fatal(err)
	}
	status, _, _ = do(t, driver, "GET", "/api/auth/me", "", "")
	if status != http.StatusUnauthorized {
		t.Errorf("deactivated me = %d, want 401", status)
	}
}

func TestAPICorrectTimes(t *testing.T) {
	admin := loginSession(t, adminEmail)
	email := "times-" + uuid.NewString()[:8] + "@test.dev"
	out, err := jsonCall(t, admin, "POST", "/api/drivers", fmt.Sprintf(
		`{"name": "Times Driver", "email": %q, "password": "pw-times", "phone": "0"}`, email))
	if err != nil {
		t.Fatal(err)
	}
	driverID := out["driver"].(map[string]any)["id"].(string)
	locations := make([]string, 0, 2)
	for i := 0; i < 2; i++ {
		out, err = jsonCall(t, admin, "POST", "/api/locations",
			fmt.Sprintf(`{"label": "Times P%d", "address": "Rua Times, %d"}`, i, i))
		if err != nil {
			t.Fatal(err)
		}
		locations = append(locations, out["location"].(map[string]any)["id"].(string))
	}
	out, err = jsonCall(t, admin, "POST", "/api/routes", fmt.Sprintf(
		`{"driver_user_id": %q, "route_date": "2026-09-02", "location_ids": ["%s", "%s"]}`,
		driverID, locations[0], locations[1]))
	if err != nil {
		t.Fatal(err)
	}
	routeID := out["route"].(map[string]any)["id"].(string)
	if got := out["route"].(map[string]any)["route_date"]; got != "2026-09-02" {
		t.Errorf("route_date = %v, want plain 2026-09-02", got)
	}
	if _, err := jsonCall(t, admin, "POST", "/api/routes/"+routeID+"/start", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := jsonCall(t, admin, "POST", "/api/routes/"+routeID+"/stops/2/arrive",
		`{"at": "2026-09-02T09:00:00-03:00"}`); err != nil {
		t.Fatal(err)
	}
	path := "/api/routes/" + routeID + "/stops/2/times"

	// Manager: 200 with the stop, and an update_times audit row.
	manager := loginSession(t, "manager@stoptime.dev")
	out, err = jsonCall(t, manager, "PATCH", path,
		`{"arrival_at": "2026-09-02T09:05:00-03:00", "departure_at": "2026-09-02T09:20:00-03:00"}`)
	if err != nil {
		t.Fatal(err)
	}
	stop, _ := out["stop"].(map[string]any)
	if stop == nil || !sameInstant(stop["arrival_at"], "2026-09-02T12:05:00Z") ||
		!sameInstant(stop["departure_at"], "2026-09-02T12:20:00Z") {
		t.Errorf("corrected stop = %v", stop)
	}
	out, err = jsonCall(t, admin, "GET", "/api/audit?entity=route_stop", "")
	if err != nil {
		t.Fatal(err)
	}
	raw := jsonString(out)
	if !strings.Contains(raw, "update_times") || !strings.Contains(raw, "2026-09-02T09:20:00-03:00") {
		t.Errorf("audit missing the JSON correction: %s", raw[:min(len(raw), 400)])
	}

	// Driver (even the route's own): 403 — corrections are manager/admin.
	driver := loginSessionAs(t, email, "pw-times")
	status, _, _ := do(t, driver, "PATCH", path, "application/json",
		`{"departure_at": "2026-09-02T09:30:00-03:00"}`)
	if status != http.StatusForbidden {
		t.Errorf("driver correction = %d, want 403", status)
	}

	// RN02: departure before arrival is 422.
	status, body, _ := do(t, manager, "PATCH", path, "application/json",
		`{"departure_at": "2026-09-02T08:00:00-03:00"}`)
	if status != http.StatusUnprocessableEntity || !strings.Contains(body, `"error"`) {
		t.Errorf("RN02 correction = %d %s, want 422", status, body)
	}

	// Malformed timestamp: 400.
	status, _, _ = do(t, manager, "PATCH", path, "application/json", `{"arrival_at": "09:00"}`)
	if status != http.StatusBadRequest {
		t.Errorf("malformed correction = %d, want 400", status)
	}
}

// sameInstant compares a JSON timestamp to an RFC 3339 instant,
// offset-insensitively.
func sameInstant(v any, want string) bool {
	got, ok := v.(string)
	if !ok {
		return false
	}
	g, err1 := time.Parse(time.RFC3339, got)
	w, err2 := time.Parse(time.RFC3339, want)
	return err1 == nil && err2 == nil && g.Equal(w)
}
