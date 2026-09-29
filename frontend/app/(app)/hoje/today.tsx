"use client";

import { RouteScreen } from "@/components/route/route-screen";
import { useUser } from "@/components/session-context";
import { ButtonLink, EmptyState, ErrorState, PageHeader, Skeleton } from "@/components/ui";
import { api } from "@/lib/api";
import { describeError } from "@/lib/errors";
import { fmtDateLong, todayISO } from "@/lib/format";
import { useApi } from "@/lib/use-api";

export function Today() {
  const user = useUser();
  const today = todayISO();
  const routes = useApi(user.role === "driver" ? today : null, () => api.routes({ from: today, to: today }));

  if (user.role !== "driver") {
    return (
      <EmptyState title="Esta tela é do motorista" action={<ButtonLink href="/historico">Ver roteiros no histórico</ButtonLink>}>
        Gestores acompanham os roteiros pelo histórico e pelo painel.
      </EmptyState>
    );
  }
  if (routes.state === "error") return <ErrorState message={describeError(routes.error)} onRetry={routes.reload} />;
  if (!routes.data)
    return (
      <div aria-busy="true" aria-label="Carregando roteiro" className="flex flex-col gap-3">
        <Skeleton className="h-9 w-64" />
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-24 w-full" />
      </div>
    );

  const route = routes.data[0];
  if (!route)
    return (
      <>
        <PageHeader title="Meu roteiro de hoje" eyebrow={fmtDateLong(today)} />
        <EmptyState
          title="Nenhum roteiro para hoje"
          action={<ButtonLink href="/historico">Ver meus roteiros anteriores</ButtonLink>}
        >
          O gestor ainda não montou o seu roteiro de hoje. Quando ele montar, os pontos aparecem aqui.
        </EmptyState>
      </>
    );
  return <RouteScreen id={route.id} title="Meu roteiro de hoje" />;
}
