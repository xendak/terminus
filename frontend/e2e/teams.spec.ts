import { expect, test } from "@playwright/test";
import { PASSWORD, signIn, users } from "./helpers";

const GUSTAVO = "aa000000-0000-4000-8000-000000000002"; // db/seed/golden.sql manager

test("history and dashboard filter by team, carried in the URL", async ({ page }) => {
  await signIn(page, users.manager);
  await page.goto(`/historico?from=2026-06-15&to=2026-06-15&manager_user_id=${GUSTAVO}`);
  await expect(page.getByLabel("Equipe", { exact: true })).toHaveValue(GUSTAVO);
  await expect(page.getByRole("row").filter({ hasText: "15/06/2026" })).toHaveCount(3);
  await expect(page.getByRole("link", { name: "Exportar CSV" })).toHaveAttribute("href", new RegExp(`manager_user_id=${GUSTAVO}`));

  await page.goto("/painel?from=2026-06-15&to=2026-06-15&aba=periodo");
  await page.getByLabel("Equipe", { exact: true }).selectOption(GUSTAVO);
  await expect(page).toHaveURL(new RegExp(`manager_user_id=${GUSTAVO}`));
  await expect(page.getByText("Total em 15/06/2026 · 3 roteiros")).toBeVisible();
});

test("admin assigns a driver to a new manager's team", async ({ page, request }) => {
  const stamp = `${Date.now()}-${Math.floor(Math.random() * 1e6)}`;
  const managerName = `Gestora Equipe ${stamp}`;
  const driverName = `Motorista Equipe ${stamp}`;
  await request.post("/api/auth/login", { data: { email: users.admin, password: PASSWORD } });
  const m = await request.post("/api/managers", {
    data: { name: managerName, email: `equipe-${stamp}@teste.dev`, password: "senha-teste-123", phone: "31 90000-3333" },
  });
  expect(m.status()).toBe(201);
  const managerId = ((await m.json()) as { manager: { id: string } }).manager.id;
  await request.post("/api/auth/logout");

  await signIn(page, users.admin);
  await page.goto("/motoristas");
  await page.getByRole("button", { name: "Novo motorista" }).click();
  await page.getByLabel("Nome", { exact: true }).fill(driverName);
  await page.getByLabel("E-mail", { exact: true }).fill(`motorista-${stamp}@teste.dev`);
  await page.getByLabel("Senha inicial").fill("senha-teste-123");
  await page.getByLabel("Telefone", { exact: true }).fill("31 90000-4444");
  await expect(page.getByLabel("Gestor responsável")).toBeEnabled();
  await page.getByLabel("Gestor responsável").selectOption({ label: managerName });
  await page.getByRole("button", { name: "Cadastrar motorista" }).click();
  await expect(page.getByRole("row").filter({ hasText: driverName })).toContainText(managerName);

  await page.goto("/gerentes");
  await expect(page.getByRole("row").filter({ hasText: managerName })).toContainText("1 motorista");

  // Clearing the team sends null.
  await page.goto("/motoristas");
  await page.getByRole("button", { name: `Editar ${driverName}` }).click();
  await page.getByLabel("Gestor responsável").selectOption({ label: "Sem gestor" });
  await page.getByRole("button", { name: "Salvar alterações" }).click();
  await expect(page.getByRole("row").filter({ hasText: driverName })).toContainText("sem gestor");
  expect(managerId).toBeTruthy();
});
