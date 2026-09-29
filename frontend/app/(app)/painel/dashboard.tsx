"use client";

import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useId, useState, type FormEvent } from "react";
import { BarSeries, type BarPoint } from "@/components/bar-series";
import { useUser } from "@/components/session-context";
import {
  Button,
  Card,
  cx,
  EmptyState,
  ErrorState,
  Field,
  Input,
  JourneyRuler,
  PageHeader,
  Skeleton,
} from "@/components/ui";
import { api, type DayPoint, type MonthPoint, type PeriodSummary } from "@/lib/api";
import { describeError } from "@/lib/errors";
import {
  addDaysISO,
  addMonthsISO,
  fmtDate,
  fmtMinutes,
  fmtMonth,
  fmtNumber,
  fmtPercent,
  monthStartISO,
  todayISO,
} from "@/lib/format";
import { isStaff } from "@/lib/roles";
import { useApi } from "@/lib/use-api";

type Tab = "dia" | "mes" | "periodo";

const tabs: { id: Tab; label: string }[] = [
  { id: "dia", label: "Por dia" },
  { id: "mes", label: "Por mês" },
  { id: "periodo", label: "Período" },
];

function presets(today: string) {
  return [
    { id: "hoje", label: "Hoje", from: today, to: today },
    { id: "7d", label: "7 dias", from: addDaysISO(today, -6), to: today },
    { id: "mes", label: "Este mês", from: monthStartISO(today), to: today },
    { id: "12m", label: "12 meses", from: addDaysISO(addMonthsISO(today, -12), 1), to: today },
  ];
}

interface DashboardData {
  day: DayPoint[];
  month: MonthPoint[];
  period: PeriodSummary;
  journeyHours: number;
}

export function Dashboard() {
  const user = useUser();
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const today = todayISO();
  const from = params.get("from") ?? monthStartISO(today);
  const to = params.get("to") ?? today;
  const tabParam = params.get("aba");
  const tab: Tab = tabParam === "mes" || tabParam === "periodo" ? tabParam : "dia";
  const baseId = useId();

  const data = useApi<DashboardData>(`${from}|${to}`, async () => {
    const w = { from, to };
    const [day, month, period, journeyHours] = await Promise.all([
      api.dashboardDay(w),
      api.dashboardMonth(w),
      api.dashboardPeriod(w),
      isStaff(user.role)
        ? api
            .params()
            .then((ps) => Number(ps.find((p) => p.key === "standard_journey_hours")?.value ?? 8))
            .catch(() => 8)
        : Promise.resolve(8),
    ]);
    return { day, month, period, journeyHours };
  });

  function setQuery(next: Record<string, string>) {
    const q = new URLSearchParams(params.toString());
    for (const [k, v] of Object.entries(next)) q.set(k, v);
    router.replace(`${pathname}?${q.toString()}`, { scroll: false });
  }

  const presetList = presets(today);
  const activePreset = presetList.find((p) => p.from === from && p.to === to)?.id;

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

      <div role="tablist" aria-label="Agrupamento" className="mb-4 flex gap-1 border-b border-line">
        {tabs.map((t) => (
          <button
            key={t.id}
            role="tab"
            type="button"
            id={`${baseId}-tab-${t.id}`}
            aria-selected={tab === t.id}
            aria-controls={`${baseId}-panel`}
            onClick={() => setQuery({ aba: t.id })}
            className={cx(
              "-mb-px border-b-2 px-4 py-2.5 text-sm font-semibold transition-colors",
              tab === t.id ? "border-placa text-ink" : "border-transparent text-ink-3 hover:text-ink",
            )}
          >
            {t.label}
          </button>
        ))}
      </div>

      <div id={`${baseId}-panel`} role="tabpanel" aria-labelledby={`${baseId}-tab-${tab}`}>
        {data.state === "error" ? (
          <ErrorState message={describeError(data.error)} onRetry={data.reload} />
        ) : !data.data || (data.state === "loading" && !data.data) ? (
          <DashboardSkeleton />
        ) : data.data.day.length === 0 && data.data.month.length === 0 ? (
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
          <div className={cx("transition-opacity", data.state === "loading" && "opacity-60")}>
            <Panels tab={tab} data={data.data} showRanking={isStaff(user.role)} from={from} to={to} />
          </div>
        )}
      </div>
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
          <Input id="range-to" name="to" type="date" defaultValue={to} className="sm:w-40" />
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

/**
 * The series carry only buckets with data. Days and months without stops
 * are drawn as empty slots (0 min) so the time axis stays honest; each
 * value itself comes straight from the API.
 */
function fillDays(series: DayPoint[], from: string, to: string) {
  const byDay = new Map(series.map((p) => [p.date.slice(0, 10), p.total_stopped_minutes]));
  const out: { key: string; minutes: number }[] = [];
  for (let d = from; d <= to && out.length < 400; d = addDaysISO(d, 1)) out.push({ key: d, minutes: byDay.get(d) ?? 0 });
  return out;
}

function fillMonths(series: MonthPoint[], from: string, to: string) {
  const byMonth = new Map(series.map((p) => [p.month.slice(0, 7), p.total_stopped_minutes]));
  const out: { key: string; minutes: number }[] = [];
  for (let m = `${from.slice(0, 7)}-01`; m.slice(0, 7) <= to.slice(0, 7) && out.length < 60; m = addMonthsISO(m, 1)) {
    out.push({ key: m.slice(0, 7), minutes: byMonth.get(m.slice(0, 7)) ?? 0 });
  }
  return out;
}

function Panels({
  tab,
  data,
  showRanking,
  from,
  to,
}: {
  tab: Tab;
  data: DashboardData;
  showRanking: boolean;
  from: string;
  to: string;
}) {
  const journeyMinutes = Math.round(data.journeyHours * 60);
  const [asTable, setAsTable] = useState(false);

  if (tab === "periodo") return <PeriodPanel period={data.period} hours={data.journeyHours} showRanking={showRanking} />;

  const points: BarPoint[] =
    tab === "dia"
      ? fillDays(data.day, from, to).map((p) => ({ key: p.key, label: fmtDate(p.key).slice(0, 5), minutes: p.minutes }))
      : fillMonths(data.month, from, to).map((p) => ({ key: p.key, label: fmtMonth(p.key), minutes: p.minutes }));
  const longLabel = (p: BarPoint) => (tab === "dia" ? fmtDate(p.key) : fmtMonth(p.key));

  return (
    <div className="grid grid-cols-1 gap-6 lg:grid-cols-[minmax(0,1fr)_280px]">
      <Card className="p-4 sm:p-6">
        <div className="mb-4 flex flex-wrap items-baseline justify-between gap-2">
          <h2 className="display text-lg font-semibold">
            {tab === "dia" ? "Minutos parados por dia" : "Minutos parados por mês"}
          </h2>
          <button
            type="button"
            onClick={() => setAsTable((v) => !v)}
            className="text-sm font-semibold text-placa underline-offset-4 hover:underline"
          >
            {asTable ? "Ver gráfico" : "Ver como tabela"}
          </button>
        </div>
        {points.length === 0 ? (
          <p className="py-10 text-center text-ink-3">Sem paradas neste agrupamento.</p>
        ) : asTable ? (
          <div className="max-h-80 overflow-auto">
            <table className="w-full text-sm">
              <thead className="sticky top-0 bg-surface text-left text-ink-3">
                <tr>
                  <th className="py-2 font-medium">{tab === "dia" ? "Dia" : "Mês"}</th>
                  <th className="py-2 text-right font-medium">Parado</th>
                  <th className="py-2 text-right font-medium">% de uma jornada</th>
                </tr>
              </thead>
              <tbody className="tnum">
                {points.map((p) => (
                  <tr key={p.key} className="border-t border-line">
                    <td className="py-2">{longLabel(p)}</td>
                    <td className="py-2 text-right">{fmtMinutes(p.minutes)}</td>
                    <td className="py-2 text-right">{fmtPercent((p.minutes / journeyMinutes) * 100)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <BarSeries
            points={points}
            journeyMinutes={journeyMinutes}
            showJourneyLine={tab === "dia"}
            ariaLabel={tab === "dia" ? "Gráfico de minutos parados por dia" : "Gráfico de minutos parados por mês"}
          />
        )}
      </Card>
      <PeriodTotals period={data.period} hours={data.journeyHours} />
    </div>
  );
}

function PeriodTotals({ period, hours }: { period: PeriodSummary; hours: number }) {
  const pct = Number(period.journey_percent);
  return (
    <Card className="flex flex-col gap-5 p-5">
      <div>
        <p className="text-sm text-ink-3">Total parado no período</p>
        <p className="display mt-1 text-3xl font-bold tnum">{fmtMinutes(period.total_stopped_minutes)}</p>
      </div>
      <div>
        {pct > 100 ? (
          <p className="text-sm text-ink-2">
            Equivale a{" "}
            <span className="font-semibold text-ink tnum">{fmtNumber(pct / 100, 1)} jornadas</span> de {hours} h
            <span className="block text-ink-3 tnum">({fmtPercent(period.journey_percent)} de uma jornada)</span>
          </p>
        ) : (
          <>
            <p className="mb-2 text-sm text-ink-2">
              <span className="font-semibold text-ink tnum">{fmtPercent(period.journey_percent)}</span> de uma jornada
              de {hours} h
            </p>
            <JourneyRuler percent={pct} hours={hours} />
          </>
        )}
      </div>
      <p className="border-t border-line pt-4 text-sm text-ink-2">
        <span className="font-semibold text-ink tnum">{period.routes_count}</span>{" "}
        {period.routes_count === 1 ? "roteiro" : "roteiros"} no período
      </p>
    </Card>
  );
}

function PeriodPanel({ period, hours, showRanking }: { period: PeriodSummary; hours: number; showRanking: boolean }) {
  const ranking = [...(period.by_driver ?? [])].sort((a, b) => b.total_stopped_minutes - a.total_stopped_minutes);
  const max = Math.max(1, ...ranking.map((r) => r.total_stopped_minutes));
  return (
    <div className="grid grid-cols-1 gap-6 lg:grid-cols-[280px_minmax(0,1fr)]">
      <PeriodTotals period={period} hours={hours} />
      <Card className="p-4 sm:p-6">
        <h2 className="display mb-1 text-lg font-semibold">{showRanking ? "Por motorista" : "Seus roteiros"}</h2>
        <p className="mb-5 text-sm text-ink-3">Do mais parado ao menos parado, com a parte de uma jornada de {hours} h.</p>
        {ranking.length === 0 ? (
          <p className="py-6 text-ink-3">Sem motoristas com paradas no período.</p>
        ) : (
          <ol className="flex flex-col gap-4">
            {ranking.map((r, i) => (
              <li key={r.driver_name} className="grid grid-cols-[1.5rem_1fr] items-start gap-3">
                <span className="pt-0.5 text-sm font-semibold text-ink-3 tnum">{i + 1}º</span>
                <div className="min-w-0">
                  <div className="flex items-baseline justify-between gap-3">
                    <span className="truncate font-semibold">{r.driver_name}</span>
                    <span className="shrink-0 text-sm tnum">
                      <span className="font-semibold">{fmtMinutes(r.total_stopped_minutes)}</span>
                      <span className="text-ink-3"> · {fmtPercent(r.journey_percent)}</span>
                    </span>
                  </div>
                  <div className="mt-1.5 h-2 rounded-sm bg-surface-2">
                    <div
                      className="h-2 rounded-sm bg-chart"
                      style={{ width: `${(r.total_stopped_minutes / max) * 100}%` }}
                    />
                  </div>
                </div>
              </li>
            ))}
          </ol>
        )}
      </Card>
    </div>
  );
}

function DashboardSkeleton() {
  return (
    <div className="grid grid-cols-1 gap-6 lg:grid-cols-[minmax(0,1fr)_280px]" aria-busy="true" aria-label="Carregando painel">
      <Card className="p-6">
        <Skeleton className="mb-6 h-5 w-52" />
        <div className="flex h-64 items-end gap-3">
          {[40, 65, 30, 80, 55, 70, 45, 60].map((h, i) => (
            <Skeleton key={i} className="flex-1" style={{ height: `${h}%` }} />
          ))}
        </div>
      </Card>
      <Card className="flex flex-col gap-4 p-5">
        <Skeleton className="h-4 w-32" />
        <Skeleton className="h-9 w-40" />
        <Skeleton className="h-3 w-full" />
      </Card>
    </div>
  );
}
