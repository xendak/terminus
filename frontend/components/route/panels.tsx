"use client";

import { useMemo, useState, type FormEvent } from "react";
import { Button, Card, Field, Input, JourneyRuler, Select } from "@/components/ui";
import { api, type RouteView } from "@/lib/api";
import { fieldErrors } from "@/lib/errors";
import { decimalForInput, fmtBRL, fmtMinutes, fmtNumber, fmtPercent, parseDecimalInput } from "@/lib/format";
import { useApi } from "@/lib/use-api";
import type { RouteMutation } from "./route-screen";

function Stat({ label, value, sub }: { label: string; value: string; sub?: string }) {
  return (
    <div className="rounded-xl border border-line bg-surface px-4 py-3 shadow-card">
      <p className="text-xs font-medium text-ink-3">{label}</p>
      <p className="display mt-0.5 text-xl font-bold tnum">{value}</p>
      {sub && <p className="text-xs text-ink-3">{sub}</p>}
    </div>
  );
}

/** Running totals, always from the API's SQL aggregates. */
export function TotalsStrip({ route, narrow = false }: { route: RouteView; narrow?: boolean }) {
  const counted = route.stops.filter((s) => s.counted);
  const done = counted.filter((s) => s.departure_at).length;
  return (
    <div className={narrow ? "grid grid-cols-2 gap-3" : "grid grid-cols-2 gap-3 sm:grid-cols-4"}>
      <Stat label="Paradas concluídas" value={`${done} de ${counted.length}`} />
      <Stat label="Tempo parado" value={fmtMinutes(route.total_stopped_minutes)} />
      <Stat label="Parte da jornada" value={fmtPercent(route.journey_percent)} />
      <Stat
        label="Distância"
        value={route.distance_km ? `${fmtNumber(route.distance_km, 1)} km` : "—"}
        sub={route.distance_km ? undefined : "informada ao encerrar"}
      />
    </div>
  );
}

/** The closed-route summary: total stopped, journey share, estimated cost. */
export function ClosedSummary({ route }: { route: RouteView }) {
  const pct = Number(route.journey_percent);
  return (
    <Card className="overflow-hidden">
      <div className="grid gap-px bg-line sm:grid-cols-3">
        <div className="bg-surface p-5">
          <p className="text-sm text-ink-3">Tempo parado no dia</p>
          <p className="display mt-1 text-4xl font-bold tnum">{fmtMinutes(route.total_stopped_minutes)}</p>
          <p className="mt-1 text-sm text-ink-3 tnum">{route.total_stopped_seconds.toLocaleString("pt-BR")} segundos somados</p>
        </div>
        <div className="bg-surface p-5">
          <p className="text-sm text-ink-3">Parte da jornada</p>
          <p className="display mt-1 text-4xl font-bold tnum">{fmtPercent(route.journey_percent)}</p>
          <div className="mt-3">
            <JourneyRuler percent={pct} scale="percent" />
          </div>
        </div>
        <div className="bg-surface p-5">
          <p className="text-sm text-ink-3">Custo estimado</p>
          <p className="display mt-1 text-4xl font-bold tnum">
            {route.estimated_cost_brl ? fmtBRL(route.estimated_cost_brl) : "—"}
          </p>
          <p className="mt-1 text-sm text-ink-3 tnum">
            {route.distance_km
              ? `${fmtNumber(route.distance_km, 1)} km rodados`
              : "Sem distância informada, o custo não é calculado."}
          </p>
        </div>
      </div>
    </Card>
  );
}

/** Add a registered location to the route (draft or active; staff only). */
export function ComposePanel({ route, mutate }: { route: RouteView; mutate: RouteMutation }) {
  const locations = useApi("locations", () => api.locations());
  const [query, setQuery] = useState("");
  const [picked, setPicked] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const options = useMemo(() => {
    const q = query.trim().toLowerCase();
    const all = locations.data ?? [];
    return q ? all.filter((l) => `${l.label} ${l.address}`.toLowerCase().includes(q)) : all;
  }, [locations.data, query]);

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (!picked) return setError("Escolha um ponto.");
    setError(null);
    setBusy(true);
    const ok = await mutate(() => api.addStop(route.id, picked), "Ponto adicionado ao fim do roteiro.");
    if (ok) setPicked("");
    setBusy(false);
  }

  return (
    <Card className="p-5">
      <h2 className="display text-lg font-semibold">Adicionar ponto</h2>
      <p className="mt-1 text-sm text-ink-2">Entra no fim da lista. Use as setas para mudar a ordem.</p>
      <form method="post" action="/sem-js" onSubmit={submit} className="mt-4 flex flex-col gap-3" noValidate>
        <Field label="Buscar" htmlFor="add-q">
          <Input
            id="add-q"
            type="search"
            placeholder="Nome ou endereço"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </Field>
        <Field label="Ponto" htmlFor="add-loc" error={error ?? undefined}>
          <Select id="add-loc" value={picked} onChange={(e) => setPicked(e.target.value)} invalid={!!error}>
            <option value="">{locations.state === "loading" ? "Carregando…" : `${options.length} pontos`}</option>
            {options.map((l) => (
              <option key={l.id} value={l.id}>
                {l.label} — {l.address}
              </option>
            ))}
          </Select>
        </Field>
        <Button type="submit" busy={busy}>
          Adicionar ao roteiro
        </Button>
      </form>
    </Card>
  );
}

/** Completed state: distance (RN07) and close. Staff may close early. */
export function ClosePanel({
  route,
  mutate,
  pendingStops,
}: {
  route: RouteView;
  mutate: RouteMutation;
  pendingStops: number;
}) {
  const [distance, setDistance] = useState(route.distance_km ? decimalForInput(route.distance_km) : "");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [confirming, setConfirming] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    let km: string | undefined;
    if (distance.trim()) {
      const parsed = parseDecimalInput(distance);
      if (!parsed || Number(parsed) <= 0) return setError("Informe a distância em km, maior que zero. Ex.: 42,5");
      km = parsed;
    }
    if (pendingStops > 0 && !confirming) return setConfirming(true);
    setError(null);
    setBusy(true);
    try {
      await mutate(async () => {
        try {
          return await api.closeRoute(route.id, km);
        } catch (err) {
          const fe = fieldErrors(err);
          if (fe.distance_km) setError(fe.distance_km);
          throw err;
        }
      }, "Roteiro encerrado. Os horários estão congelados.");
    } finally {
      setBusy(false);
      setConfirming(false);
    }
  }

  return (
    <Card className="p-5">
      <h2 className="display text-lg font-semibold">
        {pendingStops === 0 ? "Todos os pontos concluídos" : "Encerrar roteiro"}
      </h2>
      <p className="mt-1 text-sm text-ink-2">
        {pendingStops === 0
          ? "Informe os km rodados no dia para calcular o custo e encerre o roteiro."
          : `Ainda ${pendingStops === 1 ? "falta 1 ponto" : `faltam ${pendingStops} pontos`}. Encerrar agora congela o roteiro como está.`}
      </p>
      <form method="post" action="/sem-js" onSubmit={submit} className="mt-4 flex flex-col gap-3" noValidate>
        <Field
          label="Distância percorrida (km)"
          htmlFor="close-km"
          error={error ?? undefined}
          hint="Hodômetro ou estimativa. Sem distância, o custo fica em branco."
        >
          <Input
            id="close-km"
            inputMode="decimal"
            placeholder="Ex.: 42,5"
            value={distance}
            onChange={(e) => setDistance(e.target.value)}
            invalid={!!error}
            className="h-12 text-lg tnum"
          />
        </Field>
        {confirming && (
          <p role="alert" className="rounded-lg bg-warn-soft px-3 py-2 text-sm text-warn-ink">
            Há pontos sem saída registrada. Toque de novo para encerrar mesmo assim.
          </p>
        )}
        <Button type="submit" variant={pendingStops === 0 ? "primary" : "danger"} size="lg" busy={busy}>
          {confirming ? "Encerrar mesmo assim" : "Encerrar roteiro"}
        </Button>
      </form>
    </Card>
  );
}

export function ReopenPanel({ route, mutate }: { route: RouteView; mutate: RouteMutation }) {
  const [busy, setBusy] = useState(false);
  return (
    <Card className="p-5">
      <h2 className="display text-lg font-semibold">Reabrir roteiro</h2>
      <p className="mt-1 text-sm text-ink-2">
        Volta o roteiro para “em andamento” para corrigir horários. A reabertura fica na auditoria.
      </p>
      <Button
        className="mt-4 w-full"
        busy={busy}
        onClick={async () => {
          setBusy(true);
          await mutate(() => api.reopenRoute(route.id), "Roteiro reaberto.");
          setBusy(false);
        }}
      >
        Reabrir roteiro
      </Button>
    </Card>
  );
}
