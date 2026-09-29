import { expect, test } from "@playwright/test";
import { createRoute, signIn, users } from "./helpers";

test("driver starts, records arrival and departure, and closes a fresh route", async ({ page, request }) => {
  const id = await createRoute(request, users.driverC, 2);
  await signIn(page, users.driverC);
  await page.goto(`/roteiros/${id}`);

  await page.getByRole("button", { name: "Iniciar roteiro" }).click();
  await expect(page.getByText("Em andamento").first()).toBeVisible();

  // RN01: the departure point has no stopwatch.
  const current = page.locator('li[aria-current="step"]');
  await expect(current).toContainText("Partida");
  await expect(page.getByRole("timer")).toHaveCount(0);
  await page.getByRole("button", { name: "Registrar saída da base" }).click();

  await expect(current).toContainText("Parada 1");
  await page.getByRole("button", { name: "Cheguei aqui" }).click();
  await expect(page.getByRole("timer")).toBeVisible();
  await expect(page.getByRole("timer")).toHaveText(/^00:00:0\d$/);
  await page.getByRole("button", { name: "Saí deste ponto" }).click();

  await expect(page.getByRole("heading", { name: "Todos os pontos concluídos" })).toBeVisible();
  await page.getByLabel("Distância percorrida (km)").fill("12,5");
  await page.getByRole("button", { name: "Encerrar roteiro" }).click();

  await expect(page.getByText("Tempo parado no dia")).toBeVisible();
  await expect(page.getByText("Encerrado", { exact: true })).toBeVisible();
  await expect(page.getByText("12,5 km rodados")).toBeVisible();
});

test("manual arrival entry requires a date and time", async ({ page, request }) => {
  const id = await createRoute(request, users.driverC, 2);
  await signIn(page, users.driverC);
  await page.goto(`/roteiros/${id}`);
  await page.getByRole("button", { name: "Iniciar roteiro" }).click();
  await page.getByRole("button", { name: "Registrar saída da base" }).click();
  await page.getByRole("button", { name: "Informar chegada manualmente" }).click();
  await page.getByLabel("Horário da chegada").fill("");
  await page.getByRole("button", { name: "Registrar chegada" }).click();
  await expect(page.getByText("Informe data e hora.")).toBeVisible();
});
