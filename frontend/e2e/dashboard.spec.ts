import { expect, test } from "@playwright/test";
import { signIn, users } from "./helpers";

test.beforeEach(async ({ page }) => {
  await signIn(page, users.manager);
});

test("period tab shows the golden June totals and driver ranking", async ({ page }) => {
  // tp.md §5 golden fixture: three closed routes on 15/06/2026 (75 + 45 + 41 min).
  await page.goto("/painel?from=2026-06-15&to=2026-06-15&aba=periodo");
  await expect(page.getByText("Total parado no período")).toBeVisible();
  const ranking = page.getByRole("listitem").filter({ hasText: "Marcos Motorista" });
  await expect(ranking).toContainText("1 h 15 min");
  await expect(ranking).toContainText("15,6%");
});

test("day tab renders a chart and a table view", async ({ page }) => {
  await page.goto("/painel?from=2026-06-01&to=2026-06-30");
  await expect(page.getByRole("tab", { name: "Por dia" })).toHaveAttribute("aria-selected", "true");
  await expect(page.getByRole("img", { name: /minutos parados por dia/i })).toBeVisible();
  await page.getByRole("button", { name: "Ver como tabela" }).click();
  await expect(page.getByRole("row").filter({ hasText: "15/06/2026" })).toContainText("2 h 41 min");
});

test("an empty window shows the empty state", async ({ page }) => {
  await page.goto("/painel?from=2039-01-01&to=2039-01-02");
  await expect(page.getByText("Nenhuma parada registrada neste período")).toBeVisible();
});

test("drill-down: a day row opens history for that day, down to route A's addresses", async ({ page }) => {
  await page.goto("/painel?from=2026-06-01&to=2026-06-30");
  // Wait for the data (not just the shell) before switching to the table.
  await expect(page.getByRole("img", { name: /minutos parados por dia/i })).toBeVisible();
  await page.getByRole("button", { name: "Ver como tabela" }).click();
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
  await page.goto("/painel?from=2026-06-15&to=2026-06-15");
  await page.locator(".recharts-bar-rectangle").first().click();
  await expect(page).toHaveURL(/\/historico\?from=2026-06-15&to=2026-06-15$/);

  await page.goto("/painel?from=2026-06-15&to=2026-06-15&aba=periodo");
  await page.getByRole("link", { name: "Ver roteiros de Bianca Batista no período" }).click();
  // Seed id of driver B (db/seed/golden.sql).
  await expect(page).toHaveURL(/driver_user_id=aa000000-0000-4000-8000-000000000004/);
  await expect(page.getByLabel("Motorista", { exact: true })).toHaveValue(/.+/);
  const rows = page.getByRole("row").filter({ hasText: "15/06/2026" });
  await expect(rows).toHaveCount(1);
  await expect(rows).toContainText("Bianca Batista");
});

test("drill-down: a driver's own ranking row links to their history", async ({ page }) => {
  await page.context().clearCookies();
  await signIn(page, users.driverA);
  await page.goto("/painel?from=2026-06-15&to=2026-06-15&aba=periodo");
  await page.getByRole("link", { name: "Ver roteiros de Marcos Motorista no período" }).click();
  await expect(page).toHaveURL(/\/historico\?from=2026-06-15&to=2026-06-15&driver_user_id=aa000000-0000-4000-8000-000000000003$/);
  await expect(page.getByRole("link", { name: /Abrir roteiro de Marcos Motorista em 15\/06\/2026/ })).toHaveCount(1);
});
