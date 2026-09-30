"use client";

import { useState, type FormEvent } from "react";
import { useUser } from "@/components/session-context";
import { Button, Card, ErrorState, Forbidden, Input, Notice, PageHeader, Skeleton } from "@/components/ui";
import { api, type Param, type ParamKey } from "@/lib/api";
import { fieldErrors } from "@/lib/errors";
import { decimalForInput, fmtDateTime, parseDecimalInput } from "@/lib/format";
import { isStaff } from "@/lib/roles";
import { useApi } from "@/lib/use-api";

const meta: Record<ParamKey, { label: string; unit: string; help: string; integer?: boolean; positive?: boolean }> = {
  fuel_price_brl: {
    label: "Preço do combustível",
    unit: "R$ por litro",
    help: "Entra no custo estimado: litros gastos × preço.",
  },
  cost_per_km_brl: {
    label: "Custo por km",
    unit: "R$ por km",
    help: "Desgaste, manutenção e outros custos por km rodado, somados ao combustível.",
  },
  default_km_per_l: {
    label: "Consumo padrão",
    unit: "km por litro",
    help: "Usado quando o veículo do motorista não tem consumo próprio.",
    positive: true,
  },
  standard_journey_hours: {
    label: "Jornada padrão",
    unit: "horas por dia",
    help: "Base do percentual de jornada: essa quantidade de horas vale 100%.",
    positive: true,
  },
  stop_warn_minutes: {
    label: "Parada longa a partir de (min)",
    unit: "minutos",
    help: "No painel, paradas a partir deste tempo aparecem como “Atenção”. Não muda nenhum total.",
    integer: true,
    positive: true,
  },
  stop_alert_minutes: {
    label: "Parada crítica a partir de (min)",
    unit: "minutos",
    help: "No painel, paradas a partir deste tempo aparecem em vermelho, como “Acima do limite”.",
    integer: true,
    positive: true,
  },
  min_stop_minutes: {
    label: "Parada mínima",
    unit: "minutos",
    help: "Paradas mais curtas que isso ficam registradas, mas não entram nos totais. Zero conta todas.",
    integer: true,
  },
};

const order: ParamKey[] = [
  "fuel_price_brl",
  "cost_per_km_brl",
  "default_km_per_l",
  "standard_journey_hours",
  "min_stop_minutes",
  "stop_warn_minutes",
  "stop_alert_minutes",
];

export function Params() {
  const user = useUser();
  const params = useApi(isStaff(user.role) ? "params" : null, () => api.params());
  if (!isStaff(user.role)) return <Forbidden />;

  return (
    <>
      <PageHeader title="Parâmetros">
        Valores usados nos cálculos de custo e de jornada. Uma mudança vale para todos os relatórios a partir da próxima
        consulta e fica registrada na auditoria.
      </PageHeader>
      {params.state === "error" ? (
        <ErrorState message="Não foi possível carregar os parâmetros." onRetry={params.reload} />
      ) : !params.data ? (
        <div className="flex flex-col gap-3">
          {order.map((k) => (
            <Skeleton key={k} className="h-24 w-full" />
          ))}
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          {order
            .map((k) => params.data!.find((p) => p.key === k))
            .filter((p): p is Param => !!p)
            .map((p) => (
              <ParamRow key={p.key} param={p} userId={user.id} />
            ))}
        </div>
      )}
    </>
  );
}

function ParamRow({ param: initial, userId }: { param: Param; userId: string }) {
  const [param, setParam] = useState(initial);
  const [value, setValue] = useState(decimalForInput(initial.value));
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);
  const [busy, setBusy] = useState(false);
  const m = meta[param.key];
  const id = `param-${param.key}`;
  const dirty = value.trim() !== decimalForInput(param.value);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setSaved(false);
    const parsed = parseDecimalInput(value);
    if (parsed === null) return setError("Informe um número. Ex.: 6,09");
    if (Number(parsed) < 0) return setError("Valores negativos não são aceitos.");
    if (m.positive && Number(parsed) === 0) return setError("Informe um valor maior que zero.");
    if (m.integer && !Number.isInteger(Number(parsed))) return setError("Use um número inteiro de minutos.");
    setError(null);
    setBusy(true);
    try {
      const updated = await api.updateParam(param.key, parsed);
      setParam(updated);
      setValue(decimalForInput(updated.value));
      setSaved(true);
    } catch (err) {
      setError(Object.values(fieldErrors(err))[0]);
    } finally {
      setBusy(false);
    }
  }

  return (
    <Card className="p-4 sm:p-5">
      <form method="post" action="/sem-js" onSubmit={submit} noValidate className="grid grid-cols-1 gap-3 md:grid-cols-[minmax(0,1fr)_auto] md:items-center md:gap-6">
        <div className="min-w-0">
          <label htmlFor={id} className="display text-base font-semibold">
            {m.label}
          </label>
          <p className="mt-0.5 text-sm text-ink-2">{m.help}</p>
          <p className="mt-1 text-xs text-ink-3 tnum">
            Atualizado em {fmtDateTime(param.updated_at)}
            {param.updated_by === userId
              ? " por você"
              : param.updated_by_name
                ? ` por ${param.updated_by_name}`
                : ""}
          </p>
        </div>
        <div className="flex flex-col gap-1.5">
          <div className="flex items-center gap-2">
            <div className="relative">
              <Input
                id={id}
                inputMode="decimal"
                value={value}
                onChange={(e) => {
                  setValue(e.target.value);
                  setSaved(false);
                }}
                invalid={!!error}
                aria-describedby={`${id}-unit`}
                className="w-32 text-right text-lg font-semibold tnum"
              />
            </div>
            <span id={`${id}-unit`} className="w-28 text-sm text-ink-2">
              {m.unit}
            </span>
            <Button type="submit" variant={dirty ? "primary" : "secondary"} disabled={!dirty} busy={busy}>
              Salvar
            </Button>
          </div>
          {error && (
            <p id={`${id}-error`} role="alert" className="text-sm font-medium text-danger">
              {error}
            </p>
          )}
          {saved && !error && (
            <Notice tone="success" className="px-3 py-1.5">
              Salvo. Os relatórios já usam o novo valor.
            </Notice>
          )}
        </div>
      </form>
    </Card>
  );
}
