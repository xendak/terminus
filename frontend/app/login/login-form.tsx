"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { useState, type FormEvent } from "react";
import { Button, Field, Input, Notice } from "@/components/ui";
import { api, ApiError } from "@/lib/api";
import { describeError } from "@/lib/errors";
import { homeFor, safeNext } from "@/lib/roles";

export function LoginForm() {
  const router = useRouter();
  const params = useSearchParams();
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = new FormData(e.currentTarget);
    setBusy(true);
    setError(null);
    try {
      const me = await api.login(String(form.get("email") ?? "").trim(), String(form.get("password") ?? ""));
      router.replace(safeNext(params.get("next"), window.location.origin) ?? homeFor(me.user.role));
      router.refresh();
    } catch (err) {
      setError(
        err instanceof ApiError && (err.status === 401 || err.status === 400)
          ? "E-mail ou senha incorretos."
          : describeError(err),
      );
      setBusy(false);
    }
  }

  return (
    <form method="post" action="/sem-js" onSubmit={onSubmit} className="mt-8 flex flex-col gap-4" noValidate>
      {error && <Notice tone="error">{error}</Notice>}
      <Field label="E-mail" htmlFor="email">
        <Input id="email" name="email" type="email" autoComplete="username" inputMode="email" required autoFocus />
      </Field>
      <Field label="Senha" htmlFor="password">
        <Input id="password" name="password" type="password" autoComplete="current-password" required />
      </Field>
      <Button type="submit" variant="primary" size="lg" busy={busy} className="mt-2">
        {busy ? "Entrando…" : "Entrar"}
      </Button>
    </form>
  );
}
