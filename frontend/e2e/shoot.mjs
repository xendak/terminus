// Dev helper: log in as a role and screenshot pages.
// node e2e/shoot.mjs <role|anon> <outPrefix> <width> <path> [path...]
import { chromium } from "@playwright/test";

const [role, prefix, width, ...paths] = process.argv.slice(2);
const base = process.env.BASE_URL ?? "http://localhost:3210";
const out = process.env.SHOTS ?? "/tmp/claude-1000/terminus-shots";
const scheme = process.env.SCHEME ?? "light";
const browser = await chromium.launch({ args: ["--lang=pt-BR"] });
const ctx = await browser.newContext({
  viewport: { width: Number(width), height: Number(width) < 600 ? 812 : 900 },
  colorScheme: scheme,
  deviceScaleFactor: 1,
  locale: "pt-BR",
  timezoneId: "America/Sao_Paulo",
});
const page = await ctx.newPage();
page.on("console", (m) => m.type() === "error" && console.log("console:", m.text()));
page.on("pageerror", (e) => console.log("pageerror:", e.message));
if (role !== "anon") {
  const email = role === "driver" ? "driver-a@stoptime.dev" : `${role}@stoptime.dev`;
  const res = await page.request.post(`${base}/api/auth/login`, { data: { email, password: "stoptime-dev" } });
  if (!res.ok()) throw new Error(`login ${res.status()}`);
}
let i = 0;
for (const p of paths) {
  await page.goto(base + p, { waitUntil: "networkidle" });
  await page.waitForTimeout(400);
  const file = `${out}/${prefix}-${i++}.png`;
  await page.screenshot({ path: file, fullPage: true });
  console.log(file, page.url());
}
await browser.close();
