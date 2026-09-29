"use client";

import { useMemo, useState, type FormEvent } from "react";
import { useUser } from "@/components/session-context";
import {
  Button,
  Card,
  EmptyState,
  ErrorState,
  Field,
  Forbidden,
  Input,
  Notice,
  PageHeader,
  Skeleton,
} from "@/components/ui";
import { api, type Location } from "@/lib/api";
import { fieldErrors } from "@/lib/errors";
import { isStaff } from "@/lib/roles";
import { useApi } from "@/lib/use-api";

type Mode = { kind: "list" } | { kind: "new" } | { kind: "edit"; location: Location };

export function Locations() {
  const user = useUser();
  const locations = useApi(isStaff(user.role) ? "locations" : null, () => api.locations());
  const [mode, setMode] = useState<Mode>({ kind: "list" });
  const [flash, setFlash] = useState<string | null>(null);
  const [query, setQuery] = useState("");

  const rows = useMemo(() => {
    const q = query.trim().toLowerCase();
    const all = locations.data ?? [];
    return q ? all.filter((l) => `${l.label} ${l.address}`.toLowerCase().includes(q)) : all;
  }, [locations.data, query]);

  if (!isStaff(user.role)) return <Forbidden />;

  return (
    <>
      <PageHeader
        title="Pontos"
        actions={
          mode.kind === "list" && (
            <Button variant="primary" onClick={() => { setFlash(null); setMode({ kind: "new" }); }}>
              Novo ponto
            </Button>
          )
        }
      >
        Endereços que entram nos roteiros: a base de partida, clientes e centros de distribuição.
      </PageHeader>
      <div aria-live="polite" className="mb-4 empty:hidden">
        {flash && <Notice tone="success">{flash}</Notice>}
      </div>
      {mode.kind !== "list" && (
        <LocationForm
          key={mode.kind === "edit" ? mode.location.id : "new"}
          location={mode.kind === "edit" ? mode.location : null}
          onCancel={() => setMode({ kind: "list" })}
          onSaved={(msg) => {
            setMode({ kind: "list" });
            setFlash(msg);
            locations.reload();
          }}
        />
      )}
      <div className="mb-4 max-w-sm">
        <Field label="Buscar" htmlFor="p-q">
          <Input id="p-q" type="search" placeholder="Nome ou endereço" value={query} onChange={(e) => setQuery(e.target.value)} />
        </Field>
      </div>
      {locations.state === "error" ? (
        <ErrorState message="Não foi possível carregar os pontos." onRetry={locations.reload} />
      ) : !locations.data ? (
        <div className="flex flex-col gap-2">
          {[0, 1, 2].map((i) => (
            <Skeleton key={i} className="h-14 w-full" />
          ))}
        </div>
      ) : rows.length === 0 ? (
        <EmptyState title={query ? "Nenhum ponto com esse nome ou endereço" : "Nenhum ponto cadastrado"}>
          {query ? "Tente outra busca." : "Cadastre a base de partida e os endereços de entrega."}
        </EmptyState>
      ) : (
        <Card className="overflow-x-auto">
          <table className="w-full min-w-[560px] text-sm">
            <thead className="bg-surface-2 text-left text-ink-2">
              <tr>
                <th scope="col" className="px-4 py-3 font-semibold">Nome</th>
                <th scope="col" className="px-4 py-3 font-semibold">Endereço</th>
                <th scope="col" className="px-4 py-3 font-semibold">Coordenadas</th>
                <th scope="col" className="px-4 py-3"><span className="sr-only">Ações</span></th>
              </tr>
            </thead>
            <tbody>
              {rows.map((l) => (
                <tr key={l.id} className="border-t border-line">
                  <td className="px-4 py-3 font-semibold">{l.label}</td>
                  <td className="px-4 py-3">{l.address}</td>
                  <td className="px-4 py-3 font-mono text-xs text-ink-2 tnum">
                    {l.latitude && l.longitude ? `${l.latitude}, ${l.longitude}` : "—"}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <Button size="sm" variant="ghost" onClick={() => { setFlash(null); setMode({ kind: "edit", location: l }); }}>
                      Editar
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </Card>
      )}
    </>
  );
}

function coordinate(value: string, min: number, max: number): string | null | "invalid" {
  const v = value.trim().replace(",", ".");
  if (!v) return null;
  const n = Number(v);
  return Number.isFinite(n) && n >= min && n <= max ? v : "invalid";
}

function LocationForm({
  location,
  onCancel,
  onSaved,
}: {
  location: Location | null;
  onCancel: () => void;
  onSaved: (message: string) => void;
}) {
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const f = new FormData(e.currentTarget);
    const s = (k: string) => String(f.get(k) ?? "").trim();
    const errs: Record<string, string> = {};
    if (!s("label")) errs.label = "Dê um nome ao ponto.";
    if (!s("address")) errs.address = "Informe o endereço.";
    const lat = coordinate(s("latitude"), -90, 90);
    const lng = coordinate(s("longitude"), -180, 180);
    if (lat === "invalid") errs.latitude = "Latitude entre −90 e 90. Ex.: −19,9191";
    if (lng === "invalid") errs.longitude = "Longitude entre −180 e 180. Ex.: −43,9386";
    setErrors(errs);
    if (Object.keys(errs).length) return;
    const body = {
      label: s("label"),
      address: s("address"),
      latitude: lat === "invalid" ? null : lat,
      longitude: lng === "invalid" ? null : lng,
    };
    setBusy(true);
    try {
      if (location) {
        await api.updateLocation(location.id, body);
        onSaved(`Ponto “${body.label}” atualizado.`);
      } else {
        await api.createLocation(body);
        onSaved(`Ponto “${body.label}” cadastrado.`);
      }
    } catch (err) {
      setErrors(fieldErrors(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Card className="rise mb-6 p-5 sm:p-6">
      <h2 className="display text-lg font-semibold">{location ? `Editar ${location.label}` : "Novo ponto"}</h2>
      <form method="post" action="/sem-js" onSubmit={submit} noValidate className="mt-4 grid gap-4 sm:grid-cols-2">
        <Field label="Nome" htmlFor="l-label" error={errors.label}>
          <Input id="l-label" name="label" defaultValue={location?.label} placeholder="Ex.: Mercado Central" invalid={!!errors.label} />
        </Field>
        <Field label="Endereço" htmlFor="l-address" error={errors.address}>
          <Input id="l-address" name="address" defaultValue={location?.address} placeholder="Rua, número" invalid={!!errors.address} />
        </Field>
        <Field label="Latitude (opcional)" htmlFor="l-lat" error={errors.latitude}>
          <Input id="l-lat" name="latitude" inputMode="decimal" defaultValue={location?.latitude ?? ""} invalid={!!errors.latitude} />
        </Field>
        <Field label="Longitude (opcional)" htmlFor="l-lng" error={errors.longitude}>
          <Input id="l-lng" name="longitude" inputMode="decimal" defaultValue={location?.longitude ?? ""} invalid={!!errors.longitude} />
        </Field>
        {errors.form && <Notice tone="error" className="sm:col-span-2">{errors.form}</Notice>}
        <div className="flex gap-2 sm:col-span-2">
          <Button type="submit" variant="primary" busy={busy}>
            {location ? "Salvar alterações" : "Cadastrar ponto"}
          </Button>
          <Button type="button" variant="ghost" onClick={onCancel}>
            Cancelar
          </Button>
        </div>
      </form>
    </Card>
  );
}
