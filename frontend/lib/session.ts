import "server-only";
import { cookies } from "next/headers";
import type { Me } from "./api";

const backend = process.env.BACKEND_URL ?? "http://127.0.0.1:8080";

export type SessionResult =
  | { kind: "ok"; me: Me }
  | { kind: "anonymous" }
  | { kind: "unavailable" };

/** Resolves the session by forwarding the browser's cookie to the API. */
export async function getSession(): Promise<SessionResult> {
  const jar = await cookies();
  const token = jar.get("st_session");
  if (!token) return { kind: "anonymous" };
  try {
    const res = await fetch(`${backend}/api/auth/me`, {
      headers: { cookie: `st_session=${token.value}` },
      cache: "no-store",
    });
    if (res.status === 401) return { kind: "anonymous" };
    if (!res.ok) return { kind: "unavailable" };
    return { kind: "ok", me: (await res.json()) as Me };
  } catch {
    return { kind: "unavailable" };
  }
}
