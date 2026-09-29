import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";
import { redirect } from "next/navigation";
import { Wordmark } from "@/components/ui";
import { getSession } from "@/lib/session";
import { homeFor } from "@/lib/roles";
import { LoginForm } from "./login-form";

export const metadata: Metadata = { title: "Entrar" };

export default async function LoginPage() {
  const s = await getSession();
  if (s.kind === "ok") redirect(homeFor(s.me.user.role));

  return (
    <main className="grid min-h-dvh lg:grid-cols-[minmax(0,1fr)_minmax(0,1.1fr)]">
      <div className="flex flex-col justify-between gap-10 px-6 py-8 sm:px-12">
        <Link href="/sobre" aria-label="Sobre o Terminus" className="self-start">
          <Wordmark />
        </Link>
        <div className="w-full max-w-sm">
          <h1 className="display text-3xl font-bold tracking-tight">Entrar</h1>
          <p className="mt-2 text-ink-2">Use o e-mail e a senha cadastrados pelo gestor da sua operação.</p>
          <Suspense>
            <LoginForm />
          </Suspense>
        </div>
        <p className="text-sm text-ink-3">
          Primeira vez por aqui?{" "}
          <Link href="/sobre" className="font-semibold text-placa underline-offset-4 hover:underline">
            Conheça o Terminus
          </Link>
        </p>
      </div>
      <SchematicPanel />
    </main>
  );
}

/** A day's route drawn as a transit line: the product in one picture. */
function SchematicPanel() {
  const stops = [
    { t: "07:40", name: "Base — Av. Partida, 100", dwell: null },
    { t: "08:15", name: "Rua Peru, 55", dwell: 20 },
    { t: "09:05", name: "Av. Brasil, 1200", dwell: 35 },
    { t: "10:10", name: "Rua Goiás, 18", dwell: 20 },
  ];
  return (
    <aside aria-hidden className="relative hidden overflow-hidden bg-placa text-on-placa lg:flex lg:flex-col lg:justify-center lg:px-16">
      <div className="absolute inset-4 rounded-2xl border-2 border-on-placa/70" />
      <div className="relative max-w-md">
        <p className="display text-sm font-semibold uppercase tracking-[0.2em] opacity-80">Roteiro de hoje</p>
        <ol className="mt-8">
          {stops.map((s, i) => (
            <li key={s.t} className="relative flex gap-5 pb-9 last:pb-0">
              {i < stops.length - 1 && <span className="absolute left-[9px] top-5 h-full w-[3px] bg-on-placa/60" />}
              <span
                className={
                  i === 0
                    ? "relative z-10 mt-0.5 h-[21px] w-[21px] shrink-0 rounded-[4px] bg-on-placa"
                    : "relative z-10 mt-0.5 h-[21px] w-[21px] shrink-0 rounded-full border-[3px] border-on-placa bg-placa"
                }
              />
              <div className="flex flex-1 items-baseline justify-between gap-4">
                <div>
                  <p className="display text-lg font-semibold">{s.name}</p>
                  <p className="text-sm opacity-80 tnum">{i === 0 ? `Partida ${s.t}` : `Chegada ${s.t}`}</p>
                </div>
                {s.dwell !== null && (
                  <span className="rounded-md bg-cone px-2 py-1 font-mono text-sm font-semibold text-white tnum">
                    {s.dwell} min
                  </span>
                )}
              </div>
            </li>
          ))}
        </ol>
        <p className="mt-10 border-t border-on-placa/40 pt-5 text-lg">
          <span className="display text-3xl font-bold tnum">1 h 15 min</span>
          <span className="ml-3 opacity-85">parado · 15,6% da jornada</span>
        </p>
      </div>
    </aside>
  );
}
