package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Read queries: each read service is backed by ONE SQL aggregation query
// (docs/spec/architecture.md: SQL owns queries and aggregation; the
// parameters below are read at query time, never hardcoded). Parameters
// arrive pivoted in a CTE; the min_stop_minutes threshold applies inside
// the SUM CASE (RN03/RF10); journey percent is rounded once to 3 places
// (RN04, pinned by the golden fixture); cost once to 2 (RN07). No
// distance means no cost (NULL), not zero cost.

// paramsPivot is shared by every read query.
const paramsPivot = `
  SELECT max(value) FILTER (WHERE key = 'min_stop_minutes')       AS m,
         max(value) FILTER (WHERE key = 'standard_journey_hours') AS h,
         max(value) FILTER (WHERE key = 'fuel_price_brl')         AS f,
         max(value) FILTER (WHERE key = 'cost_per_km_brl')        AS c,
         max(value) FILTER (WHERE key = 'default_km_per_l')       AS k
    FROM parameter`

// costExpr computes the estimated cost (RN07) from a route alias and the
// parameter pivot p; NULL when the route has no distance yet.
const costExpr = `
       CASE WHEN r.distance_km IS NULL THEN NULL
            ELSE round(r.distance_km / coalesce(dp.km_per_l, p.k) * p.f
                       + r.distance_km * p.c, 2)::text
       END`

// StopDetail is one stop of a route detail. Label, address and
// coordinates are the stop's snapshot of its location (migration 0003),
// not the location's current values.
type StopDetail struct {
	StopOrder   int        `db:"stop_order" json:"stop_order"`
	// Counted: the stop adds to the route total — false for stop 1 (RN01)
	// and for a completed stop under min_stop_minutes (BelowMin), whose
	// StopSeconds stay as recorded.
	Counted     bool       `db:"counted" json:"counted"`
	BelowMin    bool       `db:"below_min" json:"below_min"`
	Label       string     `db:"label" json:"label"`
	Address     string     `db:"address" json:"address"`
	Latitude    *string    `db:"latitude" json:"latitude"`
	Longitude   *string    `db:"longitude" json:"longitude"`
	ArrivalAt   *time.Time `db:"arrival_at" json:"arrival_at"`
	DepartureAt *time.Time `db:"departure_at" json:"departure_at"`
	StopSeconds *int       `db:"stop_seconds" json:"stop_seconds"`
}

func (s *Store) RouteStopDetails(ctx context.Context, routeID uuid.UUID) ([]StopDetail, error) {
	rows, err := s.db.Query(ctx, `
WITH p AS (`+paramsPivot+`)
SELECT rs.stop_order,
       (rs.stop_order > 1 AND coalesce(rs.stop_seconds / 60 >= p.m, true)) AS counted,
       (rs.stop_order > 1 AND coalesce(rs.stop_seconds / 60 <  p.m, false)) AS below_min,
       rs.label_snapshot, rs.address_snapshot,
       rs.latitude_snapshot::text, rs.longitude_snapshot::text,
       rs.arrival_at, rs.departure_at, rs.stop_seconds
  FROM route_stop rs
 CROSS JOIN p
 WHERE rs.route_id = $1
 ORDER BY rs.stop_order`, routeID)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	var stops []StopDetail
	for rows.Next() {
		var st StopDetail
		if err := rows.Scan(&st.StopOrder, &st.Counted, &st.BelowMin, &st.Label, &st.Address,
			&st.Latitude, &st.Longitude, &st.ArrivalAt, &st.DepartureAt, &st.StopSeconds); err != nil {
			return nil, translate(err)
		}
		stops = append(stops, st)
	}
	return stops, translate(rows.Err())
}

// RouteWithTotals is one route with driver name and its SQL-computed
// aggregates (RN03, RN04, RN07).
type RouteWithTotals struct {
	ID                 uuid.UUID `db:"id" json:"id"`
	DriverUserID       uuid.UUID `db:"driver_user_id" json:"driver_user_id"`
	DriverName         string    `db:"driver_name" json:"driver_name"`
	RouteDate          Date      `db:"route_date" json:"route_date"`
	Status             string    `db:"status" json:"status"`
	DistanceKm         *string   `db:"distance_km" json:"distance_km"`
	Note               *string   `db:"note" json:"note"`
	TotalSeconds       int64     `db:"total_seconds" json:"total_stopped_seconds"`
	TotalStoppedMinut  int       `db:"total_stopped_minutes" json:"total_stopped_minutes"`
	JourneyPercent     string    `db:"journey_percent" json:"journey_percent"`
	EstimatedCostBRL   *string   `db:"estimated_cost_brl" json:"estimated_cost_brl"`
}

// RouteWithTotals returns the route row plus aggregates in one query.
func (s *Store) RouteWithTotals(ctx context.Context, routeID uuid.UUID) (RouteWithTotals, error) {
	var rt RouteWithTotals
	err := s.db.QueryRow(ctx, `
WITH p AS (`+paramsPivot+`),
     agg AS (
  SELECT coalesce(sum(CASE WHEN rs.stop_seconds / 60 >= p.m THEN rs.stop_seconds END), 0) AS total_seconds
    FROM route_stop rs CROSS JOIN p
   WHERE rs.route_id = $1 AND rs.stop_order > 1
 )
SELECT r.id, r.driver_user_id, u.name AS driver_name, r.route_date, r.status,
       r.distance_km::text, r.note,
       agg.total_seconds                     AS total_seconds,
       (agg.total_seconds / 60)::int        AS total_stopped_minutes,
       round(agg.total_seconds / (p.h * 3600) * 100, 3)::text AS journey_percent,`+
		costExpr+`
  FROM route r
  JOIN app_user u             ON u.id = r.driver_user_id
  LEFT JOIN driver_profile dp ON dp.user_id = r.driver_user_id
  CROSS JOIN agg
  CROSS JOIN p
 WHERE r.id = $1`, routeID).
		Scan(&rt.ID, &rt.DriverUserID, &rt.DriverName, &rt.RouteDate, &rt.Status,
			&rt.DistanceKm, &rt.Note, &rt.TotalSeconds, &rt.TotalStoppedMinut,
			&rt.JourneyPercent, &rt.EstimatedCostBRL)
	if err != nil {
		return RouteWithTotals{}, scanOne(err)
	}
	return rt, nil
}

// RouteListRow is one row of the history list (ListRoutes).
type RouteListRow struct {
	ID                 uuid.UUID `db:"id" json:"id"`
	RouteDate          Date      `db:"route_date" json:"route_date"`
	DriverName         string    `db:"driver_name" json:"driver_name"`
	Status             string    `db:"status" json:"status"`
	StopCount          int       `db:"stop_count" json:"stop_count"`
	TotalStoppedMinut  int       `db:"total_stopped_minutes" json:"total_stopped_minutes"`
	JourneyPercent     string    `db:"journey_percent" json:"journey_percent"`
	EstimatedCostBRL   *string   `db:"estimated_cost_brl" json:"estimated_cost_brl"`
}

// ListRoutes lists routes in a date window with their aggregates, at
// query level filtered by driver and status (scoping is never
// post-filtering).
func (s *Store) ListRoutes(ctx context.Context, from, to string, driverUserID *uuid.UUID, status *string, managerUserID *uuid.UUID) ([]RouteListRow, error) {
	rows, err := s.db.Query(ctx, `
WITH p AS (`+paramsPivot+`),
     agg AS (
  SELECT rs.route_id,
         count(*) AS stop_count,
         coalesce(sum(CASE WHEN rs.stop_order > 1 AND rs.stop_seconds / 60 >= p.m
                           THEN rs.stop_seconds END), 0) AS total_seconds
    FROM route_stop rs CROSS JOIN p
   WHERE rs.route_id IN (SELECT r.id FROM route r
                          WHERE r.route_date BETWEEN $1::date AND $2::date
                            AND ($3::uuid IS NULL OR r.driver_user_id = $3))
   GROUP BY rs.route_id
 )
SELECT r.id, r.route_date, u.name AS driver_name, r.status,
       coalesce(agg.stop_count, 0)                       AS stop_count,
       (coalesce(agg.total_seconds, 0) / 60)::int        AS total_stopped_minutes,
       round(coalesce(agg.total_seconds, 0) / (p.h * 3600) * 100, 3)::text AS journey_percent,`+
		costExpr+`
  FROM route r
  JOIN app_user u             ON u.id = r.driver_user_id
  LEFT JOIN driver_profile dp ON dp.user_id = r.driver_user_id
  LEFT JOIN agg               ON agg.route_id = r.id
  CROSS JOIN p
 WHERE r.route_date BETWEEN $1::date AND $2::date
   AND ($3::uuid IS NULL OR r.driver_user_id = $3)
   AND ($5::uuid IS NULL OR r.driver_user_id IN (SELECT tp.user_id FROM driver_profile tp WHERE tp.manager_user_id = $5))
   AND ($4::text IS NULL OR r.status = $4)
 ORDER BY r.route_date DESC, u.name`, from, to, driverUserID, status, managerUserID)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	var routes []RouteListRow
	for rows.Next() {
		var r RouteListRow
		if err := rows.Scan(&r.ID, &r.RouteDate, &r.DriverName, &r.Status,
			&r.StopCount, &r.TotalStoppedMinut, &r.JourneyPercent, &r.EstimatedCostBRL); err != nil {
			return nil, translate(err)
		}
		routes = append(routes, r)
	}
	return routes, translate(rows.Err())
}

// DayPoint is one point of the by-day series: a day with recorded data.
// JourneyPercent uses one standard day per route that day (RN04/RN05).
type DayPoint struct {
	Date              Date   `db:"date" json:"date"`
	TotalStoppedMinut int    `db:"total_stopped_minutes" json:"total_stopped_minutes"`
	JourneyPercent    string `db:"journey_percent" json:"journey_percent"`
}

// DashboardByDaySQL is exported so tests EXPLAIN ANALYZE the exact query
// the service runs (RNF03 pattern guard).
const DashboardByDaySQL = `
WITH p AS (` + paramsPivot + `)
SELECT r.route_date AS date,
       (coalesce(sum(CASE WHEN rs.stop_seconds / 60 >= p.m THEN rs.stop_seconds END), 0) / 60)::int AS total_stopped_minutes,
       round(coalesce(sum(CASE WHEN rs.stop_seconds / 60 >= p.m THEN rs.stop_seconds END), 0)
             / (count(DISTINCT r.id) FILTER (WHERE rs.stop_seconds IS NOT NULL) * p.h * 3600) * 100, 3)::text AS journey_percent
  FROM route r
  JOIN route_stop rs ON rs.route_id = r.id AND rs.stop_order > 1
  CROSS JOIN p
 WHERE r.route_date BETWEEN $1::date AND $2::date
   AND ($3::uuid IS NULL OR r.driver_user_id = $3)
   AND ($4::uuid IS NULL OR r.driver_user_id IN (SELECT tp.user_id FROM driver_profile tp WHERE tp.manager_user_id = $4))
 GROUP BY r.route_date, p.h
HAVING count(rs.stop_seconds) > 0
 ORDER BY r.route_date`

func (s *Store) DashboardByDay(ctx context.Context, from, to string, driverUserID, managerUserID *uuid.UUID) ([]DayPoint, error) {
	rows, err := s.db.Query(ctx, DashboardByDaySQL, from, to, driverUserID, managerUserID)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	var points []DayPoint
	for rows.Next() {
		var p DayPoint
		if err := rows.Scan(&p.Date, &p.TotalStoppedMinut, &p.JourneyPercent); err != nil {
			return nil, translate(err)
		}
		points = append(points, p)
	}
	return points, translate(rows.Err())
}

// MonthPoint is one point of the by-month series. JourneyPercent uses
// one standard day per worked route in the month (RN04/RN05).
type MonthPoint struct {
	Month             string `db:"month" json:"month"`
	TotalStoppedMinut int    `db:"total_stopped_minutes" json:"total_stopped_minutes"`
	JourneyPercent    string `db:"journey_percent" json:"journey_percent"`
}

func (s *Store) DashboardByMonth(ctx context.Context, from, to string, driverUserID, managerUserID *uuid.UUID) ([]MonthPoint, error) {
	rows, err := s.db.Query(ctx, `
WITH p AS (`+paramsPivot+`)
SELECT to_char(r.route_date, 'YYYY-MM') AS month,
       (coalesce(sum(CASE WHEN rs.stop_seconds / 60 >= p.m THEN rs.stop_seconds END), 0) / 60)::int AS total_stopped_minutes,
       round(coalesce(sum(CASE WHEN rs.stop_seconds / 60 >= p.m THEN rs.stop_seconds END), 0)
             / (count(DISTINCT r.id) FILTER (WHERE rs.stop_seconds IS NOT NULL) * p.h * 3600) * 100, 3)::text AS journey_percent
  FROM route r
  JOIN route_stop rs ON rs.route_id = r.id AND rs.stop_order > 1
  CROSS JOIN p
 WHERE r.route_date BETWEEN $1::date AND $2::date
   AND ($3::uuid IS NULL OR r.driver_user_id = $3)
   AND ($4::uuid IS NULL OR r.driver_user_id IN (SELECT tp.user_id FROM driver_profile tp WHERE tp.manager_user_id = $4))
 GROUP BY to_char(r.route_date, 'YYYY-MM'), p.h
HAVING count(rs.stop_seconds) > 0
 ORDER BY 1`, from, to, driverUserID, managerUserID)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	var points []MonthPoint
	for rows.Next() {
		var p MonthPoint
		if err := rows.Scan(&p.Month, &p.TotalStoppedMinut, &p.JourneyPercent); err != nil {
			return nil, translate(err)
		}
		points = append(points, p)
	}
	return points, translate(rows.Err())
}

// PeriodRow carries the GROUPING SETS output of the period summary:
// one grand row (IsTotal = 1) plus one row per driver. RoutesCount is
// the worked routes (at least one recorded stop interval) and is the
// journey percent base: one standard day per route (RN04/RN05).
type PeriodRow struct {
	IsTotal            int        `db:"is_total"`
	DriverUserID       *uuid.UUID `db:"driver_user_id"`
	DriverName         *string    `db:"driver_name"`
	TotalStoppedMinut  int        `db:"total_stopped_minutes"`
	JourneyPercent     string     `db:"journey_percent"`
	RoutesCount        int        `db:"routes_count"`
}

func (s *Store) DashboardByPeriod(ctx context.Context, from, to string, driverUserID, managerUserID *uuid.UUID) ([]PeriodRow, error) {
	rows, err := s.db.Query(ctx, `
WITH p AS (`+paramsPivot+`),
     base AS (
  SELECT r.id, r.driver_user_id, rs.stop_seconds
    FROM route r
    JOIN route_stop rs ON rs.route_id = r.id AND rs.stop_order > 1
   WHERE r.route_date BETWEEN $1::date AND $2::date
     AND ($3::uuid IS NULL OR r.driver_user_id = $3)
     AND ($4::uuid IS NULL OR r.driver_user_id IN (SELECT tp.user_id FROM driver_profile tp WHERE tp.manager_user_id = $4))
     AND rs.stop_seconds IS NOT NULL -- worked stops only: a planned route is not a worked day
 )
SELECT GROUPING(b.driver_user_id) AS is_total,
       b.driver_user_id,
       max(u.name) AS driver_name,
       (coalesce(sum(CASE WHEN b.stop_seconds / 60 >= p.m THEN b.stop_seconds END), 0) / 60)::int AS total_stopped_minutes,
       round(coalesce(sum(CASE WHEN b.stop_seconds / 60 >= p.m THEN b.stop_seconds END), 0)
             / (count(DISTINCT b.id) * p.h * 3600) * 100, 3)::text AS journey_percent,
       count(DISTINCT b.id) AS routes_count
  FROM base b
  CROSS JOIN p
  LEFT JOIN app_user u ON u.id = b.driver_user_id
 GROUP BY GROUPING SETS ((p.m, p.h), (b.driver_user_id, p.m, p.h))
 ORDER BY is_total DESC, driver_name, b.driver_user_id`, from, to, driverUserID, managerUserID)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	var rowsOut []PeriodRow
	for rows.Next() {
		var r PeriodRow
		if err := rows.Scan(&r.IsTotal, &r.DriverUserID, &r.DriverName, &r.TotalStoppedMinut, &r.JourneyPercent, &r.RoutesCount); err != nil {
			return nil, translate(err)
		}
		rowsOut = append(rowsOut, r)
	}
	return rowsOut, translate(rows.Err())
}

// AuditEntryView is one audit_log row joined with the acting user's
// name (operations.md ListAudit output); values pass through as JSON.
type AuditEntryView struct {
	At        time.Time       `db:"at" json:"at"`
	Actor     string          `db:"actor" json:"actor"`
	Entity    string          `db:"entity" json:"entity"`
	EntityID  string          `db:"entity_id" json:"entity_id"`
	Action    string          `db:"action" json:"action"`
	OldValues json.RawMessage `db:"old_values" json:"old_values"`
	NewValues json.RawMessage `db:"new_values" json:"new_values"`
}

// ListAudit: the append-only trail (RNF05), newest first, optionally
// filtered by entity and a date window (inclusive of `to`'s day).
func (s *Store) ListAudit(ctx context.Context, entity *string, from, to *string) ([]AuditEntryView, error) {
	rows, err := s.db.Query(ctx, `
SELECT a.at, u.name AS actor, a.entity, a.entity_id, a.action, a.old_values, a.new_values
  FROM audit_log a
  JOIN app_user u ON u.id = a.actor_user_id
 WHERE ($1::text IS NULL OR a.entity = $1)
   AND ($2::date IS NULL OR a.at >= $2::date)
   AND ($3::date IS NULL OR a.at < ($3::date + 1))
 ORDER BY a.at DESC
 LIMIT 200`, entity, from, to)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	var entries []AuditEntryView
	for rows.Next() {
		var e AuditEntryView
		if err := rows.Scan(&e.At, &e.Actor, &e.Entity, &e.EntityID,
			&e.Action, &e.OldValues, &e.NewValues); err != nil {
			return nil, translate(err)
		}
		entries = append(entries, e)
	}
	return entries, translate(rows.Err())
}

// ExportRow is one CSV row: a stop with its location, timestamps, and
// the route's SQL-computed totals (operations.md ExportPeriodCSV).
type ExportRow struct {
	RouteDate         Date       `db:"route_date"`
	DriverName        string     `db:"driver_name"`
	StopOrder         int        `db:"stop_order"`
	Address           string     `db:"address"`
	ArrivalAt         *time.Time `db:"arrival_at"`
	DepartureAt       *time.Time `db:"departure_at"`
	StopMinutes       *int       `db:"stop_minutes"`
	RouteTotalMinutes int        `db:"route_total_minutes"`
	RouteCost         *string    `db:"route_cost"`
}

// ExportRows is the CSV source query: one row per stop, in date/driver/
// order, with the route total (RN03) and cost (RN07) computed by SQL.
func (s *Store) ExportRows(ctx context.Context, from, to string, driverUserID, managerUserID *uuid.UUID) ([]ExportRow, error) {
	rows, err := s.db.Query(ctx, `
WITH p AS (`+paramsPivot+`),
     agg AS (
  SELECT rs.route_id,
         coalesce(sum(CASE WHEN rs.stop_order > 1 AND rs.stop_seconds / 60 >= p.m
                           THEN rs.stop_seconds END), 0) AS total_seconds
    FROM route_stop rs CROSS JOIN p
   WHERE rs.route_id IN (SELECT r.id FROM route r
                          WHERE r.route_date BETWEEN $1::date AND $2::date
                            AND ($3::uuid IS NULL OR r.driver_user_id = $3))
   GROUP BY rs.route_id
 )
SELECT r.route_date, u.name AS driver_name,
       rs.stop_order, rs.address_snapshot AS address,
       rs.arrival_at, rs.departure_at,
       (rs.stop_seconds / 60)                     AS stop_minutes,
       (coalesce(agg.total_seconds, 0) / 60)::int AS route_total_minutes,`+
		costExpr+`
  FROM route r
  JOIN app_user u             ON u.id = r.driver_user_id
  JOIN route_stop rs          ON rs.route_id = r.id
  LEFT JOIN driver_profile dp ON dp.user_id = r.driver_user_id
  LEFT JOIN agg               ON agg.route_id = r.id
  CROSS JOIN p
 WHERE r.route_date BETWEEN $1::date AND $2::date
   AND ($3::uuid IS NULL OR r.driver_user_id = $3)
   AND ($4::uuid IS NULL OR r.driver_user_id IN (SELECT tp.user_id FROM driver_profile tp WHERE tp.manager_user_id = $4))
 ORDER BY r.route_date, u.name, rs.stop_order`, from, to, driverUserID, managerUserID)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	var out []ExportRow
	for rows.Next() {
		var e ExportRow
		if err := rows.Scan(&e.RouteDate, &e.DriverName, &e.StopOrder, &e.Address,
			&e.ArrivalAt, &e.DepartureAt, &e.StopMinutes, &e.RouteTotalMinutes, &e.RouteCost); err != nil {
			return nil, translate(err)
		}
		out = append(out, e)
	}
	return out, translate(rows.Err())
}
