import type { Metadata } from "next";
import Link from "next/link";
import { Wordmark } from "@/components/ui";

export const metadata: Metadata = {
  title: "Sobre",
  description:
    "Terminus mede quanto tempo cada entregador fica parado em cada ponto do roteiro e transforma isso em painel e custo.",
};

const signRows = [
  { place: "Rua Peru, 55", minutes: 20, arrow: "M12 20V5M6 11l6-6 6 6" },
  { place: "Av. Brasil, 1200", minutes: 35, arrow: "M4 12h15M13 6l6 6-6 6" },
  { place: "Rua Goiás, 18", minutes: 20, arrow: "M6 18 18 6M9 6h9v9" },
];

const benefits = [
  {
    title: "Tempo parado com endereço",
    body: "O motorista toca em “Cheguei” e “Saí”. O Terminus mede quanto tempo ele ficou em cada ponto, sem planilha e sem estimativa.",
  },
  {
    title: "Painel por dia, mês e período",
    body: "Veja onde a operação para mais, compare motoristas e acompanhe quanto da jornada de 8 horas vira espera.",
  },
  {
    title: "Custo real de cada rota",
    body: "Km rodados, consumo do veículo e preço do combustível viram custo estimado por roteiro. Os valores mudam na tela, sem programador.",
  },
];

const steps = [
  { title: "O gestor monta o roteiro", body: "Motorista, data e os pontos na ordem da visita. O primeiro é a partida." },
  { title: "O motorista registra no celular", body: "Chegada e saída em cada ponto, com cronômetro na tela enquanto está parado." },
  { title: "O painel mostra o resultado", body: "Totais por dia, mês e período, histórico com endereços e exportação em CSV." },
];

export default function SobrePage() {
  return (
    <div className="min-h-dvh">
      <header className="mx-auto flex max-w-6xl items-center justify-between px-5 py-5 sm:px-8">
        <Wordmark />
        <Link
          href="/login"
          className="inline-flex h-10 items-center rounded-lg border border-line-strong px-4 text-sm font-semibold hover:bg-surface-2"
        >
          Entrar
        </Link>
      </header>

      <main>
        <section className="mx-auto grid max-w-6xl items-center gap-12 px-5 pb-16 pt-8 sm:px-8 lg:grid-cols-[1.05fr_1fr] lg:pb-24 lg:pt-16">
          <div>
            <p className="text-sm font-semibold uppercase tracking-[0.16em] text-placa">Para operações de entrega urbana</p>
            <h1 className="display mt-4 text-4xl font-bold leading-[1.05] tracking-tight text-balance sm:text-6xl">
              Cada minuto parado tem um endereço.
            </h1>
            <p className="mt-6 max-w-xl text-lg text-ink-2 text-pretty">
              O Terminus registra quanto tempo cada entregador fica em cada ponto do roteiro e mostra, num painel, onde
              a sua rota perde tempo e dinheiro.
            </p>
            <div className="mt-8 flex flex-wrap items-center gap-3">
              <Link
                href="/login"
                className="inline-flex h-12 items-center rounded-lg bg-placa px-6 text-base font-bold text-on-placa hover:bg-placa-strong"
              >
                Entrar no Terminus
              </Link>
              <a href="#como-funciona" className="inline-flex h-12 items-center px-2 font-semibold text-ink-2 underline-offset-4 hover:text-ink hover:underline">
                Como funciona
              </a>
            </div>
          </div>

          {/* The signature: a Brazilian indication plate, but for dwell time. */}
          <figure aria-label="Placa com o tempo parado em três pontos de um roteiro" className="plate rotate-[-1.2deg] p-7 shadow-card sm:p-9">
            <p className="display text-xs font-semibold uppercase tracking-[0.24em] opacity-85">Tempo parado hoje</p>
            <ul className="mt-5 flex flex-col divide-y divide-on-placa/35">
              {signRows.map((r) => (
                <li key={r.place} className="flex items-center gap-4 py-4">
                  <svg aria-hidden width="30" height="30" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.6" strokeLinecap="round" strokeLinejoin="round">
                    <path d={r.arrow} />
                  </svg>
                  <span className="display flex-1 text-xl font-semibold sm:text-2xl">{r.place}</span>
                  <span className="display text-2xl font-bold tnum sm:text-3xl">
                    {r.minutes}
                    <span className="ml-1 text-base font-semibold">min</span>
                  </span>
                </li>
              ))}
            </ul>
            <figcaption className="mt-4 flex items-baseline justify-between border-t-2 border-on-placa pt-4">
              <span className="text-sm opacity-90">15,6% da jornada de 8 h</span>
              <span className="display text-3xl font-bold tnum">1 h 15</span>
            </figcaption>
          </figure>
        </section>

        <section aria-labelledby="beneficios" className="border-y border-line bg-surface">
          <div className="mx-auto max-w-6xl px-5 py-16 sm:px-8">
            <h2 id="beneficios" className="sr-only">
              Benefícios
            </h2>
            <ul className="grid gap-10 md:grid-cols-3">
              {benefits.map((b) => (
                <li key={b.title}>
                  <span aria-hidden className="mb-4 block h-1 w-10 rounded-full bg-cone" />
                  <h3 className="display text-xl font-semibold">{b.title}</h3>
                  <p className="mt-2 text-ink-2 text-pretty">{b.body}</p>
                </li>
              ))}
            </ul>
          </div>
        </section>

        <section id="como-funciona" aria-labelledby="como-funciona-titulo" className="mx-auto max-w-6xl px-5 py-16 sm:px-8 lg:py-24">
          <h2 id="como-funciona-titulo" className="display text-3xl font-bold tracking-tight">
            Como funciona
          </h2>
          <ol className="mt-10 grid gap-8 md:grid-cols-3">
            {steps.map((s, i) => (
              <li key={s.title} className="relative">
                <div className="flex items-center gap-3">
                  <span
                    aria-hidden
                    className={
                      i === 0
                        ? "flex h-9 w-9 items-center justify-center rounded-md bg-placa font-bold text-on-placa"
                        : "flex h-9 w-9 items-center justify-center rounded-full border-[3px] border-placa font-bold text-placa"
                    }
                  >
                    {i + 1}
                  </span>
                  {i < steps.length - 1 && <span aria-hidden className="hidden h-[3px] flex-1 bg-placa/40 md:block" />}
                </div>
                <h3 className="display mt-4 text-lg font-semibold">{s.title}</h3>
                <p className="mt-1.5 text-ink-2">{s.body}</p>
              </li>
            ))}
          </ol>
        </section>

        <section className="bg-ink text-paper">
          <div className="mx-auto flex max-w-6xl flex-col items-start gap-6 px-5 py-14 sm:px-8 md:flex-row md:items-center md:justify-between">
            <div>
              <h2 className="display text-2xl font-bold sm:text-3xl">Descubra onde a sua rota para.</h2>
              <p className="mt-2 opacity-80">Entre com a conta criada pelo gestor da sua operação.</p>
            </div>
            <Link
              href="/login"
              className="inline-flex h-12 items-center rounded-lg bg-cone px-6 text-base font-bold text-white hover:bg-cone-bright"
            >
              Entrar no Terminus
            </Link>
          </div>
        </section>
      </main>

      <footer className="mx-auto max-w-6xl px-5 py-8 text-sm text-ink-3 sm:px-8">
        Terminus · MVP acadêmico, Engenharia de Software II, PUC Minas.
      </footer>
    </div>
  );
}
