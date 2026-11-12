// Security audit of 2026-11, point 18: the app still works under the CSP of vercel.json. It serves the built
// app with exactly those headers and fails on any violation or on a page that does not load.
import { test, before, after } from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { createServer } from "node:http";
import { existsSync, readFileSync } from "node:fs";
import { join, dirname, extname } from "node:path";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { answer, pdfBytes } from "./responsive/mock-api.mjs";

const frontend = join(dirname(fileURLToPath(import.meta.url)), "..");
const dist = join(frontend, "dist-csp");
const API = "https://bakery-manager-ofbi.vercel.app";
const PHOTO = "https://hrwbnkikfllpzwuqklrc.supabase.co/storage/v1/object/public/recipe-photos/recipes/x/a.jpg";
const cfg = JSON.parse(readFileSync(join(frontend, "vercel.json"), "utf8"));
const headers = Object.fromEntries(((cfg.headers ?? []).find((h) => h.source === "/(.*)")?.headers ?? []).map((h) => [h.key, h.value]));
const TYPES = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".woff2": "font/woff2", ".woff": "font/woff", ".png": "image/png" };
let server, browser, base;

before(async () => {
  const build = spawnSync("npx", ["vite", "build", "--outDir", dist, "--emptyOutDir"], { cwd: frontend, encoding: "utf8", env: { ...process.env, VITE_API_URL: API } });
  assert.equal(build.status, 0, build.stderr || build.stdout);
  server = createServer((req, res) => {
    const path = join(dist, new URL(req.url, "http://x").pathname);
    const file = existsSync(path) && extname(path) ? path : join(dist, "index.html");
    res.writeHead(200, { ...headers, "content-type": TYPES[extname(file)] ?? "application/octet-stream" });
    res.end(readFileSync(file));
  }).listen(0, "127.0.0.1");
  await new Promise((r) => server.once("listening", r));
  base = `http://127.0.0.1:${server.address().port}`;
  browser = await chromium.launch({ executablePath: process.env.CHROMIUM_PATH || undefined });
});
after(async () => { await browser?.close(); server?.close(); });

test("the app loads and works with the CSP of vercel.json", { timeout: 120000 }, async () => {
  assert.ok(headers["Content-Security-Policy"], "vercel.json has no CSP");
  const context = await browser.newContext({ viewport: { width: 1280, height: 800 } });
  await context.addInitScript(() => localStorage.setItem("token", "test.token.value"));
  await context.route(`${API}/**`, (route) => {
    const req = route.request();
    const cors = { "access-control-allow-origin": base, "access-control-allow-headers": "*", "access-control-allow-methods": "*" };
    if (req.method() === "OPTIONS") return route.fulfill({ status: 204, headers: cors });
    const found = answer(req.method(), new URL(req.url()).pathname);
    if (found?.pdf) return route.fulfill({ status: 200, headers: cors, contentType: "application/pdf", body: pdfBytes });
    return route.fulfill({ status: found ? 200 : 404, headers: cors, contentType: "application/json", body: JSON.stringify(found?.body ?? {}) });
  });
  await context.route(PHOTO, (route) => route.fulfill({ status: 200, contentType: "image/svg+xml", body: "<svg xmlns='http://www.w3.org/2000/svg' width='4' height='4'/>" }));
  const violations = [];
  const page = await context.newPage();
  page.on("console", (m) => { if (/Content Security Policy|Refused to/i.test(m.text())) violations.push(m.text()); });
  page.on("pageerror", (e) => violations.push(`pageerror: ${e.message}`));
  for (const path of ["/dashboard", "/ingredients", "/recipes", "/sales", "/business"]) {
    await page.goto(base + path, { waitUntil: "networkidle" });
    await page.waitForTimeout(300);
    assert.ok((await page.locator("h1").count()) > 0, `${path} did not render`);
  }
  // A photo of the bucket is drawn; a script from anywhere else is not run.
  await page.evaluate((src) => { const i = new Image(); i.id = "probe"; i.src = src; document.body.appendChild(i); }, PHOTO);
  await page.waitForFunction(() => document.getElementById("probe")?.complete);
  assert.equal(await page.evaluate(() => document.getElementById("probe").naturalWidth > 0), true, "the photo of the bucket was blocked");
  const ran = await page.evaluate(() => new Promise((resolve) => {
    const s = document.createElement("script"); s.textContent = "window.__injected = true"; document.body.appendChild(s); setTimeout(() => resolve(window.__injected === true), 100);
  }));
  assert.equal(ran, false, "an inline script ran: the CSP does not block it");
  const csp = violations.filter((v) => !/inline script/i.test(v));
  assert.deepEqual(csp, [], `the CSP blocked something the app needs: ${csp.join(" | ")}`);
  await context.close();
});
