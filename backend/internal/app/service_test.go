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
	svc = app.New(st)

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

func createManager(t *testing.T, name string) store.User {
	t.Helper()
	u, err := svc.CreateManager(ctx, app.CreateManagerInput{
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
	d, err := svc.CreateDriver(ctx, app.CreateDriverInput{
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
	d2, err := svc.CreateDriver(ctx, app.CreateDriverInput{
		Name: "Driver With Profile", Email: "profile@test.dev", Password: "pw", Phone: "0",
		Document: &doc, VehicleName: ptr("Fiorino"), VehiclePlate: ptr("ABC1D23"), KmPerL: &km,
	})
	if err != nil {
		t.Fatalf("CreateDriver with profile: %v", err)
	}
	if d2.KmPerL == nil || *d2.KmPerL != "12.50" {
		t.Errorf("km_per_l round-trip = %v, want 12.50", d2.KmPerL)
	}

	// Duplicate emails across roles hit the same lower(email) index.
	_, err = svc.CreateDriver(ctx, app.CreateDriverInput{
		Name: "Dup", Email: mgr.Email, Password: "pw", Phone: "0",
	})
	assertErrIs(t, "duplicate manager email as driver", err, app.ErrDuplicateEmail)
	_, err = svc.CreateManager(ctx, app.CreateManagerInput{
		Name: "Dup", Email: "driver-accounts@test.dev", Password: "pw", Phone: "0",
	})
	assertErrIs(t, "duplicate driver email as manager", err, app.ErrDuplicateEmail)

	// Validation.
	_, err = svc.CreateDriver(ctx, app.CreateDriverInput{Name: "", Email: "x@test.dev", Password: "pw", Phone: "0"})
	assertErrIs(t, "missing name", err, app.ErrValidation)
	var fe *app.FieldError
	if !errors.As(err, &fe) || fe.Field != "name" {
		t.Errorf("missing name error = %v, want FieldError name", err)
	}
	_, err = svc.CreateDriver(ctx, app.CreateDriverInput{
		Name: "X", Email: "x@test.dev", Password: "pw", Phone: "0", KmPerL: ptr("abc"),
	})
	assertErrIs(t, "bad km_per_l", err, app.ErrBadInput)
	_, err = svc.CreateDriver(ctx, app.CreateDriverInput{
		Name: "X", Email: "x@test.dev", Password: "pw", Phone: "0", KmPerL: ptr("0"),
	})
	assertErrIs(t, "zero km_per_l", err, app.ErrValidation)

	// Update + active_only listing (LGPD deactivation path).
	inactive := false
	if _, err := svc.UpdateDriver(ctx, app.UpdateDriverInput{DriverID: drv.ID, Active: &inactive}); err != nil {
		t.Fatalf("UpdateDriver deactivate: %v", err)
	}
	activeOnly, err := svc.ListDrivers(ctx, true)
	if err != nil {
		t.Fatalf("ListDrivers: %v", err)
	}
	for _, d := range activeOnly {
		if d.ID == drv.ID {
			t.Error("deactivated driver still listed with activeOnly=true")
		}
	}
	all, err := svc.ListDrivers(ctx, false)
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
	_, err = svc.UpdateDriver(ctx, app.UpdateDriverInput{DriverID: uuid.New()})
	assertErrIs(t, "update unknown driver", err, app.ErrNotFound)

	managers, err := svc.ListManagers(ctx)
	if err != nil {
		t.Fatalf("ListManagers: %v", err)
	}
	if len(managers) == 0 {
		t.Error("ListManagers empty")
	}

	// Locations.
	actor := actorOf(mgr)
	locs := createLocations(t, actor, 2)
	updated, err := svc.UpdateLocation(ctx, app.UpdateLocationInput{
		LocationID: locs[0].ID, Label: ptr("Renamed"),
	})
	if err != nil {
		t.Fatalf("UpdateLocation: %v", err)
	}
	if updated.Label != "Renamed" || updated.Address != locs[0].Address {
		t.Errorf("UpdateLocation partial: %+v", updated)
	}
	listed, err := svc.ListLocations(ctx, ptr("Renamed"))
	if err != nil {
		t.Fatalf("ListLocations: %v", err)
	}
	if len(listed) != 1 {
		t.Errorf("ListLocations(q) = %d rows, want 1", len(listed))
	}
	_, err = svc.UpdateLocation(ctx, app.UpdateLocationInput{LocationID: uuid.New(), Label: ptr("X")})
	assertErrIs(t, "update unknown location", err, app.ErrNotFound)
}

// --- route creation ----------------------------------------------------

func TestCreateRouteValidation(t *testing.T) {
	mgr := createManager(t, "manager-routes")
	actor := actorOf(mgr)
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
	actor := actorOf(mgr)
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
	actor := actorOf(mgr)
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
	if _, err := svc.StartRoute(ctx, routeID); err != nil {
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
		if _, err := svc.RecordArrival(ctx, app.RecordTimeInput{RouteID: routeID, StopOrder: d.order, At: d.arrival}); err != nil {
			t.Fatalf("RecordArrival %d: %v", d.order, err)
		}
		st, err := svc.RecordDeparture(ctx, app.RecordTimeInput{RouteID: routeID, StopOrder: d.order, At: d.depart})
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
	_, err = svc.RecordArrival(ctx, app.RecordTimeInput{RouteID: routeID, StopOrder: 2, At: at(12, 0)})
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
	_, err = svc.RecordArrival(ctx, app.RecordTimeInput{RouteID: routeID, StopOrder: 2, At: at(13, 0)})
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
	_, err = svc.SetRouteDistance(ctx, routeID, "50")
	assertErrIs(t, "distance on closed", err, app.ErrRouteClosed)

	// Reopen: the day continues; audit records it.
	if _, err := svc.ReopenRoute(ctx, actorOf(mgr), routeID); err != nil {
		t.Fatalf("ReopenRoute: %v", err)
	}
	if _, err := svc.SetRouteDistance(ctx, routeID, "100"); err != nil {
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
	actor := actorOf(mgr)
	drv := createDriver(t, "driver-rn02")
	locs := createLocations(t, actor, 2)
	route, err := svc.CreateRoute(ctx, actor, app.CreateRouteInput{
		DriverUserID: drv.ID, RouteDate: "2026-07-05", LocationIDs: locationIDs(locs),
	})
	if err != nil {
		t.Fatalf("CreateRoute: %v", err)
	}
	if _, err := svc.StartRoute(ctx, route.Route.ID); err != nil {
		t.Fatalf("StartRoute: %v", err)
	}

	// Departure before arrival, rejected by the service (RN02).
	_, err = svc.RecordArrival(ctx, app.RecordTimeInput{RouteID: route.Route.ID, StopOrder: 2, At: at(10, 0)})
	if err != nil {
		t.Fatalf("RecordArrival: %v", err)
	}
	_, err = svc.RecordDeparture(ctx, app.RecordTimeInput{RouteID: route.Route.ID, StopOrder: 2, At: at(9, 0)})
	assertErrIs(t, "departure before arrival", err, app.ErrDepartureBeforeArrival)

	// Departure without arrival.
	_, err = svc.RecordDeparture(ctx, app.RecordTimeInput{RouteID: route.Route.ID, StopOrder: 1, At: at(9, 30)})
	assertErrIs(t, "departure without arrival", err, app.ErrValidation)

	// Recording on a draft route is refused.
	draftRoute, err := svc.CreateRoute(ctx, actor, app.CreateRouteInput{
		DriverUserID: drv.ID, RouteDate: "2026-07-06", LocationIDs: locationIDs(locs),
	})
	if err != nil {
		t.Fatalf("CreateRoute draft: %v", err)
	}
	_, err = svc.RecordArrival(ctx, app.RecordTimeInput{RouteID: draftRoute.Route.ID, StopOrder: 1, At: at(9, 0)})
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
	params, err := svc.GetParams(ctx)
	if err != nil {
		t.Fatalf("GetParams: %v", err)
	}
	if len(params) != 5 {
		t.Fatalf("GetParams = %d rows, want 5", len(params))
	}
	byKey := map[string]string{}
	for _, p := range params {
		byKey[p.Key] = p.Value
	}
	if byKey["fuel_price_brl"] != "6.0900" {
		t.Errorf("fuel_price_brl = %q, want 6.0900", byKey["fuel_price_brl"])
	}

	actor := adminActor()
	updated, err := svc.UpdateParam(ctx, actor, app.UpdateParamInput{Key: "fuel_price_brl", Value: "6.19"})
	if err != nil {
		t.Fatalf("UpdateParam: %v", err)
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
	actor := actorOf(mgr)
	drv := createDriver(t, "driver-distance")
	locs := createLocations(t, actor, 2)

	draft, err := svc.CreateRoute(ctx, actor, app.CreateRouteInput{
		DriverUserID: drv.ID, RouteDate: "2026-07-07", LocationIDs: locationIDs(locs),
	})
	if err != nil {
		t.Fatalf("CreateRoute: %v", err)
	}
	_, err = svc.SetRouteDistance(ctx, draft.Route.ID, "75.50")
	assertErrIs(t, "distance on draft", err, app.ErrValidation)

	if _, err := svc.StartRoute(ctx, draft.Route.ID); err != nil {
		t.Fatalf("StartRoute: %v", err)
	}
	r, err := svc.SetRouteDistance(ctx, draft.Route.ID, "75.50")
	if err != nil {
		t.Fatalf("SetRouteDistance: %v", err)
	}
	if r.DistanceKm == nil || *r.DistanceKm != "75.50" {
		t.Errorf("distance round-trip = %v, want 75.50", r.DistanceKm)
	}

	_, err = svc.SetRouteDistance(ctx, draft.Route.ID, "0")
	assertErrIs(t, "zero distance", err, app.ErrValidation)
	_, err = svc.SetRouteDistance(ctx, draft.Route.ID, "abc")
	assertErrIs(t, "bad distance", err, app.ErrBadInput)
	_, err = svc.SetRouteDistance(ctx, uuid.New(), "10")
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
