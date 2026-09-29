"use client";

import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useState, type FormEvent } from "react";
import { useUser } from "@/components/session-context";
import {
  Button,
  Card,
  cx,
  EmptyState,
  ErrorState,
  Field,
  Input,
  PageHeader,
  Select,
  Skeleton,
  StatusBadge,
} from "@/components/ui";
import { api, type RouteListRow, type RouteStatus } from "@/lib/api";
import { describeError } from "@/lib/errors";
import { fmtBRL, fmtDate, fmtMinutes, fmtPercent, monthStartISO, todayISO } from "@/lib/format";
import { isStaff } from "@/lib/roles";
import { useApi } from "@/lib/use-api";

const statuses: { value: RouteStatus | ""; label: string }[] = [
  { value: "", label: "Todos" },
  { value: "draft", label: "Rascunho" },
  { value: "active", label: "Em andamento" },
  { value: "closed", label: "Encerrado" },
];

export function History() {
  const user = useUser();
  const staff = isStaff(user.role);
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const today = todayISO();
  const from = params.get("from") ?? monthStartISO(today);
  const to = params.get("to") ?? today;
  const driver = params.get("motorista") ?? "";
  const statusParam = params.get("status") ?? "";
  const status = statuses.some((s) => s.value === statusParam) ? (statusParam as RouteStatus | "") : "";
  const [formError, setFormError] = useState<string | null>(null);

  const drivers = useApi(staff ? "drivers" : null, () => api.drivers());
  const routes = useApi(`${from}|${to}|${driver}|${status}`, () =>
    api.routes({ from, to, driver_user_id: driver || undefined, status }),
  );

  function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const f = new FormData(e.currentTarget);
    const nf = String(f.get("from") ?? "");
    const nt = String(f.get("to") ?? "");
    if (!nf || !nt) return setFormError("Informe as duas datas.");
    if (nf > nt) return setFormError("A data inicial precisa ser anterior à final.");
    setFormError(null);
    const q = new URLSearchParams({ from: nf, to: nt });
    const d = String(f.get("motorista") ?? "");
    const s = String(f.get("status") ?? "");
    if (d) q.set("motorista", d);
    if (s) q.set("status", s);
    router.replace(`${pathname}?${q.toString()}`, { scroll: false });
  }

  const exportHref = api.exportUrl({ from, to, driver_user_id: driver || undefined });

  return (
    <>
      <PageHeader
        title="Histórico"
        eyebrow={`${fmtDate(from)} a ${fmtDate(to)}`}
        actions={
          <div className="flex flex-col items-start gap-1 sm:items-end">
          <a
            href={exportHref}
            download
            className="inline-flex h-10 items-center gap-2 rounded-lg border border-line-strong bg-surface px-4 text-sm font-semibold hover:bg-surface-2"
          >
            <svg aria-hidden width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.7">
              <path d="M8 2v8M4.5 6.5 8 10l3.5-3.5M2.5 13.5h11" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
            Exportar CSV
          </a>
          <span className="text-xs text-ink-3">
            Exporta o período{staff && driver ? " do motorista escolhido" : ""}, em todas as situações.
          </span>
          </div>
        }
      >
        {staff
          ? "Roteiros do período com tempo parado, parte da jornada e custo estimado."
          : "Seus roteiros no período, com o tempo parado de cada dia."}
      </PageHeader>

      <Card className="mb-6 p-4">
        <form
          key={`${from}|${to}|${driver}|${status}`}
          onSubmit={onSubmit}
          className="grid grid-cols-2 items-end gap-3 md:flex md:flex-wrap"
          noValidate
        >
          <Field label="De" htmlFor="h-from">
            <Input id="h-from" name="from" type="date" defaultValue={from} invalid={!!formError} />
          </Field>
          <Field label="Até" htmlFor="h-to">
            <Input id="h-to" name="to" type="date" defaultValue={to} />
          </Field>
          {staff && (
            <Field label="Motorista" htmlFor="h-driver" className="col-span-2 md:min-w-52">
              <Select id="h-driver" name="motorista" defaultValue={driver}>
                <option value="">Todos os motoristas</option>
                {(drivers.data ?? []).map((d) => (
                  <option key={d.id} value={d.id}>
                    {d.name}
                    {d.active ? "" : " (inativo)"}
                  </option>
                ))}
              </Select>
            </Field>
          )}
          <Field label="Situação" htmlFor="h-status" className="col-span-2 md:col-span-1 md:min-w-44">
            <Select id="h-status" name="status" defaultValue={status}>
              {statuses.map((s) => (
                <option key={s.value} value={s.value}>
                  {s.label}
                </option>
              ))}
            </Select>
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

      {routes.state === "error" ? (
        <ErrorState message={describeError(routes.error)} onRetry={routes.reload} />
      ) : !routes.data ? (
        <div className="flex flex-col gap-2" aria-busy="true" aria-label="Carregando roteiros">
          {[0, 1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-14 w-full" />
          ))}
        </div>
      ) : routes.data.length === 0 ? (
        <EmptyState title="Nenhum roteiro neste intervalo">
          Mude as datas ou tire os filtros de motorista e situação.
        </EmptyState>
      ) : (
        <div className={cx("transition-opacity", routes.state === "loading" && "opacity-60")}>
          <RoutesTable rows={routes.data} showDriver={staff} />
        </div>
      )}
    </>
  );
}

function RoutesTable({ rows, showDriver }: { rows: RouteListRow[]; showDriver: boolean }) {
  return (
    <>
      {/* Desktop table */}
      <Card className="hidden overflow-hidden md:block">
        <table className="w-full text-sm">
          <thead className="bg-surface-2 text-left text-ink-2">
            <tr>
              <th scope="col" className="px-4 py-3 font-semibold">Data</th>
              {showDriver && <th scope="col" className="px-4 py-3 font-semibold">Motorista</th>}
              <th scope="col" className="px-4 py-3 text-right font-semibold">Pontos</th>
              <th scope="col" className="px-4 py-3 text-right font-semibold">Parado</th>
              <th scope="col" className="px-4 py-3 text-right font-semibold">Jornada</th>
              <th scope="col" className="px-4 py-3 text-right font-semibold">Custo estimado</th>
              <th scope="col" className="px-4 py-3 font-semibold">Situação</th>
              <th scope="col" className="px-4 py-3"><span className="sr-only">Abrir</span></th>
            </tr>
          </thead>
          <tbody className="tnum">
            {rows.map((r) => (
              <tr key={r.id} className="border-t border-line hover:bg-surface-2/60">
                <td className="px-4 py-3 font-medium">{fmtDate(r.route_date)}</td>
                {showDriver && <td className="px-4 py-3">{r.driver_name}</td>}
                <td className="px-4 py-3 text-right">{r.stop_count}</td>
                <td className="px-4 py-3 text-right font-semibold">{fmtMinutes(r.total_stopped_minutes)}</td>
                <td className="px-4 py-3 text-right">{fmtPercent(r.journey_percent)}</td>
                <td className="px-4 py-3 text-right">
                  {r.estimated_cost_brl ? fmtBRL(r.estimated_cost_brl) : <span className="text-ink-3">sem distância</span>}
                </td>
                <td className="px-4 py-3">
                  <StatusBadge status={r.status} />
                </td>
                <td className="px-4 py-3 text-right">
                  <Link
                    href={`/roteiros/${r.id}`}
                    className="font-semibold text-placa underline-offset-4 hover:underline"
                    aria-label={`Abrir roteiro de ${r.driver_name} em ${fmtDate(r.route_date)}`}
                  >
                    Abrir
                  </Link>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </Card>

      {/* Phone cards */}
      <ul className="flex flex-col gap-3 md:hidden">
        {rows.map((r) => (
          <li key={r.id}>
            <Link
              href={`/roteiros/${r.id}`}
              className="block rounded-xl border border-line bg-surface p-4 shadow-card active:bg-surface-2"
            >
              <div className="flex items-center justify-between gap-3">
                <span className="font-semibold tnum">{fmtDate(r.route_date)}</span>
                <StatusBadge status={r.status} />
              </div>
              {showDriver && <p className="mt-1 text-ink-2">{r.driver_name}</p>}
              <dl className="mt-3 grid grid-cols-3 gap-2 text-sm tnum">
                <div>
                  <dt className="text-ink-3">Parado</dt>
                  <dd className="font-semibold">{fmtMinutes(r.total_stopped_minutes)}</dd>
                </div>
                <div>
                  <dt className="text-ink-3">Jornada</dt>
                  <dd>{fmtPercent(r.journey_percent)}</dd>
                </div>
                <div>
                  <dt className="text-ink-3">Custo</dt>
                  <dd>{r.estimated_cost_brl ? fmtBRL(r.estimated_cost_brl) : "—"}</dd>
                </div>
              </dl>
            </Link>
          </li>
        ))}
      </ul>
    </>
  );
}
