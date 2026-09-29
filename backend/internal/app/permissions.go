package app

import (
	"fmt"

	"github.com/google/uuid"
)

// The role matrix (operations.md, "Role matrix") enforced in the service
// layer — middleware is a convenience, never the authority
// (architecture.md, Security). Three shapes:
//
//   - plain: the listed roles may call, everyone else gets ErrForbidden;
//   - own-route: the driver role is listed but the service additionally
//     requires the route to belong to the actor (ownRoute);
//   - scoped: the driver role is listed and the service FORCES the
//     driver's id into the query filter (own data only, never
//     post-filtered).
//
// Entries below list roles only; the own-route and scoped behaviors are
// enforced by the services calling allow + ownRoute / forcing the filter.

const (
	OpLogin                = "Login"
	OpLogout               = "Logout"
	OpCurrentUser          = "CurrentUser"
	OpCreateDriver         = "CreateDriver"
	OpUpdateDriver         = "UpdateDriver"
	OpListDrivers          = "ListDrivers"
	OpCreateManager        = "CreateManager"
	OpListManagers         = "ListManagers"
	OpCreateLocation       = "CreateLocation"
	OpUpdateLocation       = "UpdateLocation"
	OpListLocations        = "ListLocations"
	OpCreateRoute          = "CreateRoute"
	OpAddStop              = "AddStop"
	OpRemoveStop           = "RemoveStop"
	OpReorderStops         = "ReorderStops"
	OpStartRoute           = "StartRoute"
	OpCloseRoute           = "CloseRoute"
	OpReopenRoute          = "ReopenRoute"
	OpRecordArrival        = "RecordArrival"
	OpRecordDeparture      = "RecordDeparture"
	OpUpdateStopTimes      = "UpdateStopTimes"
	OpSetRouteDistance     = "SetRouteDistance"
	OpGetRoute             = "GetRoute"
	OpListRoutes           = "ListRoutes"
	OpGetDashboardByDay    = "GetDashboardByDay"
	OpGetDashboardByMonth  = "GetDashboardByMonth"
	OpGetDashboardByPeriod = "GetDashboardByPeriod"
	OpGetParams            = "GetParams"
	OpUpdateParam          = "UpdateParam"
	OpListAudit            = "ListAudit"
	OpExportPeriodCSV      = "ExportPeriodCSV"
)

var matrix = map[string][]string{
	OpLogin:                {"admin", "manager", "driver"}, // needs no session
	OpLogout:               {"admin", "manager", "driver"},
	OpCurrentUser:          {"admin", "manager", "driver"},
	OpCreateDriver:         {"admin", "manager"},
	OpUpdateDriver:         {"admin", "manager"},
	OpListDrivers:          {"admin", "manager"},
	OpCreateManager:        {"admin"},
	OpListManagers:         {"admin"},
	OpCreateLocation:       {"admin", "manager"},
	OpUpdateLocation:       {"admin", "manager"},
	OpListLocations:        {"admin", "manager"},
	OpCreateRoute:          {"admin", "manager"},
	OpAddStop:              {"admin", "manager"},
	OpRemoveStop:           {"admin", "manager"},
	OpReorderStops:         {"admin", "manager"},
	OpStartRoute:           {"admin", "manager", "driver"}, // own route
	OpCloseRoute:           {"admin", "manager", "driver"}, // own route
	OpReopenRoute:          {"admin"},
	OpRecordArrival:        {"admin", "manager", "driver"}, // own route
	OpRecordDeparture:      {"admin", "manager", "driver"}, // own route
	OpUpdateStopTimes:      {"admin", "manager"},
	OpSetRouteDistance:     {"admin", "manager", "driver"}, // own route
	OpGetRoute:             {"admin", "manager", "driver"}, // own only
	OpListRoutes:           {"admin", "manager", "driver"}, // scoped
	OpGetDashboardByDay:    {"admin", "manager", "driver"}, // scoped
	OpGetDashboardByMonth:  {"admin", "manager", "driver"}, // scoped
	OpGetDashboardByPeriod: {"admin", "manager", "driver"}, // scoped
	OpGetParams:            {"admin", "manager"},
	OpUpdateParam:          {"admin", "manager"},
	OpListAudit:            {"admin"},
	OpExportPeriodCSV:      {"admin", "manager", "driver"}, // scoped
}

// allow is the gate every service passes first: no session is
// ErrUnauthenticated, a role outside the matrix row is ErrForbidden.
func (s *Services) allow(actor Actor, op string) error {
	if actor.UserID == uuid.Nil || actor.Role == "" {
		return ErrUnauthenticated
	}
	roles, ok := matrix[op]
	if !ok {
		return fmt.Errorf("app: unknown operation %q", op)
	}
	for _, r := range roles {
		if actor.Role == r {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrForbidden, op)
}

// ownRoute enforces the "own route" matrix cells: drivers may only act
// on routes they drive. Admin and manager pass untouched.
func (s *Services) ownRoute(actor Actor, routeDriverUserID uuid.UUID) error {
	if actor.Role == "driver" && routeDriverUserID != actor.UserID {
		return fmt.Errorf("%w: not your route", ErrForbidden)
	}
	return nil
}
