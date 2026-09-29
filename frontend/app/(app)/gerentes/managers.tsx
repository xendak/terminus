"use client";

import { useState, type FormEvent } from "react";
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
import { AnonymizeDialog, isAnonymized } from "@/components/anonymize-dialog";
import { api, type User } from "@/lib/api";
import { fieldErrors } from "@/lib/errors";
import { roleLabel } from "@/lib/roles";
import { useApi } from "@/lib/use-api";

export function Managers() {
  const user = useUser();
  const managers = useApi(user.role === "admin" ? "managers" : null, () => api.managers());
  const [mode, setMode] = useState<Mode>({ kind: "list" });
  const [flash, setFlash] = useState<string | null>(null);

  if (user.role !== "admin") return <Forbidden />;

  return (
    <>
      <PageHeader
        title="Gerentes"
        actions={
          mode.kind === "list" && (
            <Button variant="primary" onClick={() => { setFlash(null); setMode({ kind: "new" }); }}>
              Novo gerente
            </Button>
          )
        }
      >
        Gestores montam roteiros, cadastram motoristas e pontos e ajustam os parâmetros de custo.
      </PageHeader>
      <div aria-live="polite" className="mb-4 empty:hidden">
        {flash && <Notice tone="success">{flash}</Notice>}
      </div>
      {mode.kind === "new" && (
        <ManagerForm
          onCancel={() => setMode({ kind: "list" })}
          onSaved={(name) => {
            setMode({ kind: "list" });
            setFlash(`${name} cadastrado como gestor.`);
            managers.reload();
          }}
        />
      )}
      {mode.kind === "edit" && (
        <ManagerEdit
          key={mode.manager.id}
          manager={mode.manager}
          onCancel={() => setMode({ kind: "list" })}
          onSaved={(msg) => {
            setMode({ kind: "list" });
            setFlash(msg);
            managers.reload();
          }}
        />
      )}
      {managers.state === "error" ? (
        <ErrorState message="Não foi possível carregar os gerentes." onRetry={managers.reload} />
      ) : !managers.data ? (
        <div className="flex flex-col gap-2">
          {[0, 1].map((i) => (
            <Skeleton key={i} className="h-14 w-full" />
          ))}
        </div>
      ) : managers.data.length === 0 ? (
        <EmptyState title="Nenhum gerente cadastrado" />
      ) : (
        <Card className="overflow-x-auto">
          <table className="w-full min-w-[640px] text-sm">
            <thead className="bg-surface-2 text-left text-ink-2">
              <tr>
                <th scope="col" className="px-4 py-3 font-semibold">Nome</th>
                <th scope="col" className="px-4 py-3 font-semibold">E-mail</th>
                <th scope="col" className="px-4 py-3 font-semibold">Telefone</th>
                <th scope="col" className="px-4 py-3 font-semibold">Perfil</th>
                <th scope="col" className="px-4 py-3 text-right font-semibold">Equipe</th>
                <th scope="col" className="px-4 py-3 font-semibold">Situação</th>
                <th scope="col" className="px-4 py-3"><span className="sr-only">Ações</span></th>
              </tr>
            </thead>
            <tbody>
              {managers.data.map((m) => (
                <tr key={m.id} className="border-t border-line">
                  <td className={isAnonymized(m) ? "px-4 py-3 font-semibold text-ink-3" : "px-4 py-3 font-semibold"}>
                    {m.name}
                  </td>
                  <td className="px-4 py-3">
                    {isAnonymized(m) ? <span className="text-ink-3">Dados pessoais removidos (LGPD)</span> : m.email}
                  </td>
                  <td className="px-4 py-3 tnum">{isAnonymized(m) ? "—" : m.phone}</td>
                  <td className="px-4 py-3">{roleLabel[m.role]}</td>
                  <td className="whitespace-nowrap px-4 py-3 text-right tnum">
                    {typeof m.team_size === "number"
                      ? `${m.team_size} ${m.team_size === 1 ? "motorista" : "motoristas"}`
                      : "—"}
                  </td>
                  <td className="px-4 py-3"><ActiveBadge active={m.active} /></td>
                  <td className="px-4 py-3 text-right">
                    {isAnonymized(m) ? (
                      <span className="text-xs text-ink-3">Anonimizado</span>
                    ) : (
                      <Button
                        size="sm"
                        variant="ghost"
                        aria-label={`Editar ${m.name}`}
                        onClick={() => { setFlash(null); setMode({ kind: "edit", manager: m }); }}
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

type Mode = { kind: "list" } | { kind: "new" } | { kind: "edit"; manager: User };

function ManagerEdit({
  manager,
  onCancel,
  onSaved,
}: {
  manager: User;
  onCancel: () => void;
  onSaved: (message: string) => void;
}) {
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);
  const [confirmOff, setConfirmOff] = useState(false);
  const [anonymizing, setAnonymizing] = useState(false);

  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const f = new FormData(e.currentTarget);
    const s = (k: string) => String(f.get(k) ?? "").trim();
    const errs: Record<string, string> = {};
    if (!s("name")) errs.name = "Informe o nome.";
    if (!s("phone")) errs.phone = "Informe o telefone.";
    setErrors(errs);
    if (Object.keys(errs).length) return;
    setBusy(true);
    try {
      await api.updateManager(manager.id, { name: s("name"), phone: s("phone") });
      onSaved(`${s("name")}: dados salvos.`);
    } catch (err) {
      setErrors(fieldErrors(err));
    } finally {
      setBusy(false);
    }
  }

  async function toggleActive() {
    if (manager.active && !confirmOff) return setConfirmOff(true);
    setBusy(true);
    try {
      await api.updateManager(manager.id, { active: !manager.active });
      onSaved(manager.active ? `${manager.name} desativado. O acesso foi encerrado.` : `${manager.name} reativado.`);
    } catch (err) {
      setErrors(fieldErrors(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Card className="rise mb-6 p-5 sm:p-6">
      <h2 className="display text-lg font-semibold">Editar {manager.name}</h2>
      <form method="post" action="/sem-js" onSubmit={submit} noValidate className="mt-4 grid gap-4 sm:grid-cols-3">
        <Field label="Nome" htmlFor="me-name" error={errors.name}>
          <Input id="me-name" name="name" defaultValue={manager.name} invalid={!!errors.name} autoComplete="off" />
        </Field>
        <Field label="E-mail" htmlFor="me-email" hint="O e-mail de acesso não muda.">
          <Input id="me-email" value={manager.email} readOnly disabled />
        </Field>
        <Field label="Telefone" htmlFor="me-phone" error={errors.phone}>
          <Input id="me-phone" name="phone" type="tel" defaultValue={manager.phone} invalid={!!errors.phone} />
        </Field>
        {errors.form && <Notice tone="error" className="sm:col-span-3">{errors.form}</Notice>}
        <div className="flex flex-wrap items-center gap-2 sm:col-span-3">
          <Button type="submit" variant="primary" busy={busy}>
            Salvar alterações
          </Button>
          <Button type="button" variant="ghost" onClick={onCancel}>
            Cancelar
          </Button>
          <Button type="button" variant="danger" className="ml-auto" onClick={() => setAnonymizing(true)} disabled={busy}>
            Anonimizar (LGPD)
          </Button>
          <Button type="button" variant={manager.active ? "danger" : "secondary"} onClick={toggleActive} disabled={busy}>
            {manager.active ? (confirmOff ? "Confirmar desativação" : "Desativar gerente") : "Reativar gerente"}
          </Button>
        </div>
        {confirmOff && (
          <p role="alert" className="text-sm text-warn-ink sm:col-span-3">
            Desativar encerra as sessões abertas e impede novos acessos. Roteiros e registros feitos por esse gerente
            continuam nos relatórios.
          </p>
        )}
      </form>
      {anonymizing && (
        <AnonymizeDialog
          name={manager.name}
          effects={[
            "Apaga nome, e-mail e telefone.",
            "Bloqueia o acesso: a senha deixa de funcionar e o cadastro fica inativo.",
            "Mantém roteiros, parâmetros e auditoria, sem identificar a pessoa.",
          ]}
          onConfirm={() => api.anonymizeManager(manager.id)}
          onClose={() => setAnonymizing(false)}
          onDone={() => onSaved(`${manager.name}: dados pessoais removidos.`)}
        />
      )}
    </Card>
  );
}

function ManagerForm({ onCancel, onSaved }: { onCancel: () => void; onSaved: (name: string) => void }) {
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const f = new FormData(e.currentTarget);
    const s = (k: string) => String(f.get(k) ?? "").trim();
    const errs: Record<string, string> = {};
    if (!s("name")) errs.name = "Informe o nome.";
    if (!s("email").includes("@")) errs.email = "Informe um e-mail válido.";
    if (s("password").length < 8) errs.password = "Use pelo menos 8 caracteres.";
    if (!s("phone")) errs.phone = "Informe o telefone.";
    setErrors(errs);
    if (Object.keys(errs).length) return;
    setBusy(true);
    try {
      await api.createManager({ name: s("name"), email: s("email"), password: s("password"), phone: s("phone") });
      onSaved(s("name"));
    } catch (err) {
      setErrors(fieldErrors(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Card className="rise mb-6 p-5 sm:p-6">
      <h2 className="display text-lg font-semibold">Novo gerente</h2>
      <form method="post" action="/sem-js" onSubmit={submit} noValidate className="mt-4 grid gap-4 sm:grid-cols-2">
        <Field label="Nome" htmlFor="m-name" error={errors.name}>
          <Input id="m-name" name="name" invalid={!!errors.name} autoComplete="off" />
        </Field>
        <Field label="E-mail" htmlFor="m-email" error={errors.email}>
          <Input id="m-email" name="email" type="email" invalid={!!errors.email} autoComplete="off" />
        </Field>
        <Field label="Senha inicial" htmlFor="m-password" error={errors.password} hint="Mínimo de 8 caracteres.">
          <Input id="m-password" name="password" type="password" invalid={!!errors.password} autoComplete="new-password" />
        </Field>
        <Field label="Telefone" htmlFor="m-phone" error={errors.phone}>
          <Input id="m-phone" name="phone" type="tel" invalid={!!errors.phone} />
        </Field>
        {errors.form && <Notice tone="error" className="sm:col-span-2">{errors.form}</Notice>}
        <div className="flex gap-2 sm:col-span-2">
          <Button type="submit" variant="primary" busy={busy}>
            Cadastrar gerente
          </Button>
          <Button type="button" variant="ghost" onClick={onCancel}>
            Cancelar
          </Button>
        </div>
      </form>
    </Card>
  );
}
