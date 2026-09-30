import { expect, test } from "@playwright/test";
import { PASSWORD, signIn, users } from "./helpers";

test("editing a location's address lands in the audit as old -> new", async ({ page, request }) => {
  const stamp = `${Date.now()}-${Math.floor(Math.random() * 1e6)}`;
  const label = `Ponto Auditoria ${stamp}`;
  const oldAddress = `Rua Antiga, ${stamp}`;
  const newAddress = `Rua Nova, ${stamp}`;
  await request.post("/api/auth/login", { data: { email: users.manager, password: PASSWORD } });
  const created = await request.post("/api/locations", { data: { label, address: oldAddress } });
  expect(created.status()).toBe(201);
  await request.post("/api/auth/logout");

  await signIn(page, users.manager);
  await page.goto("/pontos");
  await page.getByLabel("Buscar", { exact: true }).fill(stamp);
  await page.getByRole("row").filter({ hasText: label }).getByRole("button", { name: "Editar" }).click();
  await page.getByLabel("Endereço", { exact: true }).fill(newAddress);
  await page.getByRole("button", { name: "Salvar alterações" }).click();
  await expect(page.getByText(`Ponto “${label}” atualizado.`)).toBeVisible();

  await page.context().clearCookies();
  await signIn(page, users.admin);
  await page.goto("/auditoria?entidade=location");
  const entry = page.getByRole("row").filter({ hasText: "Ponto alterado" }).filter({ hasText: newAddress });
  await expect(entry).toHaveCount(1);
  await expect(entry).toContainText(`Endereço: ${oldAddress}→${newAddress}`);
  await expect(entry).not.toContainText("Nome do ponto");
});

test("the stop threshold parameters are editable and validated in pt-BR", async ({ page }) => {
  await signIn(page, users.admin);
  await page.goto("/parametros");
  const warn = page.getByLabel("Parada longa a partir de (min)");
  const alert = page.getByLabel("Parada crítica a partir de (min)");
  await expect(warn).toHaveValue("15");
  await expect(alert).toHaveValue("45");

  await warn.fill("50");
  await page.locator("form").filter({ has: warn }).getByRole("button", { name: "Salvar" }).click();
  await expect(page.getByText("A parada longa precisa ser menor ou igual à parada crítica.")).toBeVisible();

  await alert.fill("10");
  await page.locator("form").filter({ has: alert }).getByRole("button", { name: "Salvar" }).click();
  await expect(page.getByText("A parada crítica precisa ser maior ou igual à parada longa.")).toBeVisible();
  await alert.fill("45");

  await warn.fill("16");
  await page.locator("form").filter({ has: warn }).getByRole("button", { name: "Salvar" }).click();
  await expect(page.locator("form").filter({ has: warn }).getByText("Salvo.", { exact: false })).toBeVisible();
  await warn.fill("15");
  await page.locator("form").filter({ has: warn }).getByRole("button", { name: "Salvar" }).click();
  await expect(warn).toHaveValue("15");
  await expect(page.locator("form").filter({ has: warn }).getByText("Salvo.", { exact: false })).toBeVisible();
});
