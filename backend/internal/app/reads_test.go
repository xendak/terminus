package app_test

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"stoptime/internal/app"
	"stoptime/internal/domain"
	"stoptime/internal/store"
)

// Read services against the golden fixture (day/month/period 161,
// routes A/B/C = 75/41/45 — business-rules.md "Golden fixture").
// This file sorts before service_test.go, so these tests see the golden
// seed plus nothing else.

func goldenRouteID(t *testing.T, driverEmail string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := testDB.QueryRow(ctx, `
SELECT r.id FROM route r JOIN app_user u ON u.id = r.driver_user_id
 WHERE u.email = $1 AND r.route_date = DATE '2026-06-15'`, driverEmail).Scan(&id)
	if err != nil {
		t.Fatalf("golden route for %s: %v", driverEmail, err)
	}
	return id
}

func TestGetRouteGolden(t *testing.T) {
	v, err := svc.GetRoute(ctx, adminActor(), goldenRouteID(t, "driver-a@stoptime.dev"))
	if err != nil {
		t.Fatalf("GetRoute: %v", err)
	}
	if v.DriverName != "Marcos Motorista" {
		t.Errorf("driver name = %q", v.DriverName)
	}
	if v.RouteDate.Format("2006-01-02") != "2026-06-15" || v.Status != "closed" {
		t.Errorf("route = %s %s", v.RouteDate.Format("2006-01-02"), v.Status)
	}
	if v.DistanceKm != nil || v.EstimatedCostBRL != nil {
		t.Errorf("no distance: distance=%v cost=%v, want nil cost (not zero)", v.DistanceKm, v.EstimatedCostBRL)
	}
	if len(v.Stops) != 4 {
		t.Fatalf("stops = %d, want 4", len(v.Stops))
	}
	// Stop 1: departure point — never counted, contributes 0 even with
	// timestamps (RN01).
	if v.Stops[0].StopOrder != 1 || v.Stops[0].Counted || v.Stops[0].StopSeconds == nil || *v.Stops[0].StopSeconds != 0 {
		t.Errorf("stop 1 = %+v, want counted=false stop_seconds=0", v.Stops[0])
	}
	if v.Stops[0].Label != "Seg. Família" {
		t.Errorf("stop 1 label = %q", v.Stops[0].Label)
	}
	want := []struct {
		order  int
		label  string
		secs   int
	}{
		{2, "Ponto A2", 900},
		{3, "Ponto A3", 600},
		{4, "Ponto A4", 3000},
	}
	for i, w := range want {
		st := v.Stops[i+1]
		if st.StopOrder != w.order || !st.Counted || st.Label != w.label ||
			st.StopSeconds == nil || *st.StopSeconds != w.secs {
			t.Errorf("stop %d = %+v, want order %d %s %ds counted", i+2, st, w.order, w.label, w.secs)
		}
	}
	if v.Stops[3].Address != "Av. João César" {
		t.Errorf("stop 4 address = %q, want the tp.md address verbatim", v.Stops[3].Address)
	}

	// SQL-computed totals (RN03, RN04) and the domain oracle agree.
	if v.TotalSeconds != 4500 || v.TotalStoppedMinut != 75 {
		t.Errorf("totals = %ds / %dmin, want 4500s / 75min", v.TotalSeconds, v.TotalStoppedMinut)
	}
	if v.JourneyPercent != "15.625" {
		t.Errorf("journey percent = %q, want 15.625", v.JourneyPercent)
	}
	domainStops := make([]domain.Stop, len(v.Stops))
	for i, st := range v.Stops {
		domainStops[i] = domain.Stop{Order: st.StopOrder, Arrival: st.ArrivalAt, Departure: st.DepartureAt}
	}
	if got := domain.RouteTotalSeconds(domain.Route{Stops: domainStops}, 0); int64(got) != v.TotalSeconds {
		t.Errorf("domain oracle total = %d, SQL total = %d", got, v.TotalSeconds)
	}
}

func TestListRoutesGolden(t *testing.T) {
	rows, err := svc.ListRoutes(ctx, adminActor(), app.ListRoutesInput{
		From: ptr("2026-06-01"), To: ptr("2026-06-30"),
	})
	if err != nil {
		t.Fatalf("ListRoutes: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("ListRoutes = %d rows, want 3", len(rows))
	}
	// Same date, ordered by driver name: Bianca (41), Carla (45), Marcos (75).
	want := []struct {
		name, percent string
		minutes       int
	}{
		{"Bianca Batista", "8.542", 41},
		{"Carla Camargo", "9.375", 45},
		{"Marcos Motorista", "15.625", 75},
	}
	for i, w := range want {
		r := rows[i]
		if r.DriverName != w.name || r.TotalStoppedMinut != w.minutes ||
			r.JourneyPercent != w.percent || r.Status != "closed" || r.StopCount != 4 {
			t.Errorf("row %d = %+v, want %s %dmin %s", i, r, w.name, w.minutes, w.percent)
		}
	}

	// "Own only" (enforced, not requested): the driver's query is scoped
	// to their routes without passing a filter.
	drv := goldenRouteID(t, "driver-a@stoptime.dev")
	var driverID uuid.UUID
	if err := testDB.QueryRow(ctx, `SELECT driver_user_id FROM route WHERE id = $1`, drv).Scan(&driverID); err != nil {
		t.Fatal(err)
	}
	mine, err := svc.ListRoutes(ctx, app.Actor{UserID: driverID, Role: "driver"}, app.ListRoutesInput{
		From: ptr("2026-06-01"), To: ptr("2026-06-30"),
	})
	if err != nil {
		t.Fatalf("ListRoutes scoped: %v", err)
	}
	if len(mine) != 1 || mine[0].TotalStoppedMinut != 75 {
		t.Errorf("scoped ListRoutes = %+v, want one 75-minute route", mine)
	}

	// Status filter.
	drafts, err := svc.ListRoutes(ctx, adminActor(), app.ListRoutesInput{
		From: ptr("2026-06-01"), To: ptr("2026-06-30"), Status: ptr("draft"),
	})
	if err != nil {
		t.Fatalf("ListRoutes by status: %v", err)
	}
	if len(drafts) != 0 {
		t.Errorf("draft filter = %d rows, want 0", len(drafts))
	}
}

func TestDashboardDayGolden(t *testing.T) {
	in := app.DashboardInput{From: "2026-06-01", To: "2026-06-30"}
	points, err := svc.GetDashboardByDay(ctx, adminActor(), in)
	if err != nil {
		t.Fatalf("GetDashboardByDay: %v", err)
	}
	if len(points.Series) != 1 {
		t.Fatalf("day series = %d points, want 1 (the golden day)", len(points.Series))
	}
	if points.StandardJourneyHours != "8.0000" {
		t.Errorf("day standard_journey_hours = %q, want the parameter text 8.0000", points.StandardJourneyHours)
	}
	if points.Series[0].Date.Format("2006-01-02") != "2026-06-15" || points.Series[0].TotalStoppedMinut != 161 ||
		points.Series[0].JourneyPercent != "11.181" {
		t.Errorf("day point = %s %d %s, want 2026-06-15 161 11.181", points.Series[0].Date.Format("2006-01-02"),
			points.Series[0].TotalStoppedMinut, points.Series[0].JourneyPercent)
	}

	// "Own data only" (enforced): the driver actor sees just their day,
	// with no filter passed.
	var driverID uuid.UUID
	err = testDB.QueryRow(ctx, `SELECT id FROM app_user WHERE email = 'driver-b@stoptime.dev'`).Scan(&driverID)
	if err != nil {
		t.Fatal(err)
	}
	mine, err := svc.GetDashboardByDay(ctx, app.Actor{UserID: driverID, Role: "driver"}, app.DashboardInput{From: "2026-06-01", To: "2026-06-30"})
	if err != nil {
		t.Fatalf("GetDashboardByDay scoped: %v", err)
	}
	if len(mine.Series) != 1 || mine.Series[0].TotalStoppedMinut != 41 {
		t.Errorf("scoped day series = %+v, want one 41-minute point", mine.Series)
	}

	// A window without data has no points.
	empty, err := svc.GetDashboardByDay(ctx, adminActor(), app.DashboardInput{From: "2026-05-01", To: "2026-05-31"})
	if err != nil {
		t.Fatalf("GetDashboardByDay empty: %v", err)
	}
	if len(empty.Series) != 0 {
		t.Errorf("empty window = %d points, want 0", len(empty.Series))
	}
}

func TestDashboardMonthGolden(t *testing.T) {
	points, err := svc.GetDashboardByMonth(ctx, adminActor(), app.DashboardInput{From: "2026-01-01", To: "2026-12-31"})
	if err != nil {
		t.Fatalf("GetDashboardByMonth: %v", err)
	}
	if points.StandardJourneyHours != "8.0000" {
		t.Errorf("month standard_journey_hours = %q, want 8.0000", points.StandardJourneyHours)
	}
	if len(points.Series) != 1 || points.Series[0].Month != "2026-06" || points.Series[0].TotalStoppedMinut != 161 ||
		points.Series[0].JourneyPercent != "11.181" {
		t.Errorf("month series = %+v, want one {2026-06, 161, 11.181}", points)
	}
}

func TestDashboardPeriodGolden(t *testing.T) {
	summary, err := svc.GetDashboardByPeriod(ctx, adminActor(), app.DashboardInput{From: "2026-06-01", To: "2026-06-30"})
	if err != nil {
		t.Fatalf("GetDashboardByPeriod: %v", err)
	}
	// 9660 s over 3 routes × 8h (one standard day per route, RN04/RN05):
	// 11.180555... -> 11.181 (rounded once, in SQL).
	if summary.TotalStoppedMinut != 161 || summary.JourneyPercent != "11.181" || summary.RoutesCount != 3 {
		t.Errorf("summary = %+v, want 161min 11.181 3 routes", summary)
	}
	if summary.StandardJourneyHours != "8.0000" {
		t.Errorf("period standard_journey_hours = %q, want 8.0000", summary.StandardJourneyHours)
	}
	wantDrivers := []struct {
		name     string
		minutes  int
		percent  string
	}{
		{"Bianca Batista", 41, "8.542"},
		{"Carla Camargo", 45, "9.375"},
		{"Marcos Motorista", 75, "15.625"},
	}
	if len(summary.ByDriver) != 3 {
		t.Fatalf("by_driver = %d rows, want 3", len(summary.ByDriver))
	}
	for i, w := range wantDrivers {
		d := summary.ByDriver[i]
		if d.DriverName != w.name || d.TotalStoppedMinut != w.minutes || d.JourneyPercent != w.percent {
			t.Errorf("by_driver[%d] = %+v, want %s %d %s", i, d, w.name, w.minutes, w.percent)
		}
	}
}

// The period base counts one standard day per route: a driver with two
// routes (60 + 30 min) reads 5400 / (2 × 8h) = 9.375%, not 18.750%. A
// planned route with nothing recorded is not a worked day and stays out
// of the base. Scoped to a fresh driver, so other suites' data is
// invisible.
func TestDashboardPeriodPerRouteBase(t *testing.T) {
	drv := createDriver(t, "period-base-"+uuid.NewString()[:8])
	locs := createLocations(t, adminActor(), 2)
	record := func(date string, arrive, depart time.Time) {
		t.Helper()
		r, err := svc.CreateRoute(ctx, adminActor(), app.CreateRouteInput{
			DriverUserID: drv.ID, RouteDate: date, LocationIDs: locationIDs(locs),
		})
		if err != nil {
			t.Fatalf("CreateRoute %s: %v", date, err)
		}
		if _, err := svc.StartRoute(ctx, adminActor(), r.Route.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.RecordArrival(ctx, adminActor(), app.RecordTimeInput{RouteID: r.Route.ID, StopOrder: 2, At: &arrive}); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.RecordDeparture(ctx, adminActor(), app.RecordTimeInput{RouteID: r.Route.ID, StopOrder: 2, At: &depart}); err != nil {
			t.Fatal(err)
		}
	}
	d1 := time.Date(2025, time.May, 5, 12, 0, 0, 0, time.UTC)
	d2 := time.Date(2025, time.May, 6, 12, 0, 0, 0, time.UTC)
	record("2025-05-05", d1, d1.Add(60*time.Minute))
	record("2025-05-06", d2, d2.Add(30*time.Minute))
	// A draft for the next day: no times, not a worked day.
	if _, err := svc.CreateRoute(ctx, adminActor(), app.CreateRouteInput{
		DriverUserID: drv.ID, RouteDate: "2025-05-07", LocationIDs: locationIDs(locs),
	}); err != nil {
		t.Fatal(err)
	}

	in := app.DashboardInput{From: "2025-05-01", To: "2025-05-31", DriverUserID: &drv.ID}
	summary, err := svc.GetDashboardByPeriod(ctx, adminActor(), in)
	if err != nil {
		t.Fatalf("GetDashboardByPeriod: %v", err)
	}
	if summary.TotalStoppedMinut != 90 || summary.RoutesCount != 2 || summary.JourneyPercent != "9.375" {
		t.Errorf("summary = %+v, want 90min 2 routes 9.375", summary)
	}
	if len(summary.ByDriver) != 1 || summary.ByDriver[0].JourneyPercent != "9.375" {
		t.Errorf("by_driver = %+v, want one row at 9.375", summary.ByDriver)
	}

	// Day points carry the same base per day: 60/480 and 30/480.
	points, err := svc.GetDashboardByDay(ctx, adminActor(), in)
	if err != nil {
		t.Fatalf("GetDashboardByDay: %v", err)
	}
	if len(points.Series) != 2 || points.Series[0].JourneyPercent != "12.500" || points.Series[1].JourneyPercent != "6.250" {
		t.Errorf("day points = %+v, want 12.500 and 6.250", points)
	}

	// The month bucket holds both routes: 5400 / (2 × 8h) = 9.375%.
	months, err := svc.GetDashboardByMonth(ctx, adminActor(), in)
	if err != nil {
		t.Fatalf("GetDashboardByMonth: %v", err)
	}
	if len(months.Series) != 1 || months.Series[0].TotalStoppedMinut != 90 || months.Series[0].JourneyPercent != "9.375" {
		t.Errorf("month points = %+v, want one {90, 9.375}", months)
	}
}

func TestDashboardPeriodEmpty(t *testing.T) {
	summary, err := svc.GetDashboardByPeriod(ctx, adminActor(), app.DashboardInput{From: "2020-01-01", To: "2020-01-31"})
	if err != nil {
		t.Fatalf("GetDashboardByPeriod empty: %v", err)
	}
	if summary.TotalStoppedMinut != 0 || summary.RoutesCount != 0 || summary.JourneyPercent != "0.000" ||
		summary.ByDriver == nil || len(summary.ByDriver) != 0 {
		t.Errorf("empty summary = %+v, want zeros and an empty by_driver", summary)
	}
}

// recordRoute creates, starts and records one counted stop (stop 2)
// on a fresh route for drv; the stop lasts the given minutes.
func recordRoute(t *testing.T, drv store.Driver, locs []store.Location, date string, arrive time.Time, minutes int) {
	t.Helper()
	r, err := svc.CreateRoute(ctx, adminActor(), app.CreateRouteInput{
		DriverUserID: drv.ID, RouteDate: date, LocationIDs: locationIDs(locs),
	})
	if err != nil {
		t.Fatalf("CreateRoute %s: %v", date, err)
	}
	if _, err := svc.StartRoute(ctx, adminActor(), r.Route.ID); err != nil {
		t.Fatal(err)
	}
	depart := arrive.Add(time.Duration(minutes) * time.Minute)
	if _, err := svc.RecordArrival(ctx, adminActor(), app.RecordTimeInput{RouteID: r.Route.ID, StopOrder: 2, At: &arrive}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RecordDeparture(ctx, adminActor(), app.RecordTimeInput{RouteID: r.Route.ID, StopOrder: 2, At: &depart}); err != nil {
		t.Fatal(err)
	}
}

// H1: when min_stop_minutes exceeds every stop of a bucket, the bucket
// still answers (0 minutes, 0.000%) instead of a NULL scan error; its
// recorded routes remain worked days in the base.
func TestDashboardAllStopsBelowMinimum(t *testing.T) {
	drv := createDriver(t, "below-min-"+uuid.NewString()[:8])
	locs := createLocations(t, adminActor(), 2)
	d := time.Date(2025, time.April, 7, 12, 0, 0, 0, time.UTC)
	recordRoute(t, drv, locs, "2025-04-07", d, 10)
	recordRoute(t, drv, locs, "2025-04-08", d.AddDate(0, 0, 1), 12)

	if _, err := svc.UpdateParam(ctx, adminActor(), app.UpdateParamInput{Key: "min_stop_minutes", Value: "30"}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := svc.UpdateParam(ctx, adminActor(), app.UpdateParamInput{Key: "min_stop_minutes", Value: "0"}); err != nil {
			t.Fatal(err)
		}
	})

	in := app.DashboardInput{From: "2025-04-01", To: "2025-04-30", DriverUserID: &drv.ID}
	days, err := svc.GetDashboardByDay(ctx, adminActor(), in)
	if err != nil {
		t.Fatalf("GetDashboardByDay below minimum: %v", err)
	}
	if len(days.Series) != 2 || days.Series[0].TotalStoppedMinut != 0 || days.Series[0].JourneyPercent != "0.000" {
		t.Errorf("day series = %+v, want 2 zero points", days.Series)
	}
	months, err := svc.GetDashboardByMonth(ctx, adminActor(), in)
	if err != nil {
		t.Fatalf("GetDashboardByMonth below minimum: %v", err)
	}
	if len(months.Series) != 1 || months.Series[0].TotalStoppedMinut != 0 || months.Series[0].JourneyPercent != "0.000" {
		t.Errorf("month series = %+v, want one zero point", months.Series)
	}
	summary, err := svc.GetDashboardByPeriod(ctx, adminActor(), in)
	if err != nil {
		t.Fatalf("GetDashboardByPeriod below minimum: %v", err)
	}
	if summary.TotalStoppedMinut != 0 || summary.JourneyPercent != "0.000" || summary.RoutesCount != 2 {
		t.Errorf("summary = %+v, want 0 min, 0.000, 2 routes", summary)
	}
}

// LOW5: the SQL percents equal the domain oracle (rounded to 3 places)
// for the golden window and a multi-route fixture.
func TestJourneyPercentMatchesOracle(t *testing.T) {
	hours := new(big.Rat)
	hours.SetString("8")
	check := func(name string, seconds, routes int, got string) {
		t.Helper()
		want, err := domain.PeriodJourneyPercent(seconds, routes, hours)
		if err != nil {
			t.Fatalf("%s oracle: %v", name, err)
		}
		if got != want.FloatString(3) {
			t.Errorf("%s: SQL %s, oracle %s", name, got, want.FloatString(3))
		}
	}

	golden := app.DashboardInput{From: "2026-06-01", To: "2026-06-30"}
	summary, err := svc.GetDashboardByPeriod(ctx, adminActor(), golden)
	if err != nil {
		t.Fatal(err)
	}
	check("golden period", 9660, 3, summary.JourneyPercent)
	days, err := svc.GetDashboardByDay(ctx, adminActor(), golden)
	if err != nil || len(days.Series) != 1 {
		t.Fatalf("golden day: %v %+v", err, days)
	}
	check("golden day", 9660, 3, days.Series[0].JourneyPercent)
	months, err := svc.GetDashboardByMonth(ctx, adminActor(), golden)
	if err != nil || len(months.Series) != 1 {
		t.Fatalf("golden month: %v %+v", err, months)
	}
	check("golden month", 9660, 3, months.Series[0].JourneyPercent)

	// Three routes of one driver in one month: 25 + 40 + 7 min.
	drv := createDriver(t, "oracle-"+uuid.NewString()[:8])
	locs := createLocations(t, adminActor(), 2)
	d := time.Date(2025, time.March, 3, 12, 0, 0, 0, time.UTC)
	recordRoute(t, drv, locs, "2025-03-03", d, 25)
	recordRoute(t, drv, locs, "2025-03-04", d.AddDate(0, 0, 1), 40)
	recordRoute(t, drv, locs, "2025-03-05", d.AddDate(0, 0, 2), 7)
	in := app.DashboardInput{From: "2025-03-01", To: "2025-03-31", DriverUserID: &drv.ID}
	summary, err = svc.GetDashboardByPeriod(ctx, adminActor(), in)
	if err != nil {
		t.Fatal(err)
	}
	check("fixture period", 72*60, 3, summary.JourneyPercent)
	check("fixture by_driver", 72*60, 3, summary.ByDriver[0].JourneyPercent)
	months, err = svc.GetDashboardByMonth(ctx, adminActor(), in)
	if err != nil || len(months.Series) != 1 {
		t.Fatalf("fixture month: %v %+v", err, months)
	}
	check("fixture month", 72*60, 3, months.Series[0].JourneyPercent)
	days, err = svc.GetDashboardByDay(ctx, adminActor(), in)
	if err != nil || len(days.Series) != 3 {
		t.Fatalf("fixture day: %v %+v", err, days)
	}
	check("fixture day 3", 7*60, 1, days.Series[2].JourneyPercent)
}

func TestReadsValidation(t *testing.T) {
	_, err := svc.GetDashboardByDay(ctx, adminActor(), app.DashboardInput{From: "2026-06-01", To: "junk"})
	assertErrIs(t, "bad to", err, app.ErrBadInput)
	_, err = svc.GetDashboardByDay(ctx, adminActor(), app.DashboardInput{From: "", To: "2026-06-01"})
	assertErrIs(t, "missing from", err, app.ErrBadInput)
	_, err = svc.GetDashboardByDay(ctx, adminActor(), app.DashboardInput{From: "2026-06-30", To: "2026-06-01"})
	assertErrIs(t, "inverted window", err, app.ErrValidation)
	_, err = svc.GetRoute(ctx, adminActor(), uuid.New())
	assertErrIs(t, "unknown route", err, app.ErrNotFound)
}

// --- the 3-second rule (RNF03) -----------------------------------------

// generateSyntheticYears seeds 36 months of recorded routes (2022-2024,
// disjoint from the golden fixture and the service-test data) in one
// statement per table: 6 drivers, ~782 workdays, 6 stops each — about
// 4700 routes and 28k stops, far above course scale.
func generateSyntheticYears(t *testing.T) {
	t.Helper()
	_, err := testDB.Exec(ctx, `
INSERT INTO app_user (id, name, email, phone, password_hash, role)
SELECT ('dd000000-0000-4000-8000-00000000000' || d)::uuid,
       'Syn Driver ' || d, 'syn-' || d || '@perf.test', '0', 'x', 'driver'
  FROM generate_series(1, 6) AS d`)
	must(err)
	_, err = testDB.Exec(ctx, `
INSERT INTO driver_profile (user_id, km_per_l)
SELECT ('dd000000-0000-4000-8000-00000000000' || d)::uuid, 10.00 + d
  FROM generate_series(1, 6) AS d`)
	must(err)
	_, err = testDB.Exec(ctx, `
INSERT INTO location (id, label, address, created_by)
SELECT ('ee000000-0000-4000-8000-00000000000' || l)::uuid,
       'Ponto S' || l, 'Endereço S' || l, $1
  FROM generate_series(1, 6) AS l`, admin)
	must(err)
	_, err = testDB.Exec(ctx, `
WITH ins AS (
  INSERT INTO route (id, driver_user_id, route_date, status, distance_km, note, created_by)
  SELECT gen_random_uuid(),
         ('dd000000-0000-4000-8000-00000000000' || d)::uuid,
         g::date,
         'closed',
         10 + ((g::date - DATE '2022-01-01') % 80),
         'synthetic',
         $1
    FROM generate_series(DATE '2022-01-01', DATE '2024-12-31', INTERVAL '1 day') AS g,
         generate_series(1, 6) AS d
   WHERE extract(isodow FROM g) < 6
  RETURNING id, route_date
)
INSERT INTO route_stop (id, route_id, stop_order, location_id, arrival_at, departure_at)
SELECT gen_random_uuid(), ins.id, s,
       ('ee000000-0000-4000-8000-00000000000' || ((s % 6) + 1))::uuid,
       ins.route_date + TIME '07:00' + (s * INTERVAL '45 minutes'),
       ins.route_date + TIME '07:00' + (s * INTERVAL '45 minutes')
         + (((s * 7) % 25 + 4) * INTERVAL '1 minute')
  FROM ins CROSS JOIN generate_series(1, 6) AS s`, admin)
	must(err)
}

// explainDay runs EXPLAIN ANALYZE on the exact day-series query the
// service uses (store.DashboardByDaySQL — no drift) and returns the plan.
func explainDay(t *testing.T, from, to string) string {
	t.Helper()
	rows, err := testDB.Query(context.Background(),
		"EXPLAIN (ANALYZE) "+store.DashboardByDaySQL, from, to, nil)
	if err != nil {
		t.Fatalf("explain: %v", err)
	}
	defer rows.Close()
	var b strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatalf("explain scan: %v", err)
		}
		fmt.Fprintln(&b, line)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("explain rows: %v", err)
	}
	return b.String()
}

func TestDashboardPerfTwelveMonths(t *testing.T) {
	generateSyntheticYears(t)

	// The 12-month dashboards (year 2024 of the synthetic data) must
	// answer well under 3 seconds (RNF03).
	window := app.DashboardInput{From: "2024-01-01", To: "2024-12-31"}
	start := time.Now()
	day, err := svc.GetDashboardByDay(ctx, adminActor(), window)
	dayElapsed := time.Since(start)
	if err != nil {
		t.Fatalf("GetDashboardByDay: %v", err)
	}
	start = time.Now()
	month, err := svc.GetDashboardByMonth(ctx, adminActor(), window)
	monthElapsed := time.Since(start)
	if err != nil {
		t.Fatalf("GetDashboardByMonth: %v", err)
	}
	start = time.Now()
	period, err := svc.GetDashboardByPeriod(ctx, adminActor(), window)
	periodElapsed := time.Since(start)
	if err != nil {
		t.Fatalf("GetDashboardByPeriod: %v", err)
	}
	t.Logf("12-month dashboards: day(%d points) %s, month(%d points) %s, period(%d routes) %s",
		len(day.Series), dayElapsed, len(month.Series), monthElapsed, period.RoutesCount, periodElapsed)
	for _, m := range []struct {
		name    string
		elapsed time.Duration
	}{
		{"day", dayElapsed}, {"month", monthElapsed}, {"period", periodElapsed},
	} {
		if m.elapsed > 3*time.Second {
			t.Errorf("%s dashboard took %s, want under 3s", m.name, m.elapsed)
		}
	}
	if len(month) != 12 {
		t.Errorf("month series = %d points, want 12", len(month))
	}

	// The pattern guard: the date-windowed query uses the route(route_date)
	// index. A window covering ~3% of the data must not seq-scan route.
	// (A full-coverage window legitimately seq-scans; selectivity is what
	// makes the index the right plan — see plans/mvp/notes.md.)
	plan := explainDay(t, "2024-06-01", "2024-06-30")
	t.Logf("EXPLAIN ANALYZE (one-month window):\n%s", plan)
	if !strings.Contains(plan, "route_route_date_idx") {
		t.Errorf("day query plan does not use route_route_date_idx:\n%s", plan)
	}
	if strings.Contains(plan, "Seq Scan on route ") {
		t.Errorf("day query plan seq-scans route:\n%s", plan)
	}

	// The year-wide EXPLAIN goes in the log too (the card asks for the
	// plan printed in the test log).
	t.Logf("EXPLAIN ANALYZE (12-month window):\n%s", explainDay(t, "2024-01-01", "2024-12-31"))
}
