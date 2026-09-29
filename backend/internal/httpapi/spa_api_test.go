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

// RNF06 over JSON: manager lists see the masked document with
// document_masked; POST /api/drivers/{id}/anonymize is admin-only.
func TestAPIDriverPrivacy(t *testing.T) {
	admin := loginSession(t, adminEmail)
	manager := loginSession(t, "manager@stoptime.dev")
	email := "privacy-" + uuid.NewString()[:8] + "@test.dev"
	out, err := jsonCall(t, admin, "POST", "/api/drivers", fmt.Sprintf(
		`{"name": "Privacy Driver", "email": %q, "password": "pw-12345", "phone": "31 97777-0000", "document": "111.222.333-44"}`, email))
	if err != nil {
		t.Fatal(err)
	}
	created := out["driver"].(map[string]any)
	id := created["id"].(string)
	if created["document"] != "111.222.333-44" || created["document_masked"] != false {
		t.Errorf("admin create = %v", created)
	}

	_, body, _ := do(t, manager, "GET", "/api/drivers", "", "")
	if !strings.Contains(body, `"document":"***.***.***-44","document_masked":true`) || strings.Contains(body, "111.222.333-44") {
		t.Errorf("manager list does not mask the document")
	}

	// Manager: 403; unknown id: 404; bad id: 400.
	status, _, _ := do(t, manager, "POST", "/api/drivers/"+id+"/anonymize", "", "")
	if status != http.StatusForbidden {
		t.Errorf("manager anonymize = %d, want 403", status)
	}
	status, _, _ = do(t, admin, "POST", "/api/drivers/"+uuid.NewString()+"/anonymize", "", "")
	if status != http.StatusNotFound {
		t.Errorf("unknown anonymize = %d, want 404", status)
	}
	status, _, _ = do(t, admin, "POST", "/api/drivers/junk/anonymize", "", "")
	if status != http.StatusBadRequest {
		t.Errorf("bad id anonymize = %d, want 400", status)
	}

	for i := 0; i < 2; i++ { // idempotent: the second call answers the same 200
		out, err = jsonCall(t, admin, "POST", "/api/drivers/"+id+"/anonymize", "")
		if err != nil {
			t.Fatalf("anonymize #%d: %v", i+1, err)
		}
		d := out["driver"].(map[string]any)
		if d["name"] != "Motorista removido "+id[:8] || d["email"] != "removido-"+id+"@anonimo.invalid" ||
			d["phone"] != "" || d["active"] != false || d["document"] != nil {
			t.Errorf("anonymized #%d = %v", i+1, d)
		}
	}
	status, _, _ = do(t, newClient(), "POST", "/api/auth/login", "application/json",
		fmt.Sprintf(`{"email": %q, "password": "pw-12345"}`, email))
	if status != http.StatusUnauthorized {
		t.Errorf("login after anonymize = %d, want 401", status)
	}
}

// A deactivated or anonymized user's cookie stops working on the next
// request, for both transports, and the cookie is cleared.
func TestSessionRevokedOnDeactivation(t *testing.T) {
	admin := loginSession(t, adminEmail)
	mk := func(tag string) (string, string, *http.Client) {
		email := tag + "-" + uuid.NewString()[:8] + "@test.dev"
		out, err := jsonCall(t, admin, "POST", "/api/drivers", fmt.Sprintf(
			`{"name": "Revoke %s", "email": %q, "password": "pw-12345", "phone": "0"}`, tag, email))
		if err != nil {
			t.Fatal(err)
		}
		c := loginSessionAs(t, email, "pw-12345")
		if status, _, _ := do(t, c, "GET", "/api/auth/me", "", ""); status != http.StatusOK {
			t.Fatalf("%s me before = %d", tag, status)
		}
		return out["driver"].(map[string]any)["id"].(string), email, c
	}

	id, _, driver := mk("deact")
	if _, err := jsonCall(t, admin, "PATCH", "/api/drivers/"+id, `{"active": false}`); err != nil {
		t.Fatal(err)
	}
	status, body, h := do(t, driver, "GET", "/api/routes", "", "")
	if status != http.StatusUnauthorized || !strings.Contains(body, `"error"`) {
		t.Errorf("deactivated API call = %d %s, want 401 JSON", status, body)
	}
	if !strings.Contains(h.Get("Set-Cookie"), "st_session=;") {
		t.Errorf("deactivated call does not clear the cookie: %q", h.Get("Set-Cookie"))
	}

	id, _, driver = mk("anon")
	if _, err := jsonCall(t, admin, "POST", "/api/drivers/"+id+"/anonymize", ""); err != nil {
		t.Fatal(err)
	}
	status, _, h = do(t, driver, "GET", "/routes/today", "", "")
	if status != http.StatusSeeOther || h.Get("Location") != "/login" {
		t.Errorf("anonymized page call = %d %q, want 303 /login", status, h.Get("Location"))
	}
	status, _, _ = do(t, driver, "GET", "/api/auth/me", "", "")
	if status != http.StatusUnauthorized {
		t.Errorf("anonymized API call = %d, want 401", status)
	}
}

func TestAPIManagerUpdateAndAnonymize(t *testing.T) {
	admin := loginSession(t, adminEmail)
	email := "apim-edit-" + uuid.NewString()[:8] + "@test.dev"
	out, err := jsonCall(t, admin, "POST", "/api/managers", fmt.Sprintf(
		`{"name": "API Mgr", "email": %q, "password": "pw-12345", "phone": "31 3"}`, email))
	if err != nil {
		t.Fatal(err)
	}
	id := out["manager"].(map[string]any)["id"].(string)
	mgr := loginSessionAs(t, email, "pw-12345")

	status, _, _ := do(t, mgr, "PATCH", "/api/managers/"+id, "application/json", `{"name": "Self"}`)
	if status != http.StatusForbidden {
		t.Errorf("manager PATCH manager = %d, want 403", status)
	}
	out, err = jsonCall(t, admin, "PATCH", "/api/managers/"+id, `{"name": "API Mgr 2", "phone": "31 4"}`)
	if err != nil {
		t.Fatal(err)
	}
	m := out["manager"].(map[string]any)
	if m["name"] != "API Mgr 2" || m["phone"] != "31 4" || m["email"] != email || m["active"] != true {
		t.Errorf("PATCH manager = %v", m)
	}
	status, _, _ = do(t, admin, "PATCH", "/api/managers/"+uuid.NewString(), "application/json", `{"name": "X"}`)
	if status != http.StatusNotFound {
		t.Errorf("PATCH unknown manager = %d, want 404", status)
	}

	out, err = jsonCall(t, admin, "POST", "/api/managers/"+id+"/anonymize", "")
	if err != nil {
		t.Fatal(err)
	}
	m = out["manager"].(map[string]any)
	if m["name"] != "Gestor removido "+id[:8] || m["active"] != false || m["phone"] != "" {
		t.Errorf("anonymized manager = %v", m)
	}
	if status, _, _ := do(t, mgr, "GET", "/api/locations", "", ""); status != http.StatusUnauthorized {
		t.Errorf("anonymized manager session = %d, want 401", status)
	}
}

// RNF05 over JSON: a location edit answers the location and leaves a
// closed route's detail and CSV on the original address.
func TestAPILocationEditKeepsHistory(t *testing.T) {
	admin := loginSession(t, adminEmail)
	email := "loc-hist-" + uuid.NewString()[:8] + "@test.dev"
	out, err := jsonCall(t, admin, "POST", "/api/drivers", fmt.Sprintf(
		`{"name": "Loc Hist", "email": %q, "password": "pw-12345", "phone": "0"}`, email))
	if err != nil {
		t.Fatal(err)
	}
	driverID := out["driver"].(map[string]any)["id"].(string)
	ids := []string{}
	for i := 0; i < 2; i++ {
		out, err = jsonCall(t, admin, "POST", "/api/locations", fmt.Sprintf(`{"label": "Hist %d", "address": "Rua Velha, %d"}`, i, i))
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, out["location"].(map[string]any)["id"].(string))
	}
	out, err = jsonCall(t, admin, "POST", "/api/routes", fmt.Sprintf(
		`{"driver_user_id": %q, "route_date": "2025-01-13", "location_ids": [%q, %q]}`, driverID, ids[0], ids[1]))
	if err != nil {
		t.Fatal(err)
	}
	routeID := out["route"].(map[string]any)["id"].(string)
	for _, step := range []string{"start", "close"} {
		if _, err := jsonCall(t, admin, "POST", "/api/routes/"+routeID+"/"+step, ""); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := jsonCall(t, admin, "PATCH", "/api/locations/"+ids[1], `{"address": "Rua Nova, 1"}`); err != nil {
		t.Fatal(err)
	}
	view := routeView(t, admin, routeID)
	if addr := stopsOf(view)[1].(map[string]any)["address"]; addr != "Rua Velha, 1" {
		t.Errorf("route detail address after edit = %v, want Rua Velha, 1", addr)
	}
	_, csvBody, _ := do(t, admin, "GET", "/api/export?from=2025-01-13&to=2025-01-13&driver_user_id="+driverID, "", "")
	if !strings.Contains(csvBody, "Rua Velha, 1") || strings.Contains(csvBody, "Rua Nova, 1") {
		t.Errorf("CSV after edit does not keep the original address")
	}
	_, auditBody, _ := do(t, admin, "GET", "/api/audit?entity=location", "", "")
	if !strings.Contains(auditBody, "update_location") || !strings.Contains(auditBody, "Rua Nova, 1") {
		t.Errorf("location edit not audited")
	}
}

// Team (responsible manager) over JSON: driver fields, PATCH null,
// team_size, and the manager_user_id filter on reads.
func TestAPIManagerTeam(t *testing.T) {
	admin := loginSession(t, adminEmail)
	gustavo := "aa000000-0000-4000-8000-000000000002"

	_, body, _ := do(t, admin, "GET", "/api/drivers", "", "")
	if !strings.Contains(body, `"manager_user_id":"`+gustavo+`","manager_name":"Gustavo Gerente"`) {
		t.Errorf("driver list lacks the golden responsible manager")
	}
	_, body, _ = do(t, admin, "GET", "/api/managers", "", "")
	if !strings.Contains(body, `"team_size":`) {
		t.Errorf("manager list lacks team_size: %s", body)
	}

	email := "team-api-" + uuid.NewString()[:8] + "@test.dev"
	out, err := jsonCall(t, admin, "POST", "/api/drivers", fmt.Sprintf(
		`{"name": "Team API", "email": %q, "password": "pw-12345", "phone": "0", "manager_user_id": %q}`, email, gustavo))
	if err != nil {
		t.Fatal(err)
	}
	d := out["driver"].(map[string]any)
	id := d["id"].(string)
	if d["manager_user_id"] != gustavo || d["manager_name"] != "Gustavo Gerente" {
		t.Errorf("created driver = %v", d)
	}
	status, body, _ := do(t, admin, "PATCH", "/api/drivers/"+id, "application/json", `{"manager_user_id": "`+id+`"}`)
	if status != http.StatusUnprocessableEntity || !strings.Contains(body, `"field":"manager_user_id"`) {
		t.Errorf("driver as manager = %d %s", status, body)
	}
	out, err = jsonCall(t, admin, "PATCH", "/api/drivers/"+id, `{"phone": "1"}`)
	if err != nil || out["driver"].(map[string]any)["manager_user_id"] != gustavo {
		t.Errorf("absent manager_user_id must keep: %v %v", out, err)
	}
	out, err = jsonCall(t, admin, "PATCH", "/api/drivers/"+id, `{"manager_user_id": null}`)
	if err != nil || out["driver"].(map[string]any)["manager_user_id"] != nil {
		t.Errorf("null manager_user_id must clear: %v %v", out, err)
	}

	// Golden drivers are all Gustavo's: the team window equals the whole.
	q := "from=2026-06-01&to=2026-06-30&manager_user_id=" + gustavo
	out, err = jsonCall(t, admin, "GET", "/api/dashboard/period?"+q, "")
	if err != nil || out["total_stopped_minutes"] != float64(161) {
		t.Errorf("team period = %v %v", out, err)
	}
	other := "aa000000-0000-4000-8000-000000000001" // the admin: nobody's manager
	out, err = jsonCall(t, admin, "GET", "/api/routes?from=2026-06-01&to=2026-06-30&manager_user_id="+other, "")
	if err != nil || len(out["routes"].([]any)) != 0 {
		t.Errorf("empty team routes = %v %v", out, err)
	}
	for _, path := range []string{"/api/dashboard/day", "/api/dashboard/month", "/api/dashboard/period", "/api/routes", "/api/export"} {
		status, body, _ := do(t, admin, "GET", path+"?from=2026-06-01&to=2026-06-30&manager_user_id=junk", "", "")
		if status != http.StatusUnprocessableEntity || !strings.Contains(body, `"field":"manager_user_id"`) {
			t.Errorf("%s bad manager id = %d %s", path, status, body)
		}
	}
	_, csvBody, _ := do(t, admin, "GET", "/api/export?"+q, "", "")
	if !strings.Contains(csvBody, "Marcos Motorista") {
		t.Errorf("team export lacks the golden rows")
	}
}

// RN06 sequence and below_min over JSON.
func TestAPIStopSequence(t *testing.T) {
	admin := loginSession(t, adminEmail)
	email := "seq-api-" + uuid.NewString()[:8] + "@test.dev"
	out, err := jsonCall(t, admin, "POST", "/api/drivers", fmt.Sprintf(
		`{"name": "Seq API", "email": %q, "password": "pw-12345", "phone": "0"}`, email))
	if err != nil {
		t.Fatal(err)
	}
	driverID := out["driver"].(map[string]any)["id"].(string)
	ids := []string{}
	for i := 0; i < 3; i++ {
		out, err = jsonCall(t, admin, "POST", "/api/locations", fmt.Sprintf(`{"label": "Seq %d", "address": "Rua Seq, %d"}`, i, i))
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, out["location"].(map[string]any)["id"].(string))
	}
	out, err = jsonCall(t, admin, "POST", "/api/routes", fmt.Sprintf(
		`{"driver_user_id": %q, "route_date": "2024-10-14", "location_ids": [%q, %q, %q]}`, driverID, ids[0], ids[1], ids[2]))
	if err != nil {
		t.Fatal(err)
	}
	routeID := out["route"].(map[string]any)["id"].(string)
	if _, err := jsonCall(t, admin, "POST", "/api/routes/"+routeID+"/start", ""); err != nil {
		t.Fatal(err)
	}
	driver := loginSessionAs(t, email, "pw-12345")
	status, body, _ := do(t, driver, "POST", "/api/routes/"+routeID+"/stops/3/arrive", "application/json", `{"at": "2024-10-14T09:00:00-03:00"}`)
	if status != http.StatusUnprocessableEntity || !strings.Contains(body, `"field":"arrival_at"`) {
		t.Errorf("out-of-order arrive = %d %s, want 422 arrival_at", status, body)
	}
	for _, step := range []struct{ path, at string }{
		{"stops/2/arrive", "2024-10-14T09:00:00-03:00"},
		{"stops/2/depart", "2024-10-14T09:10:00-03:00"},
		{"stops/3/arrive", "2024-10-14T09:30:00-03:00"},
		{"stops/3/depart", "2024-10-14T09:35:00-03:00"},
	} {
		if _, err := jsonCall(t, driver, "POST", "/api/routes/"+routeID+"/"+step.path, `{"at": "`+step.at+`"}`); err != nil {
			t.Fatalf("%s: %v", step.path, err)
		}
	}
	stops := stopsOf(routeView(t, admin, routeID))
	for i, s := range stops {
		st := s.(map[string]any)
		if _, ok := st["below_min"]; !ok || st["below_min"] != false || st["counted"] != (i > 0) {
			t.Errorf("stop %d counted/below_min = %v/%v", i+1, st["counted"], st["below_min"])
		}
	}
}

func TestAPIListManagersForManagers(t *testing.T) {
	status, body, _ := do(t, loginSession(t, "manager@stoptime.dev"), "GET", "/api/managers", "", "")
	if status != http.StatusOK || !strings.Contains(body, `"team_size"`) || !strings.Contains(body, "Gustavo Gerente") ||
		strings.Contains(body, `"email"`) || strings.Contains(body, `"phone"`) {
		t.Errorf("manager GET /api/managers = %d %s, want minimized rows", status, body)
	}
	_, body, _ = do(t, loginSession(t, adminEmail), "GET", "/api/managers", "", "")
	if !strings.Contains(body, `"email":"manager@stoptime.dev"`) {
		t.Errorf("admin list lacks emails: %s", body)
	}
	status, _, _ = do(t, loginSession(t, driverEmail), "GET", "/api/managers", "", "")
	if status != http.StatusForbidden {
		t.Errorf("driver GET /api/managers = %d, want 403", status)
	}
}

// The legacy /managers page follows the same minimization: managers see
// the list without email/phone and without the admin-only create form.
func TestManagersPageByRole(t *testing.T) {
	status, body, _ := do(t, loginSession(t, "manager@stoptime.dev"), "GET", "/managers", "", "")
	if status != http.StatusOK || !strings.Contains(body, "Gustavo Gerente") {
		t.Fatalf("manager /managers = %d", status)
	}
	if strings.Contains(body, `action="/managers"`) || strings.Contains(body, "manager@stoptime.dev") ||
		strings.Contains(body, `name="password"`) {
		t.Errorf("manager page shows admin-only controls or contact data")
	}
	_, body, _ = do(t, loginSession(t, adminEmail), "GET", "/managers", "", "")
	if !strings.Contains(body, `action="/managers"`) || !strings.Contains(body, "manager@stoptime.dev") {
		t.Errorf("admin page lacks the create form or the emails")
	}
}
