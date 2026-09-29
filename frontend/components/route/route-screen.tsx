"use client";

import Link from "next/link";
import { useState } from "react";
import { useUser } from "@/components/session-context";
import { Card, ErrorState, Forbidden, Notice, PageHeader, Skeleton, StatusBadge } from "@/components/ui";
import { api, ApiError, type RouteView } from "@/lib/api";
import { describeError } from "@/lib/errors";
import { fmtDateLong } from "@/lib/format";
import { isStaff } from "@/lib/roles";
import { useApi } from "@/lib/use-api";
import { ClosePanel, ClosedSummary, ComposePanel, ReopenPanel, TotalsStrip } from "./panels";
import { StopLine } from "./stop-line";

export type RouteMutation = (run: () => Promise<RouteView | void>, success?: string) => Promise<boolean>;

/**
 * The route screen: tracker for the driver (phone-first) and route
 * detail for managers/admins. Which controls appear follows the route
 * status and the role matrix; the backend re-checks every action.
 */
export function RouteScreen({ id, title }: { id: string; title?: string }) {
  const user = useUser();
  const staff = isStaff(user.role);
  const route = useApi(id, () => api.route(id));
  const [flash, setFlash] = useState<{ tone: "success" | "error"; text: string } | null>(null);

  const mutate: RouteMutation = async (run, success) => {
    setFlash(null);
    try {
      const next = await run();
      if (next) route.set(next);
      else route.reload();
      if (success) setFlash({ tone: "success", text: success });
      return true;
    } catch (err) {
      setFlash({ tone: "error", text: describeError(err) });
      route.reload();
      return false;
    }
  };

  if (route.state === "error" && !route.data) {
    if (route.error instanceof ApiError && route.error.status === 403) return <Forbidden />;
    if (route.error instanceof ApiError && route.error.status === 404)
      return <ErrorState message="Roteiro não encontrado. Ele pode ter sido removido." />;
    return <ErrorState message={describeError(route.error)} onRetry={route.reload} />;
  }
  if (!route.data) return <RouteSkeleton />;

  const r = route.data;
  const pendingStops = r.stops.filter((s) => !s.departure_at).length;
  const completed = r.status === "active" && pendingStops === 0;
  const canCompose = staff && (r.status === "draft" || r.status === "active");

  return (
    <>
      <PageHeader
        title={title ?? `Roteiro de ${r.driver_name}`}
        eyebrow={
          staff ? (
            <Link href="/historico" className="underline-offset-4 hover:text-ink hover:underline">
              ← Histórico
            </Link>
          ) : undefined
        }
        actions={<StatusBadge status={r.status} />}
      >
        <span>{fmtDateLong(r.route_date)}</span>
        {r.note && <span className="mt-1 block text-sm text-ink-3">Observação: {r.note}</span>}
      </PageHeader>

      <div aria-live="polite" className="mb-4 empty:hidden">
        {flash && <Notice tone={flash.tone}>{flash.text}</Notice>}
      </div>

      {r.status === "closed" ? <ClosedSummary route={r} /> : staff && <TotalsStrip route={r} />}

      <div className={staff || r.status === "closed" ? "mt-6 grid grid-cols-1 gap-6 lg:grid-cols-[minmax(0,1fr)_320px]" : "grid grid-cols-1 gap-6 lg:grid-cols-[minmax(0,1fr)_320px]"}>
        <Card className="p-3 sm:p-5">
          <StopLine
            route={r}
            mutate={mutate}
            canCompose={staff && r.status === "draft"}
            canCorrect={staff}
            collapseDone={!staff}
          />
        </Card>
        <div className="flex flex-col gap-4">
          {r.status === "draft" && (
            <DraftStart route={r} mutate={mutate} canStart={staff || r.driver_user_id === user.id} />
          )}
          {canCompose && <ComposePanel route={r} mutate={mutate} />}
          {r.status === "active" && (completed || staff) && (
            <ClosePanel route={r} mutate={mutate} pendingStops={pendingStops} />
          )}
          {!staff && r.status !== "closed" && <TotalsStrip route={r} narrow />}
          {r.status === "closed" && user.role === "admin" && <ReopenPanel route={r} mutate={mutate} />}
          {r.status === "closed" && user.role === "manager" && (
            <p className="text-sm text-ink-3">
              Roteiro encerrado: horários e pontos estão congelados. Só o administrador pode reabrir para correção.
            </p>
          )}
        </div>
      </div>
    </>
  );
}

function DraftStart({ route, mutate, canStart }: { route: RouteView; mutate: RouteMutation; canStart: boolean }) {
  const [busy, setBusy] = useState(false);
  if (!canStart) return null;
  return (
    <Card className="p-5">
      <h2 className="display text-lg font-semibold">Pronto para sair?</h2>
      <p className="mt-1 text-sm text-ink-2">
        Ao iniciar, o roteiro passa a aceitar chegadas e saídas. O primeiro ponto é a partida e não conta tempo parado.
      </p>
      <button
        type="button"
        disabled={busy || route.stops.length < 2}
        onClick={async () => {
          setBusy(true);
          await mutate(() => api.startRoute(route.id), "Roteiro iniciado. Registre a saída da base.");
          setBusy(false);
        }}
        className="mt-4 flex min-h-14 w-full items-center justify-center rounded-xl bg-placa px-5 text-lg font-bold text-on-placa hover:bg-placa-strong disabled:opacity-55"
      >
        {busy ? "Iniciando…" : "Iniciar roteiro"}
      </button>
    </Card>
  );
}

function RouteSkeleton() {
  return (
    <div aria-busy="true" aria-label="Carregando roteiro" className="flex flex-col gap-4">
      <Skeleton className="h-9 w-72" />
      <Skeleton className="h-5 w-48" />
      <div className="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-4">
        {[0, 1, 2, 3].map((i) => (
          <Skeleton key={i} className="h-20" />
        ))}
      </div>
      {[0, 1, 2, 3].map((i) => (
        <Skeleton key={i} className="h-16 w-full" />
      ))}
    </div>
  );
}
