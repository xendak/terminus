import { expect, type APIRequestContext, type Page } from "@playwright/test";

export const PASSWORD = "stoptime-dev";
export const users = {
  admin: "admin@stoptime.dev",
  manager: "manager@stoptime.dev",
  driverA: "driver-a@stoptime.dev",
  driverC: "driver-c@stoptime.dev",
};

/** Signs the page's browser context in through the API (cookie is shared). */
export async function signIn(page: Page, email: string) {
  const res = await page.request.post("/api/auth/login", { data: { email, password: PASSWORD } });
  expect(res.ok()).toBeTruthy();
}

/**
 * A far-future date no seed or earlier run uses, so RN05 (one route per
 * driver per date) never collides and dashboards for real dates stay clean.
 */
export function uniqueFutureDate(offset = 0): string {
  const base = Date.UTC(2040, 0, 1);
  const day = Math.floor(Date.now() / 1000) % 15000;
  return new Date(base + (day + offset) * 86_400_000).toISOString().slice(0, 10);
}

interface Json {
  [k: string]: unknown;
}

async function json<T>(res: { ok(): boolean; status(): number; json(): Promise<unknown> }): Promise<T> {
  expect(res.ok(), `HTTP ${res.status()}`).toBeTruthy();
  return (await res.json()) as T;
}

/** Creates a draft route as the manager; returns its id. */
export async function createRoute(request: APIRequestContext, driverEmail: string, date: string, stops = 2) {
  await json(await request.post("/api/auth/login", { data: { email: users.manager, password: PASSWORD } }));
  const { drivers } = await json<{ drivers: (Json & { id: string; email: string })[] }>(await request.get("/api/drivers"));
  const driver = drivers.find((d) => d.email === driverEmail);
  expect(driver, `driver ${driverEmail}`).toBeTruthy();
  const { locations } = await json<{ locations: { id: string }[] }>(await request.get("/api/locations"));
  expect(locations.length).toBeGreaterThanOrEqual(stops);
  const { route } = await json<{ route: { id: string } }>(
    await request.post("/api/routes", {
      data: { driver_user_id: driver!.id, route_date: date, location_ids: locations.slice(0, stops).map((l) => l.id) },
    }),
  );
  await request.post("/api/auth/logout");
  return route.id;
}
