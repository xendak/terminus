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
