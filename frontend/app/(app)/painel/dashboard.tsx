"use client";

import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useState, type FormEvent } from "react";
import {
  Button,
  Card,
  cx,
  ErrorState,
  Field,
  Input,
  PageHeader,
  Skeleton,
} from "@/components/ui";
import { useUser } from "@/components/session-context";
import { api, type DayPoint, type MonthPoint, type PeriodSummary, type StopDetail } from "@/lib/api";
import { describeError } from "@/lib/errors";
import { addDaysISO, addMonthsISO, fmtDate, fmtMinutes, fmtMonth, fmtPercent, fmtTime, monthStartISO, todayISO } from "@/lib/format";
import { isStaff } from "@/lib/roles";
import { useApi } from "@/lib/use-api";

// ---- shapes for the per-day detail (timeline + points list) ---------------

interface DriverDay {
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
  address: string;
  arrivalAt: string;
  departureAt: string;
  stopSeconds: number;
}

interface DayDetail {
  drivers: DriverDay[];
  stops: DayStop[];
}

interface DashboardData {
  day: DayPoint[];
  month: MonthPoint[];
  /** Last 12 days ending at `to` — the single-day chart context. */
  chartDay: DayPoint[];
  period: PeriodSummary;
  journeyHours: number;
}

// A stop at or above this is "prolonged" (red in the timeline). 45 min.
const PROLONGED_S = 45 * 60;
// Points-list status thresholds (seconds).
const WARN_S = 15 * 60;
const OVER_S = 45 * 60;
// Ranges up to this many days chart by day; longer ones chart by month.
const DAY_GROUP_MAX = 31;

interface Segment {
  type: "origin" | "idle" | "long";
  left: number;
  width: number;
  min?: number;
}

/** Build a 0–100% track from a driver's counted stops (arrival→departure). */
function buildTimeline(driver: DriverDay): { segments: Segment[]; alert: boolean } | null {
  const stops = driver.stops
    .filter((s) => s.counted && s.arrival_at && s.departure_at)
    .sort((a, b) => a.stop_order - b.stop_order);
  if (stops.length === 0) return null;
  const start = new Date(stops[0].arrival_at as string).getTime();
  const end = new Date(stops[stops.length - 1].departure_at as string).getTime();
  const span = Math.max(1, end - start);
  const pct = (t: number) => ((t - start) / span) * 100;
  const segments: Segment[] = [{ type: "origin", left: 0, width: 3 }];
  let alert = false;
  for (const s of stops) {
    const a = new Date(s.arrival_at as string).getTime();
    const d = new Date(s.departure_at as string).getTime();
    const left = Math.max(4, pct(a));
    const width = Math.max(1.5, pct(d) - left);
    const long = (s.stop_seconds ?? 0) >= PROLONGED_S;
    if (long) alert = true;
    segments.push({ type: long ? "long" : "idle", left, width, min: Math.round((d - a) / 60000) });
  }
  return { segments, alert };
}

function stopStatus(seconds: number) {
  if (seconds >= OVER_S) return { label: "Acima do limite", cls: "bg-danger-soft text-danger" };
  if (seconds >= WARN_S) return { label: "Atenção", cls: "bg-cone-soft text-cone" };
  return { label: "Normal", cls: "bg-placa-soft text-placa-strong" };
}

function presets(today: string) {
  return [
    { id: "hoje", label: "Hoje", from: today, to: today },
    { id: "7d", label: "7 dias", from: addDaysISO(today, -6), to: today },
    { id: "mes", label: "Este mês", from: monthStartISO(today), to: today },
    { id: "12m", label: "12 meses", from: addDaysISO(addMonthsISO(today, -12), 1), to: today },
  ];
}

// ---- screen ----------------------------------------------------------------

export function Dashboard() {
  const user = useUser();
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const today = todayISO();
  const from = params.get("from") ?? monthStartISO(today);
  const to = params.get("to") ?? today;

  const spanDays = Math.max(0, Math.round((Date.parse(to) - Date.parse(from)) / 86400000));
  const isSingleDay = from === to;
  const groupBy: "day" | "month" = spanDays <= DAY_GROUP_MAX ? "day" : "month";
  const [driverFilter, setDriverFilter] = useState("");

  const data = useApi<DashboardData>(`${from}|${to}`, async () => {
    const w = { from, to };
    const [day, month, period, chartDay] = await Promise.all([
      api.dashboardDay(w),
      api.dashboardMonth(w),
      api.dashboardPeriod(w),
      api.dashboardDay({ from: addDaysISO(to, -11), to: to }),
    ]);
    let hours = day.hours ?? month.hours ?? period.standard_journey_hours;
    if (hours === undefined && isStaff(user.role)) {
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
    };
  });

  // Timeline + points are single-day views, anchored on `to`. Skipped (null key)
  // for multi-day ranges so we don't pay the per-route fetch.
  const detail = useApi<DayDetail>(isSingleDay ? to : null, async () => {
    const rows = (await api.routes({ from: to, to })).filter((r) => r.status === "active" || r.status === "closed");
    const views = await Promise.all(rows.map((r) => api.route(r.id)));
    let vehicles = new Map<string, string>();
    if (isStaff(user.role)) {
      try {
        vehicles = new Map((await api.drivers()).map((d) => [d.id, d.vehicle_plate ?? ""]));
      } catch {
        /* a driver can't list peers; the timeline just omits the plate */
      }
    }
    const drivers: DriverDay[] = views.map((v) => ({
      driverId: v.driver_user_id,
      driverName: v.driver_name,
      vehicle: vehicles.get(v.driver_user_id) || undefined,
      stops: v.stops,
    }));
    const stops: DayStop[] = views.flatMap((v) =>
      v.stops
        .filter((s) => s.counted && s.arrival_at && s.departure_at)
        .map((s) => ({
          routeId: v.id,
          driverId: v.driver_user_id,
          driverName: v.driver_name,
          stopOrder: s.stop_order,
          address: s.address,
          arrivalAt: s.arrival_at as string,
          departureAt: s.departure_at as string,
          stopSeconds: s.stop_seconds ?? 0,
        })),
    );
    return { drivers, stops };
  });

  function setQuery(next: Record<string, string>) {
    const q = new URLSearchParams(params.toString());
    for (const [k, v] of Object.entries(next)) q.set(k, v);
    router.replace(`${pathname}?${q.toString()}`, { scroll: false });
  }

  const presetList = presets(today);
  const activePreset = presetList.find((p) => p.from === from && p.to === to)?.id;
  const caption = from === to ? (from === today ? "hoje" : fmtDate(from)) : `${fmtDate(from)} – ${fmtDate(to)}`;

  const dd = data.data;
  const chartPoints =
    isSingleDay
      ? (dd?.chartDay ?? []).map((d) => ({ key: d.date, label: fmtDate(d.date).slice(0, 5), minutes: d.total_stopped_minutes }))
      : groupBy === "day"
        ? (dd?.day ?? []).map((d) => ({ key: d.date, label: fmtDate(d.date).slice(0, 5), minutes: d.total_stopped_minutes }))
        : (dd?.month ?? []).map((m) => ({ key: m.month, label: fmtMonth(m.month), minutes: m.total_stopped_minutes }));

  const driverOptions = (detail.data?.drivers ?? []).map((d) => ({ value: d.driverId, label: d.driverName }));
  const filteredDrivers = (detail.data?.drivers ?? []).filter((d) => !driverFilter || d.driverId === driverFilter);
  const filteredStops = (detail.data?.stops ?? []).filter((s) => !driverFilter || s.driverId === driverFilter);

  return (
    <>
      <PageHeader
        title={user.role === "driver" ? "Meu tempo parado" : "Painel"}
        eyebrow={`${fmtDate(from)} a ${fmtDate(to)}`}
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
      ) : data.data == null ? (
        <DashboardSkeleton />
      ) : (
        <div className={cx("flex flex-col gap-6 transition-opacity", data.state === "loading" && "opacity-60")}>
          <KpiStrip period={dd!.period} hours={dd!.journeyHours} caption={caption} />

          {isSingleDay && (
            <TimelineByDriver
              drivers={filteredDrivers}
              driverOptions={driverOptions}
              driverFilter={driverFilter}
              onDriverFilter={setDriverFilter}
              day={to}
              loading={detail.data == null}
            />
          )}

          <div className={cx("grid grid-cols-1 gap-6", isSingleDay && "lg:grid-cols-2")}>
            <DailyChart points={chartPoints} unit={groupBy} />
            {isSingleDay && <PointsList stops={filteredStops} day={to} loading={detail.data == null} />}
          </div>

          <DashboardTable
            groupBy={groupBy}
            rows={groupBy === "day" ? dd!.day : dd!.month}
            period={dd!.period}
            caption={caption}
          />
        </div>
      )}
    </>
  );
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
  const total = period.total_stopped_minutes;
  const routes = period.routes_count;
  const avg = routes > 0 ? Math.round(total / routes) : 0;
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <KpiHero value={fmtMinutes(total)} caption={`no período ${caption}`} />
      <Kpi label="Média por roteiro" value={routes > 0 ? fmtMinutes(avg) : "—"} caption={`${routes} ${routes === 1 ? "roteiro" : "roteiros"}`} />
      <Kpi label="% da jornada" value={fmtPercent(period.journey_percent)} caption={`de ${hours.toLocaleString("pt-BR")} h por roteiro`} />
      <Kpi label="Roteiros no período" value={String(routes)} caption="com paradas registradas" />
    </div>
  );
}

function KpiHero({ value, caption }: { value: string; caption: string }) {
  return (
    <div className="rounded-xl border border-ink/20 bg-ink p-5 text-paper shadow-card dark:border-paper/20 dark:bg-paper dark:text-ink">
      <p className="text-[11px] font-semibold uppercase tracking-wide text-paper/60 dark:text-ink/60">Tempo parado total</p>
      <p className="display mt-1.5 text-3xl font-bold text-cone tnum">{value}</p>
      <p className="mt-1 text-xs text-paper/60 dark:text-ink/60">{caption}</p>
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

function TimelineByDriver({
  drivers,
  driverOptions,
  driverFilter,
  onDriverFilter,
  day,
  loading,
}: {
  drivers: DriverDay[];
  driverOptions: { value: string; label: string }[];
  driverFilter: string;
  onDriverFilter: (v: string) => void;
  day: string;
  loading: boolean;
}) {
  const rows = drivers
    .map((d) => ({ d, tl: buildTimeline(d) }))
    .filter((x): x is { d: DriverDay; tl: { segments: Segment[]; alert: boolean } } => x.tl !== null);
  return (
    <Card className="p-4 sm:p-5">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="display text-base font-semibold">Linha do tempo por motorista</h2>
          <p className="text-xs text-ink-3">laranja = parado · vermelho = parada prolongada</p>
        </div>
        {driverOptions.length > 1 && (
          <label className="flex items-center gap-2 text-xs font-medium text-ink-3">
            Motorista
            <select
              value={driverFilter}
              onChange={(e) => onDriverFilter(e.target.value)}
              className="h-9 rounded-md border border-line-strong bg-surface px-2.5 text-sm font-normal text-ink"
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
      {loading ? (
        <div className="flex flex-col gap-4" aria-busy="true">
          {[0, 1, 2].map((i) => (
            <div key={i} className="grid grid-cols-[180px_1fr] items-center gap-4">
              <Skeleton className="h-6 w-40" />
              <Skeleton className="h-3 w-full" />
            </div>
          ))}
        </div>
      ) : rows.length === 0 ? (
        <p className="py-6 text-center text-sm text-ink-3">
          {driverFilter ? "Nenhuma parada registrada para este motorista." : `Nenhum roteiro com paradas registradas em ${fmtDate(day)}.`}
        </p>
      ) : (
        <div className="flex flex-col gap-4">
          {rows.map(({ d, tl }) => (
            <div key={d.driverId} className="grid grid-cols-[minmax(0,180px)_1fr] items-center gap-4">
              <div className="min-w-0">
                <p className="truncate text-sm font-semibold">{d.driverName}</p>
                <p className="flex items-center gap-1.5 text-xs text-ink-3">
                  <span aria-hidden className={cx("h-1.5 w-1.5 rounded-full", tl.alert ? "bg-danger" : "bg-placa")} />
                  {d.vehicle ?? "em rota"}
                </p>
              </div>
              <div className="relative h-2.5 rounded-full bg-surface-2">
                {tl.segments.map((s, i) => (
                  <div
                    key={i}
                    title={s.min ? `${s.min} min parado` : "partida"}
                    className={cx(
                      "absolute top-0 h-2.5 rounded-full",
                      s.type === "origin" ? "bg-line-strong" : s.type === "long" ? "bg-danger" : "bg-cone",
                    )}
                    style={{ left: `${s.left}%`, width: `${s.width}%` }}
                  />
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </Card>
  );
}

// ---- bar chart -------------------------------------------------------------

function DailyChart({ points, unit }: { points: { key: string; label: string; minutes: number }[]; unit: "day" | "month" }) {
  const max = Math.max(1, ...points.map((p) => p.minutes));
  const peak = points.findIndex((p) => p.minutes === max && p.minutes > 0);
  return (
    <Card className="p-4 sm:p-5">
      <h2 className="display mb-1 text-base font-semibold">{unit === "month" ? "Tempo parado por mês" : "Tempo parado por dia"}</h2>
      <p className="mb-4 text-xs text-ink-3">
        minutos parados {unit === "month" ? "por mês" : "por dia"} no período
      </p>
      {points.length === 0 ? (
        <p className="py-10 text-center text-sm text-ink-3">Sem paradas neste agrupamento.</p>
      ) : (
        <>
          <div className="flex h-48 items-end gap-1.5">
            {points.map((p, i) => {
              const h = Math.round((p.minutes / max) * 100);
              const isPeak = i === peak;
              return (
                <div
                  key={p.key}
                  className="flex h-full flex-1 flex-col items-center justify-end gap-1"
                  title={`${p.label}: ${fmtMinutes(p.minutes)}`}
                >
                  <div className="relative w-full flex-1 overflow-hidden rounded-t-sm bg-placa-soft">
                    <div
                      className={cx("absolute bottom-0 w-full rounded-t-sm", isPeak ? "bg-danger" : "bg-cone")}
                      style={{ height: `${Math.max(p.minutes > 0 ? 4 : 0, h)}%` }}
                    />
                  </div>
                  <span className="text-[10px] text-ink-3 tnum">{p.label}</span>
                </div>
              );
            })}
          </div>
          <div className="mt-4 flex flex-wrap gap-x-4 gap-y-1 text-xs text-ink-3">
            <span className="flex items-center gap-1.5"><span className="h-2.5 w-2.5 rounded-sm bg-placa-soft" /> base</span>
            <span className="flex items-center gap-1.5"><span className="h-2.5 w-2.5 rounded-sm bg-cone" /> tempo parado</span>
            <span className="flex items-center gap-1.5"><span className="h-2.5 w-2.5 rounded-sm bg-danger" /> pico</span>
          </div>
        </>
      )}
    </Card>
  );
}

// ---- points list -----------------------------------------------------------

function PointsList({ stops, day, loading }: { stops: DayStop[]; day: string; loading: boolean }) {
  const sorted = [...stops].sort((a, b) => b.stopSeconds - a.stopSeconds);
  return (
    <Card className="p-4 sm:p-5">
      <h2 className="display mb-1 text-base font-semibold">Histórico de pontos</h2>
      <p className="mb-3 text-xs text-ink-3">paradas registradas em {fmtDate(day)}</p>
      {loading ? (
        <div className="flex flex-col gap-3" aria-busy="true">
          {[0, 1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-10 w-full" />
          ))}
        </div>
      ) : sorted.length === 0 ? (
        <p className="py-6 text-center text-sm text-ink-3">Nenhuma parada com registro de horário.</p>
      ) : (
        <ul className="flex flex-col">
          {sorted.map((s, i) => {
            const st = stopStatus(s.stopSeconds);
            return (
              <li key={`${s.routeId}-${s.stopOrder}`} className={cx(i > 0 && "border-t border-line")}>
                <div className="flex items-center gap-3 py-3">
                  <span className="grid h-9 w-9 shrink-0 place-items-center rounded-md bg-surface-2 text-sm font-semibold text-ink-2 tnum">
                    {s.stopOrder}
                  </span>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium">{s.address}</p>
                    <p className="truncate text-xs text-ink-3">
                      {s.driverName} · chegada {fmtTime(s.arrivalAt)} · saída {fmtTime(s.departureAt)}
                    </p>
                  </div>
                  <span className="shrink-0 text-sm font-semibold tnum">{fmtMinutes(Math.round(s.stopSeconds / 60))}</span>
                  <span className={cx("shrink-0 rounded-full px-2.5 py-0.5 text-xs font-semibold", st.cls)}>{st.label}</span>
                </div>
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
  rows,
  period,
  caption,
}: {
  groupBy: "day" | "month";
  rows: (DayPoint | MonthPoint)[];
  period: PeriodSummary;
  caption: string;
}) {
  return (
    <Card className="overflow-x-auto p-0">
      <table className="w-full min-w-[520px] text-sm">
        <thead>
          <tr className="border-b border-line text-left">
            <th className="px-4 py-3 font-semibold text-ink-3">{groupBy === "day" ? "Dia" : "Mês"}</th>
            <th className="px-4 py-3 text-right font-semibold text-ink-3">Tempo parado</th>
            <th className="px-4 py-3 text-right font-semibold text-ink-3">% da jornada</th>
            <th className="px-4 py-3 text-right font-semibold text-ink-3">Roteiros</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r, i) => {
            const day = r as DayPoint;
            const month = r as MonthPoint;
            return (
              <tr key={day.date ?? month.month} className={cx("border-b border-line last:border-0", i % 2 === 1 && "bg-surface-2/40")}>
                <td className="px-4 py-2.5">{day.date ? fmtDate(day.date) : fmtMonth(month.month!)}</td>
                <td className="px-4 py-2.5 text-right tnum">{fmtMinutes(r.total_stopped_minutes)}</td>
                <td className="px-4 py-2.5 text-right tnum">{fmtPercent(r.journey_percent)}</td>
                <td className="px-4 py-2.5 text-right text-ink-3">—</td>
              </tr>
            );
          })}
          <tr className="border-t-2 border-line-strong bg-surface-2/60 font-semibold">
            <td className="px-4 py-3">Total ({caption})</td>
            <td className="px-4 py-3 text-right tnum">{fmtMinutes(period.total_stopped_minutes)}</td>
            <td className="px-4 py-3 text-right tnum">{fmtPercent(period.journey_percent)}</td>
            <td className="px-4 py-3 text-right tnum">{period.routes_count}</td>
          </tr>
        </tbody>
      </table>
    </Card>
  );
}

// ---- loading ---------------------------------------------------------------

function DashboardSkeleton() {
  return (
    <div aria-busy="true" aria-label="Carregando painel" className="flex flex-col gap-6">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {[0, 1, 2, 3].map((i) => (
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
