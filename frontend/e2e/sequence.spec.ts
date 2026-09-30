import { expect, test } from "@playwright/test";
import { createRoute, signIn, users } from "./helpers";

test("a manual arrival before the previous departure is rejected inline (RN06)", async ({ page, request }) => {
  const id = await createRoute(request, users.driverC, 3);
  await signIn(page, users.driverC);
  await page.goto(`/roteiros/${id}`);
  await page.getByRole("button", { name: "Iniciar roteiro" }).click();
  await page.getByRole("button", { name: "Registrar saída da base" }).click();
  const current = page.locator('li[aria-current="step"]');
  await expect(current).toContainText("Ponto 2");

  await page.getByRole("button", { name: "Informar chegada manualmente" }).click();
  await page.getByLabel("Horário da chegada").fill("2020-01-01T08:00");
  await page.getByRole("button", { name: "Registrar chegada" }).click();
  await expect(page.getByText("Horário fora de ordem: a chegada não pode ser antes da saída do ponto anterior.")).toBeVisible();
  await expect(page.getByRole("timer")).toHaveCount(0);

  // The normal flow still works right after.
  await page.getByRole("button", { name: "Cancelar horário manual" }).click();
  await page.getByRole("button", { name: "Cheguei aqui" }).click();
  await expect(page.getByRole("timer")).toBeVisible();
});

test("the driver may skip the base departure and start at stop 2", async ({ page, request }) => {
  const id = await createRoute(request, users.driverC, 3);
  await signIn(page, users.driverC);
  await page.goto(`/roteiros/${id}`);
  await page.getByRole("button", { name: "Iniciar roteiro" }).click();
  // The start is async: wait until the route is active before using the API.
  await expect(page.getByRole("button", { name: "Registrar saída da base" })).toBeVisible();
  // Arrive at stop 2 through the API without leaving the base.
  const res = await page.request.post(`/api/routes/${id}/stops/2/arrive`);
  expect(res.ok()).toBeTruthy();
  await page.reload();
  const current = page.locator('li[aria-current="step"]');
  await expect(current).toContainText("Ponto 2");
  await expect(page.getByRole("timer")).toBeVisible();
});
