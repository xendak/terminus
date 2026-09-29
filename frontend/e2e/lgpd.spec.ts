import { expect, test } from "@playwright/test";
import { PASSWORD, signIn, users } from "./helpers";

test("managers see the CPF masked; admins see it in full", async ({ page }) => {
  await signIn(page, users.manager);
  await page.goto("/motoristas");
  const managerRow = page.getByRole("row").filter({ hasText: "Bianca Batista" });
  await expect(managerRow).toContainText(/CPF \*{3}\.\*{3}\.\*{3}-\d{2}/);

  await page.getByRole("button", { name: "Editar Bianca Batista" }).click();
  await expect(page.getByLabel("Novo CPF (opcional)")).toHaveValue("");
  await expect(page.getByText("protegido, visível só para administradores")).toBeVisible();
  await expect(page.getByRole("button", { name: "Anonimizar (LGPD)" })).toHaveCount(0);

  await page.context().clearCookies();
  await signIn(page, users.admin);
  await page.goto("/motoristas");
  await expect(page.getByRole("row").filter({ hasText: "Bianca Batista" })).toContainText(/CPF \d{3}\.\d{3}\.\d{3}-\d{2}/);
});

test("admin anonymizes a driver: data removed, row inactive, login blocked", async ({ page, request }) => {
  const stamp = `${Date.now()}-${Math.floor(Math.random() * 1e6)}`;
  const email = `lgpd-${stamp}@teste.dev`;
  const name = `Teste LGPD ${stamp}`;
  await request.post("/api/auth/login", { data: { email: users.admin, password: PASSWORD } });
  const created = await request.post("/api/drivers", {
    data: { name, email, password: "senha-teste-123", phone: "31 90000-0000", document: "111.222.333-44", vehicle_name: "Kombi" },
  });
  expect(created.status()).toBe(201);
  await request.post("/api/auth/logout");

  await signIn(page, users.admin);
  await page.goto("/motoristas");
  await page.getByRole("button", { name: `Editar ${name}` }).click();
  await page.getByRole("button", { name: "Anonimizar (LGPD)" }).click();

  const dialog = page.getByRole("dialog", { name: `Anonimizar ${name}?` });
  await expect(dialog).toContainText("Não tem como desfazer.");
  const confirm = dialog.getByRole("button", { name: "Anonimizar definitivamente" });
  await expect(confirm).toBeDisabled();
  await dialog.getByLabel("Entendo que os dados pessoais serão apagados de forma definitiva.").check();
  await confirm.click();

  await expect(page.getByText(`${name}: dados pessoais removidos.`)).toBeVisible();
  await expect(page.getByRole("row").filter({ hasText: name })).toHaveCount(0);
  const removed = page.getByRole("row").filter({ hasText: "Dados pessoais removidos (LGPD)" }).filter({ hasText: "Motorista removido" });
  await expect(removed.first()).toContainText("Inativo");

  const login = await page.request.post("/api/auth/login", { data: { email, password: "senha-teste-123" } });
  expect(login.status()).toBe(401);

  await page.goto("/auditoria?entidade=app_user");
  const entry = page.getByRole("row").filter({ hasText: "Anonimização (LGPD)" }).first();
  await expect(entry).toContainText("Dados apagados: Nome, E-mail, Telefone, Senha, CPF, Veículo, Placa");
});

test("admin edits, deactivates and anonymizes a manager", async ({ page, request }) => {
  const stamp = `${Date.now()}-${Math.floor(Math.random() * 1e6)}`;
  const email = `gerente-${stamp}@teste.dev`;
  const name = `Gerente Teste ${stamp}`;
  await request.post("/api/auth/login", { data: { email: users.admin, password: PASSWORD } });
  const created = await request.post("/api/managers", {
    data: { name, email, password: "senha-teste-123", phone: "31 90000-1111" },
  });
  expect(created.status()).toBe(201);
  await request.post("/api/auth/logout");

  await signIn(page, users.admin);
  await page.goto("/gerentes");
  await page.getByRole("button", { name: `Editar ${name}` }).click();
  await page.getByLabel("Telefone", { exact: true }).fill("31 98888-2222");
  await page.getByRole("button", { name: "Salvar alterações" }).click();
  await expect(page.getByText(`${name}: dados salvos.`)).toBeVisible();
  await expect(page.getByRole("row").filter({ hasText: name })).toContainText("31 98888-2222");

  await page.getByRole("button", { name: `Editar ${name}` }).click();
  await page.getByRole("button", { name: "Desativar gerente" }).click();
  await page.getByRole("button", { name: "Confirmar desativação" }).click();
  await expect(page.getByRole("row").filter({ hasText: name })).toContainText("Inativo");
  const blocked = await page.request.post("/api/auth/login", { data: { email, password: "senha-teste-123" } });
  expect(blocked.ok()).toBeFalsy();

  await page.getByRole("button", { name: `Editar ${name}` }).click();
  await page.getByRole("button", { name: "Anonimizar (LGPD)" }).click();
  const dialog = page.getByRole("dialog", { name: `Anonimizar ${name}?` });
  await dialog.getByLabel("Entendo que os dados pessoais serão apagados de forma definitiva.").check();
  await dialog.getByRole("button", { name: "Anonimizar definitivamente" }).click();
  await expect(page.getByText(`${name}: dados pessoais removidos.`)).toBeVisible();
  await expect(page.getByRole("row").filter({ hasText: name })).toHaveCount(0);
});
