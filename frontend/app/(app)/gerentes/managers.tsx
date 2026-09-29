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
import { api } from "@/lib/api";
import { fieldErrors } from "@/lib/errors";
import { roleLabel } from "@/lib/roles";
import { useApi } from "@/lib/use-api";

export function Managers() {
  const user = useUser();
  const managers = useApi(user.role === "admin" ? "managers" : null, () => api.managers());
  const [creating, setCreating] = useState(false);
  const [flash, setFlash] = useState<string | null>(null);

  if (user.role !== "admin") return <Forbidden />;

  return (
    <>
      <PageHeader
        title="Gerentes"
        actions={
          !creating && (
            <Button variant="primary" onClick={() => { setFlash(null); setCreating(true); }}>
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
      {creating && (
        <ManagerForm
          onCancel={() => setCreating(false)}
          onSaved={(name) => {
            setCreating(false);
            setFlash(`${name} cadastrado como gestor.`);
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
          <table className="w-full min-w-[520px] text-sm">
            <thead className="bg-surface-2 text-left text-ink-2">
              <tr>
                <th scope="col" className="px-4 py-3 font-semibold">Nome</th>
                <th scope="col" className="px-4 py-3 font-semibold">E-mail</th>
                <th scope="col" className="px-4 py-3 font-semibold">Telefone</th>
                <th scope="col" className="px-4 py-3 font-semibold">Perfil</th>
                <th scope="col" className="px-4 py-3 font-semibold">Situação</th>
              </tr>
            </thead>
            <tbody>
              {managers.data.map((m) => (
                <tr key={m.id} className="border-t border-line">
                  <td className="px-4 py-3 font-semibold">{m.name}</td>
                  <td className="px-4 py-3">{m.email}</td>
                  <td className="px-4 py-3 tnum">{m.phone}</td>
                  <td className="px-4 py-3">{roleLabel[m.role]}</td>
                  <td className="px-4 py-3"><ActiveBadge active={m.active} /></td>
                </tr>
              ))}
            </tbody>
          </table>
        </Card>
      )}
    </>
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
      <form onSubmit={submit} noValidate className="mt-4 grid gap-4 sm:grid-cols-2">
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
