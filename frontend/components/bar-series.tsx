"use client";

import { Bar, BarChart, CartesianGrid, ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { fmtMinutes, fmtPercent } from "@/lib/format";

export interface BarPoint {
  key: string;
  label: string;
  minutes: number;
}

interface TooltipPayload {
  payload: BarPoint;
}

function ChartTooltip({
  active,
  payload,
  journeyMinutes,
}: {
  active?: boolean;
  payload?: TooltipPayload[];
  journeyMinutes: number;
}) {
  if (!active || !payload?.length) return null;
  const p = payload[0].payload;
  return (
    <div className="rounded-lg border border-line bg-surface px-3 py-2 text-sm shadow-card">
      <p className="font-semibold">{p.label}</p>
      <p className="tnum text-ink">{fmtMinutes(p.minutes)} parados</p>
      <p className="tnum text-ink-3">{fmtPercent((p.minutes / journeyMinutes) * 100)} de uma jornada</p>
    </div>
  );
}

/** 0..top in 4–6 round steps (1, 2, 2.5, 5 × 10ⁿ minutes). */
function niceTicks(top: number): number[] {
  if (top <= 0) return [0, 5, 10];
  const raw = top / 4;
  const mag = 10 ** Math.floor(Math.log10(raw));
  const step = [1, 2, 2.5, 5, 10].map((m) => m * mag).find((s) => s >= raw) ?? 10 * mag;
  const out: number[] = [];
  for (let v = 0; v < top + step; v += step) out.push(Math.round(v));
  return out;
}

/**
 * One aggregate series (SQL-computed minutes per bucket) as bars. The
 * chart only plots what the API returned — it never sums rows.
 */
export function BarSeries({
  points,
  journeyMinutes,
  showJourneyLine,
  ariaLabel,
}: {
  points: BarPoint[];
  journeyMinutes: number;
  showJourneyLine: boolean;
  ariaLabel: string;
}) {
  const max = Math.max(0, ...points.map((p) => p.minutes));
  const top = showJourneyLine ? Math.max(max, journeyMinutes) : max;
  const ticks = niceTicks(top);
  return (
    <div role="img" aria-label={ariaLabel} className="h-72 w-full sm:h-80">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={points} margin={{ top: 16, right: 8, bottom: 0, left: 0 }} barCategoryGap={points.length > 40 ? 1 : "22%"}>
          <CartesianGrid vertical={false} stroke="var(--line)" strokeDasharray="0" />
          <XAxis
            dataKey="label"
            tickLine={false}
            axisLine={{ stroke: "var(--line-strong)" }}
            tick={{ fill: "var(--ink-3)", fontSize: 12 }}
            minTickGap={16}
          />
          <YAxis
            width={44}
            tickLine={false}
            axisLine={false}
            domain={[0, ticks[ticks.length - 1]]}
            ticks={ticks}
            tick={{ fill: "var(--ink-3)", fontSize: 12 }}
            tickFormatter={(v: number) => v.toLocaleString("pt-BR")}
          />
          <Tooltip
            cursor={{ fill: "var(--surface-2)" }}
            content={(props) => (
              <ChartTooltip
                active={props.active}
                payload={props.payload as unknown as TooltipPayload[] | undefined}
                journeyMinutes={journeyMinutes}
              />
            )}
          />
          {showJourneyLine && (
            <ReferenceLine
              y={journeyMinutes}
              stroke="var(--ink-3)"
              strokeDasharray="5 4"
              label={{
                value: `jornada de ${fmtMinutes(journeyMinutes)}`,
                position: "insideTopRight",
                fill: "var(--ink-2)",
                fontSize: 12,
              }}
            />
          )}
          <Bar dataKey="minutes" fill="var(--chart)" radius={[4, 4, 0, 0]} maxBarSize={48} isAnimationActive={false} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
