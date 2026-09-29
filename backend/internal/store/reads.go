package store

import (
	"context"
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

// StopDetail is one stop of a route detail, joined with its location.
type StopDetail struct {
	StopOrder   int        `db:"stop_order"`
	Counted     bool       `db:"counted"`
	Label       string     `db:"label"`
	Address     string     `db:"address"`
	Latitude    *string    `db:"latitude"`
	Longitude   *string    `db:"longitude"`
	ArrivalAt   *time.Time `db:"arrival_at"`
	DepartureAt *time.Time `db:"departure_at"`
	StopSeconds *int       `db:"stop_seconds"`
}

func (s *Store) RouteStopDetails(ctx context.Context, routeID uuid.UUID) ([]StopDetail, error) {
	rows, err := s.db.Query(ctx, `
SELECT rs.stop_order,
       (rs.stop_order > 1)          AS counted,
       l.label, l.address,
       l.latitude::text, l.longitude::text,
       rs.arrival_at, rs.departure_at, rs.stop_seconds
  FROM route_stop rs
  JOIN location l ON l.id = rs.location_id
 WHERE rs.route_id = $1
 ORDER BY rs.stop_order`, routeID)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	var stops []StopDetail
	for rows.Next() {
		var st StopDetail
		if err := rows.Scan(&st.StopOrder, &st.Counted, &st.Label, &st.Address,
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
	ID                 uuid.UUID `db:"id"`
	DriverUserID       uuid.UUID `db:"driver_user_id"`
	DriverName         string    `db:"driver_name"`
	RouteDate          time.Time `db:"route_date"`
	Status             string    `db:"status"`
	DistanceKm         *string   `db:"distance_km"`
	Note               *string   `db:"note"`
	TotalSeconds       int64     `db:"total_seconds"`
	TotalStoppedMinut  int       `db:"total_stopped_minutes"`
	JourneyPercent     string    `db:"journey_percent"`
	EstimatedCostBRL   *string   `db:"estimated_cost_brl"`
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
	ID                 uuid.UUID `db:"id"`
	RouteDate          time.Time `db:"route_date"`
	DriverName         string    `db:"driver_name"`
	Status             string    `db:"status"`
	StopCount          int       `db:"stop_count"`
	TotalStoppedMinut  int       `db:"total_stopped_minutes"`
	JourneyPercent     string    `db:"journey_percent"`
	EstimatedCostBRL   *string   `db:"estimated_cost_brl"`
}

// ListRoutes lists routes in a date window with their aggregates, at
// query level filtered by driver and status (scoping is never
// post-filtering).
func (s *Store) ListRoutes(ctx context.Context, from, to string, driverUserID *uuid.UUID, status *string) ([]RouteListRow, error) {
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
   AND ($4::text IS NULL OR r.status = $4)
 ORDER BY r.route_date DESC, u.name`, from, to, driverUserID, status)
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
type DayPoint struct {
	Date                time.Time `db:"date"`
	TotalStoppedMinut   int       `db:"total_stopped_minutes"`
}

// DashboardByDaySQL is exported so tests EXPLAIN ANALYZE the exact query
// the service runs (RNF03 pattern guard).
const DashboardByDaySQL = `
WITH p AS (` + paramsPivot + `)
SELECT r.route_date AS date,
       (sum(CASE WHEN rs.stop_seconds / 60 >= p.m THEN rs.stop_seconds END) / 60)::int AS total_stopped_minutes
  FROM route r
  JOIN route_stop rs ON rs.route_id = r.id AND rs.stop_order > 1
  CROSS JOIN p
 WHERE r.route_date BETWEEN $1::date AND $2::date
   AND ($3::uuid IS NULL OR r.driver_user_id = $3)
 GROUP BY r.route_date
HAVING count(rs.stop_seconds) > 0
 ORDER BY r.route_date`

func (s *Store) DashboardByDay(ctx context.Context, from, to string, driverUserID *uuid.UUID) ([]DayPoint, error) {
	rows, err := s.db.Query(ctx, DashboardByDaySQL, from, to, driverUserID)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	var points []DayPoint
	for rows.Next() {
		var p DayPoint
		if err := rows.Scan(&p.Date, &p.TotalStoppedMinut); err != nil {
			return nil, translate(err)
		}
		points = append(points, p)
	}
	return points, translate(rows.Err())
}

// MonthPoint is one point of the by-month series.
type MonthPoint struct {
	Month             string `db:"month"`
	TotalStoppedMinut int    `db:"total_stopped_minutes"`
}

func (s *Store) DashboardByMonth(ctx context.Context, from, to string, driverUserID *uuid.UUID) ([]MonthPoint, error) {
	rows, err := s.db.Query(ctx, `
WITH p AS (`+paramsPivot+`)
SELECT to_char(r.route_date, 'YYYY-MM') AS month,
       (sum(CASE WHEN rs.stop_seconds / 60 >= p.m THEN rs.stop_seconds END) / 60)::int AS total_stopped_minutes
  FROM route r
  JOIN route_stop rs ON rs.route_id = r.id AND rs.stop_order > 1
  CROSS JOIN p
 WHERE r.route_date BETWEEN $1::date AND $2::date
   AND ($3::uuid IS NULL OR r.driver_user_id = $3)
 GROUP BY to_char(r.route_date, 'YYYY-MM')
HAVING count(rs.stop_seconds) > 0
 ORDER BY 1`, from, to, driverUserID)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	var points []MonthPoint
	for rows.Next() {
		var p MonthPoint
		if err := rows.Scan(&p.Month, &p.TotalStoppedMinut); err != nil {
			return nil, translate(err)
		}
		points = append(points, p)
	}
	return points, translate(rows.Err())
}

// PeriodRow carries the GROUPING SETS output of the period summary:
// one grand row (IsTotal = 1) plus one row per driver.
type PeriodRow struct {
	IsTotal            int     `db:"is_total"`
	DriverName         *string `db:"driver_name"`
	TotalStoppedMinut  int     `db:"total_stopped_minutes"`
	JourneyPercent     string  `db:"journey_percent"`
	RoutesCount        int     `db:"routes_count"`
}

func (s *Store) DashboardByPeriod(ctx context.Context, from, to string, driverUserID *uuid.UUID) ([]PeriodRow, error) {
	rows, err := s.db.Query(ctx, `
WITH p AS (`+paramsPivot+`),
     base AS (
  SELECT r.id, r.driver_user_id, rs.stop_seconds
    FROM route r
    JOIN route_stop rs ON rs.route_id = r.id AND rs.stop_order > 1
   WHERE r.route_date BETWEEN $1::date AND $2::date
     AND ($3::uuid IS NULL OR r.driver_user_id = $3)
 )
SELECT GROUPING(b.driver_user_id) AS is_total,
       max(u.name) AS driver_name,
       (coalesce(sum(CASE WHEN b.stop_seconds / 60 >= p.m THEN b.stop_seconds END), 0) / 60)::int AS total_stopped_minutes,
       round(coalesce(sum(CASE WHEN b.stop_seconds / 60 >= p.m THEN b.stop_seconds END), 0) / (p.h * 3600) * 100, 3)::text AS journey_percent,
       count(DISTINCT b.id) AS routes_count
  FROM base b
  CROSS JOIN p
  LEFT JOIN app_user u ON u.id = b.driver_user_id
 GROUP BY GROUPING SETS ((p.m, p.h), (b.driver_user_id, p.m, p.h))
 ORDER BY is_total DESC, driver_name`, from, to, driverUserID)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	var rowsOut []PeriodRow
	for rows.Next() {
		var r PeriodRow
		if err := rows.Scan(&r.IsTotal, &r.DriverName, &r.TotalStoppedMinut, &r.JourneyPercent, &r.RoutesCount); err != nil {
			return nil, translate(err)
		}
		rowsOut = append(rowsOut, r)
	}
	return rowsOut, translate(rows.Err())
}
