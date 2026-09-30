import { expect, test } from "@playwright/test";
import { freeDate, signIn, users } from "./helpers";

test("manager builds a route and RN05 blocks a second one on the same date", async ({ page, request }) => {
  const date = await freeDate(request, users.driverC);
  await signIn(page, users.manager);
  await page.goto("/roteiros/novo");

  // Invalid state first: nothing chosen.
  await page.getByRole("button", { name: "Criar roteiro" }).click();
  await expect(page.getByText("Escolha o motorista.")).toBeVisible();

  await page.getByLabel("Motorista", { exact: true }).selectOption({ label: "Carla Camargo" });
  await page.getByLabel("Data", { exact: true }).fill(date);
  const results = page.getByRole("list", { name: "Resultados" }).getByRole("button");
  await results.nth(0).click();
  await results.nth(1).click();
  await results.nth(2).click();
  const order = page.getByRole("list", { name: "Pontos na ordem da visita" }).getByRole("listitem");
  await expect(order).toHaveCount(3);
  await expect(order.first()).toContainText("Ponto 1 · partida");

  // Reorder: move the last stop up one position.
  const lastLabel = (await order.nth(2).locator("p").first().textContent())!;
  await order.nth(2).getByRole("button", { name: /^Subir/ }).click();
  await expect(order.nth(1)).toContainText(lastLabel);

  await page.getByRole("button", { name: "Criar roteiro" }).click();
  await expect(page.getByText("Roteiro criado")).toBeVisible();
  await page.getByRole("link", { name: "Abrir roteiro" }).click();
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("Roteiro de Carla Camargo");
  await expect(page.getByText("Rascunho")).toBeVisible();

  // Same driver + date again: the pre-check links to the existing route.
  await page.goto("/roteiros/novo");
  await page.getByLabel("Motorista", { exact: true }).selectOption({ label: "Carla Camargo" });
  await page.getByLabel("Data", { exact: true }).fill(date);
  await expect(page.getByRole("link", { name: "Abrir o roteiro existente" })).toBeVisible();
});
