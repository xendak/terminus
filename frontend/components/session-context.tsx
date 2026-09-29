"use client";

import { createContext, useContext, type ReactNode } from "react";
import type { User } from "@/lib/api";

const SessionContext = createContext<User | null>(null);

export function SessionProvider({ user, children }: { user: User; children: ReactNode }) {
  return <SessionContext.Provider value={user}>{children}</SessionContext.Provider>;
}

/** The signed-in user; only valid under the (app) layout. */
export function useUser(): User {
  const u = useContext(SessionContext);
  if (!u) throw new Error("useUser outside SessionProvider");
  return u;
}
