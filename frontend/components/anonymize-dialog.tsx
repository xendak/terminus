"use client";

import { useEffect, useRef, useState } from "react";
import type { User } from "@/lib/api";
import { describeError } from "@/lib/errors";
import { Button, Notice } from "./ui";

/** Anonymized accounts carry the backend's reserved, undeliverable e-mail. */
export function isAnonymized(u: Pick<User, "email">): boolean {
  return u.email.endsWith("@anonimo.invalid");
}

/**
 * LGPD erasure confirmation: a native modal (focus trap, Esc) that states
 * what is erased and kept, and only arms the action after an explicit tick.
 */
export function AnonymizeDialog({
  name,
  effects,
  onConfirm,
  onClose,
  onDone,
}: {
  name: string;
  effects: string[];
  onConfirm: () => Promise<unknown>;
  onClose: () => void;
  onDone: () => void;
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
      await onConfirm();
      onDone();
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
          Anonimizar {name}?
        </h2>
        <p className="mt-3 text-sm text-ink-2">
          Atende a um pedido de exclusão (LGPD). <strong className="text-ink">Não tem como desfazer.</strong>
        </p>
        <ul className="mt-3 flex list-disc flex-col gap-1 pl-5 text-sm text-ink-2">
          {effects.map((e) => (
            <li key={e}>{e}</li>
          ))}
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
        {error && (
          <Notice tone="error" className="mt-4">
            {error}
          </Notice>
        )}
        <div className="mt-6 flex flex-wrap justify-end gap-2">
          <Button type="button" variant="ghost" onClick={() => ref.current?.close()} disabled={busy}>
            Cancelar
          </Button>
          <Button type="button" variant="destructive" onClick={confirm} disabled={!agree} busy={busy}>
            Anonimizar definitivamente
          </Button>
        </div>
      </div>
    </dialog>
  );
}
