package app_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"stoptime/internal/app"
	"stoptime/internal/domain"
	"stoptime/internal/store"
)

// Integration tests run against stoptime_test (scripts/testdb.sh). They
// own their data: TestMain truncates and bootstraps the default
// parameters, so the suite never depends on the golden seed.

var (
	svc    *app.Services
	testDB *pgxpool.Pool
	// The golden seed's admin (Ana Administradora) doubles as the test
	// actor for parameter updates and synthetic-data provenance.
	admin = uuid.MustParse("aa000000-0000-4000-8000-000000000001")
)

func TestMain(m *testing.M) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		fmt.Fprintln(os.Stderr, "TEST_DATABASE_URL is not set — run scripts/testdb.sh and eval \"$(scripts/db-up.sh)\" first")
		os.Exit(1)
	}
	ctx := context.Background()
	var err error
	testDB, err = pgxpool.New(ctx, url)
	must(err)
	must(testDB.Ping(ctx))

	_, err = testDB.Exec(ctx, `
TRUNCATE app_user, driver_profile, location, route, route_stop, audit_log, parameter`)
	must(err)

	// The golden fixture (tp.md section 5) is the shared test data:
	// day/month/period 161, routes A/B/C = 75/41/45, five parameters at
	// defaults, and the demo users. The app tests create their own data
	// on other dates, so the suites do not collide. pgx takes one
	// statement per Exec, so strip -- comments and split on ';' (seed
	// files keep ';' only as a statement terminator).
	seed, err := os.ReadFile("../../../db/seed/golden.sql")
	must(err)
	var clean strings.Builder
	for _, line := range strings.Split(string(seed), "\n") {
		if i := strings.Index(line, "--"); i >= 0 {
			line = line[:i]
		}
		clean.WriteString(line)
		clean.WriteString("\n")
	}
	for _, stmt := range strings.Split(clean.String(), ";") {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		_, err := testDB.Exec(ctx, stmt)
		must(err)
	}

	st, err := store.Open(ctx, url)
	must(err)
	svc = app.New(st, []byte("test session key — 32+ bytes — stoptime"))

	code := m.Run()
	st.Close()
	testDB.Close()
	os.Exit(code)
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "test setup:", err)
		os.Exit(1)
	}
}

// --- fixtures -----------------------------------------------------------

func adminActor() app.Actor { return app.Actor{UserID: admin, Role: "admin"} }

var ctx = context.Background()

func createManager(t *testing.T, name string) store.Manager {
	t.Helper()
	u, err := svc.CreateManager(ctx, adminActor(), app.CreateManagerInput{
		Name: name, Email: name + "@test.dev", Password: "pw-" + name, Phone: "0",
	})
	if err != nil {
		t.Fatalf("CreateManager: %v", err)
	}
	return u
}

func actorOf(u store.User) app.Actor { return app.Actor{UserID: u.ID, Role: u.Role} }

func createDriver(t *testing.T, name string) store.Driver {
	t.Helper()
	d, err := svc.CreateDriver(ctx, adminActor(), app.CreateDriverInput{
		Name: name, Email: name + "@test.dev", Password: "pw-" + name, Phone: "0",
	})
	if err != nil {
		t.Fatalf("CreateDriver: %v", err)
	}
	return d
}

func createLocations(t *testing.T, actor app.Actor, n int) []store.Location {
	t.Helper()
	locations := make([]store.Location, 0, n)
	for i := 0; i < n; i++ {
		l, err := svc.CreateLocation(ctx, actor, app.CreateLocationInput{
			Label:   fmt.Sprintf("Ponto %c", 'A'+i),
			Address: fmt.Sprintf("Rua %c, %d", 'A'+i, 100+i),
		})
		if err != nil {
			t.Fatalf("CreateLocation: %v", err)
		}
		locations = append(locations, l)
	}
	return locations
}

func locationIDs(ls []store.Location) []uuid.UUID {
	ids := make([]uuid.UUID, len(ls))
	for i, l := range ls {
		ids[i] = l.ID
	}
	return ids
}

// at builds a fixed timestamp on the test date (UTC).
func at(hour, min int) *time.Time {
	t := time.Date(2026, time.July, 1, hour, min, 0, 0, time.UTC)
	return &t
}

func assertErrIs(t *testing.T, name string, err, want error) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Errorf("%s = %v, want %v", name, err, want)
	}
}

// auditRows fetches audit rows for one entity, newest last.
func auditRows(t *testing.T, entity, entityID string) []map[string]any {
	t.Helper()
	rows, err := testDB.Query(ctx, `
SELECT action, old_values, new_values FROM audit_log
 WHERE entity = $1 AND entity_id = $2 ORDER BY at`, entity, entityID)
	if err != nil {
		t.Fatalf("audit query: %v", err)
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var action string
		var oldV, newV []byte
		if err := rows.Scan(&action, &oldV, &newV); err != nil {
			t.Fatalf("audit scan: %v", err)
		}
		var oldM, newM map[string]any
		_ = json.Unmarshal(oldV, &oldM)
		_ = json.Unmarshal(newV, &newM)
		out = append(out, map[string]any{"action": action, "old": oldM, "new": newM})
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("audit rows: %v", err)
	}
	return out
}

// stopSecondsFromDB reads the generated stop_seconds column directly.
func stopSecondsFromDB(t *testing.T, routeID uuid.UUID) []int {
	t.Helper()
	rows, err := testDB.Query(ctx, `
SELECT stop_order, stop_seconds FROM route_stop
 WHERE route_id = $1 ORDER BY stop_order`, routeID)
	if err != nil {
		t.Fatalf("stop query: %v", err)
	}
	defer rows.Close()
	var out []int
	for rows.Next() {
		var order, secs int
		if err := rows.Scan(&order, &secs); err != nil {
			t.Fatalf("stop scan: %v", err)
		}
		out = append(out, secs)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("stop rows: %v", err)
	}
	return out
}

// --- directories -------------------------------------------------------

func TestAccountsAndDirectories(t *testing.T) {
	mgr := createManager(t, "manager-accounts")
	drv := createDriver(t, "driver-accounts")

	if drv.Role != "driver" || drv.Active != true {
		t.Fatalf("created driver: role=%q active=%v", drv.Role, drv.Active)
	}
	km := "12.50"
	doc := "123.456.789-00"
	d2, err := svc.CreateDriver(ctx, adminActor(), app.CreateDriverInput{
		Name: "Driver With Profile", Email: "profile@test.dev", Password: "pw-12345", Phone: "0",
		Document: &doc, VehicleName: ptr("Fiorino"), VehiclePlate: ptr("ABC1D23"), KmPerL: &km,
	})
	if err != nil {
		t.Fatalf("CreateDriver with profile: %v", err)
	}
	if d2.KmPerL == nil || *d2.KmPerL != "12.50" {
		t.Errorf("km_per_l round-trip = %v, want 12.50", d2.KmPerL)
	}

	// Duplicate emails across roles hit the same lower(email) index.
	_, err = svc.CreateDriver(ctx, adminActor(), app.CreateDriverInput{
		Name: "Dup", Email: mgr.Email, Password: "pw-12345", Phone: "0",
	})
	assertErrIs(t, "duplicate manager email as driver", err, app.ErrDuplicateEmail)
	_, err = svc.CreateManager(ctx, adminActor(), app.CreateManagerInput{
		Name: "Dup", Email: "driver-accounts@test.dev", Password: "pw-12345", Phone: "0",
	})
	assertErrIs(t, "duplicate driver email as manager", err, app.ErrDuplicateEmail)

	// Validation.
	_, err = svc.CreateDriver(ctx, adminActor(), app.CreateDriverInput{Name: "", Email: "x@test.dev", Password: "pw-12345", Phone: "0"})
	assertErrIs(t, "missing name", err, app.ErrValidation)
	var fe *app.FieldError
	if !errors.As(err, &fe) || fe.Field != "name" {
		t.Errorf("missing name error = %v, want FieldError name", err)
	}
	_, err = svc.CreateDriver(ctx, adminActor(), app.CreateDriverInput{
		Name: "X", Email: "x@test.dev", Password: "pw-12345", Phone: "0", KmPerL: ptr("abc"),
	})
	assertErrIs(t, "bad km_per_l", err, app.ErrBadInput)
	_, err = svc.CreateDriver(ctx, adminActor(), app.CreateDriverInput{
		Name: "X", Email: "x@test.dev", Password: "pw-12345", Phone: "0", KmPerL: ptr("0"),
	})
	assertErrIs(t, "zero km_per_l", err, app.ErrValidation)

	// Passwords: at least 8 characters, a field error on "password".
	_, err = svc.CreateDriver(ctx, adminActor(), app.CreateDriverInput{
		Name: "Short", Email: "short-pw@test.dev", Password: "1234567", Phone: "0",
	})
	assertErrIs(t, "short driver password", err, app.ErrValidation)
	if !errors.As(err, &fe) || fe.Field != "password" {
		t.Errorf("short driver password error = %v, want FieldError password", err)
	}
	_, err = svc.CreateManager(ctx, adminActor(), app.CreateManagerInput{
		Name: "Short", Email: "short-pw-m@test.dev", Password: "1234567", Phone: "0",
	})
	assertErrIs(t, "short manager password", err, app.ErrValidation)
	if !errors.As(err, &fe) || fe.Field != "password" {
		t.Errorf("short manager password error = %v, want FieldError password", err)
	}
	if _, err := svc.CreateManager(ctx, adminActor(), app.CreateManagerInput{
		Name: "Eight", Email: "eight-pw-m@test.dev", Password: "12345678", Phone: "0",
	}); err != nil {
		t.Errorf("8-character password rejected: %v", err)
	}

	// Clearing optional profile fields: Clear wins over keep; km_per_l
	// cleared falls back to the default_km_per_l parameter (NULL).
	cleared, err := svc.UpdateDriver(ctx, adminActor(), app.UpdateDriverInput{
		DriverID: d2.ID,
		Clear:    app.DriverClear{Document: true, KmPerL: true},
	})
	if err != nil {
		t.Fatalf("UpdateDriver clear: %v", err)
	}
	if cleared.Document != nil || cleared.KmPerL != nil {
		t.Errorf("cleared driver = doc %v km %v, want both nil", cleared.Document, cleared.KmPerL)
	}
	if cleared.VehicleName == nil || *cleared.VehicleName != "Fiorino" ||
		cleared.VehiclePlate == nil || *cleared.VehiclePlate != "ABC1D23" {
		t.Errorf("untouched fields changed: %v %v", cleared.VehicleName, cleared.VehiclePlate)
	}
	cleared, err = svc.UpdateDriver(ctx, adminActor(), app.UpdateDriverInput{
		DriverID: d2.ID,
		Clear:    app.DriverClear{VehicleName: true, VehiclePlate: true},
	})
	if err != nil || cleared.VehicleName != nil || cleared.VehiclePlate != nil {
		t.Errorf("clear vehicle = %v %v (%v), want nil nil", cleared.VehicleName, cleared.VehiclePlate, err)
	}

	// Update + active_only listing (LGPD deactivation path).
	inactive := false
	if _, err := svc.UpdateDriver(ctx, adminActor(), app.UpdateDriverInput{DriverID: drv.ID, Active: &inactive}); err != nil {
		t.Fatalf("UpdateDriver deactivate: %v", err)
	}
	activeOnly, err := svc.ListDrivers(ctx, adminActor(), true)
	if err != nil {
		t.Fatalf("ListDrivers: %v", err)
	}
	for _, d := range activeOnly {
		if d.ID == drv.ID {
			t.Error("deactivated driver still listed with activeOnly=true")
		}
	}
	all, err := svc.ListDrivers(ctx, adminActor(), false)
	if err != nil {
		t.Fatalf("ListDrivers: %v", err)
	}
	found := false
	for _, d := range all {
		if d.ID == drv.ID {
			found = true
		}
	}
	if !found {
		t.Error("deactivated driver missing from full list")
	}
	_, err = svc.UpdateDriver(ctx, adminActor(), app.UpdateDriverInput{DriverID: uuid.New()})
	assertErrIs(t, "update unknown driver", err, app.ErrNotFound)

	managers, err := svc.ListManagers(ctx, adminActor())
	if err != nil {
		t.Fatalf("ListManagers: %v", err)
	}
	if len(managers) == 0 {
		t.Error("ListManagers empty")
	}

	// Locations.
	actor := actorOf(mgr.User)
	locs := createLocations(t, actor, 2)
	updated, err := svc.UpdateLocation(ctx, actor, app.UpdateLocationInput{
		LocationID: locs[0].ID, Label: ptr("Renamed"),
	})
	if err != nil {
		t.Fatalf("UpdateLocation: %v", err)
	}
	if updated.Label != "Renamed" || updated.Address != locs[0].Address {
		t.Errorf("UpdateLocation partial: %+v", updated)
	}
	listed, err := svc.ListLocations(ctx, actor, ptr("Renamed"))
	if err != nil {
		t.Fatalf("ListLocations: %v", err)
	}
	if len(listed) != 1 {
		t.Errorf("ListLocations(q) = %d rows, want 1", len(listed))
	}
	_, err = svc.UpdateLocation(ctx, actor, app.UpdateLocationInput{LocationID: uuid.New(), Label: ptr("X")})
	assertErrIs(t, "update unknown location", err, app.ErrNotFound)
}

// --- route creation ----------------------------------------------------

func TestCreateRouteValidation(t *testing.T) {
	mgr := createManager(t, "manager-routes")
	actor := actorOf(mgr.User)
	drv := createDriver(t, "driver-routes")
	locs := createLocations(t, actor, 3)

	_, err := svc.CreateRoute(ctx, actor, app.CreateRouteInput{
		DriverUserID: drv.ID, RouteDate: "2026-07-02", LocationIDs: locationIDs(locs[:1]),
	})
	assertErrIs(t, "one stop", err, app.ErrValidation)

	_, err = svc.CreateRoute(ctx, actor, app.CreateRouteInput{
		DriverUserID: uuid.New(), RouteDate: "2026-07-02", LocationIDs: locationIDs(locs[:2]),
	})
	assertErrIs(t, "unknown driver", err, app.ErrNotFound)

	_, err = svc.CreateRoute(ctx, actor, app.CreateRouteInput{
		DriverUserID: mgr.ID, RouteDate: "2026-07-02", LocationIDs: locationIDs(locs[:2]),
	})
	assertErrIs(t, "manager as driver", err, app.ErrValidation)

	_, err = svc.CreateRoute(ctx, actor, app.CreateRouteInput{
		DriverUserID: drv.ID, RouteDate: "02/07/2026", LocationIDs: locationIDs(locs[:2]),
	})
	assertErrIs(t, "bad date", err, app.ErrBadInput)

	unknownLoc := []uuid.UUID{locs[0].ID, uuid.New()}
	_, err = svc.CreateRoute(ctx, actor, app.CreateRouteInput{
		DriverUserID: drv.ID, RouteDate: "2026-07-02", LocationIDs: unknownLoc,
	})
	assertErrIs(t, "unknown location", err, app.ErrNotFound)
}

func TestCreateRouteRN05Conflict(t *testing.T) {
	mgr := createManager(t, "manager-rn05")
	actor := actorOf(mgr.User)
	drv := createDriver(t, "driver-rn05")
	other := createDriver(t, "driver-rn05-b")
	locs := createLocations(t, actor, 2)

	if _, err := svc.CreateRoute(ctx, actor, app.CreateRouteInput{
		DriverUserID: drv.ID, RouteDate: "2026-07-03", LocationIDs: locationIDs(locs),
	}); err != nil {
		t.Fatalf("CreateRoute: %v", err)
	}
	_, err := svc.CreateRoute(ctx, actor, app.CreateRouteInput{
		DriverUserID: drv.ID, RouteDate: "2026-07-03", LocationIDs: locationIDs(locs),
	})
	assertErrIs(t, "same driver same date", err, app.ErrDriverDateConflict)

	// Different date and different driver are both fine.
	if _, err := svc.CreateRoute(ctx, actor, app.CreateRouteInput{
		DriverUserID: drv.ID, RouteDate: "2026-07-04", LocationIDs: locationIDs(locs),
	}); err != nil {
		t.Fatalf("same driver other date: %v", err)
	}
	if _, err := svc.CreateRoute(ctx, actor, app.CreateRouteInput{
		DriverUserID: other.ID, RouteDate: "2026-07-03", LocationIDs: locationIDs(locs),
	}); err != nil {
		t.Fatalf("other driver same date: %v", err)
	}
}

// --- the full day (route A shape) --------------------------------------

func TestFullDayRouteA(t *testing.T) {
	mgr := createManager(t, "manager-fullday")
	actor := actorOf(mgr.User)
	drv := createDriver(t, "driver-fullday")
	locs := createLocations(t, actor, 5) // four stops + one for add/remove

	route, err := svc.CreateRoute(ctx, actor, app.CreateRouteInput{
		DriverUserID: drv.ID,
		RouteDate:    "2026-07-01",
		LocationIDs:  locationIDs(locs[:4]),
		Note:         ptr("full day"),
	})
	if err != nil {
		t.Fatalf("CreateRoute: %v", err)
	}
	routeID := route.Route.ID
	if route.Route.Status != "draft" {
		t.Errorf("status = %q, want draft", route.Route.Status)
	}
	if route.Route.CreatedBy != mgr.ID {
		t.Errorf("created_by = %v, want manager", route.Route.CreatedBy)
	}
	if len(route.Stops) != 4 || route.Stops[0].StopOrder != 1 || route.Stops[3].StopOrder != 4 {
		t.Fatalf("initial composition wrong: %+v", route.Stops)
	}

	// Composition edits: add in the middle, remove it, ids stay stable.
	originalIDs := []uuid.UUID{route.Stops[0].ID, route.Stops[1].ID, route.Stops[2].ID, route.Stops[3].ID}
	pos := 2
	route, err = svc.AddStop(ctx, actor, app.AddStopInput{
		RouteID: routeID, LocationID: locs[4].ID, Position: &pos,
	})
	if err != nil {
		t.Fatalf("AddStop: %v", err)
	}
	if len(route.Stops) != 5 {
		t.Fatalf("after AddStop: %d stops, want 5", len(route.Stops))
	}
	orders := make([]int, len(route.Stops))
	for i, st := range route.Stops {
		orders[i] = st.StopOrder
	}
	if fmt.Sprint(orders) != "[1 2 3 4 5]" {
		t.Errorf("orders after AddStop = %v, want [1 2 3 4 5]", orders)
	}

	route, err = svc.RemoveStop(ctx, actor, app.RemoveStopInput{RouteID: routeID, StopOrder: 2})
	if err != nil {
		t.Fatalf("RemoveStop: %v", err)
	}
	if len(route.Stops) != 4 {
		t.Fatalf("after RemoveStop: %d stops, want 4", len(route.Stops))
	}
	for i, st := range route.Stops {
		if st.ID != originalIDs[i] {
			t.Errorf("stop %d id changed by renumbering: %v != %v", i+1, st.ID, originalIDs[i])
		}
	}

	// Reorder: move stop 3 up, then back down.
	route, err = svc.ReorderStops(ctx, actor, app.ReorderStopsInput{
		RouteID: routeID, StopOrder: 3, Direction: "up",
	})
	if err != nil {
		t.Fatalf("ReorderStops up: %v", err)
	}
	if route.Stops[1].ID != originalIDs[2] || route.Stops[2].ID != originalIDs[1] {
		t.Errorf("reorder up did not swap: %+v", route.Stops)
	}
	route, err = svc.ReorderStops(ctx, actor, app.ReorderStopsInput{
		RouteID: routeID, StopOrder: 2, Direction: "down",
	})
	if err != nil {
		t.Fatalf("ReorderStops down: %v", err)
	}
	for i, st := range route.Stops {
		if st.ID != originalIDs[i] {
			t.Errorf("reorder back id mismatch at %d", i+1)
		}
	}

	// Start and record the day: 0 + 15 + 10 + 50 minutes. The recorded
	// stops come back from the services with both timestamps set.
	if _, err := svc.StartRoute(ctx, actor, routeID); err != nil {
		t.Fatalf("StartRoute: %v", err)
	}
	day := []struct {
		order          int
		arrival, depart *time.Time
	}{
		{1, at(8, 0), at(8, 5)},
		{2, at(9, 0), at(9, 15)},
		{3, at(10, 0), at(10, 10)},
		{4, at(11, 0), at(11, 50)},
	}
	var recorded []store.RouteStop
	for _, d := range day {
		if _, err := svc.RecordArrival(ctx, actor, app.RecordTimeInput{RouteID: routeID, StopOrder: d.order, At: d.arrival}); err != nil {
			t.Fatalf("RecordArrival %d: %v", d.order, err)
		}
		st, err := svc.RecordDeparture(ctx, actor, app.RecordTimeInput{RouteID: routeID, StopOrder: d.order, At: d.depart})
		if err != nil {
			t.Fatalf("RecordDeparture %d: %v", d.order, err)
		}
		recorded = append(recorded, st)
	}

	// The generated column mirrors the rules (RN01: stop 1 = 0).
	got := stopSecondsFromDB(t, routeID)
	want := []int{0, 900, 600, 3000}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("stop_seconds = %v, want %v", got, want)
	}

	// The domain oracle agrees on the total: 75 minutes, 15.625%.
	domainStops := make([]domain.Stop, len(recorded))
	for i, st := range recorded {
		domainStops[i] = domain.Stop{Order: st.StopOrder, Arrival: st.ArrivalAt, Departure: st.DepartureAt}
	}
	total := domain.RouteTotalSeconds(domain.Route{Stops: domainStops}, 0)
	if total != 4500 || domain.WholeMinutes(total) != 75 {
		t.Errorf("domain total = %d seconds, want 4500 (75 min)", total)
	}
	pct, err := domain.JourneyPercent(total, mustRat("8"))
	if err != nil || pct.Cmp(mustRat("15.625")) != 0 {
		t.Errorf("domain journey percent = %v (err %v), want 15.625", pct, err)
	}

	// Double record is a correction, not a record.
	_, err = svc.RecordArrival(ctx, actor, app.RecordTimeInput{RouteID: routeID, StopOrder: 2, At: at(12, 0)})
	assertErrIs(t, "double arrival", err, app.ErrValidation)

	// Correction (audited): stop 3 becomes 10:05 → 10:20 (15 minutes).
	stop3 := route.Stops[2]
	// (The stop used for the correction: the third of the current
	// composition — original ids held after the edits.)
	corrected, err := svc.UpdateStopTimes(ctx, actor, app.UpdateStopTimesInput{
		RouteID:     routeID,
		StopOrder:   3,
		ArrivalAt:   at(10, 5),
		DepartureAt: at(10, 20),
	})
	if err != nil {
		t.Fatalf("UpdateStopTimes: %v", err)
	}
	if !corrected.ArrivalAt.Equal(*at(10, 5)) {
		t.Errorf("corrected arrival = %v", corrected.ArrivalAt)
	}
	times := auditRows(t, "route_stop", stop3.ID.String())
	if len(times) != 1 || times[0]["action"] != "update_times" {
		t.Fatalf("update_times audit rows = %+v", times)
	}
	wantOld := map[string]string{"arrival_at": "2026-07-01T10:00:00Z", "departure_at": "2026-07-01T10:10:00Z"}
	wantNew := map[string]string{"arrival_at": "2026-07-01T10:05:00Z", "departure_at": "2026-07-01T10:20:00Z"}
	assertAuditInstants(t, times[0]["old"], wantOld)
	assertAuditInstants(t, times[0]["new"], wantNew)

	// Close with a distance, then every mutation is rejected.
	closed, err := svc.CloseRoute(ctx, actor, app.CloseRouteInput{RouteID: routeID, DistanceKm: ptr("100")})
	if err != nil {
		t.Fatalf("CloseRoute: %v", err)
	}
	if closed.Status != "closed" || closed.DistanceKm == nil || *closed.DistanceKm != "100.00" {
		t.Errorf("closed route = %+v", closed)
	}
	_, err = svc.RecordArrival(ctx, actor, app.RecordTimeInput{RouteID: routeID, StopOrder: 2, At: at(13, 0)})
	assertErrIs(t, "record on closed", err, app.ErrRouteClosed)
	_, err = svc.AddStop(ctx, actor, app.AddStopInput{RouteID: routeID, LocationID: locs[4].ID})
	assertErrIs(t, "add on closed", err, app.ErrRouteClosed)
	_, err = svc.UpdateStopTimes(ctx, actor, app.UpdateStopTimesInput{
		RouteID: routeID, StopOrder: 2, ArrivalAt: at(13, 0),
	})
	assertErrIs(t, "correct on closed", err, app.ErrRouteClosed)
	_, err = svc.ReorderStops(ctx, actor, app.ReorderStopsInput{RouteID: routeID, StopOrder: 2, Direction: "up"})
	assertErrIs(t, "reorder on closed", err, app.ErrRouteClosed)
	_, err = svc.RemoveStop(ctx, actor, app.RemoveStopInput{RouteID: routeID, StopOrder: 2})
	assertErrIs(t, "remove on closed", err, app.ErrRouteClosed)
	_, err = svc.SetRouteDistance(ctx, actor, routeID, "50")
	assertErrIs(t, "distance on closed", err, app.ErrRouteClosed)

	// Reopen: the day continues; audit records it.
	// Reopen is admin-only per the role matrix (the manager who closed
	// cannot reopen).
	if _, err := svc.ReopenRoute(ctx, adminActor(), routeID); err != nil {
		t.Fatalf("ReopenRoute: %v", err)
	}
	if _, err := svc.SetRouteDistance(ctx, actor, routeID, "100"); err != nil {
		t.Fatalf("SetRouteDistance after reopen: %v", err)
	}

	// Route-level audit trail: composition edits, close, reopen.
	routeAudit := auditRows(t, "route", routeID.String())
	actions := make([]string, len(routeAudit))
	for i, row := range routeAudit {
		actions[i] = row["action"].(string)
	}
	if fmt.Sprint(actions) != "[reorder reorder close_route reopen_route]" {
		t.Errorf("route audit actions = %v, want [reorder reorder close_route reopen_route]", actions)
	}

	// Stop-level audit: only stop 3's correction survives — the added
	// stop was removed again, and its audit row belongs to its gone id.
	stopAuditCount := 0
	for _, st := range route.Stops {
		stopAuditCount += len(auditRows(t, "route_stop", st.ID.String()))
	}
	if stopAuditCount != 1 {
		t.Errorf("stop audit rows = %d, want 1", stopAuditCount)
	}
}

// --- RN02 and the database backstop ------------------------------------

func TestRN02Rejections(t *testing.T) {
	mgr := createManager(t, "manager-rn02")
	actor := actorOf(mgr.User)
	drv := createDriver(t, "driver-rn02")
	locs := createLocations(t, actor, 2)
	route, err := svc.CreateRoute(ctx, actor, app.CreateRouteInput{
		DriverUserID: drv.ID, RouteDate: "2026-07-05", LocationIDs: locationIDs(locs),
	})
	if err != nil {
		t.Fatalf("CreateRoute: %v", err)
	}
	if _, err := svc.StartRoute(ctx, actor, route.Route.ID); err != nil {
		t.Fatalf("StartRoute: %v", err)
	}

	// Departure before arrival, rejected by the service (RN02).
	_, err = svc.RecordArrival(ctx, actor, app.RecordTimeInput{RouteID: route.Route.ID, StopOrder: 2, At: at(10, 0)})
	if err != nil {
		t.Fatalf("RecordArrival: %v", err)
	}
	_, err = svc.RecordDeparture(ctx, actor, app.RecordTimeInput{RouteID: route.Route.ID, StopOrder: 2, At: at(9, 0)})
	assertErrIs(t, "departure before arrival", err, app.ErrDepartureBeforeArrival)

	// Departure without arrival.
	_, err = svc.RecordDeparture(ctx, actor, app.RecordTimeInput{RouteID: route.Route.ID, StopOrder: 1, At: at(9, 30)})
	assertErrIs(t, "departure without arrival", err, app.ErrValidation)

	// Recording on a draft route is refused.
	draftRoute, err := svc.CreateRoute(ctx, actor, app.CreateRouteInput{
		DriverUserID: drv.ID, RouteDate: "2026-07-06", LocationIDs: locationIDs(locs),
	})
	if err != nil {
		t.Fatalf("CreateRoute draft: %v", err)
	}
	_, err = svc.RecordArrival(ctx, actor, app.RecordTimeInput{RouteID: draftRoute.Route.ID, StopOrder: 1, At: at(9, 0)})
	assertErrIs(t, "record on draft", err, app.ErrValidation)

	// The database check constraint is the backstop (constraint name
	// pinned from data-model.md).
	_, err = testDB.Exec(ctx, `
UPDATE route_stop SET departure_at = arrival_at - interval '1 hour'
 WHERE route_id = $1 AND stop_order = 2`, route.Route.ID)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23514" || pgErr.ConstraintName != "route_stop_times_order" {
		t.Errorf("raw violation = %v, want 23514 route_stop_times_order", err)
	}
}

// --- parameters ---------------------------------------------------------

func TestParams(t *testing.T) {
	params, err := svc.GetParams(ctx, adminActor())
	if err != nil {
		t.Fatalf("GetParams: %v", err)
	}
	if len(params) != 7 {
		t.Fatalf("GetParams = %d rows, want 7", len(params))
	}
	byKey := map[string]string{}
	for _, p := range params {
		byKey[p.Key] = p.Value
	}
	if byKey["fuel_price_brl"] != "6.0900" {
		t.Errorf("fuel_price_brl = %q, want 6.0900", byKey["fuel_price_brl"])
	}
	for _, p := range params {
		if p.UpdatedByName == "" {
			t.Errorf("param %s has no updated_by_name", p.Key)
		}
	}

	actor := adminActor()
	updated, err := svc.UpdateParam(ctx, actor, app.UpdateParamInput{Key: "fuel_price_brl", Value: "6.19"})
	if err != nil {
		t.Fatalf("UpdateParam: %v", err)
	}
	if updated.UpdatedByName != "Ana Administradora" {
		t.Errorf("updated_by_name = %q, want Ana Administradora", updated.UpdatedByName)
	}
	if updated.UpdatedBy != admin {
		t.Errorf("updated_by = %v, want admin", updated.UpdatedBy)
	}
	// The parameter audit row carries the TEXT key as entity_id.
	rows := auditRows(t, "parameter", "fuel_price_brl")
	if len(rows) != 1 || rows[0]["action"] != "update_param" {
		t.Fatalf("update_param audit = %+v", rows)
	}
	if fmt.Sprint(rows[0]["old"]) != fmt.Sprint(map[string]any{"value": "6.0900"}) ||
		fmt.Sprint(rows[0]["new"]) != fmt.Sprint(map[string]any{"value": "6.19"}) {
		t.Errorf("update_param audit values: %v -> %v", rows[0]["old"], rows[0]["new"])
	}

	_, err = svc.UpdateParam(ctx, actor, app.UpdateParamInput{Key: "no_such_key", Value: "1"})
	assertErrIs(t, "unknown key", err, app.ErrNotFound)
	_, err = svc.UpdateParam(ctx, actor, app.UpdateParamInput{Key: "fuel_price_brl", Value: "abc"})
	assertErrIs(t, "bad value", err, app.ErrBadInput)
	_, err = svc.UpdateParam(ctx, actor, app.UpdateParamInput{Key: "fuel_price_brl", Value: "-1"})
	assertErrIs(t, "negative value", err, app.ErrValidation)
	_, err = svc.UpdateParam(ctx, actor, app.UpdateParamInput{Key: "standard_journey_hours", Value: "0"})
	assertErrIs(t, "zero journey hours", err, app.ErrValidation)
	if _, err := svc.UpdateParam(ctx, actor, app.UpdateParamInput{Key: "min_stop_minutes", Value: "0"}); err != nil {
		t.Errorf("min_stop_minutes = 0 should be allowed: %v", err)
	}
}

func TestSetRouteDistance(t *testing.T) {
	mgr := createManager(t, "manager-distance")
	actor := actorOf(mgr.User)
	drv := createDriver(t, "driver-distance")
	locs := createLocations(t, actor, 2)

	draft, err := svc.CreateRoute(ctx, actor, app.CreateRouteInput{
		DriverUserID: drv.ID, RouteDate: "2026-07-07", LocationIDs: locationIDs(locs),
	})
	if err != nil {
		t.Fatalf("CreateRoute: %v", err)
	}
	_, err = svc.SetRouteDistance(ctx, actor, draft.Route.ID, "75.50")
	assertErrIs(t, "distance on draft", err, app.ErrValidation)

	if _, err := svc.StartRoute(ctx, actor, draft.Route.ID); err != nil {
		t.Fatalf("StartRoute: %v", err)
	}
	r, err := svc.SetRouteDistance(ctx, actor, draft.Route.ID, "75.50")
	if err != nil {
		t.Fatalf("SetRouteDistance: %v", err)
	}
	if r.DistanceKm == nil || *r.DistanceKm != "75.50" {
		t.Errorf("distance round-trip = %v, want 75.50", r.DistanceKm)
	}

	_, err = svc.SetRouteDistance(ctx, actor, draft.Route.ID, "0")
	assertErrIs(t, "zero distance", err, app.ErrValidation)
	_, err = svc.SetRouteDistance(ctx, actor, draft.Route.ID, "abc")
	assertErrIs(t, "bad distance", err, app.ErrBadInput)
	_, err = svc.SetRouteDistance(ctx, actor, uuid.New(), "10")
	assertErrIs(t, "unknown route", err, app.ErrNotFound)
}

// --- helpers ------------------------------------------------------------

// assertAuditInstants compares a jsonb audit side (RFC 3339 strings,
// any offset) against expected instants.
func assertAuditInstants(t *testing.T, gotSide any, want map[string]string) {
	t.Helper()
	got, _ := gotSide.(map[string]any)
	if len(got) != len(want) {
		t.Errorf("audit side = %v, want keys %v", gotSide, want)
		return
	}
	for k, wantS := range want {
		gotS, _ := got[k].(string)
		g, err := time.Parse(time.RFC3339Nano, gotS)
		if err != nil {
			t.Errorf("audit %s = %q: %v", k, gotS, err)
			continue
		}
		w, err := time.Parse(time.RFC3339Nano, wantS)
		if err != nil {
			panic(err)
		}
		if !g.Equal(w) {
			t.Errorf("audit %s = %s, want %s", k, gotS, wantS)
		}
	}
}

func ptr[T any](v T) *T { return &v }

func mustRat(s string) *big.Rat {
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		panic("bad decimal literal: " + s)
	}
	return r
}

func managerActor() app.Actor {
	return app.Actor{UserID: uuid.MustParse("aa000000-0000-4000-8000-000000000002"), Role: "manager"}
}

// RNF06 access: managers see the driver document masked (every digit
// but the last two), admins in full; the flag says which.
func TestDriverDocumentMasking(t *testing.T) {
	doc := "123.456.789-11"
	d, err := svc.CreateDriver(ctx, managerActor(), app.CreateDriverInput{
		Name: "Masked " + uuid.NewString()[:8], Email: "masked-" + uuid.NewString()[:8] + "@test.dev",
		Password: "pw-12345", Phone: "0", Document: &doc,
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.Document == nil || *d.Document != "***.***.***-11" || !d.DocumentMasked {
		t.Errorf("manager create response document = %v masked=%v", d.Document, d.DocumentMasked)
	}

	find := func(actor app.Actor) store.Driver {
		t.Helper()
		all, err := svc.ListDrivers(ctx, actor, false)
		if err != nil {
			t.Fatal(err)
		}
		for _, x := range all {
			if x.ID == d.ID {
				return x
			}
		}
		t.Fatalf("driver %s not listed", d.ID)
		return store.Driver{}
	}
	if m := find(managerActor()); m.Document == nil || *m.Document != "***.***.***-11" || !m.DocumentMasked {
		t.Errorf("manager list document = %v masked=%v", m.Document, m.DocumentMasked)
	}
	if a := find(adminActor()); a.Document == nil || *a.Document != doc || a.DocumentMasked {
		t.Errorf("admin list document = %v masked=%v", a.Document, a.DocumentMasked)
	}

	// A manager may set a new document; the response is masked.
	newDoc := "987.654.321-00"
	u, err := svc.UpdateDriver(ctx, managerActor(), app.UpdateDriverInput{DriverID: d.ID, Document: &newDoc})
	if err != nil {
		t.Fatal(err)
	}
	if u.Document == nil || *u.Document != "***.***.***-00" || !u.DocumentMasked {
		t.Errorf("manager update response document = %v", u.Document)
	}
	// Re-submitting the masked value (the edit form round trip) keeps
	// the stored document instead of overwriting it with asterisks.
	masked := "***.***.***-00"
	if _, err := svc.UpdateDriver(ctx, managerActor(), app.UpdateDriverInput{DriverID: d.ID, Document: &masked, Phone: ptr("31 1")}); err != nil {
		t.Fatalf("masked round trip: %v", err)
	}
	if a := find(adminActor()); a.Document == nil || *a.Document != newDoc {
		t.Errorf("after masked round trip the stored document = %v, want %s", a.Document, newDoc)
	}
	// Any other value with a mask character is a validation error.
	bad := "***.***.***-99"
	_, err = svc.UpdateDriver(ctx, managerActor(), app.UpdateDriverInput{DriverID: d.ID, Document: &bad})
	assertErrIs(t, "foreign masked document", err, app.ErrValidation)

	// Drivers without a document are not flagged.
	plain := createDriver(t, "unmasked-"+uuid.NewString()[:8])
	all, err := svc.ListDrivers(ctx, managerActor(), false)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range all {
		if x.ID == plain.ID && (x.Document != nil || x.DocumentMasked) {
			t.Errorf("no-document driver = %v masked=%v", x.Document, x.DocumentMasked)
		}
	}
}

// RNF06 erasure: AnonymizeDriver pseudonymizes the person, keeps the
// operational history, audits which fields were cleared (not their
// values), and is admin-only.
func TestAnonymizeDriver(t *testing.T) {
	doc := "321.654.987-55"
	drv, err := svc.CreateDriver(ctx, adminActor(), app.CreateDriverInput{
		Name: "Erase Me", Email: "erase-" + uuid.NewString()[:8] + "@test.dev", Password: "pw-12345",
		Phone: "31 98888-0000", Document: &doc, VehicleName: ptr("Fiorino"), VehiclePlate: ptr("AAA1B23"), KmPerL: ptr("11.00"),
	})
	if err != nil {
		t.Fatal(err)
	}
	locs := createLocations(t, adminActor(), 2)
	d := time.Date(2025, time.February, 3, 12, 0, 0, 0, time.UTC)
	recordRoute(t, drv, locs, "2025-02-03", d, 20)

	_, err = svc.AnonymizeDriver(ctx, managerActor(), drv.ID)
	assertErrIs(t, "manager anonymize", err, app.ErrForbidden)
	_, err = svc.AnonymizeDriver(ctx, adminActor(), uuid.New())
	assertErrIs(t, "unknown driver", err, app.ErrNotFound)

	out, err := svc.AnonymizeDriver(ctx, adminActor(), drv.ID)
	if err != nil {
		t.Fatalf("AnonymizeDriver: %v", err)
	}
	short := drv.ID.String()[:8]
	if out.Name != "Motorista removido "+short || out.Email != "removido-"+drv.ID.String()+"@anonimo.invalid" ||
		out.Phone != "" || out.Active || out.Document != nil || out.VehicleName != nil || out.VehiclePlate != nil {
		t.Errorf("anonymized driver = %+v", out)
	}
	if out.KmPerL == nil || *out.KmPerL != "11.00" {
		t.Errorf("km_per_l (operational) = %v, want kept 11.00", out.KmPerL)
	}
	// Login is impossible (inactive and no usable password).
	_, _, _, err = svc.Login(ctx, app.LoginInput{Email: drv.Email, Password: "pw-12345"})
	assertErrIs(t, "login after anonymize", err, app.ErrUnauthenticated)

	// Operational history intact: the route and its 20 minutes remain.
	sum, err := svc.GetDashboardByPeriod(ctx, adminActor(), app.DashboardInput{From: "2025-02-01", To: "2025-02-28", DriverUserID: &drv.ID})
	if err != nil || sum.TotalStoppedMinut != 20 || sum.RoutesCount != 1 || sum.ByDriver[0].DriverName != out.Name {
		t.Errorf("history after anonymize = %+v, %v", sum, err)
	}

	// Audit: one anonymize row naming the cleared fields, no personal data.
	entries, err := svc.ListAudit(ctx, adminActor(), app.ListAuditInput{Entity: ptr("app_user")})
	if err != nil {
		t.Fatal(err)
	}
	var rows int
	for _, e := range entries {
		if e.EntityID != drv.ID.String() {
			continue
		}
		rows++
		raw := string(e.OldValues) + string(e.NewValues)
		if e.Action != "anonymize" || !strings.Contains(raw, "document") || !strings.Contains(raw, "email") {
			t.Errorf("audit row = %s %s", e.Action, raw)
		}
		for _, personal := range []string{"Erase Me", drv.Email, "98888", doc, "Fiorino", "AAA1B23"} {
			if strings.Contains(raw, personal) {
				t.Errorf("audit row leaks %q: %s", personal, raw)
			}
		}
	}
	if rows != 1 {
		t.Errorf("anonymize audit rows = %d, want 1", rows)
	}

	// Second call: 200-style no-op, same result, no second audit row.
	again, err := svc.AnonymizeDriver(ctx, adminActor(), drv.ID)
	if err != nil || again.Name != out.Name {
		t.Errorf("second anonymize = %+v, %v", again, err)
	}
	entries, _ = svc.ListAudit(ctx, adminActor(), app.ListAuditInput{Entity: ptr("app_user")})
	rows = 0
	for _, e := range entries {
		if e.EntityID == drv.ID.String() {
			rows++
		}
	}
	if rows != 1 {
		t.Errorf("audit rows after second call = %d, want still 1", rows)
	}
}

// Sessions: Authenticate turns a cookie into a session only for an
// existing, active user, with the role read fresh from the database.
func TestAuthenticateRequiresActiveUser(t *testing.T) {
	drv := createDriver(t, "auth-active-"+uuid.NewString()[:8])
	_, sess, cookie, err := svc.Login(ctx, app.LoginInput{Email: drv.Email, Password: "pw-" + drv.Name})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	got, err := svc.Authenticate(ctx, cookie)
	if err != nil || got.UserID != drv.ID || got.Role != "driver" || !got.ExpiresAt.Equal(sess.ExpiresAt) {
		t.Fatalf("Authenticate active = %+v, %v", got, err)
	}
	_, err = svc.Authenticate(ctx, "garbage")
	assertErrIs(t, "bad cookie", err, app.ErrUnauthenticated)

	if _, err := svc.UpdateDriver(ctx, adminActor(), app.UpdateDriverInput{DriverID: drv.ID, Active: ptr(false)}); err != nil {
		t.Fatal(err)
	}
	_, err = svc.Authenticate(ctx, cookie)
	assertErrIs(t, "deactivated user's cookie", err, app.ErrUnauthenticated)
}

// RNF05 + acceptance criterion 3: a location edit is audited, and a
// stop keeps the label/address/coordinates it was created with — past
// routes (detail, CSV) never change when the registry changes.
func TestLocationEditKeepsStopSnapshot(t *testing.T) {
	drv := createDriver(t, "snapshot-"+uuid.NewString()[:8])
	lat, lng := "-19.900000", "-43.900000"
	a, err := svc.CreateLocation(ctx, adminActor(), app.CreateLocationInput{Label: "Snap A", Address: "Rua Antiga, 1", Latitude: &lat, Longitude: &lng})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.CreateLocation(ctx, adminActor(), app.CreateLocationInput{Label: "Snap B", Address: "Rua Antiga, 2"})
	if err != nil {
		t.Fatal(err)
	}
	route, err := svc.CreateRoute(ctx, adminActor(), app.CreateRouteInput{
		DriverUserID: drv.ID, RouteDate: "2025-01-06", LocationIDs: []uuid.UUID{a.ID, b.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.StartRoute(ctx, adminActor(), route.Route.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CloseRoute(ctx, adminActor(), app.CloseRouteInput{RouteID: route.Route.ID}); err != nil {
		t.Fatal(err)
	}

	newLat := "-20.000000"
	if _, err := svc.UpdateLocation(ctx, managerActor(), app.UpdateLocationInput{
		LocationID: a.ID, Label: ptr("Snap A2"), Address: ptr("Av. Nova, 99"), Latitude: &newLat,
	}); err != nil {
		t.Fatalf("UpdateLocation: %v", err)
	}

	view, err := svc.GetRoute(ctx, adminActor(), route.Route.ID)
	if err != nil {
		t.Fatal(err)
	}
	st := view.Stops[0]
	if st.Label != "Snap A" || st.Address != "Rua Antiga, 1" || st.Latitude == nil || *st.Latitude != lat {
		t.Errorf("closed route stop 1 = %s / %s / %v, want the original snapshot", st.Label, st.Address, st.Latitude)
	}
	rows, err := svc.ExportPeriodCSV(ctx, adminActor(), "2025-01-06", "2025-01-06", &drv.ID, nil)
	if err != nil || len(rows) != 2 || rows[0].Address != "Rua Antiga, 1" {
		t.Errorf("export after edit = %+v, %v; want the original address", rows, err)
	}

	// A stop added after the edit snapshots the new values.
	later, err := svc.CreateRoute(ctx, adminActor(), app.CreateRouteInput{
		DriverUserID: drv.ID, RouteDate: "2025-01-07", LocationIDs: []uuid.UUID{b.ID, a.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	view, err = svc.GetRoute(ctx, adminActor(), later.Route.ID)
	if err != nil || view.Stops[1].Address != "Av. Nova, 99" || view.Stops[1].Label != "Snap A2" {
		t.Errorf("new route stop 2 = %+v, %v; want the edited location", view.Stops, err)
	}

	// The edit is audited with old and new values.
	entries, err := svc.ListAudit(ctx, adminActor(), app.ListAuditInput{Entity: ptr("location")})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range entries {
		if e.EntityID == a.ID.String() && e.Action == "update_location" {
			found = true
			if !strings.Contains(string(e.OldValues), "Rua Antiga, 1") || !strings.Contains(string(e.NewValues), "Av. Nova, 99") ||
				e.Actor != "Gustavo Gerente" {
				t.Errorf("update_location audit = %s -> %s by %s", e.OldValues, e.NewValues, e.Actor)
			}
		}
	}
	if !found {
		t.Error("no update_location audit row")
	}
}

// Managers: admin-only UpdateManager (name/phone/active; email
// immutable) and AnonymizeManager (same pseudonymization as drivers,
// no profile to clear).
func TestManagerUpdateAndAnonymize(t *testing.T) {
	m, err := svc.CreateManager(ctx, adminActor(), app.CreateManagerInput{
		Name: "Mgr Edit", Email: "mgr-edit-" + uuid.NewString()[:8] + "@test.dev", Password: "pw-12345", Phone: "31 1",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.UpdateManager(ctx, managerActor(), app.UpdateManagerInput{ManagerID: m.ID, Name: ptr("X")})
	assertErrIs(t, "manager updates manager", err, app.ErrForbidden)
	_, err = svc.UpdateManager(ctx, adminActor(), app.UpdateManagerInput{ManagerID: uuid.MustParse("aa000000-0000-4000-8000-000000000003"), Name: ptr("X")})
	assertErrIs(t, "UpdateManager on a driver id", err, app.ErrNotFound)

	u, err := svc.UpdateManager(ctx, adminActor(), app.UpdateManagerInput{ManagerID: m.ID, Name: ptr("Mgr Renamed"), Phone: ptr("31 2"), Active: ptr(false)})
	if err != nil || u.Name != "Mgr Renamed" || u.Phone != "31 2" || u.Active || u.Email != m.Email {
		t.Errorf("UpdateManager = %+v, %v", u, err)
	}
	_, _, _, err = svc.Login(ctx, app.LoginInput{Email: m.Email, Password: "pw-12345"})
	assertErrIs(t, "login deactivated manager", err, app.ErrUnauthenticated)

	_, err = svc.AnonymizeManager(ctx, managerActor(), m.ID)
	assertErrIs(t, "manager anonymizes manager", err, app.ErrForbidden)
	_, err = svc.AnonymizeManager(ctx, adminActor(), uuid.New())
	assertErrIs(t, "unknown manager", err, app.ErrNotFound)
	a, err := svc.AnonymizeManager(ctx, adminActor(), m.ID)
	if err != nil || a.Name != "Gestor removido "+m.ID.String()[:8] || a.Email != "removido-"+m.ID.String()+"@anonimo.invalid" ||
		a.Phone != "" || a.Active {
		t.Errorf("AnonymizeManager = %+v, %v", a, err)
	}
	again, err := svc.AnonymizeManager(ctx, adminActor(), m.ID)
	if err != nil || again.Name != a.Name {
		t.Errorf("second AnonymizeManager = %+v, %v", again, err)
	}
	entries, _ := svc.ListAudit(ctx, adminActor(), app.ListAuditInput{Entity: ptr("app_user")})
	rows := 0
	for _, e := range entries {
		if e.EntityID == m.ID.String() {
			rows++
			if strings.Contains(string(e.OldValues)+string(e.NewValues), "Mgr Renamed") {
				t.Errorf("manager anonymize audit leaks the name: %s", e.OldValues)
			}
		}
	}
	if rows != 1 {
		t.Errorf("manager anonymize audit rows = %d, want 1", rows)
	}
}

// tp.md §8: a driver may have a responsible manager (the "team"). It is
// an attribute, not a permission boundary; reads can filter by it.
func TestManagerTeam(t *testing.T) {
	gustavo := uuid.MustParse("aa000000-0000-4000-8000-000000000002")
	all, err := svc.ListDrivers(ctx, adminActor(), false)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range all {
		if d.Email == "driver-a@stoptime.dev" &&
			(d.ManagerUserID == nil || *d.ManagerUserID != gustavo || d.ManagerName == nil || *d.ManagerName != "Gustavo Gerente") {
			t.Errorf("golden driver A manager = %v %v, want Gustavo", d.ManagerUserID, d.ManagerName)
		}
	}

	m, err := svc.CreateManager(ctx, adminActor(), app.CreateManagerInput{
		Name: "Team Lead", Email: "team-" + uuid.NewString()[:8] + "@test.dev", Password: "pw-12345", Phone: "0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if m.TeamSize != 0 {
		t.Errorf("new manager team_size = %d", m.TeamSize)
	}
	member, err := svc.CreateDriver(ctx, managerActor(), app.CreateDriverInput{
		Name: "Team Member", Email: "member-" + uuid.NewString()[:8] + "@test.dev", Password: "pw-12345", Phone: "0",
		ManagerUserID: &m.ID,
	})
	if err != nil {
		t.Fatalf("CreateDriver with manager: %v", err)
	}
	if member.ManagerUserID == nil || *member.ManagerUserID != m.ID || member.ManagerName == nil || *member.ManagerName != "Team Lead" {
		t.Errorf("member manager = %v %v", member.ManagerUserID, member.ManagerName)
	}
	outsider := createDriver(t, "outsider-"+uuid.NewString()[:8])

	// Only a manager can be responsible.
	_, err = svc.UpdateDriver(ctx, adminActor(), app.UpdateDriverInput{DriverID: outsider.ID, ManagerUserID: &outsider.ID})
	assertErrIs(t, "driver as manager", err, app.ErrValidation)
	var fe *app.FieldError
	if !errors.As(err, &fe) || fe.Field != "manager_user_id" {
		t.Errorf("driver as manager error = %v, want field manager_user_id", err)
	}

	managers, err := svc.ListManagers(ctx, adminActor())
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range managers {
		if x.ID == m.ID && x.TeamSize != 1 {
			t.Errorf("team_size = %d, want 1", x.TeamSize)
		}
		if x.ID == gustavo && x.TeamSize < 3 {
			t.Errorf("Gustavo team_size = %d, want the 3 golden drivers", x.TeamSize)
		}
	}

	// Team filters on history, dashboards and export.
	locs := createLocations(t, adminActor(), 2)
	day := time.Date(2021, time.March, 1, 12, 0, 0, 0, time.UTC)
	recordRoute(t, member, locs, "2021-03-01", day, 10)
	recordRoute(t, outsider, locs, "2021-03-01", day, 20)
	team := &m.ID
	routes, err := svc.ListRoutes(ctx, adminActor(), app.ListRoutesInput{From: ptr("2021-03-01"), To: ptr("2021-03-01"), ManagerUserID: team})
	if err != nil || len(routes) != 1 || routes[0].DriverName != "Team Member" {
		t.Errorf("team routes = %+v, %v", routes, err)
	}
	in := app.DashboardInput{From: "2021-03-01", To: "2021-03-01", ManagerUserID: team}
	period, err := svc.GetDashboardByPeriod(ctx, adminActor(), in)
	if err != nil || period.TotalStoppedMinut != 10 || period.RoutesCount != 1 {
		t.Errorf("team period = %+v, %v", period, err)
	}
	days, err := svc.GetDashboardByDay(ctx, adminActor(), in)
	if err != nil || len(days.Series) != 1 || days.Series[0].TotalStoppedMinut != 10 {
		t.Errorf("team day = %+v, %v", days, err)
	}
	months, err := svc.GetDashboardByMonth(ctx, adminActor(), in)
	if err != nil || len(months.Series) != 1 || months.Series[0].TotalStoppedMinut != 10 {
		t.Errorf("team month = %+v, %v", months, err)
	}
	rows, err := svc.ExportPeriodCSV(ctx, adminActor(), "2021-03-01", "2021-03-01", nil, team)
	if err != nil || len(rows) != 2 || rows[0].DriverName != "Team Member" {
		t.Errorf("team export = %+v, %v", rows, err)
	}
	// Unfiltered, both drivers count.
	all2, err := svc.GetDashboardByPeriod(ctx, adminActor(), app.DashboardInput{From: "2021-03-01", To: "2021-03-01"})
	if err != nil || all2.TotalStoppedMinut != 30 {
		t.Errorf("unfiltered period = %+v, %v", all2, err)
	}

	// Explicit clear.
	cleared, err := svc.UpdateDriver(ctx, adminActor(), app.UpdateDriverInput{DriverID: member.ID, Clear: app.DriverClear{ManagerUserID: true}})
	if err != nil || cleared.ManagerUserID != nil || cleared.ManagerName != nil {
		t.Errorf("cleared manager = %v %v, %v", cleared.ManagerUserID, cleared.ManagerName, err)
	}
}

// RN06 sequence: a driver reaches stop n only after leaving stop n-1
// (stop 1's departure is optional — no stopwatch there — but bounds stop
// 2 when recorded); corrections may skip the "left first" rule but never
// put times out of order between neighbours.
func TestStopSequenceRules(t *testing.T) {
	drv := createDriver(t, "seq-"+uuid.NewString()[:8])
	locs := createLocations(t, adminActor(), 3)
	route, err := svc.CreateRoute(ctx, adminActor(), app.CreateRouteInput{
		DriverUserID: drv.ID, RouteDate: "2024-10-07", LocationIDs: locationIDs(locs),
	})
	if err != nil {
		t.Fatal(err)
	}
	id := route.Route.ID
	if _, err := svc.StartRoute(ctx, adminActor(), id); err != nil {
		t.Fatal(err)
	}
	driver := app.Actor{UserID: drv.ID, Role: "driver"}
	tm := func(h, m int) *time.Time { v := time.Date(2024, time.October, 7, h, m, 0, 0, time.UTC); return &v }
	wantOrder := func(name string, err error, field string) {
		t.Helper()
		assertErrIs(t, name, err, app.ErrStopTimesOutOfOrder)
		var fe *app.FieldError
		if !errors.As(err, &fe) || fe.Field != field || fe.Reason == "" {
			t.Errorf("%s = %v, want field %s with a reason", name, err, field)
		}
	}

	// Stop 3 before leaving stop 2 (not yet arrived there): rejected.
	_, err = svc.RecordArrival(ctx, driver, app.RecordTimeInput{RouteID: id, StopOrder: 3, At: tm(9, 0)})
	wantOrder("arrive 3 before leaving 2", err, "arrival_at")

	// Stop 2 without a stop-1 departure is fine (departure point).
	if _, err := svc.RecordArrival(ctx, driver, app.RecordTimeInput{RouteID: id, StopOrder: 2, At: tm(9, 0)}); err != nil {
		t.Fatalf("arrive 2: %v", err)
	}
	_, err = svc.RecordArrival(ctx, driver, app.RecordTimeInput{RouteID: id, StopOrder: 3, At: tm(9, 30)})
	wantOrder("arrive 3 while still at 2", err, "arrival_at")
	if _, err := svc.RecordDeparture(ctx, driver, app.RecordTimeInput{RouteID: id, StopOrder: 2, At: tm(9, 10)}); err != nil {
		t.Fatalf("depart 2: %v", err)
	}
	_, err = svc.RecordArrival(ctx, driver, app.RecordTimeInput{RouteID: id, StopOrder: 3, At: tm(9, 5)})
	wantOrder("arrive 3 before departing 2", err, "arrival_at")
	if _, err := svc.RecordArrival(ctx, driver, app.RecordTimeInput{RouteID: id, StopOrder: 3, At: tm(9, 30)}); err != nil {
		t.Fatalf("arrive 3 in order: %v", err)
	}

	// Corrections keep neighbours in order.
	_, err = svc.UpdateStopTimes(ctx, managerActor(), app.UpdateStopTimesInput{RouteID: id, StopOrder: 2, DepartureAt: tm(9, 40)})
	wantOrder("correct departure 2 after arrival 3", err, "departure_at")
	_, err = svc.UpdateStopTimes(ctx, managerActor(), app.UpdateStopTimesInput{RouteID: id, StopOrder: 3, ArrivalAt: tm(9, 5)})
	wantOrder("correct arrival 3 before departure 2", err, "arrival_at")
	if _, err := svc.UpdateStopTimes(ctx, managerActor(), app.UpdateStopTimesInput{RouteID: id, StopOrder: 2, DepartureAt: tm(9, 20)}); err != nil {
		t.Errorf("in-order correction rejected: %v", err)
	}

	// A recorded stop-1 departure bounds stop 2.
	other, err := svc.CreateRoute(ctx, adminActor(), app.CreateRouteInput{
		DriverUserID: drv.ID, RouteDate: "2024-10-08", LocationIDs: locationIDs(locs),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.StartRoute(ctx, adminActor(), other.Route.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpdateStopTimes(ctx, managerActor(), app.UpdateStopTimesInput{RouteID: other.Route.ID, StopOrder: 1, DepartureAt: tm(8, 0)}); err != nil {
		t.Fatalf("set stop-1 departure: %v", err)
	}
	_, err = svc.RecordArrival(ctx, driver, app.RecordTimeInput{RouteID: other.Route.ID, StopOrder: 2, At: tm(7, 50)})
	wantOrder("arrive 2 before leaving the departure point", err, "arrival_at")

	// Managers may correct a stop whose predecessor has no departure yet.
	if _, err := svc.UpdateStopTimes(ctx, managerActor(), app.UpdateStopTimesInput{
		RouteID: other.Route.ID, StopOrder: 3, ArrivalAt: tm(10, 0), DepartureAt: tm(10, 10),
	}); err != nil {
		t.Errorf("manager correction without predecessor departure: %v", err)
	}
}

// Managers may list managers (to pick a driver's responsible manager),
// minimized: no email or phone (RNF06); admins see everything.
func TestListManagersMinimizedForManagers(t *testing.T) {
	full, err := svc.ListManagers(ctx, adminActor())
	if err != nil {
		t.Fatal(err)
	}
	mini, err := svc.ListManagers(ctx, managerActor())
	if err != nil {
		t.Fatalf("manager ListManagers: %v", err)
	}
	if len(full) == 0 || len(mini) != len(full) {
		t.Fatalf("lists = %d / %d", len(full), len(mini))
	}
	for i := range full {
		if full[i].Email == "" || full[i].Restricted {
			t.Errorf("admin row %d = %+v, want full", i, full[i])
		}
		m := mini[i]
		if m.ID != full[i].ID || m.Name != full[i].Name || m.TeamSize != full[i].TeamSize || !m.Restricted ||
			m.Email != "" || m.Phone != "" {
			t.Errorf("manager row %d = %+v, want minimized", i, m)
		}
	}
	raw, err := json.Marshal(mini[0])
	if err != nil {
		t.Fatal(err)
	}
	var keys map[string]any
	if err := json.Unmarshal(raw, &keys); err != nil {
		t.Fatal(err)
	}
	if len(keys) != 4 || keys["id"] == nil || keys["name"] == nil || keys["active"] == nil || keys["team_size"] == nil {
		t.Errorf("minimized JSON = %s, want exactly id, name, active, team_size", raw)
	}
	_, err = svc.ListManagers(ctx, app.Actor{UserID: uuid.MustParse("aa000000-0000-4000-8000-000000000003"), Role: "driver"})
	assertErrIs(t, "driver ListManagers", err, app.ErrForbidden)
}

// Stop colour thresholds are parameters (data, not code): 15/45 by
// default, non-negative, and warn never above alert.
func TestStopThresholdParams(t *testing.T) {
	params, err := svc.GetParams(ctx, adminActor())
	if err != nil {
		t.Fatal(err)
	}
	byKey := map[string]store.Param{}
	for _, p := range params {
		byKey[p.Key] = p
	}
	if byKey["stop_warn_minutes"].Value != "15.0000" || byKey["stop_alert_minutes"].Value != "45.0000" ||
		byKey["stop_warn_minutes"].Unit != "minutes" || byKey["stop_alert_minutes"].Unit != "minutes" {
		t.Errorf("thresholds = %+v / %+v", byKey["stop_warn_minutes"], byKey["stop_alert_minutes"])
	}
	set := func(key, value string) error {
		_, err := svc.UpdateParam(ctx, managerActor(), app.UpdateParamInput{Key: key, Value: value})
		return err
	}
	var fe *app.FieldError
	err = set("stop_warn_minutes", "50")
	assertErrIs(t, "warn above alert", err, app.ErrValidation)
	if !errors.As(err, &fe) || fe.Field != "value" {
		t.Errorf("warn above alert = %v, want field value", err)
	}
	assertErrIs(t, "alert below warn", set("stop_alert_minutes", "10"), app.ErrValidation)
	assertErrIs(t, "negative warn", set("stop_warn_minutes", "-1"), app.ErrValidation)
	if err := set("stop_warn_minutes", "45"); err != nil {
		t.Errorf("warn equal to alert rejected: %v", err)
	}
	if err := set("stop_alert_minutes", "60"); err != nil {
		t.Errorf("raise alert: %v", err)
	}
	days, err := svc.GetDashboardByDay(ctx, managerActor(), app.DashboardInput{From: "2026-06-01", To: "2026-06-30"})
	if err != nil || days.StopWarnMinutes != "45.0000" || days.StopAlertMinutes != "60.0000" {
		t.Errorf("day thresholds after update = %q/%q, %v", days.StopWarnMinutes, days.StopAlertMinutes, err)
	}
	for _, kv := range [][2]string{{"stop_warn_minutes", "15"}, {"stop_alert_minutes", "45"}} {
		if err := set(kv[0], kv[1]); err != nil {
			t.Fatalf("restore %s: %v", kv[0], err)
		}
	}
}
