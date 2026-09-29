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

interface Json {
  [k: string]: unknown;
}

async function json<T>(res: { ok(): boolean; status(): number; json(): Promise<unknown> }): Promise<T> {
  expect(res.ok(), `HTTP ${res.status()}`).toBeTruthy();
  return (await res.json()) as T;
}

function randomFarFutureDate(): string {
  // 2040-01-01 plus up to ~60 years: far from seed data and real dashboards.
  const day = Math.floor(Math.random() * 21_900);
  return new Date(Date.UTC(2040, 0, 1) + day * 86_400_000).toISOString().slice(0, 10);
}

async function managerDriverId(request: APIRequestContext, driverEmail: string): Promise<string> {
  await json(await request.post("/api/auth/login", { data: { email: users.manager, password: PASSWORD } }));
  const { drivers } = await json<{ drivers: (Json & { id: string; email: string })[] }>(await request.get("/api/drivers"));
  const driver = drivers.find((d) => d.email === driverEmail);
  expect(driver, `driver ${driverEmail}`).toBeTruthy();
  return driver!.id;
}

/**
 * A random far-future date on which the driver has no route yet (RN05),
 * checked against the API so repeated runs on a dev DB never collide.
 */
export async function freeDate(request: APIRequestContext, driverEmail: string): Promise<string> {
  const driverId = await managerDriverId(request, driverEmail);
  for (let i = 0; i < 20; i++) {
    const date = randomFarFutureDate();
    const { routes } = await json<{ routes: unknown[] | null }>(
      await request.get("/api/routes", { params: { from: date, to: date, driver_user_id: driverId } }),
    );
    if (!routes || routes.length === 0) {
      await request.post("/api/auth/logout");
      return date;
    }
  }
  throw new Error("no free date found");
}

/** Creates a draft route as the manager on a free far-future date; returns its id. */
export async function createRoute(request: APIRequestContext, driverEmail: string, stops = 2) {
  const driverId = await managerDriverId(request, driverEmail);
  const { locations } = await json<{ locations: { id: string }[] }>(await request.get("/api/locations"));
  expect(locations.length).toBeGreaterThanOrEqual(stops);
  for (let attempt = 0; attempt < 5; attempt++) {
    const res = await request.post("/api/routes", {
      data: {
        driver_user_id: driverId,
        route_date: randomFarFutureDate(),
        location_ids: locations.slice(0, stops).map((l) => l.id),
      },
    });
    if (res.status() === 409) continue; // RN05: that date is taken, draw another
    const { route } = await json<{ route: { id: string } }>(res);
    await request.post("/api/auth/logout");
    return route.id;
  }
  throw new Error("could not create a route on a free date");
}
