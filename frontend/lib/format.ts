// pt-BR display helpers. Every timestamp renders in America/Sao_Paulo;
// route dates (midnight UTC) render by their calendar date alone.

export const TZ = "America/Sao_Paulo";

// Brazil has had no daylight saving since 2019: São Paulo is a fixed
// UTC−03:00, which lets manual datetime entries become RFC 3339 exactly.
const SP_OFFSET = "-03:00";

const dateTimeFmt = new Intl.DateTimeFormat("pt-BR", {
  timeZone: TZ,
  day: "2-digit",
  month: "2-digit",
  year: "numeric",
  hour: "2-digit",
  minute: "2-digit",
  hourCycle: "h23",
});

const timeFmt = new Intl.DateTimeFormat("pt-BR", {
  timeZone: TZ,
  hour: "2-digit",
  minute: "2-digit",
  hourCycle: "h23",
});

const isoDayFmt = new Intl.DateTimeFormat("en-CA", {
  timeZone: TZ,
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
});

const brl = new Intl.NumberFormat("pt-BR", { style: "currency", currency: "BRL" });

/** "2026-06-15T00:00:00Z" or "2026-06-15" → "15/06/2026". */
export function fmtDate(value: string): string {
  const [y, m, d] = value.slice(0, 10).split("-");
  return `${d}/${m}/${y}`;
}

/** Short weekday + date for headings: "seg., 15/06/2026". */
export function fmtDateLong(value: string): string {
  const day = value.slice(0, 10);
  const weekday = new Intl.DateTimeFormat("pt-BR", { weekday: "long", timeZone: "UTC" }).format(
    new Date(`${day}T12:00:00Z`),
  );
  return `${weekday.charAt(0).toUpperCase()}${weekday.slice(1)}, ${fmtDate(day)}`;
}

export function fmtDateTime(ts: string | null | undefined): string {
  if (!ts) return "—";
  return dateTimeFmt.format(new Date(ts)).replace(",", "");
}

export function fmtTime(ts: string | null | undefined): string {
  if (!ts) return "—";
  return timeFmt.format(new Date(ts));
}

/** "2026-06" → "jun/2026". */
export function fmtMonth(ym: string): string {
  const [y, m] = ym.split("-");
  const name = new Intl.DateTimeFormat("pt-BR", { month: "short", timeZone: "UTC" })
    .format(new Date(`${y}-${m}-15T12:00:00Z`))
    .replace(".", "");
  return `${name}/${y}`;
}

/** Whole minutes → "45 min" or "1 h 15 min". */
export function fmtMinutes(min: number): string {
  if (min < 60) return `${min} min`;
  const h = Math.floor(min / 60);
  const m = min % 60;
  return m === 0 ? `${h} h` : `${h} h ${m} min`;
}

/** Seconds → "00:12:05" for the stopwatch. */
export function fmtClock(totalSeconds: number): string {
  const s = Math.max(0, Math.floor(totalSeconds));
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  return [h, m, sec].map((n) => String(n).padStart(2, "0")).join(":");
}

/** Decimal string or number → pt-BR with fixed fraction digits. */
export function fmtNumber(value: string | number | null | undefined, digits = 1): string {
  if (value === null || value === undefined || value === "") return "—";
  const n = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(n)) return "—";
  return n.toLocaleString("pt-BR", { minimumFractionDigits: digits, maximumFractionDigits: digits });
}

/** "15.625" → "15,6%". */
export function fmtPercent(value: string | number | null | undefined): string {
  const s = fmtNumber(value, 1);
  return s === "—" ? s : `${s}%`;
}

/** "48.72" → "R$ 48,72". */
export function fmtBRL(value: string | number | null | undefined): string {
  if (value === null || value === undefined || value === "") return "—";
  const n = typeof value === "number" ? value : Number(value);
  return Number.isFinite(n) ? brl.format(n) : "—";
}

/** Trim a numeric(…,4) string for editing: "6.0900" → "6,09". */
export function decimalForInput(value: string): string {
  const n = Number(value);
  if (!Number.isFinite(n)) return value;
  return String(n).replace(".", ",");
}

/** Accept "6,09" or "6.09"; return "6.09" or null when not a number. */
export function parseDecimalInput(value: string): string | null {
  const v = value.trim().replace(/\s/g, "").replace(",", ".");
  if (v === "" || !/^-?\d+(\.\d+)?$/.test(v)) return null;
  return v;
}

/** Today's date in São Paulo as YYYY-MM-DD. */
export function todayISO(now = new Date()): string {
  return isoDayFmt.format(now);
}

export function addDaysISO(iso: string, days: number): string {
  const d = new Date(`${iso}T12:00:00Z`);
  d.setUTCDate(d.getUTCDate() + days);
  return d.toISOString().slice(0, 10);
}

export function monthStartISO(iso: string): string {
  return `${iso.slice(0, 7)}-01`;
}

export function addMonthsISO(iso: string, months: number): string {
  const d = new Date(`${iso}T12:00:00Z`);
  d.setUTCMonth(d.getUTCMonth() + months);
  return d.toISOString().slice(0, 10);
}

/** RFC 3339 → value for <input type="datetime-local"> in São Paulo time. */
export function toLocalInput(ts: string | null | undefined): string {
  if (!ts) return "";
  const parts = new Intl.DateTimeFormat("en-CA", {
    timeZone: TZ,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
  }).formatToParts(new Date(ts));
  const get = (t: string) => parts.find((p) => p.type === t)?.value ?? "00";
  return `${get("year")}-${get("month")}-${get("day")}T${get("hour")}:${get("minute")}`;
}

/** datetime-local value (São Paulo wall time) → RFC 3339 with offset. */
export function fromLocalInput(value: string): string {
  return `${value.length === 16 ? `${value}:00` : value}${SP_OFFSET}`;
}

/** Current São Paulo wall time for a datetime-local default. */
export function nowLocalInput(): string {
  return toLocalInput(new Date().toISOString());
}
