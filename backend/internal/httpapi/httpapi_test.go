package httpapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"stoptime/internal/app"
	"stoptime/internal/httpapi"
	"stoptime/internal/store"
)

// HTTP shell integration tests against stoptime_test (the app suite
// runs first and leaves the golden users in place; assertions are
// presence-based, never exact counts).

const (
	adminEmail  = "admin@stoptime.dev"
	driverEmail = "driver-a@stoptime.dev"
	demoPass    = "stoptime-dev"
	testKey     = "test session key — 32+ bytes — stoptime"
)

var tsURL string

func TestMain(m *testing.M) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		fmt.Fprintln(os.Stderr, "TEST_DATABASE_URL is not set — run scripts/testdb.sh and eval \"$(scripts/db-up.sh)\" first")
		os.Exit(1)
	}
	st, err := store.Open(context.Background(), dbURL)
	must(err)
	svc := app.New(st, []byte(testKey))
	server, err := httpapi.New(svc, []byte(testKey))
	must(err)
	ts := httptest.NewServer(server.Router())
	tsURL = ts.URL
	defer ts.Close()

	os.Exit(m.Run())
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "test setup:", err)
		os.Exit(1)
	}
}

func newClient() *http.Client {
	jar, err := cookiejar.New(nil)
	must(err)
	return &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse // follow nothing; assert redirects
	}}
}

func do(t *testing.T, client *http.Client, method, path, contentType, body string) (int, string, http.Header) {
	t.Helper()
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, tsURL+path, rd)
	must(err)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	must(err)
	return resp.StatusCode, string(b), resp.Header
}

func postForm(t *testing.T, client *http.Client, path string, form url.Values) (int, string, string) {
	t.Helper()
	status, body, h := do(t, client, "POST", path, "application/x-www-form-urlencoded", form.Encode())
	return status, body, h.Get("Location")
}

// loginSession returns a client holding a session for a golden demo
// user (the login path under test is the JSON mirror).
func loginSession(t *testing.T, email string) *http.Client {
	t.Helper()
	client := newClient()
	body := fmt.Sprintf(`{"email": %q, "password": %q}`, email, demoPass)
	status, respBody, _ := do(t, client, "POST", "/api/auth/login", "application/json", body)
	if status != http.StatusOK {
		t.Fatalf("login %s: status %d, body %s", email, status, respBody)
	}
	return client
}

// --- pages --------------------------------------------------------------

func TestLoginPageAndSessionFlow(t *testing.T) {
	client := newClient()

	status, body, _ := do(t, client, "GET", "/login", "", "")
	if status != http.StatusOK || !strings.Contains(body, "Sign in") || !strings.Contains(body, "<form") {
		t.Fatalf("login page: status %d, has form: %v", status, strings.Contains(body, "<form"))
	}

	// Wrong password: the invalid state, inline.
	status, _, _ = postForm(t, client, "/login", url.Values{
		"email": {adminEmail}, "password": {"wrong"},
	})
	if status != http.StatusOK {
		t.Errorf("bad login = %d, want 200 (invalid state)", status)
	}

	// Good login: redirect + session cookie.
	status, _, location := postForm(t, client, "/login", url.Values{
		"email": {adminEmail}, "password": {demoPass},
	})
	if status != http.StatusSeeOther || location != "/" {
		t.Errorf("login = %d %q, want 303 /", status, location)
	}

	status, body, _ = do(t, client, "GET", "/", "", "")
	if status != http.StatusOK || !strings.Contains(body, "StopTime") {
		t.Errorf("home = %d, want 200 with app name", status)
	}

	// Logout clears the session; protected pages bounce to the form.
	status, _, _ = postForm(t, client, "/logout", url.Values{})
	if status != http.StatusSeeOther {
		t.Errorf("logout = %d, want 303", status)
	}
	status, _, _ = do(t, client, "GET", "/drivers", "", "")
	if status != http.StatusSeeOther {
		t.Errorf("after logout /drivers = %d, want 303 to login", status)
	}
}

func TestUnauthenticatedBounce(t *testing.T) {
	status, _, h := do(t, newClient(), "GET", "/drivers", "", "")
	if status != http.StatusSeeOther || h.Get("Location") != "/login" {
		t.Errorf("anonymous /drivers = %d %q, want 303 /login", status, h.Get("Location"))
	}
}

func TestDirectoryPages(t *testing.T) {
	admin := loginSession(t, adminEmail)

	// Drivers list shows the golden driver; the create form state works.
	status, body, _ := do(t, admin, "GET", "/drivers", "", "")
	if status != http.StatusOK || !strings.Contains(body, "Marcos Motorista") {
		t.Errorf("drivers page = %d, want the golden driver listed", status)
	}

	name := "HTTP Driver " + uuid.NewString()[:8]
	status, _, _ = postForm(t, admin, "/drivers", url.Values{
		"name": {name}, "email": {"http-" + uuid.NewString()[:8] + "@test.dev"},
		"password": {"pw"}, "phone": {"0"}, "km_per_l": {"12.50"},
	})
	if status != http.StatusSeeOther {
		t.Errorf("create driver = %d, want 303", status)
	}

	// The saved state: list + flash.
	status, body, _ = do(t, admin, "GET", "/drivers", "", "")
	if !strings.Contains(body, name) || !strings.Contains(body, "Driver created.") {
		t.Errorf("drivers page after create: name=%v flash=%v",
			strings.Contains(body, name), strings.Contains(body, "Driver created."))
	}

	// The invalid state: missing name, inline field error, no redirect.
	status, body, _ = postForm(t, admin, "/drivers", url.Values{
		"email": {"incomplete-" + uuid.NewString()[:8] + "@test.dev"},
		"password": {"pw"}, "phone": {"0"},
	})
	if status != http.StatusOK || !strings.Contains(body, "name: required") {
		t.Errorf("invalid create = %d, want 200 with inline error", status)
	}

	// Edit flow: find the created driver's id via the JSON mirror.
	_, list, _ := do(t, admin, "GET", "/api/drivers", "", "")
	var listing struct {
		Drivers []struct {
			ID    uuid.UUID `json:"id"`
			Name  string    `json:"name"`
			KmPerL *string  `json:"km_per_l"`
		} `json:"drivers"`
	}
	must(json.Unmarshal([]byte(list), &listing))
	var id string
	for _, d := range listing.Drivers {
		if d.Name == name {
			id = d.ID.String()
		}
	}
	if id == "" {
		t.Fatal("created driver not in list")
	}

	status, body, _ = do(t, admin, "GET", "/drivers/"+id+"/edit", "", "")
	if status != http.StatusOK || !strings.Contains(body, name) {
		t.Errorf("edit page = %d, want the driver form", status)
	}
	status, _, _ = postForm(t, admin, "/drivers/"+id+"/edit", url.Values{
		"name": {"Renamed " + name},
	})
	if status != http.StatusSeeOther {
		t.Errorf("edit submit = %d, want 303", status)
	}
	status, body, _ = do(t, admin, "GET", "/drivers", "", "")
	if !strings.Contains(body, "Renamed "+name) {
		t.Error("renamed driver missing from list")
	}

	// Locations page: create + list + flash.
	label := "HTTP Loc " + uuid.NewString()[:8]
	status, _, _ = postForm(t, admin, "/locations", url.Values{
		"label": {label}, "address": {"Rua HTTP, 1"},
	})
	if status != http.StatusSeeOther {
		t.Errorf("create location = %d, want 303", status)
	}
	status, body, _ = do(t, admin, "GET", "/locations", "", "")
	if !strings.Contains(body, label) || !strings.Contains(body, "Location created.") {
		t.Errorf("locations page after create: label=%v", strings.Contains(body, label))
	}

	// Managers page (admin only).
	status, body, _ = do(t, admin, "GET", "/managers", "", "")
	if status != http.StatusOK || !strings.Contains(body, "Managers") {
		t.Errorf("managers page = %d", status)
	}
}

func TestDriverSeesNoDirectories(t *testing.T) {
	driver := loginSession(t, driverEmail)
	// The service denies drivers; the page renders 403.
	status, _, _ := do(t, driver, "GET", "/drivers", "", "")
	if status != http.StatusForbidden {
		t.Errorf("driver /drivers = %d, want 403", status)
	}
	status, _, _ = do(t, driver, "GET", "/managers", "", "")
	if status != http.StatusForbidden {
		t.Errorf("driver /managers = %d, want 403", status)
	}
}

// --- JSON mirrors -------------------------------------------------------

func TestAPIAuth(t *testing.T) {
	client := newClient()

	// Anonymous: 401, JSON body.
	status, _, _ := do(t, client, "GET", "/api/drivers", "", "")
	if status != http.StatusUnauthorized {
		t.Errorf("anonymous api = %d, want 401", status)
	}

	// Wrong password: 401 without leaking which part failed.
	status, body, _ := do(t, client, "POST", "/api/auth/login", "application/json",
		fmt.Sprintf(`{"email": %q, "password": "wrong"}`, adminEmail))
	if status != http.StatusUnauthorized || strings.Contains(body, "password") {
		t.Errorf("bad login = %d %q, want 401 without field detail", status, body)
	}

	// Good login: user JSON + cookie; logout clears with 204.
	admin := loginSession(t, adminEmail)
	status, _, _ = do(t, admin, "POST", "/api/auth/logout", "application/json", "")
	if status != http.StatusNoContent {
		t.Errorf("api logout = %d, want 204", status)
	}
}

func TestDriversAPIFlow(t *testing.T) {
	admin := loginSession(t, adminEmail)

	email := "api-" + uuid.NewString()[:8] + "@test.dev"
	status, body, _ := do(t, admin, "POST", "/api/drivers", "application/json",
		fmt.Sprintf(`{"name": "API Driver", "email": %q, "password": "pw", "phone": "0", "vehicle_name": "Kombi"}`, email))
	if status != http.StatusCreated || !strings.Contains(body, `"vehicle_name":"Kombi"`) {
		t.Fatalf("api create = %d %s", status, body)
	}
	if strings.Contains(body, "password") {
		t.Error("API leaked a password field")
	}

	// Duplicate email: the sentinel maps to 409.
	status, _, _ = do(t, admin, "POST", "/api/drivers", "application/json",
		fmt.Sprintf(`{"name": "Dup", "email": %q, "password": "pw", "phone": "0"}`, email))
	if status != http.StatusConflict {
		t.Errorf("duplicate = %d, want 409", status)
	}

	// List contains the created driver.
	status, body, _ = do(t, admin, "GET", "/api/drivers", "", "")
	if status != http.StatusOK || !strings.Contains(body, email) {
		t.Errorf("api list = %d, contains created: %v", status, strings.Contains(body, email))
	}

	// PATCH updates partially.
	var created struct {
		Driver struct {
			ID uuid.UUID `json:"id"`
		} `json:"driver"`
	}
	must(json.Unmarshal([]byte(body), &created))
	// find the created driver's id
	_, list, _ := do(t, admin, "GET", "/api/drivers", "", "")
	var rows struct {
		Drivers []struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"drivers"`
	}
	must(json.Unmarshal([]byte(list), &rows))
	var id string
	for _, d := range rows.Drivers {
		if d.Email == email {
			id = d.ID
		}
	}
	status, body, _ = do(t, admin, "PATCH", "/api/drivers/"+id, "application/json", `{"phone": "31999"}`)
	if status != http.StatusOK || !strings.Contains(body, `"phone":"31999"`) {
		t.Errorf("api patch = %d %s", status, body)
	}

	// Role enforcement through the API: driver → 403.
	driver := loginSession(t, driverEmail)
	status, _, _ = do(t, driver, "GET", "/api/drivers", "", "")
	if status != http.StatusForbidden {
		t.Errorf("driver api list = %d, want 403", status)
	}
}

func TestManagersAndLocationsAPI(t *testing.T) {
	admin := loginSession(t, adminEmail)

	status, body, _ := do(t, admin, "POST", "/api/managers", "application/json",
		fmt.Sprintf(`{"name": "API Manager", "email": "apim-%s@test.dev", "password": "pw", "phone": "0"}`, uuid.NewString()[:8]))
	if status != http.StatusCreated {
		t.Fatalf("api manager create = %d %s", status, body)
	}

	status, body, _ = do(t, admin, "POST", "/api/locations", "application/json",
		fmt.Sprintf(`{"label": "API Loc", "address": "Rua API, %d"}`, 1))
	if status != http.StatusCreated {
		t.Fatalf("api location create = %d %s", status, body)
	}

	// Manager role cannot create managers (admin-only cell).
	manager := loginSession(t, "manager@stoptime.dev")
	status, _, _ = do(t, manager, "POST", "/api/managers", "application/json",
		`{"name": "X", "email": "x-mgr@test.dev", "password": "pw", "phone": "0"}`)
	if status != http.StatusForbidden {
		t.Errorf("manager api manager-create = %d, want 403", status)
	}
}

func TestStaticAndHealth(t *testing.T) {
	for _, asset := range []string{"/static/pico.min.css", "/static/htmx.min.js", "/static/chart.umd.js"} {
		status, _, h := do(t, newClient(), "GET", asset, "", "")
		if status != http.StatusOK {
			t.Errorf("%s = %d, want 200", asset, status)
		}
		if ct := h.Get("Content-Type"); asset == "/static/pico.min.css" && ct != "text/css; charset=utf-8" {
			t.Errorf("css content-type = %q", ct)
		}
	}
	status, body, _ := do(t, newClient(), "GET", "/healthz", "", "")
	if status != http.StatusOK || body != "ok" {
		t.Errorf("healthz = %d %q", status, body)
	}
}
