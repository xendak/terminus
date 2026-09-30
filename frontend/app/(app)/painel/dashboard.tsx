"use client";

import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useState, type FormEvent } from "react";
import { useUser } from "@/components/session-context";
import { TeamSelect } from "@/components/team-select";
import {
  Button,
  Card,
  cx,
  EmptyState,
  ErrorState,
  Field,
  Input,
  Notice,
  PageHeader,
  Skeleton,
} from "@/components/ui";
import { api, type DayPoint, type MonthPoint, type PeriodSummary, type StopDetail } from "@/lib/api";
import { describeError } from "@/lib/errors";
import {
  addDaysISO,
  addMonthsISO,
  fmtDate,
  fmtMinutes,
  fmtMonth,
  fmtPercent,
  fmtTime,
  monthStartISO,
  todayISO,
} from "@/lib/format";
import { isStaff } from "@/lib/roles";
import { useApi } from "@/lib/use-api";
import { useNow } from "@/lib/use-now";

// ---- shapes for the per-day detail (timeline + points list) ---------------

interface DriverDay {
  routeId: string;
  driverId: string;
  driverName: string;
  vehicle?: string;
  stops: StopDetail[];
}

interface DayStop {
  routeId: string;
  driverId: string;
  driverName: string;
  stopOrder: number;
  label: string;
  address: string;
  arrivalAt: string;
  departureAt: string;
  stopSeconds: number;
  belowMin: boolean;
}

interface DayDetail {
  drivers: DriverDay[];
  stops: DayStop[];
  /** Routes of the day whose detail could not be loaded. */
  failed: number;
}

interface DashboardData {
  day: DayPoint[];
  month: MonthPoint[];
  /** Last 12 days ending at `to`: the single-day chart context. */
  chartDay: DayPoint[];
  period: PeriodSummary;
  journeyHours: number;
  thresholds: Thresholds;
}

interface BarPoint {
  key: string;
  label: string;
  longLabel: string;
  minutes: number;
  percent?: string;
  href?: string;
}

// Display thresholds for single stops come from the API (parameters
// stop_warn_minutes / stop_alert_minutes); these are the defaults for a
// server that does not send them. Nothing is totalled or excluded by them.
const DEFAULT_WARN_MIN = 15;
const DEFAULT_ALERT_MIN = 45;

interface Thresholds {
  warnS: number;
  alertS: number;
}

function thresholdsOf(...answers: { warn?: string; alert?: string }[]): Thresholds {
  const pick = (vals: (string | undefined)[], fallback: number) => {
    // 0 is a valid threshold (every stop reaches it); only a missing or
    // unreadable value falls back to the default.
    const raw = vals.find((v) => v !== undefined && v !== "");
    const n = raw === undefined ? NaN : Number(raw);
    return (Number.isFinite(n) && n >= 0 ? n : fallback) * 60;
  };
  return {
    warnS: pick(answers.map((a) => a.warn), DEFAULT_WARN_MIN),
    alertS: pick(answers.map((a) => a.alert), DEFAULT_ALERT_MIN),
  };
}
// Ranges up to this many days chart by day; longer ones chart by month.
const DAY_GROUP_MAX = 31;
// Per-route detail is fetched for single-day views only, a handful of routes.
const DETAIL_MAX_ROUTES = 60;

function stopStatus(stop: { stopSeconds: number; belowMin: boolean }, t: Thresholds) {
  if (stop.belowMin) return { label: "Não conta", cls: "border border-line text-ink-3" };
  if (stop.stopSeconds >= t.alertS) return { label: "Acima do limite", cls: "bg-danger-soft text-danger" };
  if (stop.stopSeconds >= t.warnS) return { label: "Atenção", cls: "bg-cone-soft text-cone-ink" };
  return { label: "Normal", cls: "bg-placa-soft text-placa-ink" };
}

/** "Marcos Motorista" → "Marcos M.": enough to tell drivers apart in a row. */
function shortName(name: string): string {
  const [first, ...rest] = name.trim().split(/\s+/);
  const last = rest.at(-1);
  return last ? `${first} ${last[0]}.` : first;
}

/** Whole minutes of one stop, floored from seconds (RN02, display only). */
const stopMinutes = (seconds: number) => Math.floor(seconds / 60);

function presets(today: string) {
  return [
    { id: "hoje", label: "Hoje", from: today, to: today },
    { id: "7d", label: "7 dias", from: addDaysISO(today, -6), to: today },
    { id: "mes", label: "Este mês", from: monthStartISO(today), to: today },
    { id: "12m", label: "12 meses", from: addDaysISO(addMonthsISO(today, -12), 1), to: today },
  ];
}

/** History pre-filtered to a bucket or a driver: the drill-down target. */
function historyHref(from: string, to: string, driverUserId?: string, team?: string): string {
  const q = new URLSearchParams({ from, to });
  if (driverUserId) q.set("driver_user_id", driverUserId);
  if (team) q.set("manager_user_id", team);
  return `/historico?${q.toString()}`;
}

/** A month bucket's days, clipped to the selected window. */
function monthWindow(month: string, from: string, to: string): { from: string; to: string } {
  const first = `${month}-01`;
  const last = addDaysISO(addMonthsISO(first, 1), -1);
  return { from: first < from ? from : first, to: last > to ? to : last };
}

/**
 * The series carry only buckets with data. Days and months without stops
 * are drawn as empty slots (0 min) so the time axis stays honest; each
 * value itself comes straight from the API.
 */
function fillDays(series: DayPoint[], from: string, to: string) {
  const byDay = new Map(series.map((p) => [p.date.slice(0, 10), p]));
  const out: { key: string; minutes: number; percent?: string }[] = [];
  for (let d = from; d <= to && out.length < 400; d = addDaysISO(d, 1)) {
    const p = byDay.get(d);
    out.push({ key: d, minutes: p?.total_stopped_minutes ?? 0, percent: p?.journey_percent });
  }
  return out;
}

function fillMonths(series: MonthPoint[], from: string, to: string) {
  const byMonth = new Map(series.map((p) => [p.month.slice(0, 7), p]));
  const out: { key: string; minutes: number; percent?: string }[] = [];
  for (let m = `${from.slice(0, 7)}-01`; m.slice(0, 7) <= to.slice(0, 7) && out.length < 60; m = addMonthsISO(m, 1)) {
    const p = byMonth.get(m.slice(0, 7));
    out.push({ key: m.slice(0, 7), minutes: p?.total_stopped_minutes ?? 0, percent: p?.journey_percent });
  }
  return out;
}

// ---- screen ----------------------------------------------------------------

export function Dashboard() {
  const user = useUser();
  const staff = isStaff(user.role);
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const today = todayISO();
  const from = params.get("from") ?? monthStartISO(today);
  const to = params.get("to") ?? today;
  const team = staff ? params.get("manager_user_id") ?? "" : "";

  const spanDays = Math.max(0, Math.round((Date.parse(`${to}T12:00:00Z`) - Date.parse(`${from}T12:00:00Z`)) / 86400000));
  const isSingleDay = from === to;
  const groupBy: "day" | "month" = spanDays < DAY_GROUP_MAX ? "day" : "month";
  const [driverFilter, setDriverFilter] = useState("");

  const managers = useApi(staff ? "manager-options" : null, () => api.managerOptions());
  const data = useApi<DashboardData>(`${from}|${to}|${team}`, async () => {
    const w = { from, to, manager_user_id: team || undefined };
    const [day, month, period, chartDay] = await Promise.all([
      api.dashboardDay(w),
      api.dashboardMonth(w),
      api.dashboardPeriod(w),
      isSingleDay
        ? api.dashboardDay({ ...w, from: addDaysISO(to, -11) })
        : Promise.resolve({ series: [] as DayPoint[], hours: undefined }),
    ]);
    // The API states the journey base with every answer; only if an older
    // server does not, staff can read it from the parameters.
    let hours = day.hours ?? month.hours ?? period.standard_journey_hours;
    if (hours === undefined && staff) {
      hours = await api
        .params()
        .then((ps) => ps.find((p) => p.key === "standard_journey_hours")?.value)
        .catch(() => undefined);
    }
    const parsed = Number(hours);
    return {
      day: day.series,
      month: month.series,
      chartDay: chartDay.series,
      period,
      journeyHours: Number.isFinite(parsed) && parsed > 0 ? parsed : 8,
      thresholds: thresholdsOf(
        { warn: day.warn, alert: day.alert },
        { warn: period.stop_warn_minutes, alert: period.stop_alert_minutes },
      ),
    };
  });

  // Timeline + points are single-day views, anchored on `to`. Skipped (null
  // key) for longer ranges so they never pay the per-route fetch.
  const detail = useApi<DayDetail>(isSingleDay ? `${to}|${team}` : null, async () => {
    const rows = (await api.routes({ from: to, to, manager_user_id: team || undefined }))
      .filter((r) => r.status === "active" || r.status === "closed")
      .slice(0, DETAIL_MAX_ROUTES);
    const [settled, vehicles] = await Promise.all([
      Promise.allSettled(rows.map((r) => api.route(r.id))),
      // Plates are a staff nicety; drivers cannot list drivers (403).
      staff
        ? api
            .drivers()
            .then((ds) => new Map(ds.map((d) => [d.id, d.vehicle_plate ?? ""])))
            .catch(() => new Map<string, string>())
        : Promise.resolve(new Map<string, string>()),
    ]);
    const views = settled.flatMap((s) => (s.status === "fulfilled" ? [s.value] : []));
    const drivers: DriverDay[] = views.map((v) => ({
      routeId: v.id,
      driverId: v.driver_user_id,
      driverName: v.driver_name,
      vehicle: vehicles.get(v.driver_user_id) || undefined,
      stops: v.stops,
    }));
    const stops: DayStop[] = views.flatMap((v) =>
      v.stops
        .filter((s) => s.stop_order > 1 && s.arrival_at && s.departure_at)
        .map((s) => ({
          routeId: v.id,
          driverId: v.driver_user_id,
          driverName: v.driver_name,
          stopOrder: s.stop_order,
          label: s.label,
          address: s.address,
          arrivalAt: s.arrival_at as string,
          departureAt: s.departure_at as string,
          stopSeconds: s.stop_seconds ?? 0,
          belowMin: !!s.below_min,
        })),
    );
    return { drivers, stops, failed: settled.length - views.length };
  });

  function setQuery(next: Record<string, string>) {
    const q = new URLSearchParams(params.toString());
    for (const [k, v] of Object.entries(next)) {
      if (v) q.set(k, v);
      else q.delete(k);
    }
    router.replace(`${pathname}?${q.toString()}`, { scroll: false });
  }

  const presetList = presets(today);
  const activePreset = presetList.find((p) => p.from === from && p.to === to)?.id;
  const caption = isSingleDay ? (from === today ? "hoje" : `em ${fmtDate(from)}`) : `de ${fmtDate(from)} a ${fmtDate(to)}`;

  const dd = data.data;
  const dayHref = (key: string) => historyHref(key, key, undefined, team);
  const chartPoints: BarPoint[] = !dd
    ? []
    : isSingleDay
      ? fillDays(dd.chartDay, addDaysISO(to, -11), to).map((p) => ({
          key: p.key,
          label: fmtDate(p.key).slice(0, 5),
          longLabel: fmtDate(p.key),
          minutes: p.minutes,
          percent: p.percent,
          href: dayHref(p.key),
        }))
      : groupBy === "day"
        ? fillDays(dd.day, from, to).map((p) => ({
            key: p.key,
            label: fmtDate(p.key).slice(0, 5),
            longLabel: fmtDate(p.key),
            minutes: p.minutes,
            percent: p.percent,
            href: dayHref(p.key),
          }))
        : fillMonths(dd.month, from, to).map((p) => {
            const w = monthWindow(p.key, from, to);
            return {
              key: p.key,
              label: fmtMonth(p.key),
              longLabel: fmtMonth(p.key),
              minutes: p.minutes,
              percent: p.percent,
              href: historyHref(w.from, w.to, undefined, team),
            };
          });

  const driverOptions = (detail.data?.drivers ?? []).map((d) => ({ value: d.driverId, label: d.driverName }));
  const filteredDrivers = (detail.data?.drivers ?? []).filter((d) => !driverFilter || d.driverId === driverFilter);
  const filteredStops = (detail.data?.stops ?? []).filter((s) => !driverFilter || s.driverId === driverFilter);
  const empty =
    !!dd && dd.day.length === 0 && dd.month.length === 0 && (!isSingleDay || (detail.state === "ready" && detail.data.drivers.length === 0));

  return (
    <>
      <PageHeader
        title={user.role === "driver" ? "Meu tempo parado" : "Painel"}
        eyebrow={`${fmtDate(from)} a ${fmtDate(to)}`}
        actions={
          staff && (
            <TeamSelect
              id="painel-equipe"
              value={team}
              managers={managers.data ?? []}
              onChange={(v) => setQuery({ manager_user_id: v })}
            />
          )
        }
      >
        Tempo parado nos pontos dos roteiros, somado no banco por dia, por mês e no período.
      </PageHeader>

      <RangeBar
        from={from}
        to={to}
        presets={presetList}
        activePreset={activePreset}
        onPick={(f, t) => setQuery({ from: f, to: t })}
      />

      {data.state === "error" ? (
        <ErrorState message={describeError(data.error)} onRetry={data.reload} />
      ) : !dd ? (
        <DashboardSkeleton />
      ) : empty ? (
        <EmptyState
          title="Nenhuma parada registrada neste período"
          action={
            activePreset !== "12m" ? (
              <Button
                variant="secondary"
                onClick={() => {
                  const p = presetList[3];
                  setQuery({ from: p.from, to: p.to });
                }}
              >
                Ver os últimos 12 meses
              </Button>
            ) : undefined
          }
        >
          Amplie o intervalo de datas ou confira se os roteiros do período já foram iniciados.
        </EmptyState>
      ) : (
        <div className={cx("flex flex-col gap-6 transition-opacity", data.state === "loading" && "opacity-60")}>
          <KpiStrip period={dd.period} hours={dd.journeyHours} caption={caption} />

          {isSingleDay && (
            <TimelineByDriver
              drivers={filteredDrivers}
              driverOptions={driverOptions}
              driverFilter={driverFilter}
              onDriverFilter={setDriverFilter}
              day={to}
              live={to === today}
              state={detail.state}
              thresholds={dd.thresholds}
              failed={detail.data?.failed ?? 0}
              onRetry={detail.reload}
            />
          )}

          <div className={cx("grid grid-cols-1 gap-6", isSingleDay && "lg:grid-cols-2")}>
            <DailyChart
              points={chartPoints}
              unit={isSingleDay || groupBy === "day" ? "day" : "month"}
              context={isSingleDay ? `últimos 12 dias até ${fmtDate(to)}` : "no período"}
              current={isSingleDay ? to : undefined}
            />
            {isSingleDay && (
              <PointsList
                stops={filteredStops}
                day={to}
                state={detail.state}
                thresholds={dd.thresholds}
                onRetry={detail.reload}
              />
            )}
          </div>

          <div className="grid grid-cols-1 items-start gap-6 xl:grid-cols-[minmax(0,1fr)_400px]">
            <DashboardTable
              groupBy={groupBy}
              points={chartPoints.length && !isSingleDay ? chartPoints : singleDayRow(dd, to, team)}
              period={dd.period}
              caption={caption}
            />
            <Ranking period={dd.period} hours={dd.journeyHours} from={from} to={to} self={!staff} />
          </div>
        </div>
      )}
    </>
  );
}

/** The table row for a single-day view: the day itself, from the API. */
function singleDayRow(dd: DashboardData, day: string, team: string): BarPoint[] {
  const p = dd.day.find((d) => d.date.slice(0, 10) === day);
  return p
    ? [
        {
          key: day,
          label: fmtDate(day),
          longLabel: fmtDate(day),
          minutes: p.total_stopped_minutes,
          percent: p.journey_percent,
          href: historyHref(day, day, undefined, team),
        },
      ]
    : [];
}

function RangeBar({
  from,
  to,
  presets,
  activePreset,
  onPick,
}: {
  from: string;
  to: string;
  presets: { id: string; label: string; from: string; to: string }[];
  activePreset?: string;
  onPick: (from: string, to: string) => void;
}) {
  const [error, setError] = useState<string | null>(null);
  function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const f = new FormData(e.currentTarget);
    const nf = String(f.get("from") ?? "");
    const nt = String(f.get("to") ?? "");
    if (!nf || !nt) return setError("Informe as duas datas.");
    if (nf > nt) return setError("A data inicial precisa ser anterior à final.");
    setError(null);
    onPick(nf, nt);
  }
  return (
    <Card className="mb-6 flex flex-col gap-4 p-4 lg:flex-row lg:items-end lg:justify-between">
      <div className="flex flex-wrap gap-2" role="group" aria-label="Intervalos rápidos">
        {presets.map((p) => (
          <button
            key={p.id}
            type="button"
            aria-pressed={activePreset === p.id}
            onClick={() => onPick(p.from, p.to)}
            className={cx(
              "h-11 rounded-full border px-4 text-sm font-semibold transition-colors sm:h-9",
              activePreset === p.id
                ? "border-placa bg-placa text-on-placa"
                : "border-line-strong text-ink-2 hover:bg-surface-2 hover:text-ink",
            )}
          >
            {p.label}
          </button>
        ))}
      </div>
      <form
        key={`${from}|${to}`}
        onSubmit={onSubmit}
        className="grid grid-cols-2 items-end gap-3 sm:flex sm:flex-wrap"
        noValidate
      >
        <Field label="De" htmlFor="range-from">
          <Input id="range-from" name="from" type="date" defaultValue={from} invalid={!!error} className="sm:w-40" />
        </Field>
        <Field label="Até" htmlFor="range-to">
          <Input id="range-to" name="to" type="date" defaultValue={to} invalid={!!error} className="sm:w-40" />
        </Field>
        <Button type="submit" variant="primary" className="col-span-2 sm:col-span-1">
          Aplicar
        </Button>
        {error && (
          <p role="alert" className="col-span-2 basis-full text-sm font-medium text-danger">
            {error}
          </p>
        )}
      </form>
    </Card>
  );
}

// ---- KPI strip -------------------------------------------------------------

function KpiStrip({ period, hours, caption }: { period: PeriodSummary; hours: number; caption: string }) {
  const routes = period.routes_count;
  const avg = period.avg_stopped_minutes_per_route;
  return (
    <div className={cx("grid grid-cols-1 gap-4 sm:grid-cols-2", avg !== undefined ? "xl:grid-cols-4" : "xl:grid-cols-3")}>
      <KpiHero value={fmtMinutes(period.total_stopped_minutes)} caption={caption} />
      {avg !== undefined && (
        <Kpi
          label="Média por roteiro"
          value={routes > 0 ? fmtMinutes(avg) : "—"}
          caption={`${routes} ${routes === 1 ? "roteiro" : "roteiros"}`}
        />
      )}
      <Kpi
        label="% da jornada"
        value={fmtPercent(period.journey_percent)}
        caption={`de ${hours.toLocaleString("pt-BR")} h por roteiro`}
      />
      <Kpi label="Roteiros no período" value={String(routes)} caption="com paradas registradas" />
    </div>
  );
}

function KpiHero({ value, caption }: { value: string; caption: string }) {
  return (
    <div className="rounded-xl border border-hero bg-hero p-5 text-on-hero shadow-card">
      <p className="text-[11px] font-semibold uppercase tracking-wide opacity-75">Tempo parado total</p>
      <p className="display mt-1.5 text-3xl font-bold text-hero-accent tnum">{value}</p>
      <p className="mt-1 text-xs opacity-75">{caption}</p>
    </div>
  );
}

function Kpi({ label, value, caption }: { label: string; value: string; caption: string }) {
  return (
    <div className="rounded-xl border border-line bg-surface p-5 shadow-card">
      <p className="text-[11px] font-semibold uppercase tracking-wide text-ink-3">{label}</p>
      <p className="display mt-1.5 text-2xl font-bold tnum">{value}</p>
      <p className="mt-1 text-xs text-ink-3">{caption}</p>
    </div>
  );
}

// ---- timeline by driver (signature) ---------------------------------------

interface Segment {
  kind: "base" | "stop" | "long" | "below" | "open";
  start: number;
  end: number;
  label: string;
}

/** A driver's recorded stops as time segments (ms), stop 1 as the start mark. */
function segmentsOf(driver: DriverDay, now: number, alertS: number): Segment[] {
  const out: Segment[] = [];
  for (const s of [...driver.stops].sort((a, b) => a.stop_order - b.stop_order)) {
    if (s.stop_order === 1) {
      // RN01: the departure point has no stopped time, only a start mark.
      if (s.departure_at) {
        const t = new Date(s.departure_at).getTime();
        out.push({ kind: "base", start: t, end: t, label: `Ponto 1 · ${s.label} (partida): saída às ${fmtTime(s.departure_at)}` });
      }
      continue;
    }
    if (!s.arrival_at) continue;
    const start = new Date(s.arrival_at).getTime();
    if (!s.departure_at) {
      out.push({ kind: "open", start, end: Math.max(start, now), label: `Ponto ${s.stop_order} · ${s.label}: parado desde ${fmtTime(s.arrival_at)}` });
      continue;
    }
    const secs = s.stop_seconds ?? 0;
    out.push({
      kind: s.below_min ? "below" : secs >= alertS ? "long" : "stop",
      start,
      end: new Date(s.departure_at).getTime(),
      label: `Ponto ${s.stop_order} · ${s.label}: ${fmtTime(s.arrival_at)}–${fmtTime(s.departure_at)}, ${stopMinutes(secs)} min${
        s.below_min ? " (abaixo do mínimo, não conta)" : ""
      }`,
    });
  }
  return out;
}

const segmentCls: Record<Segment["kind"], string> = {
  base: "bg-ink-2",
  stop: "bg-cone",
  long: "bg-danger",
  below: "bg-line-strong",
  open: "bg-cone pulse",
};

function TimelineByDriver({
  drivers,
  driverOptions,
  driverFilter,
  onDriverFilter,
  day,
  live,
  state,
  thresholds,
  failed,
  onRetry,
}: {
  drivers: DriverDay[];
  driverOptions: { value: string; label: string }[];
  driverFilter: string;
  onDriverFilter: (v: string) => void;
  day: string;
  live: boolean;
  state: "loading" | "ready" | "error";
  thresholds: Thresholds;
  failed: number;
  onRetry: () => void;
}) {
  const now = useNow(live, 30_000);
  const rows = drivers
    .map((d) => ({ d, segs: segmentsOf(d, live ? now : 0, thresholds.alertS) }))
    .filter((r) => r.segs.length > 0);
  // One shared clock for every row, rounded out to whole hours, so rows
  // compare: the same x means the same time of day for every driver.
  const all = rows.flatMap((r) => r.segs);
  const hour = 3_600_000;
  const t0 = all.length ? Math.floor(Math.min(...all.map((s) => s.start)) / hour) * hour : 0;
  const t1 = all.length ? Math.ceil(Math.max(...all.map((s) => s.end)) / hour) * hour : 0;
  const span = Math.max(hour, t1 - t0);
  const pct = (t: number) => ((t - t0) / span) * 100;
  const ticks: number[] = [];
  const step = span > 10 * hour ? 2 * hour : hour;
  for (let t = t0; t <= t1; t += step) ticks.push(t);

  return (
    <Card className="p-4 sm:p-5">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="display text-base font-semibold">Linha do tempo por motorista</h2>
          <p className="text-xs text-ink-3">Cada barra é uma parada no horário em que aconteceu, em {fmtDate(day)}.</p>
        </div>
        {driverOptions.length > 1 && (
          <label className="flex items-center gap-2 text-xs font-medium text-ink-3">
            Motorista
            <select
              value={driverFilter}
              onChange={(e) => onDriverFilter(e.target.value)}
              className="h-11 rounded-md border border-line-strong bg-surface px-2.5 text-sm font-normal text-ink sm:h-9"
            >
              <option value="">Todos</option>
              {driverOptions.map((d) => (
                <option key={d.value} value={d.value}>
                  {d.label}
                </option>
              ))}
            </select>
          </label>
        )}
      </div>
      {state === "error" ? (
        <ErrorState message="Não foi possível carregar os roteiros do dia." onRetry={onRetry} />
      ) : state === "loading" && drivers.length === 0 ? (
        <div className="flex flex-col gap-4" aria-busy="true" aria-label="Carregando linha do tempo">
          {[0, 1, 2].map((i) => (
            <div key={i} className="grid grid-cols-[minmax(0,140px)_1fr] items-center gap-4">
              <Skeleton className="h-6 w-full" />
              <Skeleton className="h-3 w-full" />
            </div>
          ))}
        </div>
      ) : rows.length === 0 ? (
        <p className="py-6 text-center text-sm text-ink-3">
          {driverFilter ? "Nenhuma parada registrada para este motorista." : `Nenhuma parada registrada em ${fmtDate(day)}.`}
        </p>
      ) : (
        <>
          <ul className="flex flex-col gap-4">
            {rows.map(({ d, segs }) => {
              const alert = segs.some((s) => s.kind === "long");
              const stopped = segs.filter((s) => s.kind !== "base");
              return (
                <li key={d.routeId} className="grid grid-cols-[minmax(0,120px)_1fr] items-center gap-3 sm:grid-cols-[minmax(0,180px)_1fr] sm:gap-4">
                  <div className="min-w-0">
                    <Link
                      href={`/roteiros/${d.routeId}`}
                      className="block truncate text-sm font-semibold underline-offset-4 hover:underline"
                    >
                      {d.driverName}
                    </Link>
                    <p className="flex items-center gap-1.5 text-xs text-ink-3">
                      <span aria-hidden className={cx("h-1.5 w-1.5 shrink-0 rounded-full", alert ? "bg-danger" : "bg-placa")} />
                      <span className="truncate">{d.vehicle || "em rota"}</span>
                    </p>
                  </div>
                  <div className="relative h-3 rounded-full bg-surface-2">
                    <span className="sr-only">
                      {stopped.length} {stopped.length === 1 ? "parada" : "paradas"}: {stopped.map((s) => s.label).join("; ")}
                    </span>
                    {segs.map((s, i) =>
                      s.kind === "base" ? (
                        <span
                          key={i}
                          aria-hidden
                          title={s.label}
                          className="absolute -top-0.5 h-4 w-1 -translate-x-1/2 rounded-sm bg-ink-2"
                          style={{ left: `${pct(s.start)}%` }}
                        />
                      ) : (
                        <span
                          key={i}
                          aria-hidden
                          title={s.label}
                          className={cx("absolute top-0 h-3 rounded-full", segmentCls[s.kind])}
                          style={{ left: `${pct(s.start)}%`, width: `max(4px, ${pct(s.end) - pct(s.start)}%)` }}
                        />
                      ),
                    )}
                  </div>
                </li>
              );
            })}
          </ul>
          <div aria-hidden className="relative mt-2 ml-[calc(120px+0.75rem)] h-4 text-[10px] text-ink-3 tnum sm:ml-[calc(180px+1rem)]">
            {ticks.map((t, i) => (
              <span
                key={t}
                className={cx(
                  "absolute",
                  i === 0 ? "translate-x-0" : i === ticks.length - 1 ? "-translate-x-full" : "-translate-x-1/2",
                  // On a phone the track is ~170px: keep every other hour.
                  ticks.length > 3 && i % 2 === 1 && i !== ticks.length - 1 && "max-sm:hidden",
                )}
                style={{ left: `${pct(t)}%` }}
              >
                {fmtTime(new Date(t).toISOString())}
              </span>
            ))}
          </div>
          <ul className="mt-4 flex flex-wrap gap-x-4 gap-y-1 text-xs text-ink-3">
            <Legend cls="h-3 w-1 bg-ink-2" label="saída da base" />
            <Legend cls="bg-cone" label="parado" />
            <Legend cls="bg-danger" label={`parada de ${thresholds.alertS / 60} min ou mais`} />
            <Legend cls="bg-line-strong" label="abaixo do mínimo (não conta)" />
          </ul>
        </>
      )}
      {failed > 0 && (
        <Notice tone="warn" className="mt-4">
          {failed === 1 ? "1 roteiro não pôde ser carregado" : `${failed} roteiros não puderam ser carregados`}.{" "}
          <button type="button" onClick={onRetry} className="font-semibold underline underline-offset-4">
            Tentar de novo
          </button>
        </Notice>
      )}
    </Card>
  );
}

function Legend({ cls, label }: { cls: string; label: string }) {
  return (
    <li className="flex items-center gap-1.5">
      <span aria-hidden className={cx("inline-block h-2.5 w-2.5 rounded-sm", cls)} />
      {label}
    </li>
  );
}

// ---- bar chart -------------------------------------------------------------

function DailyChart({
  points,
  unit,
  context,
  current,
}: {
  points: BarPoint[];
  unit: "day" | "month";
  context: string;
  current?: string;
}) {
  const max = Math.max(1, ...points.map((p) => p.minutes));
  const peak = points.findIndex((p) => p.minutes === max && p.minutes > 0);
  const dense = points.length > 16;
  return (
    <Card className="p-4 sm:p-5">
      <h2 className="display mb-1 text-base font-semibold">{unit === "month" ? "Tempo parado por mês" : "Tempo parado por dia"}</h2>
      <p className="mb-4 text-xs text-ink-3">
        Minutos parados {unit === "month" ? "por mês" : "por dia"}, {context}. Toque numa barra para ver os roteiros.
      </p>
      {points.every((p) => p.minutes === 0) ? (
        <p className="py-10 text-center text-sm text-ink-3">Sem paradas neste agrupamento.</p>
      ) : (
        <>
          <ol className="flex h-48 items-end gap-[3px] sm:gap-1.5" aria-label={`Tempo parado ${unit === "month" ? "por mês" : "por dia"}`}>
            {points.map((p, i) => {
              const h = Math.round((p.minutes / max) * 100);
              const isPeak = i === peak;
              const label = `${p.longLabel}: ${fmtMinutes(p.minutes)} parados${
                p.percent !== undefined ? `, ${fmtPercent(p.percent)} da jornada` : ""
              }`;
              const bar = (
                <>
                  <span className="relative block w-full flex-1 overflow-hidden rounded-t-sm bg-placa-soft">
                    <span
                      className={cx(
                        "absolute bottom-0 block w-full rounded-t-sm transition-opacity group-hover:opacity-80",
                        isPeak ? "bg-danger" : "bg-cone",
                      )}
                      style={{ height: `${Math.max(p.minutes > 0 ? 4 : 0, h)}%` }}
                    />
                  </span>
                  <span
                    className={cx(
                      "text-[10px] tnum",
                      p.key === current ? "font-bold text-ink" : "text-ink-3",
                      dense && i % Math.ceil(points.length / 8) !== 0 && "invisible",
                      !dense && points.length > 6 && i % 2 === 1 && "max-sm:invisible",
                    )}
                  >
                    {p.label}
                  </span>
                </>
              );
              return (
                <li key={p.key} className="flex h-full min-w-0 flex-1" title={label}>
                  {p.href && p.minutes > 0 ? (
                    <Link href={p.href} aria-label={`${label}. Ver roteiros`} className="group flex h-full w-full flex-col items-center justify-end gap-1 rounded-sm">
                      {bar}
                    </Link>
                  ) : (
                    <span className="flex h-full w-full flex-col items-center justify-end gap-1" aria-label={label}>
                      {bar}
                    </span>
                  )}
                </li>
              );
            })}
          </ol>
          <ul className="mt-4 flex flex-wrap gap-x-4 gap-y-1 text-xs text-ink-3">
            <Legend cls="bg-cone" label="tempo parado" />
            <Legend cls="bg-danger" label="pico" />
          </ul>
        </>
      )}
    </Card>
  );
}

// ---- points list -----------------------------------------------------------

function PointsList({
  stops,
  day,
  state,
  thresholds,
  onRetry,
}: {
  stops: DayStop[];
  day: string;
  state: "loading" | "ready" | "error";
  thresholds: Thresholds;
  onRetry: () => void;
}) {
  const sorted = [...stops].sort((a, b) => b.stopSeconds - a.stopSeconds);
  return (
    <Card className="p-4 sm:p-5">
      <h2 className="display mb-1 text-base font-semibold">Pontos do dia</h2>
      <p className="mb-3 text-xs text-ink-3">Paradas concluídas em {fmtDate(day)}, da mais longa para a mais curta.</p>
      {state === "error" ? (
        <ErrorState message="Não foi possível carregar os pontos do dia." onRetry={onRetry} />
      ) : state === "loading" && stops.length === 0 ? (
        <div className="flex flex-col gap-3" aria-busy="true" aria-label="Carregando pontos">
          {[0, 1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-10 w-full" />
          ))}
        </div>
      ) : sorted.length === 0 ? (
        <p className="py-6 text-center text-sm text-ink-3">Nenhuma parada concluída neste dia.</p>
      ) : (
        <ul className="flex max-h-[26rem] flex-col overflow-y-auto">
          {sorted.map((s, i) => {
            const st = stopStatus(s, thresholds);
            return (
              <li key={`${s.routeId}-${s.stopOrder}`} className={cx(i > 0 && "border-t border-line")}>
                <Link
                  href={`/roteiros/${s.routeId}`}
                  className="flex items-center gap-3 rounded-md py-3 hover:bg-surface-2/60"
                  aria-label={`${s.label}, ponto ${s.stopOrder} de ${s.driverName}: ${stopMinutes(s.stopSeconds)} min, ${st.label}. Abrir roteiro`}
                >
                  {/* RN06 numbering, same as the tracker and the CSV (the base is point 1). */}
                  <span className="flex h-11 w-12 shrink-0 flex-col items-center justify-center rounded-md bg-surface-2 leading-none text-ink-2">
                    <span className="text-[9px] font-semibold uppercase tracking-wide">Ponto</span>
                    <span className="mt-0.5 text-base font-bold tnum">{s.stopOrder}</span>
                  </span>
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-sm font-medium">{s.label}</span>
                    <span className="block truncate text-xs text-ink-3" title={s.address}>
                      <span className="font-semibold text-ink-2">{shortName(s.driverName)}</span> ·{" "}
                      {fmtTime(s.arrivalAt)}–{fmtTime(s.departureAt)} · {s.address}
                    </span>
                  </span>
                  <span className={cx("shrink-0 text-sm font-semibold tnum", s.belowMin && "text-ink-3 line-through")}>
                    {stopMinutes(s.stopSeconds)} min
                  </span>
                  <span className={cx("hidden shrink-0 rounded-full px-2.5 py-0.5 text-xs font-semibold sm:inline", st.cls)}>
                    {st.label}
                  </span>
                </Link>
              </li>
            );
          })}
        </ul>
      )}
    </Card>
  );
}

// ---- table (aggregate view) ------------------------------------------------

function DashboardTable({
  groupBy,
  points,
  period,
  caption,
}: {
  groupBy: "day" | "month";
  points: BarPoint[];
  period: PeriodSummary;
  caption: string;
}) {
  const rows = points.filter((p) => p.minutes > 0 || p.percent !== undefined);
  return (
    <Card className="overflow-x-auto p-0">
      <table className="w-full min-w-[300px] text-sm">
        <caption className="sr-only">Tempo parado {groupBy === "day" ? "por dia" : "por mês"}</caption>
        <thead>
          <tr className="border-b border-line text-left">
            <th scope="col" className="px-4 py-3 font-semibold text-ink-3">{groupBy === "day" ? "Dia" : "Mês"}</th>
            <th scope="col" className="px-4 py-3 text-right font-semibold text-ink-3">Tempo parado</th>
            <th scope="col" className="px-4 py-3 text-right font-semibold text-ink-3">% da jornada</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r, i) => (
            <tr key={r.key} className={cx("border-b border-line last:border-0", i % 2 === 1 && "bg-surface-2/40")}>
              <td className="px-4 py-2.5">
                {r.href ? (
                  <Link
                    href={r.href}
                    className="font-medium text-placa-ink underline-offset-4 hover:underline"
                    aria-label={`Ver roteiros de ${r.longLabel}`}
                  >
                    {r.longLabel}
                  </Link>
                ) : (
                  r.longLabel
                )}
              </td>
              <td className="px-4 py-2.5 text-right tnum">{fmtMinutes(r.minutes)}</td>
              <td className="px-4 py-2.5 text-right tnum">{r.percent !== undefined ? fmtPercent(r.percent) : "—"}</td>
            </tr>
          ))}
        </tbody>
        <tfoot>
          <tr className="border-t-2 border-line-strong bg-surface-2/60 font-semibold">
            <th scope="row" className="px-4 py-3 text-left">
              Total {caption} · {period.routes_count} {period.routes_count === 1 ? "roteiro" : "roteiros"}
            </th>
            <td className="px-4 py-3 text-right tnum">{fmtMinutes(period.total_stopped_minutes)}</td>
            <td className="px-4 py-3 text-right tnum">{fmtPercent(period.journey_percent)}</td>
          </tr>
        </tfoot>
      </table>
    </Card>
  );
}

// ---- per-driver ranking ------------------------------------------------------

function Ranking({
  period,
  hours,
  from,
  to,
  self,
}: {
  period: PeriodSummary;
  hours: number;
  from: string;
  to: string;
  self: boolean;
}) {
  const ranking = [...(period.by_driver ?? [])].sort((a, b) => b.total_stopped_minutes - a.total_stopped_minutes);
  const max = Math.max(1, ...ranking.map((r) => r.total_stopped_minutes));
  return (
    <Card className="p-4 sm:p-5">
      <h2 className="display mb-1 text-base font-semibold">{self ? "Seu tempo no período" : "Por motorista"}</h2>
      <p className="mb-4 text-xs text-ink-3">
        {self ? "Seu tempo parado" : "Do mais parado ao menos parado"}, com a parte da jornada de{" "}
        {hours.toLocaleString("pt-BR")} h (por roteiro).
      </p>
      {ranking.length === 0 ? (
        <p className="py-6 text-sm text-ink-3">Sem motoristas com paradas no período.</p>
      ) : (
        <ol className="flex flex-col gap-4">
          {ranking.map((r, i) => (
            <li key={r.driver_user_id} className="grid grid-cols-[1.5rem_1fr] items-start gap-3">
              <span className="pt-0.5 text-sm font-semibold text-ink-3 tnum">{i + 1}º</span>
              <div className="min-w-0">
                <div className="flex items-baseline justify-between gap-3">
                  <Link
                    href={historyHref(from, to, r.driver_user_id)}
                    className="truncate font-semibold underline-offset-4 hover:text-placa-ink hover:underline"
                    aria-label={`Ver roteiros de ${r.driver_name} no período`}
                  >
                    {r.driver_name}
                  </Link>
                  <span className="shrink-0 text-sm tnum">
                    <span className="font-semibold">{fmtMinutes(r.total_stopped_minutes)}</span>
                    <span className="text-ink-3"> · {fmtPercent(r.journey_percent)}</span>
                  </span>
                </div>
                <div className="mt-1.5 h-2 rounded-sm bg-surface-2">
                  <div className="h-2 rounded-sm bg-cone" style={{ width: `${(r.total_stopped_minutes / max) * 100}%` }} />
                </div>
              </div>
            </li>
          ))}
        </ol>
      )}
    </Card>
  );
}

// ---- loading ---------------------------------------------------------------

function DashboardSkeleton() {
  return (
    <div aria-busy="true" aria-label="Carregando painel" className="flex flex-col gap-6">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {[0, 1, 2].map((i) => (
          <Skeleton key={i} className="h-24 w-full" />
        ))}
      </div>
      <Skeleton className="h-40 w-full" />
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Skeleton className="h-72 w-full" />
        <Skeleton className="h-72 w-full" />
      </div>
      <Skeleton className="h-48 w-full" />
    </div>
  );
}
