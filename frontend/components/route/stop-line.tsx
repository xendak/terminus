"use client";

import { useState, type FormEvent } from "react";
import { useUser } from "@/components/session-context";
import { Button, cx, Field, Input } from "@/components/ui";
import { api, type RouteView, type StopDetail } from "@/lib/api";
import { fieldErrors } from "@/lib/errors";
import { fmtClock, fmtDateTime, fmtTime, fromLocalInput, nowLocalInput, toLocalInput } from "@/lib/format";
import { useNow } from "@/lib/use-now";
import type { RouteMutation } from "./route-screen";

type StopState = "done" | "current" | "upcoming" | "idle";

/** Whole minutes of a finished stop, floored from seconds (RN02, display only). */
function stopMinutes(s: StopDetail): number | null {
  return s.stop_seconds === null ? null : Math.floor(s.stop_seconds / 60);
}

/**
 * The route as a transit line: stop 1 is the square terminal (departure,
 * never counted — RN01), the others are round stations. The current stop
 * opens up with the actions a driver needs, sized for a thumb.
 */
export function StopLine({
  route,
  mutate,
  canCompose,
  canCorrect,
  collapseDone = false,
}: {
  route: RouteView;
  mutate: RouteMutation;
  canCompose: boolean;
  canCorrect: boolean;
  /** Fold finished stops into one line so the current stop sits near the top. */
  collapseDone?: boolean;
}) {
  const [showDone, setShowDone] = useState(false);
  const currentOrder =
    route.status === "active" ? route.stops.find((s) => !s.departure_at)?.stop_order ?? null : null;
  const doneStops = route.stops.filter((s) => s.departure_at);
  const folded = collapseDone && currentOrder !== null && doneStops.length >= 2 && !showDone;

  return (
    <ol aria-label="Pontos do roteiro" className="flex flex-col">
      {folded && (
        <li className="grid grid-cols-[2.5rem_minmax(0,1fr)] gap-x-2">
          <div className="relative flex justify-center pt-3">
            <span aria-hidden className="absolute left-1/2 top-6 bottom-[-0.5rem] w-[3px] -translate-x-1/2 bg-placa" />
            <span aria-hidden className="relative z-10 flex h-6 w-6 items-center justify-center rounded-full bg-placa text-[11px] font-bold text-on-placa tnum">
              {doneStops.length}
            </span>
          </div>
          <button
            type="button"
            onClick={() => setShowDone(true)}
            className="mb-2 flex min-h-12 items-center justify-between gap-3 rounded-xl px-3 py-2 text-left hover:bg-surface-2"
          >
            <span>
              <span className="block font-semibold">
                {doneStops.length} {doneStops.length === 1 ? "ponto concluído" : "pontos concluídos"}
              </span>
              <span className="block text-sm text-ink-3">Última saída às {fmtTime(doneStops[doneStops.length - 1].departure_at)}</span>
            </span>
            <span className="shrink-0 text-sm font-semibold text-placa">Mostrar</span>
          </button>
        </li>
      )}
      {route.stops.map((stop, i) => {
        if (folded && stop.departure_at) return null;
        const last = i === route.stops.length - 1;
        const state: StopState =
          route.status === "draft"
            ? "idle"
            : stop.departure_at
              ? "done"
              : stop.stop_order === currentOrder
                ? "current"
                : "upcoming";
        return (
          <StopRow
            key={`${stop.stop_order}-${stop.label}`}
            route={route}
            stop={stop}
            state={state}
            last={last}
            mutate={mutate}
            canCompose={canCompose}
            canCorrect={canCorrect && route.status !== "closed" && !!(stop.arrival_at || stop.departure_at)}
          />
        );
      })}
    </ol>
  );
}

function Node({ stop, state }: { stop: StopDetail; state: StopState }) {
  const departure = stop.stop_order === 1;
  const base = "relative z-10 flex h-6 w-6 shrink-0 items-center justify-center";
  if (departure) {
    return (
      <span
        aria-hidden
        className={cx(
          base,
          "rounded-[5px]",
          state === "upcoming" || state === "idle" ? "border-[3px] border-placa bg-surface" : "bg-placa",
          state === "current" && "ring-4 ring-placa-soft",
        )}
      />
    );
  }
  return (
    <span
      aria-hidden
      className={cx(
        base,
        "rounded-full border-[3px]",
        state === "done" && "border-placa bg-placa",
        state === "current" && "pulse border-cone bg-surface",
        (state === "upcoming" || state === "idle") && "border-line-strong bg-surface",
      )}
    >
      {state === "done" && (
        <svg width="12" height="12" viewBox="0 0 12 12">
          <path d="M2.5 6.2 5 8.5l4.5-5" fill="none" stroke="var(--on-placa)" strokeWidth="2" strokeLinecap="round" />
        </svg>
      )}
      {state === "current" && stop.arrival_at && <span className="h-2 w-2 rounded-full bg-cone" />}
    </span>
  );
}

function StopRow({
  route,
  stop,
  state,
  last,
  mutate,
  canCompose,
  canCorrect,
}: {
  route: RouteView;
  stop: StopDetail;
  state: StopState;
  last: boolean;
  mutate: RouteMutation;
  canCompose: boolean;
  canCorrect: boolean;
}) {
  const departure = stop.stop_order === 1;
  const minutes = stopMinutes(stop);
  const [correcting, setCorrecting] = useState(false);

  return (
    <li
      aria-current={state === "current" ? "step" : undefined}
      className="relative grid grid-cols-[2.5rem_minmax(0,1fr)] gap-x-2"
    >
      {/* rail */}
      <div className="relative flex justify-center pt-4">
        {!last && (
          <span
            aria-hidden
            className={cx(
              "absolute left-1/2 top-7 bottom-[-1rem] w-[3px] -translate-x-1/2",
              state === "done" ? "bg-placa" : "bg-[repeating-linear-gradient(var(--line-strong)_0_6px,transparent_6px_11px)]",
            )}
          />
        )}
        <Node stop={stop} state={state} />
      </div>

      <div
        className={cx(
          "mb-2 min-w-0 rounded-xl px-3 py-3 transition-colors",
          state === "current" && "border-2 border-cone/70 bg-cone-soft/40 px-4 py-4",
          state === "upcoming" && "opacity-75",
        )}
      >
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="text-xs font-semibold uppercase tracking-[0.12em] text-ink-3">
              {departure ? "Partida" : `Parada ${stop.stop_order - 1}`}
              {state === "current" && <span className="ml-2 text-cone">agora</span>}
            </p>
            <p className={cx("display font-semibold leading-snug", state === "current" ? "text-xl" : "text-base")}>
              {stop.label}
            </p>
            <p className="text-sm text-ink-2">{stop.address}</p>
          </div>
          {departure ? (
            <span className="mt-1 shrink-0 rounded-md border border-line px-2 py-0.5 text-xs text-ink-3">não conta</span>
          ) : minutes !== null ? (
            <span className="mt-1 shrink-0 rounded-md bg-cone-soft px-2 py-1 font-mono text-sm font-semibold text-cone tnum">
              {minutes} min
            </span>
          ) : null}
        </div>

        {(stop.arrival_at || stop.departure_at) && state !== "current" && (
          <p className="mt-1.5 font-mono text-[13px] text-ink-2 tnum">
            {departure ? (
              <>Saída {fmtTime(stop.departure_at)}</>
            ) : (
              <>
                {fmtTime(stop.arrival_at)} → {fmtTime(stop.departure_at)}
              </>
            )}
          </p>
        )}

        {state === "current" && <TrackerActions route={route} stop={stop} mutate={mutate} />}

        {(canCompose || canCorrect) && (
          <div className="mt-2 flex flex-wrap items-center gap-1">
            {canCompose && <ComposeControls route={route} stop={stop} last={last} mutate={mutate} />}
            {canCorrect && (
              <button
                type="button"
                onClick={() => setCorrecting((v) => !v)}
                aria-expanded={correcting}
                className="h-11 rounded-lg px-2.5 text-sm font-semibold text-placa hover:bg-placa-soft sm:h-9"
              >
                {correcting ? "Fechar correção" : "Corrigir horários"}
              </button>
            )}
          </div>
        )}
        {correcting && canCorrect && (
          <CorrectionForm route={route} stop={stop} mutate={mutate} onDone={() => setCorrecting(false)} />
        )}
      </div>
    </li>
  );
}

function IconButton({
  label,
  onClick,
  disabled,
  children,
  danger,
}: {
  label: string;
  onClick: () => void;
  disabled?: boolean;
  children: React.ReactNode;
  danger?: boolean;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      onClick={onClick}
      disabled={disabled}
      className={cx(
        "flex h-11 w-11 items-center justify-center rounded-lg border border-line text-ink-2 hover:bg-surface-2 hover:text-ink disabled:opacity-35 sm:h-9 sm:w-9",
        danger && "hover:bg-danger-soft hover:text-danger",
      )}
    >
      {children}
    </button>
  );
}

function ComposeControls({
  route,
  stop,
  last,
  mutate,
}: {
  route: RouteView;
  stop: StopDetail;
  last: boolean;
  mutate: RouteMutation;
}) {
  const [busy, setBusy] = useState(false);
  const run = async (fn: () => Promise<RouteView>) => {
    setBusy(true);
    await mutate(fn);
    setBusy(false);
  };
  return (
    <>
      <IconButton
        label={`Subir ${stop.label}`}
        disabled={busy || stop.stop_order === 1}
        onClick={() => run(() => api.moveStop(route.id, stop.stop_order, "up"))}
      >
        <svg aria-hidden width="14" height="14" viewBox="0 0 14 14">
          <path d="M7 11V3M3.5 6.5 7 3l3.5 3.5" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
        </svg>
      </IconButton>
      <IconButton
        label={`Descer ${stop.label}`}
        disabled={busy || last}
        onClick={() => run(() => api.moveStop(route.id, stop.stop_order, "down"))}
      >
        <svg aria-hidden width="14" height="14" viewBox="0 0 14 14">
          <path d="M7 3v8M3.5 7.5 7 11l3.5-3.5" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
        </svg>
      </IconButton>
      <IconButton
        label={`Remover ${stop.label}`}
        danger
        disabled={busy || route.stops.length <= 2}
        onClick={() => run(() => api.removeStop(route.id, stop.stop_order))}
      >
        <svg aria-hidden width="14" height="14" viewBox="0 0 14 14">
          <path d="M3.5 3.5l7 7M10.5 3.5l-7 7" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
        </svg>
      </IconButton>
    </>
  );
}

/** The driver's controls for the current stop: one big action at a time. */
function TrackerActions({ route, stop, mutate }: { route: RouteView; stop: StopDetail; mutate: RouteMutation }) {
  const departure = stop.stop_order === 1;
  const arrived = !!stop.arrival_at;
  const now = useNow(arrived && !departure);
  const [busy, setBusy] = useState(false);
  const [manual, setManual] = useState(false);

  async function act(fn: () => Promise<unknown>, success: string) {
    setBusy(true);
    await mutate(async () => {
      await fn();
    }, success);
    setBusy(false);
  }

  const recordDeparturePoint = () =>
    act(async () => {
      // Stop 1 has no stopwatch: arrival and departure share one instant
      // when the driver leaves the base (RN01 keeps it at zero anyway).
      const at = new Date().toISOString();
      if (!stop.arrival_at) await api.arrive(route.id, 1, at);
      await api.depart(route.id, 1, stop.arrival_at ? undefined : at);
    }, "Saída da base registrada.");

  return (
    <div className="mt-4 flex flex-col gap-4">
      {departure ? (
        <p className="text-sm text-ink-2">
          Ponto de partida. O tempo aqui não conta como parado; registre quando sair.
        </p>
      ) : arrived ? (
        <div className="flex flex-col gap-1">
          <p className="text-sm text-ink-2">
            Chegou às <span className="font-mono font-semibold text-ink tnum">{fmtTime(stop.arrival_at)}</span>. Parado há
          </p>
          <p
            className="font-mono text-5xl font-semibold tracking-tight text-cone tnum sm:text-6xl"
            role="timer"
            aria-label="Tempo parado neste ponto"
          >
            {fmtClock((now - new Date(stop.arrival_at!).getTime()) / 1000)}
          </p>
        </div>
      ) : (
        <p className="text-sm text-ink-2">Toque ao chegar. O cronômetro começa na hora.</p>
      )}

      {departure ? (
        <BigButton tone="placa" busy={busy} onClick={recordDeparturePoint}>
          Registrar saída da base
        </BigButton>
      ) : arrived ? (
        <BigButton tone="cone" busy={busy} onClick={() => act(() => api.depart(route.id, stop.stop_order), "Saída registrada.")}>
          Saí deste ponto
        </BigButton>
      ) : (
        <BigButton tone="placa" busy={busy} onClick={() => act(() => api.arrive(route.id, stop.stop_order), "Chegada registrada.")}>
          Cheguei aqui
        </BigButton>
      )}

      {!departure && (
        <div>
          <button
            type="button"
            aria-expanded={manual}
            onClick={() => setManual((v) => !v)}
            className="min-h-11 text-sm font-semibold text-ink-2 underline underline-offset-4 hover:text-ink"
          >
            {manual ? "Cancelar horário manual" : `Informar ${arrived ? "saída" : "chegada"} manualmente`}
          </button>
          {manual && (
            <ManualTimeForm
              kind={arrived ? "departure" : "arrival"}
              onSubmit={async (at) => {
                const ok = await mutate(async () => {
                  if (arrived) await api.depart(route.id, stop.stop_order, at);
                  else await api.arrive(route.id, stop.stop_order, at);
                }, arrived ? "Saída registrada." : "Chegada registrada.");
                if (ok) setManual(false);
              }}
            />
          )}
        </div>
      )}
    </div>
  );
}

function BigButton({
  tone,
  busy,
  onClick,
  children,
}: {
  tone: "placa" | "cone";
  busy: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={busy}
      className={cx(
        "flex min-h-16 w-full items-center justify-center gap-3 rounded-xl px-6 text-lg font-bold shadow-card transition-transform active:scale-[0.99] disabled:opacity-60",
        tone === "placa" ? "bg-placa text-on-placa hover:bg-placa-strong" : "bg-cone text-white hover:bg-cone-bright",
      )}
    >
      {busy && <span aria-hidden className="h-5 w-5 animate-spin rounded-full border-2 border-current border-r-transparent" />}
      {children}
    </button>
  );
}

function ManualTimeForm({
  kind,
  onSubmit,
}: {
  kind: "arrival" | "departure";
  onSubmit: (at: string) => Promise<void>;
}) {
  const [value, setValue] = useState(nowLocalInput);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const id = `manual-${kind}`;
  async function submit(e: FormEvent) {
    e.preventDefault();
    if (!value) return setError("Informe data e hora.");
    setError(null);
    setBusy(true);
    await onSubmit(fromLocalInput(value));
    setBusy(false);
  }
  return (
    <form onSubmit={submit} className="mt-2 flex flex-wrap items-end gap-3" noValidate>
      <Field
        label={kind === "arrival" ? "Horário da chegada" : "Horário da saída"}
        htmlFor={id}
        error={error ?? undefined}
        hint="Horário de Brasília."
      >
        <Input
          id={id}
          type="datetime-local"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          invalid={!!error}
          className="h-11"
        />
      </Field>
      <Button type="submit" variant="primary" size="lg" busy={busy}>
        Registrar {kind === "arrival" ? "chegada" : "saída"}
      </Button>
    </form>
  );
}

function CorrectionForm({
  route,
  stop,
  mutate,
  onDone,
}: {
  route: RouteView;
  stop: StopDetail;
  mutate: RouteMutation;
  onDone: () => void;
}) {
  const user = useUser();
  const [arrival, setArrival] = useState(toLocalInput(stop.arrival_at));
  const [departure, setDeparture] = useState(toLocalInput(stop.departure_at));
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);
  const idBase = `corr-${stop.stop_order}`;

  async function submit(e: FormEvent) {
    e.preventDefault();
    if ((stop.arrival_at && !arrival) || (stop.departure_at && !departure)) {
      return setErrors({
        [stop.arrival_at && !arrival ? "arrival_at" : "departure_at"]:
          "Um horário registrado não pode ficar em branco. Informe o horário correto.",
      });
    }
    if (arrival && departure && departure < arrival) {
      return setErrors({ departure_at: "A saída não pode ser antes da chegada." });
    }
    const times: { arrival_at?: string; departure_at?: string } = {};
    if (arrival && arrival !== toLocalInput(stop.arrival_at)) times.arrival_at = fromLocalInput(arrival);
    if (departure && departure !== toLocalInput(stop.departure_at)) times.departure_at = fromLocalInput(departure);
    if (!times.arrival_at && !times.departure_at) return setErrors({ form: "Nenhum horário foi alterado." });
    setErrors({});
    setBusy(true);
    try {
      await api.correctTimes(route.id, stop.stop_order, times);
      await mutate(async () => undefined, `Horários de “${stop.label}” corrigidos. A mudança foi para a auditoria.`);
      onDone();
    } catch (err) {
      const fe = fieldErrors(err);
      setErrors(fe.arrival_at || fe.departure_at ? fe : { form: Object.values(fe)[0] });
    } finally {
      setBusy(false);
    }
  }

  return (
    <form onSubmit={submit} className="rise mt-3 rounded-lg border border-line bg-surface p-4" noValidate>
      <p className="mb-3 text-sm text-ink-2">
        Correção feita por {user.name}, em horário de Brasília. Os valores antigos e novos ficam registrados na
        auditoria. Um horário já registrado não pode ser apagado, só trocado.
      </p>
      <div className="grid gap-3 sm:grid-cols-2">
        <Field
          label="Chegada"
          htmlFor={`${idBase}-a`}
          error={errors.arrival_at}
          hint={stop.arrival_at ? "Deixe como está para manter." : "Opcional."}
        >
          <Input
            id={`${idBase}-a`}
            type="datetime-local"
            value={arrival}
            onChange={(e) => setArrival(e.target.value)}
            invalid={!!errors.arrival_at}
          />
        </Field>
        <Field
          label="Saída"
          htmlFor={`${idBase}-d`}
          error={errors.departure_at}
          hint={stop.departure_at ? "Deixe como está para manter." : "Opcional."}
        >
          <Input
            id={`${idBase}-d`}
            type="datetime-local"
            value={departure}
            onChange={(e) => setDeparture(e.target.value)}
            invalid={!!errors.departure_at}
          />
        </Field>
      </div>
      {errors.form && (
        <p role="alert" className="mt-3 text-sm font-medium text-danger">
          {errors.form}
        </p>
      )}
      <div className="mt-4 flex flex-wrap items-center gap-2">
        <Button type="submit" variant="primary" busy={busy}>
          Salvar correção
        </Button>
        <Button type="button" variant="ghost" onClick={onDone}>
          Cancelar
        </Button>
        {stop.arrival_at && (
          <span className="text-xs text-ink-3 tnum">
            Atual: {fmtDateTime(stop.arrival_at)} → {fmtDateTime(stop.departure_at)}
          </span>
        )}
      </div>
    </form>
  );
}
