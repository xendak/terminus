// Typed client for the Go JSON API (backend/internal/httpapi). Shapes
// mirror the Go structs' json tags exactly; decimals arrive as strings
// (SQL numeric as exact text), timestamps as RFC 3339, route dates as
// midnight UTC ("2026-06-15T00:00:00Z").

export type Role = "admin" | "manager" | "driver";
export type RouteStatus = "draft" | "active" | "closed";

export interface User {
  id: string;
  name: string;
  email: string;
  phone: string;
  role: Role;
  active: boolean;
}

export interface Me {
  user: User;
  expires_at: string;
}

export interface Driver extends User {
  document?: string;
  vehicle_name?: string;
  vehicle_plate?: string;
  km_per_l?: string;
}

export interface Location {
  id: string;
  label: string;
  address: string;
  latitude: string | null;
  longitude: string | null;
  created_by: string;
}

export interface StopDetail {
  stop_order: number;
  counted: boolean;
  label: string;
  address: string;
  latitude: string | null;
  longitude: string | null;
  arrival_at: string | null;
  departure_at: string | null;
  stop_seconds: number | null;
}

export interface RouteView {
  id: string;
  driver_user_id: string;
  driver_name: string;
  /** "YYYY-MM-DD" (tolerates a full timestamp). */
  route_date: string;
  status: RouteStatus;
  distance_km: string | null;
  note: string | null;
  total_stopped_seconds: number;
  total_stopped_minutes: number;
  journey_percent: string;
  estimated_cost_brl: string | null;
  stops: StopDetail[];
}

export interface RouteStop {
  id: string;
  route_id: string;
  stop_order: number;
  location_id: string;
  arrival_at: string | null;
  departure_at: string | null;
  note: string | null;
}

export interface RouteListRow {
  id: string;
  route_date: string;
  driver_name: string;
  status: RouteStatus;
  stop_count: number;
  total_stopped_minutes: number;
  journey_percent: string;
  estimated_cost_brl: string | null;
}

export interface DayPoint {
  /** "YYYY-MM-DD" (older servers sent a full timestamp; read the first 10 chars). */
  date: string;
  total_stopped_minutes: number;
  /** SQL-computed share of the journeys in that day; absent on older servers. */
  journey_percent?: string;
}

export interface MonthPoint {
  month: string;
  total_stopped_minutes: number;
}

export interface DriverSummary {
  driver_name: string;
  total_stopped_minutes: number;
  journey_percent: string;
}

export interface PeriodSummary {
  total_stopped_minutes: number;
  journey_percent: string;
  routes_count: number;
  by_driver: DriverSummary[] | null;
}

export type ParamKey =
  | "fuel_price_brl"
  | "cost_per_km_brl"
  | "standard_journey_hours"
  | "min_stop_minutes"
  | "default_km_per_l";

export interface Param {
  key: ParamKey;
  value: string;
  unit: string;
  updated_by: string;
  updated_at: string;
}

export type AuditValues = Record<string, string | number | boolean | null>;

export interface AuditEntry {
  at: string;
  actor: string;
  entity: string;
  entity_id: string;
  action: string;
  old_values: AuditValues | null;
  new_values: AuditValues | null;
}

// ---- errors -------------------------------------------------------------

/** A non-2xx answer, carrying the backend's field detail when present. */
export class ApiError extends Error {
  constructor(
    readonly status: number,
    message: string,
    readonly field?: string,
    readonly reason?: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

interface ErrorBody {
  error?: string;
  field?: string;
  reason?: string;
}

function isErrorBody(v: unknown): v is ErrorBody {
  return typeof v === "object" && v !== null;
}

// ---- transport ----------------------------------------------------------

type Query = Record<string, string | undefined | null>;

function withQuery(path: string, q?: Query): string {
  if (!q) return path;
  const params = new URLSearchParams();
  for (const [k, v] of Object.entries(q)) {
    if (v !== undefined && v !== null && v !== "") params.set(k, v);
  }
  const s = params.toString();
  return s ? `${path}?${s}` : path;
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  let res: Response;
  try {
    res = await fetch(path, {
      method,
      credentials: "same-origin",
      headers: body === undefined ? undefined : { "Content-Type": "application/json" },
      body: body === undefined ? undefined : JSON.stringify(body),
      cache: "no-store",
    });
  } catch {
    throw new ApiError(0, "network");
  }
  if (res.status === 401 && typeof window !== "undefined" && !path.startsWith("/api/auth/login")) {
    const next = window.location.pathname + window.location.search;
    // A full reload on session loss drops every piece of client state.
    // eslint-disable-next-line @next/next/no-location-assign-relative-destination
    window.location.href = `/login?next=${encodeURIComponent(next)}`;
  }
  if (res.status === 204) return undefined as T;
  const text = await res.text();
  let data: unknown = null;
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = null;
    }
  }
  if (!res.ok) {
    const b = isErrorBody(data) ? data : {};
    throw new ApiError(res.status, b.error ?? res.statusText, b.field, b.reason);
  }
  return data as T;
}

const get = <T>(path: string, q?: Query) => request<T>("GET", withQuery(path, q));

// ---- operations ---------------------------------------------------------

export interface CreateDriverInput {
  name: string;
  email: string;
  password: string;
  phone: string;
  document?: string | null;
  vehicle_name?: string | null;
  vehicle_plate?: string | null;
  km_per_l?: string | null;
}

export interface UpdateDriverInput {
  name?: string;
  phone?: string;
  document?: string | null;
  vehicle_name?: string | null;
  vehicle_plate?: string | null;
  km_per_l?: string | null;
  active?: boolean;
}

export interface CreateManagerInput {
  name: string;
  email: string;
  password: string;
  phone: string;
}

export interface LocationInput {
  label: string;
  address: string;
  latitude?: string | null;
  longitude?: string | null;
}

export interface CreateRouteInput {
  driver_user_id: string;
  route_date: string;
  location_ids: string[];
  note?: string | null;
}

export interface RoutesFilter {
  from?: string;
  to?: string;
  driver_user_id?: string;
  status?: RouteStatus | "";
}

export interface Window {
  from: string;
  to: string;
}

export const api = {
  login: (email: string, password: string) =>
    request<Me>("POST", "/api/auth/login", { email, password }),
  logout: () => request<void>("POST", "/api/auth/logout"),
  me: () => get<Me>("/api/auth/me"),

  drivers: (activeOnly = false) =>
    get<{ drivers: Driver[] | null }>("/api/drivers", { active_only: activeOnly ? "true" : undefined }).then(
      (r) => r.drivers ?? [],
    ),
  createDriver: (input: CreateDriverInput) =>
    request<{ driver: Driver }>("POST", "/api/drivers", input).then((r) => r.driver),
  updateDriver: (id: string, input: UpdateDriverInput) =>
    request<{ driver: Driver }>("PATCH", `/api/drivers/${id}`, input).then((r) => r.driver),

  managers: () => get<{ managers: User[] | null }>("/api/managers").then((r) => r.managers ?? []),
  createManager: (input: CreateManagerInput) =>
    request<{ manager: User }>("POST", "/api/managers", input).then((r) => r.manager),

  locations: () => get<{ locations: Location[] | null }>("/api/locations").then((r) => r.locations ?? []),
  createLocation: (input: LocationInput) =>
    request<{ location: Location }>("POST", "/api/locations", input).then((r) => r.location),
  updateLocation: (id: string, input: Partial<LocationInput>) =>
    request<{ location: Location }>("PATCH", `/api/locations/${id}`, input).then((r) => r.location),

  routes: (f: RoutesFilter = {}) =>
    get<{ routes: RouteListRow[] | null }>("/api/routes", {
      from: f.from,
      to: f.to,
      driver_user_id: f.driver_user_id,
      status: f.status,
    }).then((r) => r.routes ?? []),
  route: (id: string) => get<{ route: RouteView }>(`/api/routes/${id}`).then((r) => r.route),
  createRoute: (input: CreateRouteInput) =>
    request<{ route: RouteView }>("POST", "/api/routes", input).then((r) => r.route),
  addStop: (id: string, locationId: string, position?: number) =>
    request<{ route: RouteView }>("POST", `/api/routes/${id}/stops`, {
      location_id: locationId,
      position,
    }).then((r) => r.route),
  removeStop: (id: string, order: number) =>
    request<{ route: RouteView }>("DELETE", `/api/routes/${id}/stops/${order}`).then((r) => r.route),
  moveStop: (id: string, order: number, direction: "up" | "down") =>
    request<{ route: RouteView }>("POST", `/api/routes/${id}/stops/${order}/move`, { direction }).then(
      (r) => r.route,
    ),
  startRoute: (id: string) =>
    request<{ route: RouteView }>("POST", `/api/routes/${id}/start`).then((r) => r.route),
  closeRoute: (id: string, distanceKm?: string) =>
    request<{ route: RouteView }>(
      "POST",
      `/api/routes/${id}/close`,
      distanceKm ? { distance_km: distanceKm } : undefined,
    ).then((r) => r.route),
  reopenRoute: (id: string) =>
    request<{ route: RouteView }>("POST", `/api/routes/${id}/reopen`).then((r) => r.route),
  setDistance: (id: string, distanceKm: string) =>
    request<{ route: RouteView }>("PUT", `/api/routes/${id}/distance`, { distance_km: distanceKm }).then(
      (r) => r.route,
    ),
  arrive: (id: string, order: number, at?: string) =>
    request<{ stop: RouteStop }>("POST", `/api/routes/${id}/stops/${order}/arrive`, at ? { at } : undefined).then(
      (r) => r.stop,
    ),
  depart: (id: string, order: number, at?: string) =>
    request<{ stop: RouteStop }>("POST", `/api/routes/${id}/stops/${order}/depart`, at ? { at } : undefined).then(
      (r) => r.stop,
    ),
  correctTimes: (id: string, order: number, times: { arrival_at?: string; departure_at?: string }) =>
    request<{ stop: RouteStop }>("PATCH", `/api/routes/${id}/stops/${order}/times`, times).then((r) => r.stop),

  dashboardDay: (w: Window) =>
    get<{ series: DayPoint[] | null }>("/api/dashboard/day", { ...w }).then((r) => r.series ?? []),
  dashboardMonth: (w: Window) =>
    get<{ series: MonthPoint[] | null }>("/api/dashboard/month", { ...w }).then((r) => r.series ?? []),
  dashboardPeriod: (w: Window) =>
    get<PeriodSummary | { series: PeriodSummary }>("/api/dashboard/period", { ...w }).then((r) =>
      "series" in r ? r.series : r,
    ),

  params: () => get<{ params: Param[] | null }>("/api/params").then((r) => r.params ?? []),
  updateParam: (key: ParamKey, value: string) =>
    request<{ param: Param }>("PUT", `/api/params/${key}`, { value }).then((r) => r.param),

  audit: (f: { entity?: string; from?: string; to?: string }) =>
    get<{ entries: AuditEntry[] | null }>("/api/audit", f).then((r) => r.entries ?? []),

  exportUrl: (f: { from: string; to: string; driver_user_id?: string }) => withQuery("/api/export", f),
};
