package app_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"stoptime/internal/app"
	"stoptime/internal/store"
)

// The role matrix, walked cell by cell (T6): every operation × every
// role, called for real, asserted against a table transcribed from
// operations.md — NOT read from the enforcement's own matrix, so the
// two must agree.
//
// Conventions that make every allowed cell a genuine success:
//   - each invocation builds fresh fixtures (unique driver, unique
//     date in 2025-08 — invisible to the other suites' windows);
//   - driver cells of own-route operations act as the route's owner
//     (the "own route"/"own only" cells);
//   - driver cells of scoped queries pass no filter — the service must
//     force the scope;
//   - the anonymous column uses a zero Actor: ErrUnauthenticated.
//
// The file sorts last (z…), so its data never reaches the other suites.

// wantMatrix: {admin, manager, driver} per operation, transcribed from
// operations.md "Role matrix".
var wantMatrix = map[string][3]bool{
	app.OpLogin:                {true, true, true},
	app.OpLogout:               {true, true, true},
	app.OpCurrentUser:          {true, true, true},
	app.OpCreateDriver:         {true, true, false},
	app.OpUpdateDriver:         {true, true, false},
	app.OpListDrivers:          {true, true, false},
	app.OpAnonymizeDriver:      {true, false, false},
	app.OpCreateManager:        {true, false, false},
	app.OpListManagers:         {true, false, false},
	app.OpCreateLocation:       {true, true, false},
	app.OpUpdateLocation:       {true, true, false},
	app.OpListLocations:        {true, true, false},
	app.OpCreateRoute:          {true, true, false},
	app.OpAddStop:              {true, true, false},
	app.OpRemoveStop:           {true, true, false},
	app.OpReorderStops:         {true, true, false},
	app.OpStartRoute:           {true, true, true}, // driver: own route
	app.OpCloseRoute:           {true, true, true}, // driver: own route
	app.OpReopenRoute:          {true, false, false},
	app.OpRecordArrival:        {true, true, true}, // driver: own route
	app.OpRecordDeparture:      {true, true, true}, // driver: own route
	app.OpUpdateStopTimes:      {true, true, false},
	app.OpSetRouteDistance:     {true, true, true}, // driver: own route
	app.OpGetRoute:             {true, true, true}, // driver: own only
	app.OpListRoutes:           {true, true, true}, // driver: scoped
	app.OpGetDashboardByDay:    {true, true, true}, // driver: scoped
	app.OpGetDashboardByMonth:  {true, true, true}, // driver: scoped
	app.OpGetDashboardByPeriod: {true, true, true}, // driver: scoped
	app.OpGetParams:            {true, true, false},
	app.OpUpdateParam:          {true, true, false},
}

var mxDateSeq int

// mxDate returns a valid route date in 2025-08. Each fixture creates a
// fresh driver, so dates may repeat without tripping RN05.
func mxDate() string {
	mxDateSeq++
	return fmt.Sprintf("2025-08-%02d", (mxDateSeq-1)%28+1)
}

func mxTag() string {
	return uuid.NewString()[:8]
}

// mxRoute creates a driver and a fresh draft route (2 stops) for the
// driver to own.
func mxRoute(t *testing.T, tag string) (store.Driver, app.RouteDetail) {
	t.Helper()
	drv := createDriver(t, "mx-driver-"+tag)
	locs := createLocations(t, adminActor(), 2)
	route, err := svc.CreateRoute(ctx, adminActor(), app.CreateRouteInput{
		DriverUserID: drv.ID,
		RouteDate:    mxDate(),
		LocationIDs:  locationIDs(locs),
	})
	if err != nil {
		t.Fatalf("mx fixture route: %v", err)
	}
	return drv, route
}

// ownRouteInvoker wraps a call so the driver cell acts as the owner.
func ownRouteInvoker(
	call func(t *testing.T, actor app.Actor, drv store.Driver, route app.RouteDetail) error,
) func(t *testing.T, role string, actor app.Actor) error {
	return func(t *testing.T, role string, actor app.Actor) error {
		drv, route := mxRoute(t, mxTag())
		if role == "driver" {
			actor = app.Actor{UserID: drv.ID, Role: "driver"}
		}
		return call(t, actor, drv, route)
	}
}

func mxInvokers() map[string]func(t *testing.T, role string, actor app.Actor) error {
	return map[string]func(t *testing.T, role string, actor app.Actor) error{
		app.OpLogin: func(t *testing.T, _ string, _ app.Actor) error {
			_, _, _, err := svc.Login(ctx, app.LoginInput{Email: "admin@stoptime.dev", Password: "stoptime-dev"})
			return err
		},
		app.OpLogout: func(t *testing.T, _ string, actor app.Actor) error {
			return svc.Logout(ctx, actor)
		},
		app.OpCurrentUser: func(t *testing.T, _ string, actor app.Actor) error {
			_, err := svc.CurrentUser(ctx, actor)
			return err
		},
		app.OpCreateDriver: func(t *testing.T, _ string, actor app.Actor) error {
			_, err := svc.CreateDriver(ctx, actor, app.CreateDriverInput{
				Name: "MX " + mxTag(), Email: "mx-" + mxTag() + "@test.dev", Password: "pw-12345", Phone: "0",
			})
			return err
		},
		app.OpUpdateDriver: func(t *testing.T, _ string, actor app.Actor) error {
			drv := createDriver(t, "mx-upd-"+mxTag())
			_, err := svc.UpdateDriver(ctx, actor, app.UpdateDriverInput{DriverID: drv.ID, Name: ptr("Renamed")})
			return err
		},
		app.OpAnonymizeDriver: func(t *testing.T, _ string, actor app.Actor) error {
			drv := createDriver(t, "mx-anon-"+mxTag())
			_, err := svc.AnonymizeDriver(ctx, actor, drv.ID)
			return err
		},
		app.OpListDrivers: func(t *testing.T, _ string, actor app.Actor) error {
			_, err := svc.ListDrivers(ctx, actor, false)
			return err
		},
		app.OpCreateManager: func(t *testing.T, _ string, actor app.Actor) error {
			_, err := svc.CreateManager(ctx, actor, app.CreateManagerInput{
				Name: "MX M " + mxTag(), Email: "mx-m-" + mxTag() + "@test.dev", Password: "pw-12345", Phone: "0",
			})
			return err
		},
		app.OpListManagers: func(t *testing.T, _ string, actor app.Actor) error {
			_, err := svc.ListManagers(ctx, actor)
			return err
		},
		app.OpCreateLocation: func(t *testing.T, _ string, actor app.Actor) error {
			_, err := svc.CreateLocation(ctx, actor, app.CreateLocationInput{Label: "MX " + mxTag(), Address: "Rua MX, 1"})
			return err
		},
		app.OpUpdateLocation: func(t *testing.T, _ string, actor app.Actor) error {
			l, err := svc.CreateLocation(ctx, adminActor(), app.CreateLocationInput{Label: "MX " + mxTag(), Address: "Rua MX, 1"})
			if err != nil {
				return err
			}
			_, err = svc.UpdateLocation(ctx, actor, app.UpdateLocationInput{LocationID: l.ID, Label: ptr("Renamed")})
			return err
		},
		app.OpListLocations: func(t *testing.T, _ string, actor app.Actor) error {
			_, err := svc.ListLocations(ctx, actor, nil)
			return err
		},
		app.OpCreateRoute: func(t *testing.T, _ string, actor app.Actor) error {
			drv := createDriver(t, "mx-cr-"+mxTag())
			locs := createLocations(t, adminActor(), 2)
			_, err := svc.CreateRoute(ctx, actor, app.CreateRouteInput{
				DriverUserID: drv.ID, RouteDate: mxDate(), LocationIDs: locationIDs(locs),
			})
			return err
		},
		app.OpAddStop: func(t *testing.T, _ string, actor app.Actor) error {
			_, route := mxRoute(t, mxTag())
			extra, err := svc.CreateLocation(ctx, adminActor(), app.CreateLocationInput{Label: "MX " + mxTag(), Address: "Rua MX, 2"})
			if err != nil {
				return err
			}
			_, err = svc.AddStop(ctx, actor, app.AddStopInput{RouteID: route.Route.ID, LocationID: extra.ID})
			return err
		},
		app.OpRemoveStop: func(t *testing.T, _ string, actor app.Actor) error {
			_, route := mxRoute(t, mxTag())
			_, err := svc.RemoveStop(ctx, actor, app.RemoveStopInput{RouteID: route.Route.ID, StopOrder: 2})
			return err
		},
		app.OpReorderStops: func(t *testing.T, _ string, actor app.Actor) error {
			_, route := mxRoute(t, mxTag())
			_, err := svc.ReorderStops(ctx, actor, app.ReorderStopsInput{
				RouteID: route.Route.ID, StopOrder: 2, Direction: "up",
			})
			return err
		},
		app.OpStartRoute: ownRouteInvoker(func(t *testing.T, actor app.Actor, _ store.Driver, route app.RouteDetail) error {
			_, err := svc.StartRoute(ctx, actor, route.Route.ID)
			return err
		}),
		app.OpCloseRoute: ownRouteInvoker(func(t *testing.T, actor app.Actor, _ store.Driver, route app.RouteDetail) error {
			if _, err := svc.StartRoute(ctx, adminActor(), route.Route.ID); err != nil {
				return err
			}
			_, err := svc.CloseRoute(ctx, actor, app.CloseRouteInput{RouteID: route.Route.ID})
			return err
		}),
		app.OpReopenRoute: func(t *testing.T, _ string, actor app.Actor) error {
			_, route := mxRoute(t, mxTag())
			if _, err := svc.StartRoute(ctx, adminActor(), route.Route.ID); err != nil {
				return err
			}
			if _, err := svc.CloseRoute(ctx, adminActor(), app.CloseRouteInput{RouteID: route.Route.ID}); err != nil {
				return err
			}
			_, err := svc.ReopenRoute(ctx, actor, route.Route.ID)
			return err
		},
		app.OpRecordArrival: ownRouteInvoker(func(t *testing.T, actor app.Actor, _ store.Driver, route app.RouteDetail) error {
			if _, err := svc.StartRoute(ctx, adminActor(), route.Route.ID); err != nil {
				return err
			}
			_, err := svc.RecordArrival(ctx, actor, app.RecordTimeInput{RouteID: route.Route.ID, StopOrder: 2, At: at(9, 0)})
			return err
		}),
		app.OpRecordDeparture: ownRouteInvoker(func(t *testing.T, actor app.Actor, _ store.Driver, route app.RouteDetail) error {
			if _, err := svc.StartRoute(ctx, adminActor(), route.Route.ID); err != nil {
				return err
			}
			if _, err := svc.RecordArrival(ctx, adminActor(), app.RecordTimeInput{RouteID: route.Route.ID, StopOrder: 2, At: at(9, 0)}); err != nil {
				return err
			}
			_, err := svc.RecordDeparture(ctx, actor, app.RecordTimeInput{RouteID: route.Route.ID, StopOrder: 2, At: at(9, 15)})
			return err
		}),
		app.OpUpdateStopTimes: func(t *testing.T, _ string, actor app.Actor) error {
			_, route := mxRoute(t, mxTag())
			_, err := svc.UpdateStopTimes(ctx, actor, app.UpdateStopTimesInput{
				RouteID: route.Route.ID, StopOrder: 2, ArrivalAt: at(9, 0), DepartureAt: at(9, 10),
			})
			return err
		},
		app.OpSetRouteDistance: ownRouteInvoker(func(t *testing.T, actor app.Actor, _ store.Driver, route app.RouteDetail) error {
			if _, err := svc.StartRoute(ctx, adminActor(), route.Route.ID); err != nil {
				return err
			}
			_, err := svc.SetRouteDistance(ctx, actor, route.Route.ID, "42.50")
			return err
		}),
		app.OpGetRoute: ownRouteInvoker(func(t *testing.T, actor app.Actor, _ store.Driver, route app.RouteDetail) error {
			_, err := svc.GetRoute(ctx, actor, route.Route.ID)
			return err
		}),
		app.OpListRoutes: func(t *testing.T, _ string, actor app.Actor) error {
			_, err := svc.ListRoutes(ctx, actor, app.ListRoutesInput{From: ptr("2025-08-01"), To: ptr("2025-08-31")})
			return err
		},
		app.OpGetDashboardByDay: func(t *testing.T, _ string, actor app.Actor) error {
			_, err := svc.GetDashboardByDay(ctx, actor, app.DashboardInput{From: "2025-08-01", To: "2025-08-31"})
			return err
		},
		app.OpGetDashboardByMonth: func(t *testing.T, _ string, actor app.Actor) error {
			_, err := svc.GetDashboardByMonth(ctx, actor, app.DashboardInput{From: "2025-08-01", To: "2025-08-31"})
			return err
		},
		app.OpGetDashboardByPeriod: func(t *testing.T, _ string, actor app.Actor) error {
			_, err := svc.GetDashboardByPeriod(ctx, actor, app.DashboardInput{From: "2025-08-01", To: "2025-08-31"})
			return err
		},
		app.OpGetParams: func(t *testing.T, _ string, actor app.Actor) error {
			_, err := svc.GetParams(ctx, actor)
			return err
		},
		app.OpUpdateParam: func(t *testing.T, _ string, actor app.Actor) error {
			_, err := svc.UpdateParam(ctx, actor, app.UpdateParamInput{Key: "fuel_price_brl", Value: "6.09"})
			return err
		},
	}
}

func TestRoleMatrix(t *testing.T) {
	roles := []string{"admin", "manager", "driver", ""}
	actorFor := map[string]app.Actor{
		"admin":   adminActor(),
		"manager": {UserID: uuid.MustParse("aa000000-0000-4000-8000-000000000002"), Role: "manager"},
		"driver":  {UserID: uuid.MustParse("aa000000-0000-4000-8000-000000000003"), Role: "driver"},
		"":        {},
	}
	invokers := mxInvokers()

	// Deterministic log order.
	ops := []string{
		app.OpLogin, app.OpLogout, app.OpCurrentUser,
		app.OpCreateDriver, app.OpUpdateDriver, app.OpListDrivers, app.OpAnonymizeDriver,
		app.OpCreateManager, app.OpListManagers,
		app.OpCreateLocation, app.OpUpdateLocation, app.OpListLocations,
		app.OpCreateRoute, app.OpAddStop, app.OpRemoveStop, app.OpReorderStops,
		app.OpStartRoute, app.OpCloseRoute, app.OpReopenRoute,
		app.OpRecordArrival, app.OpRecordDeparture, app.OpUpdateStopTimes,
		app.OpSetRouteDistance,
		app.OpGetRoute, app.OpListRoutes,
		app.OpGetDashboardByDay, app.OpGetDashboardByMonth, app.OpGetDashboardByPeriod,
		app.OpGetParams, app.OpUpdateParam,
	}

	header := "operation                admin    manager  driver   anonymous"
	t.Log(header)
	for _, op := range ops {
		cells := [4]string{}
		for i, role := range roles {
			err := invokers[op](t, role, actorFor[role])
			switch {
			case err == nil:
				cells[i] = "ALLOW"
			case errors.Is(err, app.ErrForbidden):
				cells[i] = "FORBID"
			case errors.Is(err, app.ErrUnauthenticated):
				cells[i] = "NOAUTH"
			default:
				cells[i] = fmt.Sprintf("ERR(%v)", err)
			}

			wantAllowed := false
			if role == "" {
				wantAllowed = op == app.OpLogin // login IS the entry point
			} else {
				idx := map[string]int{"admin": 0, "manager": 1, "driver": 2}[role]
				wantAllowed = wantMatrix[op][idx]
			}
			if wantAllowed && err != nil {
				t.Errorf("%s / %q: want allowed, got %v", op, role, err)
			}
			if !wantAllowed {
				wantErr := app.ErrUnauthenticated
				if role != "" {
					wantErr = app.ErrForbidden
				}
				if !errors.Is(err, wantErr) {
					t.Errorf("%s / %q: want %v, got %v", op, role, wantErr, err)
				}
			}
		}
		t.Logf("%-24s %-7s  %-7s  %-7s  %s", op, cells[0], cells[1], cells[2], cells[3])
	}
}

// The "own route" cells deny drivers acting on someone else's route —
// the invoker above proves the allowed side; this proves the boundary.
func TestOwnRouteBoundary(t *testing.T) {
	_, route := mxRoute(t, "boundary")
	stranger := app.Actor{UserID: uuid.MustParse("aa000000-0000-4000-8000-000000000003"), Role: "driver"}

	_, err := svc.StartRoute(ctx, stranger, route.Route.ID)
	assertErrIs(t, "StartRoute on other's route", err, app.ErrForbidden)

	_, err = svc.GetRoute(ctx, stranger, route.Route.ID)
	assertErrIs(t, "GetRoute on other's route", err, app.ErrForbidden)

	_, err = svc.RecordArrival(ctx, stranger, app.RecordTimeInput{RouteID: route.Route.ID, StopOrder: 2, At: at(9, 0)})
	// RecordArrival loads the route first; the ownership gate fires
	// before the status check.
	assertErrIs(t, "RecordArrival on other's route", err, app.ErrForbidden)

	_, err = svc.CloseRoute(ctx, stranger, app.CloseRouteInput{RouteID: route.Route.ID})
	assertErrIs(t, "CloseRoute on other's route", err, app.ErrForbidden)

	_, err = svc.SetRouteDistance(ctx, stranger, route.Route.ID, "10")
	assertErrIs(t, "SetRouteDistance on other's route", err, app.ErrForbidden)
}
