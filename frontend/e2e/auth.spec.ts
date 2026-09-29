import { expect, test } from "@playwright/test";
import { PASSWORD, users } from "./helpers";

async function loginViaForm(page: import("@playwright/test").Page, email: string, password = PASSWORD) {
  await page.goto("/login");
  await page.getByLabel("E-mail").fill(email);
  await page.getByLabel("Senha").fill(password);
  await page.getByRole("button", { name: "Entrar" }).click();
}

test("anonymous visitors are sent to login", async ({ page }) => {
  await page.goto("/painel");
  await expect(page).toHaveURL(/\/login\?next=%2Fpainel/);
});

test("wrong password shows an inline error", async ({ page }) => {
  await loginViaForm(page, users.manager, "senha-errada");
  await expect(page.getByRole("alert").filter({ hasText: "E-mail ou senha incorretos" })).toBeVisible();
});

test("driver lands on today's tracker with driver navigation", async ({ page }) => {
  await loginViaForm(page, users.driverA);
  await expect(page).toHaveURL(/\/hoje$/);
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("Meu roteiro de hoje");
  const nav = page.getByRole("navigation", { name: "Principal" });
  await expect(nav.getByRole("link", { name: "Histórico" })).toBeVisible();
  await expect(nav.getByRole("link", { name: "Motoristas" })).toHaveCount(0);
  await expect(nav.getByRole("link", { name: "Parâmetros" })).toHaveCount(0);
});

test("manager lands on the dashboard without admin-only items", async ({ page }) => {
  await loginViaForm(page, users.manager);
  await expect(page).toHaveURL(/\/painel/);
  const nav = page.getByRole("navigation", { name: "Principal" });
  await expect(nav.getByRole("link", { name: "Novo roteiro" })).toBeVisible();
  await expect(nav.getByRole("link", { name: "Auditoria" })).toHaveCount(0);
  await expect(nav.getByRole("link", { name: "Gerentes" })).toHaveCount(0);
});

test("admin sees audit and managers, and can log out", async ({ page }) => {
  await loginViaForm(page, users.admin);
  await expect(page).toHaveURL(/\/painel/);
  const nav = page.getByRole("navigation", { name: "Principal" });
  await expect(nav.getByRole("link", { name: "Auditoria" })).toBeVisible();
  await expect(nav.getByRole("link", { name: "Gerentes" })).toBeVisible();
  await page.getByRole("button", { name: "Sair" }).click();
  await expect(page).toHaveURL(/\/login/);
});

test("a foreign ?next= is ignored after login", async ({ browser, baseURL }) => {
  for (const next of ["/%5Cevil.com", "//evil.com", "/%09/evil.com"]) {
    const context = await browser.newContext({ baseURL });
    const page = await context.newPage();
    await page.goto(`/login?next=${next}`);
    await page.getByLabel("E-mail").fill(users.manager);
    await page.getByLabel("Senha").fill(PASSWORD);
    await page.getByRole("button", { name: "Entrar" }).click();
    await page.waitForURL("**/painel**");
    expect(new URL(page.url()).origin).toBe(new URL(baseURL!).origin);
    await context.close();
  }
});

test("a same-origin ?next= is honoured after login", async ({ page }) => {
  await page.goto("/login?next=%2Fhistorico%3Fstatus%3Dclosed");
  await page.getByLabel("E-mail").fill(users.manager);
  await page.getByLabel("Senha").fill(PASSWORD);
  await page.getByRole("button", { name: "Entrar" }).click();
  await expect(page).toHaveURL(/\/historico\?status=closed$/);
});
