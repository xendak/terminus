"use client";

import { useEffect, useRef, useState, type FormEvent } from "react";
import { useUser } from "@/components/session-context";
import {
  ActiveBadge,
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
import { api, type Driver } from "@/lib/api";
import { describeError, fieldErrors } from "@/lib/errors";
import { decimalForInput, fmtNumber, parseDecimalInput } from "@/lib/format";
import { isStaff } from "@/lib/roles";
import { useApi } from "@/lib/use-api";

type Mode = { kind: "list" } | { kind: "new" } | { kind: "edit"; driver: Driver };

export function Drivers() {
  const user = useUser();
  const drivers = useApi(isStaff(user.role) ? "drivers" : null, () => api.drivers());
  const [mode, setMode] = useState<Mode>({ kind: "list" });
  const [flash, setFlash] = useState<string | null>(null);

  if (!isStaff(user.role)) return <Forbidden />;

  function done(message: string) {
    setMode({ kind: "list" });
    setFlash(message);
    drivers.reload();
  }

  return (
    <>
      <PageHeader
        title="Motoristas"
        actions={
          mode.kind === "list" && (
            <Button variant="primary" onClick={() => { setFlash(null); setMode({ kind: "new" }); }}>
              Novo motorista
            </Button>
          )
        }
      >
        Quem roda os roteiros. O consumo do veículo (km/l) entra no custo; sem ele, vale o padrão dos parâmetros.
      </PageHeader>

      <div aria-live="polite" className="mb-4 empty:hidden">
        {flash && <Notice tone="success">{flash}</Notice>}
      </div>

      {mode.kind !== "list" && (
        <DriverForm
          key={mode.kind === "edit" ? mode.driver.id : "new"}
          driver={mode.kind === "edit" ? mode.driver : null}
          canAnonymize={user.role === "admin"}
          onCancel={() => setMode({ kind: "list" })}
          onSaved={done}
        />
      )}

      {drivers.state === "error" ? (
        <ErrorState message="Não foi possível carregar os motoristas." onRetry={drivers.reload} />
      ) : !drivers.data ? (
        <div className="flex flex-col gap-2">
          {[0, 1, 2].map((i) => (
            <Skeleton key={i} className="h-16 w-full" />
          ))}
        </div>
      ) : drivers.data.length === 0 ? (
        <EmptyState title="Nenhum motorista cadastrado">Cadastre o primeiro para montar roteiros.</EmptyState>
      ) : (
        <Card className="overflow-x-auto">
          <table className="w-full min-w-[640px] text-sm">
            <thead className="bg-surface-2 text-left text-ink-2">
              <tr>
                <th scope="col" className="px-4 py-3 font-semibold">Nome</th>
                <th scope="col" className="px-4 py-3 font-semibold">Contato</th>
                <th scope="col" className="px-4 py-3 font-semibold">Veículo</th>
                <th scope="col" className="px-4 py-3 text-right font-semibold">Consumo</th>
                <th scope="col" className="px-4 py-3 font-semibold">Situação</th>
                <th scope="col" className="px-4 py-3"><span className="sr-only">Ações</span></th>
              </tr>
            </thead>
            <tbody>
              {drivers.data.map((d) => (
                <tr key={d.id} className="border-t border-line">
                  <td className="px-4 py-3">
                    <span className={isAnonymized(d) ? "font-semibold text-ink-3" : "font-semibold"}>{d.name}</span>
                    {d.document && (
                      <span className="block font-mono text-xs text-ink-3 tnum">
                        CPF {d.document}
                        {d.document_masked && <span className="sr-only"> (protegido)</span>}
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    {isAnonymized(d) ? (
                      <span className="text-ink-3">Dados pessoais removidos (LGPD)</span>
                    ) : (
                      <>
                        <span className="block">{d.email}</span>
                        <span className="block text-ink-3 tnum">{d.phone}</span>
                      </>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    {d.vehicle_name ?? "—"}
                    {d.vehicle_plate && <span className="ml-2 rounded border border-line-strong px-1.5 py-0.5 font-mono text-xs">{d.vehicle_plate}</span>}
                  </td>
                  <td className="px-4 py-3 text-right tnum">
                    {d.km_per_l ? `${fmtNumber(d.km_per_l, 1)} km/l` : <span className="text-ink-3">padrão</span>}
                  </td>
                  <td className="px-4 py-3"><ActiveBadge active={d.active} /></td>
                  <td className="px-4 py-3 text-right">
                    {isAnonymized(d) ? (
                      <span className="text-xs text-ink-3">Anonimizado</span>
                    ) : (
                      <Button
                        size="sm"
                        variant="ghost"
                        aria-label={`Editar ${d.name}`}
                        onClick={() => { setFlash(null); setMode({ kind: "edit", driver: d }); }}
                      >
                        Editar
                      </Button>
                    )}
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

/** The backend's anonymized drivers carry a reserved, undeliverable e-mail. */
function isAnonymized(d: Driver): boolean {
  return d.email.endsWith("@anonimo.invalid");
}

function DriverForm({
  driver,
  canAnonymize,
  onCancel,
  onSaved,
}: {
  driver: Driver | null;
  canAnonymize: boolean;
  onCancel: () => void;
  onSaved: (message: string) => void;
}) {
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);
  const [confirmOff, setConfirmOff] = useState(false);
  const editing = driver !== null;
  const maskedDoc = !!driver?.document_masked;
  const [anonymizing, setAnonymizing] = useState(false);

  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const f = new FormData(e.currentTarget);
    const s = (k: string) => String(f.get(k) ?? "").trim();
    const opt = (k: string) => s(k) || null;
    const errs: Record<string, string> = {};
    if (!s("name")) errs.name = "Informe o nome.";
    if (!s("phone")) errs.phone = "Informe o telefone.";
    if (!editing) {
      if (!s("email").includes("@")) errs.email = "Informe um e-mail válido.";
      if (s("password").length < 8) errs.password = "Use pelo menos 8 caracteres.";
    }
    let km: string | null = null;
    if (s("km_per_l")) {
      km = parseDecimalInput(s("km_per_l"));
      if (!km || Number(km) <= 0) errs.km_per_l = "Informe o consumo em km/l, maior que zero. Ex.: 11,5";
    }
    setErrors(errs);
    if (Object.keys(errs).length) return;
    setBusy(true);
    try {
      if (editing) {
        await api.updateDriver(driver.id, {
          name: s("name"),
          phone: s("phone"),
          // A masked CPF is never round-tripped: absent keeps it, a typed value replaces it.
          document: maskedDoc ? s("document") || undefined : opt("document"),
          vehicle_name: opt("vehicle_name"),
          vehicle_plate: opt("vehicle_plate"),
          km_per_l: km,
        });
        onSaved(`${s("name")}: dados salvos.`);
      } else {
        await api.createDriver({
          name: s("name"),
          email: s("email"),
          password: s("password"),
          phone: s("phone"),
          document: opt("document"),
          vehicle_name: opt("vehicle_name"),
          vehicle_plate: opt("vehicle_plate"),
          km_per_l: km,
        });
        onSaved(`${s("name")} cadastrado. Já pode entrar com o e-mail e a senha informados.`);
      }
    } catch (err) {
      setErrors(fieldErrors(err));
    } finally {
      setBusy(false);
    }
  }

  async function toggleActive() {
    if (!driver) return;
    if (driver.active && !confirmOff) return setConfirmOff(true);
    setBusy(true);
    try {
      await api.updateDriver(driver.id, { active: !driver.active });
      onSaved(driver.active ? `${driver.name} desativado. O histórico continua disponível.` : `${driver.name} reativado.`);
    } catch (err) {
      setErrors(fieldErrors(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Card className="rise mb-6 p-5 sm:p-6">
      <h2 className="display text-lg font-semibold">{editing ? `Editar ${driver.name}` : "Novo motorista"}</h2>
      <form onSubmit={submit} noValidate className="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <Field label="Nome" htmlFor="d-name" error={errors.name}>
          <Input id="d-name" name="name" defaultValue={driver?.name} invalid={!!errors.name} autoComplete="off" />
        </Field>
        {editing ? (
          <Field label="E-mail" htmlFor="d-email" hint="O e-mail de acesso não muda.">
            <Input id="d-email" value={driver.email} readOnly disabled />
          </Field>
        ) : (
          <Field label="E-mail" htmlFor="d-email" error={errors.email}>
            <Input id="d-email" name="email" type="email" invalid={!!errors.email} autoComplete="off" />
          </Field>
        )}
        {!editing && (
          <Field label="Senha inicial" htmlFor="d-password" error={errors.password} hint="Mínimo de 8 caracteres.">
            <Input id="d-password" name="password" type="password" invalid={!!errors.password} autoComplete="new-password" />
          </Field>
        )}
        <Field label="Telefone" htmlFor="d-phone" error={errors.phone}>
          <Input id="d-phone" name="phone" type="tel" defaultValue={driver?.phone} invalid={!!errors.phone} />
        </Field>
        {maskedDoc ? (
          <Field
            label="Novo CPF (opcional)"
            htmlFor="d-document"
            error={errors.document}
            hint={
              <>
                CPF atual <span className="font-mono">{driver?.document}</span>: protegido, visível só para
                administradores. Em branco, mantém o atual.
              </>
            }
          >
            <Input id="d-document" name="document" invalid={!!errors.document} autoComplete="off" />
          </Field>
        ) : (
          <Field
            label="CPF (opcional)"
            htmlFor="d-document"
            error={errors.document}
            hint={editing ? "Deixe em branco para remover." : undefined}
          >
            <Input id="d-document" name="document" defaultValue={driver?.document} invalid={!!errors.document} />
          </Field>
        )}
        <Field label="Veículo (opcional)" htmlFor="d-vehicle" error={errors.vehicle_name} hint={editing ? "Deixe em branco para remover." : undefined}>
          <Input id="d-vehicle" name="vehicle_name" placeholder="Ex.: Fiorino" defaultValue={driver?.vehicle_name} />
        </Field>
        <Field label="Placa (opcional)" htmlFor="d-plate" error={errors.vehicle_plate} hint={editing ? "Deixe em branco para remover." : undefined}>
          <Input id="d-plate" name="vehicle_plate" placeholder="ABC1D23" defaultValue={driver?.vehicle_plate} className="uppercase" />
        </Field>
        <Field label="Consumo km/l (opcional)" htmlFor="d-kml" error={errors.km_per_l} hint="Em branco, vale o consumo padrão dos parâmetros.">
          <Input
            id="d-kml"
            name="km_per_l"
            inputMode="decimal"
            defaultValue={driver?.km_per_l ? decimalForInput(driver.km_per_l) : ""}
            invalid={!!errors.km_per_l}
          />
        </Field>
        {errors.form && <Notice tone="error" className="sm:col-span-2 lg:col-span-3">{errors.form}</Notice>}
        <div className="flex flex-wrap items-center gap-2 sm:col-span-2 lg:col-span-3">
          <Button type="submit" variant="primary" busy={busy}>
            {editing ? "Salvar alterações" : "Cadastrar motorista"}
          </Button>
          <Button type="button" variant="ghost" onClick={onCancel}>
            Cancelar
          </Button>
          {editing && canAnonymize && (
            <Button type="button" variant="danger" className="ml-auto" onClick={() => setAnonymizing(true)} disabled={busy}>
              Anonimizar (LGPD)
            </Button>
          )}
          {editing && (
            <Button
              type="button"
              variant={driver.active ? "danger" : "secondary"}
              className={canAnonymize ? undefined : "ml-auto"}
              onClick={toggleActive}
              disabled={busy}
            >
              {driver.active ? (confirmOff ? "Confirmar desativação" : "Desativar motorista") : "Reativar motorista"}
            </Button>
          )}
        </div>
        {confirmOff && (
          <p role="alert" className="text-sm text-warn-ink sm:col-span-2 lg:col-span-3">
            Desativar impede o acesso e tira o motorista da montagem de roteiros. Os roteiros antigos continuam no histórico.
          </p>
        )}
      </form>
      {anonymizing && driver && (
        <AnonymizeDialog
          driver={driver}
          onClose={() => setAnonymizing(false)}
          onDone={(name) => onSaved(`${name}: dados pessoais removidos. O histórico de roteiros continua nos relatórios.`)}
        />
      )}
    </Card>
  );
}

function AnonymizeDialog({
  driver,
  onClose,
  onDone,
}: {
  driver: Driver;
  onClose: () => void;
  onDone: (name: string) => void;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  const [agree, setAgree] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const d = ref.current;
    if (d && !d.open) d.showModal();
  }, []);

  async function confirm() {
    setBusy(true);
    setError(null);
    try {
      await api.anonymizeDriver(driver.id);
      onDone(driver.name);
    } catch (err) {
      setError(describeError(err));
      setBusy(false);
    }
  }

  return (
    <dialog
      ref={ref}
      onClose={onClose}
      aria-labelledby="anon-title"
      className="m-auto w-[min(32rem,calc(100vw-2rem))] rounded-xl border border-line bg-surface p-0 text-ink shadow-card backdrop:bg-ink/40"
    >
      <div className="p-6">
        <h2 id="anon-title" className="display text-xl font-bold">
          Anonimizar {driver.name}?
        </h2>
        <p className="mt-3 text-sm text-ink-2">
          Atende a um pedido de exclusão (LGPD). <strong className="text-ink">Não tem como desfazer.</strong>
        </p>
        <ul className="mt-3 flex list-disc flex-col gap-1 pl-5 text-sm text-ink-2">
          <li>Apaga nome, e-mail, telefone, CPF, veículo e placa.</li>
          <li>Bloqueia o acesso: a senha deixa de funcionar e o cadastro fica inativo.</li>
          <li>Mantém os roteiros, horários e totais, sem identificar a pessoa.</li>
        </ul>
        <label className="mt-5 flex items-start gap-3 text-sm">
          <input
            type="checkbox"
            checked={agree}
            onChange={(e) => setAgree(e.target.checked)}
            className="mt-0.5 h-5 w-5 accent-[var(--danger)]"
          />
          Entendo que os dados pessoais serão apagados de forma definitiva.
        </label>
        {error && <Notice tone="error" className="mt-4">{error}</Notice>}
        <div className="mt-6 flex flex-wrap justify-end gap-2">
          <Button type="button" variant="ghost" onClick={() => ref.current?.close()} disabled={busy}>
            Cancelar
          </Button>
          <Button
            type="button"
            variant="destructive"
            onClick={confirm}
            disabled={!agree}
            busy={busy}
          >
            Anonimizar definitivamente
          </Button>
        </div>
      </div>
    </dialog>
  );
}
