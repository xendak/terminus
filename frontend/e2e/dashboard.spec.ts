import { expect, type Page, test } from "@playwright/test";
import { signIn, users } from "./helpers";

// tp.md §5 golden fixture: three closed routes on 15/06/2026 (A 75, C 45, B 41 min).
const GOLDEN_DAY = "/painel?from=2026-06-15&to=2026-06-15";

const card = (page: Page, heading: string) =>
  page.locator("section").filter({ has: page.getByRole("heading", { name: heading, exact: true }) });

test.beforeEach(async ({ page }) => {
  await signIn(page, users.manager);
});

test("KPIs and the driver ranking show the golden day's server totals", async ({ page }) => {
  await page.goto(GOLDEN_DAY);
  await expect(page.getByText("Tempo parado total")).toBeVisible();
  await expect(page.getByText("2 h 41 min").first()).toBeVisible();
  // Server-computed mean per route (avg_stopped_minutes_per_route), golden 53.
  const avg = page.getByText("Média por roteiro").locator("..");
  await expect(avg).toContainText("53 min");
  await expect(avg).toContainText("3 roteiros");
  const ranking = card(page, "Por motorista").getByRole("listitem").filter({ hasText: "Marcos Motorista" });
  await expect(ranking).toContainText("1 h 15 min");
  await expect(ranking).toContainText("15,6%");
});

test("single day: timeline has one row per driver and the points list ranks stops", async ({ page }) => {
  await page.goto(GOLDEN_DAY);
  const timeline = card(page, "Linha do tempo por motorista");
  for (const name of ["Marcos Motorista", "Carla Camargo", "Bianca Batista"]) {
    await expect(timeline.getByRole("link", { name })).toBeVisible();
  }
  // Route A's third stop: 11:00 → 11:50, 50 min, over the 45 min mark.
  await expect(timeline).toContainText("Ponto A4: 11:00–11:50, 50 min");

  const points = card(page, "Pontos do dia").getByRole("listitem");
  await expect(points).toHaveCount(9); // 3 counted stops × 3 routes
  await expect(points.first()).toContainText("Ponto A4");
  await expect(points.first()).toContainText("50 min");
  await expect(points.first()).toContainText("Acima do limite");

  await timeline.getByRole("combobox").selectOption({ label: "Bianca Batista" });
  await expect(points).toHaveCount(3);
  await expect(timeline.getByRole("link", { name: "Marcos Motorista" })).toHaveCount(0);
});

test("a month range charts by day and lists each day with the server percent", async ({ page }) => {
  await page.goto("/painel?from=2026-06-01&to=2026-06-30");
  await expect(page.getByRole("heading", { name: "Tempo parado por dia" })).toBeVisible();
  const row = page.getByRole("row").filter({ hasText: "15/06/2026" });
  await expect(row).toContainText("2 h 41 min");
  await expect(row).toContainText("11,2%");
});

test("an empty window shows the empty state", async ({ page }) => {
  await page.goto("/painel?from=2039-01-01&to=2039-01-02");
  await expect(page.getByText("Nenhuma parada registrada neste período")).toBeVisible();
});

test("drill-down: a day row opens history for that day, down to route A's addresses", async ({ page }) => {
  await page.goto("/painel?from=2026-06-01&to=2026-06-30");
  await page.getByRole("link", { name: "Ver roteiros de 15/06/2026" }).click();
  await expect(page).toHaveURL(/\/historico\?from=2026-06-15&to=2026-06-15$/);
  await expect(page.getByLabel("De", { exact: true })).toHaveValue("2026-06-15");
  const rows = page.getByRole("row").filter({ hasText: "15/06/2026" });
  await expect(rows).toHaveCount(3);
  await page.getByRole("link", { name: "Abrir roteiro de Marcos Motorista em 15/06/2026" }).click();
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("Roteiro de Marcos Motorista");
  await expect(page.getByText("Rua Peru, 55")).toBeVisible();
});

test("drill-down: clicking a day bar and a ranking row pre-filter history", async ({ page }) => {
  await page.goto("/painel?from=2026-06-01&to=2026-06-30");
  await page.getByRole("link", { name: /^15\/06\/2026: 2 h 41 min parados.*Ver roteiros$/ }).click();
  await expect(page).toHaveURL(/\/historico\?from=2026-06-15&to=2026-06-15$/);

  await page.goto(GOLDEN_DAY);
  await page.getByRole("link", { name: "Ver roteiros de Bianca Batista no período" }).click();
  // Seed id of driver B (db/seed/golden.sql).
  await expect(page).toHaveURL(/driver_user_id=aa000000-0000-4000-8000-000000000004/);
  await expect(page.getByLabel("Motorista", { exact: true })).toHaveValue(/.+/);
  const rows = page.getByRole("row").filter({ hasText: "15/06/2026" });
  await expect(rows).toHaveCount(1);
  await expect(rows).toContainText("Bianca Batista");
});

test("a points-list row opens the route it belongs to", async ({ page }) => {
  await page.goto(GOLDEN_DAY);
  await card(page, "Pontos do dia").getByRole("link", { name: /^Ponto A4, Marcos Motorista/ }).click();
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("Roteiro de Marcos Motorista");
});

test("drill-down: a driver's own ranking row links to their history", async ({ page }) => {
  await page.context().clearCookies();
  await signIn(page, users.driverA);
  await page.goto(GOLDEN_DAY);
  await page.getByRole("link", { name: "Ver roteiros de Marcos Motorista no período" }).click();
  await expect(page).toHaveURL(/\/historico\?from=2026-06-15&to=2026-06-15&driver_user_id=aa000000-0000-4000-8000-000000000003$/);
  await expect(page.getByRole("link", { name: /Abrir roteiro de Marcos Motorista em 15\/06\/2026/ })).toHaveCount(1);
});

test("the driver's dashboard loads without staff-only calls failing it", async ({ page }) => {
  await page.context().clearCookies();
  await signIn(page, users.driverA);
  const forbidden: string[] = [];
  page.on("response", (r) => r.status() === 403 && forbidden.push(r.url()));
  await page.goto(GOLDEN_DAY);
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("Meu tempo parado");
  await expect(page.getByText("1 h 15 min").first()).toBeVisible();
  const timeline = card(page, "Linha do tempo por motorista");
  await expect(timeline.getByRole("link", { name: "Marcos Motorista" })).toBeVisible();
  await expect(timeline.getByRole("link", { name: "Carla Camargo" })).toHaveCount(0);
  await expect(page.getByText("Não foi possível")).toHaveCount(0);
  await expect(page.getByLabel("Equipe", { exact: true })).toHaveCount(0);
  expect(forbidden).toEqual([]);
});
