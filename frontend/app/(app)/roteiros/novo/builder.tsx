"use client";

import Link from "next/link";
import { useMemo, useState, type FormEvent } from "react";
import { useUser } from "@/components/session-context";
import {
  Button,
  ButtonLink,
  Card,
  cx,
  ErrorState,
  Field,
  Forbidden,
  Input,
  Notice,
  PageHeader,
  Select,
} from "@/components/ui";
import { api, ApiError, type Location, type RouteListRow, type RouteView } from "@/lib/api";
import { describeError, fieldErrors } from "@/lib/errors";
import { addDaysISO, fmtDate, todayISO } from "@/lib/format";
import { isStaff } from "@/lib/roles";
import { useApi } from "@/lib/use-api";

export function Builder() {
  const user = useUser();
  if (!isStaff(user.role)) return <Forbidden />;
  return <BuilderForm />;
}

function BuilderForm() {
  const drivers = useApi("drivers-active", () => api.drivers(true));
  const locations = useApi("locations", () => api.locations());
  const [driverId, setDriverId] = useState("");
  const [date, setDate] = useState(() => addDaysISO(todayISO(), 1));
  const [stops, setStops] = useState<Location[]>([]);
  const [note, setNote] = useState("");
  const [query, setQuery] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState<RouteView | null>(null);

  // RN05 pre-check: does this driver already have a route that day?
  const conflict = useApi<RouteListRow | null>(driverId && date ? `${driverId}|${date}` : null, () =>
    api.routes({ from: date, to: date, driver_user_id: driverId }).then((rs) => rs[0] ?? null),
  );
  const existing = driverId && date && conflict.state === "ready" ? conflict.data : null;

  const results = useMemo(() => {
    const q = query.trim().toLowerCase();
    const all = locations.data ?? [];
    return (q ? all.filter((l) => `${l.label} ${l.address}`.toLowerCase().includes(q)) : all).slice(0, 30);
  }, [locations.data, query]);

  function move(i: number, dir: -1 | 1) {
    setStops((s) => {
      const next = [...s];
      const j = i + dir;
      [next[i], next[j]] = [next[j], next[i]];
      return next;
    });
  }

  async function submit(e: FormEvent) {
    e.preventDefault();
    const errs: Record<string, string> = {};
    if (!driverId) errs.driver_user_id = "Escolha o motorista.";
    if (!date) errs.route_date = "Escolha a data.";
    if (stops.length < 2) errs.location_ids = "Adicione a partida e pelo menos uma parada.";
    if (existing) errs.route_date = "Esse motorista já tem roteiro nessa data.";
    setErrors(errs);
    if (Object.keys(errs).length) return;
    setSaving(true);
    try {
      const route = await api.createRoute({
        driver_user_id: driverId,
        route_date: date,
        location_ids: stops.map((s) => s.id),
        note: note.trim() || null,
      });
      setSaved(route);
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        conflict.reload();
        setErrors({ route_date: describeError(err) });
      } else {
        const fe = fieldErrors(err);
        setErrors(fe.location_ids || fe.driver_user_id || fe.route_date ? fe : { form: Object.values(fe)[0] });
      }
    } finally {
      setSaving(false);
    }
  }

  function reset() {
    setSaved(null);
    setStops([]);
    setNote("");
    setDriverId("");
    setErrors({});
  }

  if (saved) {
    return (
      <>
        <PageHeader title="Novo roteiro" />
        <Card className="rise max-w-xl p-6">
          <p className="text-sm font-semibold text-placa">Roteiro criado</p>
          <h2 className="display mt-1 text-2xl font-bold">
            {saved.driver_name}, {fmtDate(saved.route_date)}
          </h2>
          <p className="mt-2 text-ink-2">
            {saved.stops.length} pontos, a partir de {saved.stops[0]?.label}. O motorista vê o roteiro na tela “Meu
            roteiro de hoje” no dia.
          </p>
          <div className="mt-5 flex flex-wrap gap-2">
            <ButtonLink href={`/roteiros/${saved.id}`} variant="primary">
              Abrir roteiro
            </ButtonLink>
            <Button onClick={reset}>Montar outro</Button>
          </div>
        </Card>
      </>
    );
  }

  if (drivers.state === "error" || locations.state === "error") {
    return (
      <ErrorState
        message={describeError(drivers.state === "error" ? drivers.error : locations.state === "error" ? locations.error : null)}
        onRetry={() => {
          drivers.reload();
          locations.reload();
        }}
      />
    );
  }

  const inList = new Set(stops.map((s) => s.id));

  return (
    <>
      <PageHeader title="Novo roteiro">
        Escolha o motorista, a data e os pontos na ordem da visita. O primeiro ponto é a partida e não conta tempo
        parado.
      </PageHeader>
      <form onSubmit={submit} noValidate className="grid grid-cols-1 gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
        <div className="flex flex-col gap-6">
          <Card className="grid gap-4 p-5 sm:grid-cols-2">
            <Field label="Motorista" htmlFor="b-driver" error={errors.driver_user_id}>
              <Select
                id="b-driver"
                value={driverId}
                onChange={(e) => setDriverId(e.target.value)}
                invalid={!!errors.driver_user_id}
                disabled={saving}
              >
                <option value="">{drivers.state === "loading" ? "Carregando…" : "Escolha"}</option>
                {(drivers.data ?? []).map((d) => (
                  <option key={d.id} value={d.id}>
                    {d.name}
                  </option>
                ))}
              </Select>
            </Field>
            <Field label="Data" htmlFor="b-date" error={errors.route_date}>
              <Input
                id="b-date"
                type="date"
                value={date}
                onChange={(e) => setDate(e.target.value)}
                invalid={!!errors.route_date}
                disabled={saving}
              />
            </Field>
            {existing && (
              <Notice tone="warn" className="sm:col-span-2">
                {existing.driver_name} já tem um roteiro em {fmtDate(existing.route_date)}.{" "}
                <Link href={`/roteiros/${existing.id}`} className="font-semibold underline underline-offset-4">
                  Abrir o roteiro existente
                </Link>
              </Notice>
            )}
            <Field label="Observação (opcional)" htmlFor="b-note" className="sm:col-span-2">
              <Input id="b-note" value={note} onChange={(e) => setNote(e.target.value)} maxLength={200} disabled={saving} />
            </Field>
          </Card>

          <Card className="p-5">
            <div className="flex items-baseline justify-between gap-3">
              <h2 className="display text-lg font-semibold">Ordem da visita</h2>
              <span className="text-sm text-ink-3 tnum">{stops.length} pontos</span>
            </div>
            {errors.location_ids && (
              <p role="alert" className="mt-2 text-sm font-medium text-danger">
                {errors.location_ids}
              </p>
            )}
            {stops.length === 0 ? (
              <p className="mt-4 rounded-lg border border-dashed border-line-strong px-4 py-8 text-center text-sm text-ink-3">
                Nenhum ponto ainda. Escolha a partida na lista de pontos cadastrados.
              </p>
            ) : (
              <ol className="mt-4 flex flex-col gap-2" aria-label="Pontos na ordem da visita">
                {stops.map((s, i) => (
                  <li
                    key={s.id}
                    className={cx(
                      "flex items-center gap-3 rounded-lg border px-3 py-2",
                      i === 0 ? "border-placa/50 bg-placa-soft" : "border-line bg-surface",
                    )}
                  >
                    <span
                      aria-hidden
                      className={cx(
                        "flex h-7 w-7 shrink-0 items-center justify-center text-xs font-bold tnum",
                        i === 0 ? "rounded-[5px] bg-placa text-on-placa" : "rounded-full border-2 border-line-strong",
                      )}
                    >
                      {i === 0 ? "P" : i}
                    </span>
                    <div className="min-w-0 flex-1">
                      <p className="truncate font-semibold">{s.label}</p>
                      <p className="truncate text-sm text-ink-2">
                        {i === 0 ? "Partida · " : ""}
                        {s.address}
                      </p>
                    </div>
                    <div className="flex shrink-0 gap-1">
                      <MiniButton label={`Subir ${s.label}`} disabled={i === 0 || saving} onClick={() => move(i, -1)}>
                        <path d="M7 11V3M3.5 6.5 7 3l3.5 3.5" />
                      </MiniButton>
                      <MiniButton
                        label={`Descer ${s.label}`}
                        disabled={i === stops.length - 1 || saving}
                        onClick={() => move(i, 1)}
                      >
                        <path d="M7 3v8M3.5 7.5 7 11l3.5-3.5" />
                      </MiniButton>
                      <MiniButton
                        label={`Remover ${s.label}`}
                        disabled={saving}
                        onClick={() => setStops((all) => all.filter((x) => x.id !== s.id))}
                      >
                        <path d="M3.5 3.5l7 7M10.5 3.5l-7 7" />
                      </MiniButton>
                    </div>
                  </li>
                ))}
              </ol>
            )}
            {errors.form && <Notice tone="error" className="mt-4">{errors.form}</Notice>}
            <Button type="submit" variant="primary" size="lg" busy={saving} className="mt-5 w-full">
              {saving ? "Criando roteiro…" : "Criar roteiro"}
            </Button>
          </Card>
        </div>

        <Card className="flex max-h-[42rem] flex-col p-5">
          <h2 className="display text-lg font-semibold">Pontos cadastrados</h2>
          <p className="mt-1 text-sm text-ink-2">
            {stops.length === 0 ? "O primeiro que você escolher vira a partida." : "Toque para incluir no fim da lista."}
          </p>
          <Field label="Buscar ponto" htmlFor="b-q" className="mt-4">
            <Input
              id="b-q"
              type="search"
              placeholder="Nome ou endereço"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
            />
          </Field>
          <ul className="mt-3 -mx-1 flex-1 overflow-y-auto px-1" aria-label="Resultados">
            {locations.state === "loading" && !locations.data && <li className="py-4 text-sm text-ink-3">Carregando…</li>}
            {locations.data && results.length === 0 && (
              <li className="py-4 text-sm text-ink-3">
                Nenhum ponto encontrado.{" "}
                <Link href="/pontos" className="font-semibold text-placa underline underline-offset-4">
                  Cadastrar ponto
                </Link>
              </li>
            )}
            {results.map((l) => (
              <li key={l.id}>
                <button
                  type="button"
                  disabled={inList.has(l.id) || saving}
                  onClick={() => setStops((s) => [...s, l])}
                  className="flex min-h-12 w-full items-center justify-between gap-3 rounded-lg px-3 py-2 text-left hover:bg-surface-2 disabled:cursor-default disabled:opacity-50"
                >
                  <span className="min-w-0">
                    <span className="block truncate font-medium">{l.label}</span>
                    <span className="block truncate text-sm text-ink-3">{l.address}</span>
                  </span>
                  <span className="shrink-0 text-sm font-semibold text-placa">{inList.has(l.id) ? "Incluído" : "Incluir"}</span>
                </button>
              </li>
            ))}
          </ul>
        </Card>
      </form>
    </>
  );
}

function MiniButton({
  label,
  disabled,
  onClick,
  children,
}: {
  label: string;
  disabled?: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      disabled={disabled}
      onClick={onClick}
      className="flex h-11 w-11 items-center justify-center rounded-lg border border-line bg-surface text-ink-2 hover:bg-surface-2 hover:text-ink disabled:opacity-35 sm:h-9 sm:w-9"
    >
      <svg aria-hidden width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round">
        {children}
      </svg>
    </button>
  );
}
