import { defineConfig, devices } from "@playwright/test";

// E2E against the real stack: this Next app on :3210 proxying to the Go
// API (BACKEND_URL, default http://127.0.0.1:8080) with the dev seed.
const port = 3210;

export default defineConfig({
  testDir: "./e2e",
  testMatch: /.*\.spec\.ts/,
  timeout: 45_000,
  expect: { timeout: 10_000 },
  fullyParallel: false,
  workers: 1,
  reporter: [["list"]],
  use: {
    baseURL: process.env.BASE_URL ?? `http://localhost:${port}`,
    locale: "pt-BR",
    timezoneId: "America/Sao_Paulo",
    trace: "retain-on-failure",
  },
  projects: [
    { name: "desktop", use: { ...devices["Desktop Chrome"] } },
    { name: "phone", use: { ...devices["Pixel 7"], viewport: { width: 375, height: 812 } }, testMatch: /tracker\.spec\.ts/ },
  ],
  webServer: {
    command: `pnpm dev`,
    url: `http://localhost:${port}/login`,
    reuseExistingServer: true,
    timeout: 120_000,
  },
});
