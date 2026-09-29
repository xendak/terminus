"use client";

import { useCallback, useEffect, useRef, useState } from "react";

export type Loadable<T> =
  | { state: "loading"; data?: T }
  | { state: "ready"; data: T }
  | { state: "error"; error: unknown; data?: T };

interface Settled<T> {
  key: string;
  tick: number;
  data?: T;
  error?: unknown;
  failed: boolean;
}

/**
 * Minimal fetch-on-key hook: re-runs when `key` changes (null = skip),
 * drops stale answers, keeps the previous data while reloading, and
 * exposes reload() for retry buttons and post-mutation refreshes.
 */
export function useApi<T>(key: string | null, load: () => Promise<T>) {
  const [tick, setTick] = useState(0);
  const [settled, setSettled] = useState<Settled<T> | null>(null);
  const loadRef = useRef(load);

  useEffect(() => {
    loadRef.current = load;
  });

  useEffect(() => {
    if (key === null) return;
    let live = true;
    loadRef.current().then(
      (data) => live && setSettled({ key, tick, data, failed: false }),
      (error: unknown) => live && setSettled((prev) => ({ key, tick, data: prev?.data, error, failed: true })),
    );
    return () => {
      live = false;
    };
  }, [key, tick]);

  const reload = useCallback(() => setTick((t) => t + 1), []);
  const set = useCallback(
    (data: T) => setSettled((prev) => ({ key: prev?.key ?? "", tick: prev?.tick ?? 0, data, failed: false })),
    [],
  );

  const current = settled !== null && settled.key === key && settled.tick === tick;
  let result: Loadable<T>;
  if (!current) result = { state: "loading", data: settled?.data };
  else if (settled.failed) result = { state: "error", error: settled.error, data: settled.data };
  else result = { state: "ready", data: settled.data as T };

  return { ...result, reload, set };
}
