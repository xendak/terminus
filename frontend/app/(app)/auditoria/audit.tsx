"use client";

import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useState, type FormEvent } from "react";
import { useUser } from "@/components/session-context";
import {
  Button,
  Card,
  EmptyState,
  ErrorState,
  Field,
  Forbidden,
  Input,
  PageHeader,
  Select,
  Skeleton,
} from "@/components/ui";
import { api, type AuditEntry, type AuditValues } from "@/lib/api";
import { describeError } from "@/lib/errors";
import { fmtDateTime } from "@/lib/format";
import { useApi } from "@/lib/use-api";

const entities = [
  { value: "", label: "Tudo" },
  { value: "route_stop", label: "Horários e pontos" },
  { value: "route", label: "Roteiros" },
  { value: "parameter", label: "Parâmetros" },
  { value: "location", label: "Pontos" },
  { value: "app_user", label: "Motoristas (LGPD)" },
];

const entityLabel: Record<string, string> = {
  app_user: "Motorista",
  location: "Ponto",
  route_stop: "Ponto do roteiro",
  route: "Roteiro",
  parameter: "Parâmetro",
};

const actionLabel: Record<string, string> = {
  update_times: "Correção de horários",
  add_stop: "Ponto incluído",
  remove_stop: "Ponto removido",
  reorder: "Ordem alterada",
  close_route: "Roteiro encerrado",
  reopen_route: "Roteiro reaberto",
  update_param: "Parâmetro alterado",
  anonymize: "Anonimização (LGPD)",
  update_location: "Ponto alterado",
};

const fieldLabel: Record<string, string> = {
  name: "Nome",
  label: "Nome do ponto",
  address: "Endereço",
  latitude: "Latitude",
  longitude: "Longitude",
  email: "E-mail",
  phone: "Telefone",
  password_hash: "Senha",
  document: "CPF",
  vehicle_name: "Veículo",
  vehicle_plate: "Placa",
  active: "Ativo",
  arrival_at: "Chegada",
  departure_at: "Saída",
  stop_order: "Ordem",
  location_id: "Ponto",
  status: "Situação",
  value: "Valor",
};

const statusLabel: Record<string, string> = { draft: "rascunho", active: "em andamento", closed: "encerrado" };

function show(v: AuditValues[string] | undefined): string {
  if (v === null || v === undefined || v === "") return "—";
  if (Array.isArray(v)) return v.map((k) => fieldLabel[k] ?? k).join(", ");
  if (typeof v === "boolean") return v ? "sim" : "não";
  if (typeof v === "string") {
    if (/^\d{4}-\d{2}-\d{2}T/.test(v)) return fmtDateTime(v);
    if (statusLabel[v]) return statusLabel[v];
    if (/^[0-9a-f-]{36}$/.test(v)) return `${v.slice(0, 8)}…`;
    return v;
  }
  return String(v);
}

function Anonymization({ entry }: { entry: AuditEntry }) {
  const cleared = entry.old_values?.cleared;
  const fields = Array.isArray(cleared) ? cleared : [];
  return (
    <div className="flex flex-col gap-1 text-xs">
      <p>
        <span className="font-medium text-ink-2">Dados apagados: </span>
        {fields.length ? fields.map((k) => fieldLabel[k] ?? k).join(", ") : "—"}
      </p>
      <p>
        <span className="font-medium text-ink-2">Cadastro: </span>
        {entry.old_values?.active === false ? "já estava inativo" : "ativo → inativo"}; acesso bloqueado
      </p>
    </div>
  );
}

function Diff({ entry }: { entry: AuditEntry }) {
  if (entry.action === "anonymize") return <Anonymization entry={entry} />;
  const oldV = entry.old_values ?? {};
  const newV = entry.new_values ?? {};
  // Show what changed; a correction that kept the arrival lists only the departure.
  const all = Array.from(new Set([...Object.keys(oldV), ...Object.keys(newV)]));
  const changed = all.filter((k) => show(oldV[k]) !== show(newV[k]));
  const keys = changed.length ? changed : all;
  if (keys.length === 0) return <span className="text-ink-3">—</span>;
  return (
    <ul className="flex flex-col gap-1">
      {keys.map((k) => {
        const label = fieldLabel[k] ?? (/^[0-9a-f-]{36}$/.test(k) ? `Parada ${k.slice(0, 8)}…` : k);
        return (
          <li key={k} className="font-mono text-xs tnum">
            <span className="font-sans font-medium text-ink-2">{label}: </span>
            <span className="text-ink-3 line-through decoration-ink-3/50">{show(oldV[k])}</span>
            <span aria-label="para" className="px-1.5 text-ink-3">→</span>
            <span className="font-semibold text-ink">{show(newV[k])}</span>
          </li>
        );
      })}
    </ul>
  );
}

export function Audit() {
  const user = useUser();
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const entity = params.get("entidade") ?? "";
  const from = params.get("from") ?? "";
  const to = params.get("to") ?? "";
  const [formError, setFormError] = useState<string | null>(null);
  const entries = useApi(user.role === "admin" ? `${entity}|${from}|${to}` : null, () =>
    api.audit({ entity: entity || undefined, from: from || undefined, to: to || undefined }),
  );

  if (user.role !== "admin") return <Forbidden />;

  function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const f = new FormData(e.currentTarget);
    const nf = String(f.get("from") ?? "");
    const nt = String(f.get("to") ?? "");
    if (nf && nt && nf > nt) return setFormError("A data inicial precisa ser anterior à final.");
    setFormError(null);
    const q = new URLSearchParams();
    const en = String(f.get("entidade") ?? "");
    if (en) q.set("entidade", en);
    if (nf) q.set("from", nf);
    if (nt) q.set("to", nt);
    router.replace(`${pathname}${q.size ? `?${q.toString()}` : ""}`, { scroll: false });
  }

  return (
    <>
      <PageHeader title="Auditoria">
        Toda correção de horário, mudança na composição de roteiros e alteração de parâmetro, com o valor anterior e o
        novo.
      </PageHeader>
      <Card className="mb-6 p-4">
        <form key={`${entity}|${from}|${to}`} onSubmit={submit} noValidate className="grid grid-cols-2 items-end gap-3 md:flex md:flex-wrap">
          <Field label="Tipo de registro" htmlFor="a-entity" className="col-span-2 md:min-w-52">
            <Select id="a-entity" name="entidade" defaultValue={entity}>
              {entities.map((e) => (
                <option key={e.value} value={e.value}>
                  {e.label}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="De" htmlFor="a-from">
            <Input id="a-from" name="from" type="date" defaultValue={from} invalid={!!formError} />
          </Field>
          <Field label="Até" htmlFor="a-to">
            <Input id="a-to" name="to" type="date" defaultValue={to} />
          </Field>
          <Button type="submit" variant="primary" className="col-span-2 md:col-span-1">
            Filtrar
          </Button>
          {formError && (
            <p role="alert" className="col-span-2 basis-full text-sm font-medium text-danger">
              {formError}
            </p>
          )}
        </form>
      </Card>
      {entries.state === "error" ? (
        <ErrorState message={describeError(entries.error)} onRetry={entries.reload} />
      ) : !entries.data ? (
        <div className="flex flex-col gap-2">
          {[0, 1, 2].map((i) => (
            <Skeleton key={i} className="h-16 w-full" />
          ))}
        </div>
      ) : entries.data.length === 0 ? (
        <EmptyState title="Nenhuma alteração registrada">Mude o tipo de registro ou o intervalo de datas.</EmptyState>
      ) : (
        <Card className="overflow-x-auto">
          <table className="w-full min-w-[720px] text-sm">
            <thead className="bg-surface-2 text-left text-ink-2">
              <tr>
                <th scope="col" className="px-4 py-3 font-semibold">Quando</th>
                <th scope="col" className="px-4 py-3 font-semibold">Quem</th>
                <th scope="col" className="px-4 py-3 font-semibold">O quê</th>
                <th scope="col" className="px-4 py-3 font-semibold">Antes → depois</th>
              </tr>
            </thead>
            <tbody>
              {entries.data.map((e, i) => (
                <tr key={`${e.at}-${e.entity_id}-${i}`} className="border-t border-line align-top">
                  <td className="whitespace-nowrap px-4 py-3 tnum">{fmtDateTime(e.at)}</td>
                  <td className="px-4 py-3">{e.actor}</td>
                  <td className="px-4 py-3">
                    <span className="block font-semibold">{actionLabel[e.action] ?? e.action}</span>
                    <span className="block text-xs text-ink-3">
                      {entityLabel[e.entity] ?? e.entity} · <span className="font-mono">{e.entity_id.length > 12 ? `${e.entity_id.slice(0, 8)}…` : e.entity_id}</span>
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <Diff entry={e} />
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
