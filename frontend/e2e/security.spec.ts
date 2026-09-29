import { expect, test } from "@playwright/test";
import { PASSWORD, users } from "./helpers";

// The dev server serves its JS only to allowed origins; 127.0.0.1 must be one.
const ipBase = (process.env.BASE_URL ?? "http://localhost:3210").replace("localhost", "127.0.0.1");

test("login works when the app is opened by IP (127.0.0.1)", async ({ browser }) => {
  const context = await browser.newContext({ baseURL: ipBase });
  const page = await context.newPage();
  await page.goto("/login");
  await page.getByLabel("E-mail", { exact: true }).fill(users.manager);
  await page.getByLabel("Senha", { exact: true }).fill(PASSWORD);
  await page.getByRole("button", { name: "Entrar" }).click();
  await page.waitForURL("**/painel**");
  expect(page.url()).not.toContain(PASSWORD);
  await context.close();
});

test("without JavaScript, the login form never puts credentials in the URL", async ({ browser, baseURL }) => {
  const context = await browser.newContext({ baseURL, javaScriptEnabled: false });
  const page = await context.newPage();
  await page.goto("/login");
  await page.getByLabel("E-mail", { exact: true }).fill(users.manager);
  await page.getByLabel("Senha", { exact: true }).fill(PASSWORD);
  const [request] = await Promise.all([
    page.waitForRequest((r) => r.url().includes("/sem-js")),
    page.getByRole("button", { name: "Entrar" }).click(),
  ]);
  expect(request.method()).toBe("POST");
  await expect(page.getByRole("heading", { name: "Nada foi enviado" })).toBeVisible();
  expect(page.url()).not.toContain(PASSWORD);
  expect(page.url()).not.toContain("password");
  expect(page.url()).not.toContain("email");
  await context.close();
});
