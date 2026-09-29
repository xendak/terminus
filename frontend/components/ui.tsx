import Link from "next/link";
import type { ButtonHTMLAttributes, CSSProperties, InputHTMLAttributes, ReactNode, SelectHTMLAttributes } from "react";
import type { RouteStatus } from "@/lib/api";

export function cx(...parts: (string | false | null | undefined)[]): string {
  return parts.filter(Boolean).join(" ");
}

// ---- brand ----------------------------------------------------------------

/** The Terminus plate: a green indication sign with a white inset rule. */
export function Wordmark({ size = "md" }: { size?: "sm" | "md" | "lg" }) {
  const pad = size === "lg" ? "px-5 py-2.5 text-2xl" : size === "sm" ? "px-2.5 py-1 text-sm" : "px-3.5 py-1.5 text-base";
  return (
    <span className={cx("plate display inline-flex items-center gap-2 font-bold uppercase tracking-[0.14em]", pad)}>
      <span aria-hidden className="inline-block h-[0.55em] w-[0.55em] rounded-[2px] bg-current" />
      Terminus
    </span>
  );
}

// ---- buttons ----------------------------------------------------------------

type Variant = "primary" | "secondary" | "ghost" | "danger" | "cone";

const variants: Record<Variant, string> = {
  primary: "bg-placa text-on-placa hover:bg-placa-strong border border-transparent",
  secondary: "bg-surface text-ink border border-line-strong hover:bg-surface-2",
  ghost: "bg-transparent text-ink-2 hover:bg-surface-2 hover:text-ink border border-transparent",
  danger: "bg-surface text-danger border border-line-strong hover:bg-danger-soft",
  cone: "bg-cone text-white hover:bg-cone-bright border border-transparent",
};

const sizes = {
  sm: "h-11 px-3 text-sm gap-1.5 sm:h-8",
  md: "h-10 px-4 text-sm gap-2",
  lg: "h-12 px-5 text-base gap-2",
  xl: "min-h-16 px-6 text-lg gap-3",
};

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  size?: keyof typeof sizes;
  busy?: boolean;
}

export function Button({ variant = "secondary", size = "md", busy, className, children, disabled, ...rest }: ButtonProps) {
  return (
    <button
      {...rest}
      disabled={disabled || busy}
      aria-busy={busy || undefined}
      className={cx(
        "inline-flex select-none items-center justify-center rounded-lg font-semibold transition-colors duration-150",
        "disabled:cursor-not-allowed disabled:opacity-55",
        variants[variant],
        sizes[size],
        className,
      )}
    >
      {busy && <Spinner />}
      {children}
    </button>
  );
}

export function ButtonLink({
  href,
  variant = "secondary",
  size = "md",
  className,
  children,
}: {
  href: string;
  variant?: Variant;
  size?: keyof typeof sizes;
  className?: string;
  children: ReactNode;
}) {
  return (
    <Link
      href={href}
      className={cx(
        "inline-flex items-center justify-center rounded-lg font-semibold transition-colors duration-150",
        variants[variant],
        sizes[size],
        className,
      )}
    >
      {children}
    </Link>
  );
}

export function Spinner() {
  return (
    <span
      aria-hidden
      className="inline-block h-4 w-4 animate-spin rounded-full border-2 border-current border-r-transparent"
    />
  );
}

// ---- form fields ------------------------------------------------------------

const control =
  "w-full rounded-lg border bg-surface px-3 text-ink placeholder:text-ink-3 transition-colors " +
  "focus:outline-none focus-visible:outline-2 focus-visible:outline-offset-1 disabled:opacity-60";

export function Field({
  label,
  htmlFor,
  error,
  hint,
  children,
  className,
}: {
  label: string;
  htmlFor: string;
  error?: string;
  hint?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <div className={cx("flex flex-col gap-1.5", className)}>
      <label htmlFor={htmlFor} className="text-sm font-medium text-ink-2">
        {label}
      </label>
      {children}
      {error ? (
        <p id={`${htmlFor}-error`} role="alert" className="text-sm font-medium text-danger">
          {error}
        </p>
      ) : hint ? (
        <p className="text-xs text-ink-3">{hint}</p>
      ) : null}
    </div>
  );
}

export function Input({ invalid, className, ...rest }: InputHTMLAttributes<HTMLInputElement> & { invalid?: boolean }) {
  return (
    <input
      {...rest}
      aria-invalid={invalid || undefined}
      aria-describedby={invalid && rest.id ? `${rest.id}-error` : rest["aria-describedby"]}
      className={cx(control, "h-10", invalid ? "border-danger" : "border-line-strong", className)}
    />
  );
}

export function Select({
  invalid,
  className,
  children,
  ...rest
}: SelectHTMLAttributes<HTMLSelectElement> & { invalid?: boolean }) {
  return (
    <select
      {...rest}
      aria-invalid={invalid || undefined}
      aria-describedby={invalid && rest.id ? `${rest.id}-error` : undefined}
      className={cx(control, "h-10 pr-8", invalid ? "border-danger" : "border-line-strong", className)}
    >
      {children}
    </select>
  );
}

// ---- surfaces ---------------------------------------------------------------

export function Card({ className, children }: { className?: string; children: ReactNode }) {
  return <section className={cx("relative rounded-xl border border-line bg-surface shadow-card", className)}>{children}</section>;
}

export function PageHeader({
  title,
  eyebrow,
  actions,
  children,
}: {
  title: string;
  eyebrow?: ReactNode;
  actions?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <header className="mb-6 flex flex-wrap items-end justify-between gap-4">
      <div className="min-w-0">
        {eyebrow && <div className="mb-1 text-sm font-medium text-ink-3">{eyebrow}</div>}
        <h1 className="display text-2xl font-bold tracking-tight text-balance sm:text-3xl">{title}</h1>
        {children && <div className="mt-1.5 max-w-2xl text-ink-2">{children}</div>}
      </div>
      {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
    </header>
  );
}

// ---- feedback ---------------------------------------------------------------

export function Notice({
  tone = "info",
  children,
  className,
}: {
  tone?: "info" | "success" | "error" | "warn";
  children: ReactNode;
  className?: string;
}) {
  const tones = {
    info: "border-line bg-surface-2 text-ink",
    success: "border-placa/40 bg-placa-soft text-ink",
    error: "border-danger/40 bg-danger-soft text-ink",
    warn: "border-warn-ink/30 bg-warn-soft text-ink",
  };
  return (
    <div
      role={tone === "error" ? "alert" : "status"}
      className={cx("rise rounded-lg border px-4 py-3 text-sm", tones[tone], className)}
    >
      {children}
    </div>
  );
}

export function EmptyState({ title, children, action }: { title: string; children?: ReactNode; action?: ReactNode }) {
  return (
    <div className="flex flex-col items-center gap-3 rounded-xl border border-dashed border-line-strong px-6 py-12 text-center">
      <RouteGlyph />
      <p className="display text-lg font-semibold">{title}</p>
      {children && <div className="max-w-md text-sm text-ink-2">{children}</div>}
      {action}
    </div>
  );
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div role="alert" className="flex flex-col items-start gap-3 rounded-xl border border-danger/40 bg-danger-soft px-5 py-4">
      <p className="font-medium">{message}</p>
      {onRetry && (
        <Button size="sm" onClick={onRetry}>
          Tentar de novo
        </Button>
      )}
    </div>
  );
}

export function Forbidden() {
  return (
    <EmptyState title="Esta área não faz parte do seu perfil">
      Se você precisa desse acesso, fale com o administrador da operação.
    </EmptyState>
  );
}

export function Skeleton({ className, style }: { className?: string; style?: CSSProperties }) {
  return <div aria-hidden className={cx("skeleton", className)} style={style} />;
}

/** Three stops on a line: the empty-state mark. */
function RouteGlyph() {
  return (
    <svg aria-hidden width="88" height="20" viewBox="0 0 88 20" className="text-line-strong">
      <line x1="8" y1="10" x2="80" y2="10" stroke="currentColor" strokeWidth="2" strokeDasharray="4 4" />
      <rect x="2" y="4" width="12" height="12" rx="2" fill="var(--placa)" />
      <circle cx="44" cy="10" r="6" fill="var(--surface)" stroke="currentColor" strokeWidth="2" />
      <circle cx="80" cy="10" r="6" fill="var(--surface)" stroke="currentColor" strokeWidth="2" />
    </svg>
  );
}

// ---- status -----------------------------------------------------------------

export const statusLabel: Record<RouteStatus, string> = {
  draft: "Rascunho",
  active: "Em andamento",
  closed: "Encerrado",
};

export function StatusBadge({ status }: { status: RouteStatus }) {
  const tone = {
    draft: "border-line-strong text-ink-2 bg-surface",
    active: "border-transparent bg-cone-soft text-cone",
    closed: "border-transparent bg-placa-soft text-placa-strong",
  }[status];
  return (
    <span className={cx("inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-semibold", tone)}>
      {status === "active" && <span aria-hidden className="h-1.5 w-1.5 rounded-full bg-cone" />}
      {statusLabel[status]}
    </span>
  );
}

export function ActiveBadge({ active }: { active: boolean }) {
  return active ? (
    <span className="inline-flex items-center rounded-full bg-placa-soft px-2.5 py-0.5 text-xs font-semibold text-placa-strong">
      Ativo
    </span>
  ) : (
    <span className="inline-flex items-center rounded-full border border-line-strong px-2.5 py-0.5 text-xs font-semibold text-ink-3">
      Inativo
    </span>
  );
}

/**
 * Share of the standard workday (100%) as a ruler, in hour or percent ticks.
 * The stopped share fills in cone orange; beyond 100% it caps and says so.
 */
export function JourneyRuler({
  percent,
  hours = 8,
  label,
  scale = "hours",
}: {
  percent: number;
  hours?: number;
  label?: string;
  /** "hours": one workday in hour ticks; "percent": a share of many workdays. */
  scale?: "hours" | "percent";
}) {
  const clamped = Math.max(0, Math.min(100, percent));
  const ticks =
    scale === "percent"
      ? [0, 25, 50, 75, 100]
      : Array.from({ length: Math.max(1, Math.round(hours)) + 1 }, (_, i) => i);
  return (
    <div className="w-full">
      <div
        role="meter"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={Math.round(percent * 10) / 10}
        aria-label={label ?? "Parte da jornada parada"}
        className="relative h-3 overflow-hidden rounded-sm bg-surface-2"
      >
        <div className="absolute inset-y-0 left-0 bg-cone transition-[width] duration-500" style={{ width: `${clamped}%` }} />
      </div>
      <div className="relative mt-1 h-4 text-[10px] text-ink-3 tnum">
        {ticks.map((h, i) => (
          <span
            key={h}
            className="absolute -translate-x-1/2 first:translate-x-0 last:-translate-x-full"
            style={{ left: `${(i / (ticks.length - 1)) * 100}%` }}
          >
            {scale === "percent" ? `${h}%` : `${h}h`}
          </span>
        ))}
      </div>
    </div>
  );
}
