// Sprint 9: the screens work on a phone and on a computer. This suite drives the real app (vite dev
// server) in a real browser (Playwright) with a fake backend (mock-api.mjs), in four widths:
// 360 and 390 (phones), 768 (tablet) and 1280 (computer).
//
// Run with: npm run test:responsive     (needs Chromium; set CHROMIUM_PATH to use another binary)
//
// The screens must offer these hooks to be measured (they are part of the contract):
//   data-testid="bottom-nav" / "sidebar"        the phone navigation bar and the desktop side bar
//   data-testid="nav-more" + "nav-more-menu"    optional "Más" menu of the bottom bar for what does not fit
//   data-testid="ingredients-list"              list of ingredients; data-view="cards" | "table"
//   data-testid="quotations-list"               list of quotations; data-view="cards" | "table"
//   [data-card]                                 each card of those lists
//   role="dialog" + [data-primary-action]       every window, and its main button
//   [data-kpi]                                  every big figure (KPIs, totals)
//   data-testid="trend-chart"                   the trend chart
//   data-testid="orders-list"                   list of orders (tab "Pedidos" of Ventas); data-view="cards" | "table"
//   button "Nuevo pedido" -> window (role=dialog) with "Agregar producto" and the main button "Registrar pedido"
import { test, before, after } from "node:test";
import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { existsSync, mkdirSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { answer, pdfBytes, recipes } from "./mock-api.mjs";

const frontendDir = join(dirname(fileURLToPath(import.meta.url)), "..", "..");
const shots = join(frontendDir, "test-results", "responsive");
const PORT = 4399;
const BASE = `http://127.0.0.1:${PORT}`;
const API = "http://localhost:8080";
const WIDTHS = [360, 390, 768, 1280];
const PHONES = [360, 390];
const isPhone = (w) => w < 768;
const HEIGHT = 800;

const PAGES = [
  { name: "login", path: "/login", auth: false },
  { name: "dashboard", path: "/dashboard" },
  { name: "ingredients", path: "/ingredients" },
  { name: "recipes", path: "/recipes" },
  { name: "costs", path: "/costs" },
  { name: "fixed-costs", path: "/fixed-costs" },
  { name: "sales", path: "/sales" },
  { name: "expenses", path: "/expenses" },
  { name: "profitability", path: "/profitability" },
];
const SECTIONS = ["/dashboard", "/ingredients", "/recipes", "/costs", "/fixed-costs", "/sales", "/expenses", "/profitability"];

let server;
let browser;

before(async () => {
  server = spawn("npx", ["vite", "--port", String(PORT), "--strictPort", "--host", "127.0.0.1"], {
    cwd: frontendDir, stdio: "ignore", env: { ...process.env, VITE_API_URL: "" },
  });
  for (let i = 0; i < 120; i++) {
    try { if ((await fetch(BASE)).ok) break; } catch { /* not up yet */ }
    await new Promise((r) => setTimeout(r, 500));
  }
  browser = await chromium.launch({ executablePath: process.env.CHROMIUM_PATH || undefined });
  mkdirSync(shots, { recursive: true });
});

after(async () => {
  await browser?.close();
  server?.kill();
});

const CORS = {
  "access-control-allow-origin": "*", "access-control-allow-headers": "*",
  "access-control-allow-methods": "*", "access-control-expose-headers": "*",
};

async function open(width, path, { auth = true, height = HEIGHT } = {}) {
  const context = await browser.newContext({ viewport: { width, height }, acceptDownloads: true });
  if (auth) await context.addInitScript(() => localStorage.setItem("token", "test.token.value"));
  await context.route(`${API}/**`, (route) => {
    const req = route.request();
    const url = new URL(req.url());
    if (req.method() === "OPTIONS") return route.fulfill({ status: 204, headers: CORS });
    const found = answer(req.method(), url.pathname);
    if (!found) return route.fulfill({ status: 404, headers: CORS, contentType: "application/json", body: "{}" });
    if (found.pdf) return route.fulfill({ status: 200, headers: CORS, contentType: "application/pdf", body: pdfBytes });
    return route.fulfill({ status: 200, headers: CORS, contentType: "application/json", body: JSON.stringify(found.body) });
  });
  const page = await context.newPage();
  await page.goto(BASE + path, { waitUntil: "networkidle" });
  await page.waitForTimeout(300);
  return { page, context };
}

const withPage = (width, path, opts, fn) => async () => {
  const { page, context } = await open(width, path, opts);
  try { await fn(page); } finally { await context.close(); }
};

const interactive = () => {
  const sel = 'button, a[href], input:not([type=hidden]):not([type=file]):not([type=checkbox]):not([type=radio]), select, textarea, [role="button"]';
  const out = [];
  for (const el of document.querySelectorAll(sel)) {
    const r = el.getBoundingClientRect();
    const cs = getComputedStyle(el);
    if (r.width === 0 || r.height === 0 || cs.visibility === "hidden" || cs.display === "none") continue;
    const label = (el.getAttribute("aria-label") || el.textContent || el.placeholder || el.name || "").trim().slice(0, 28);
    const what = `${el.tagName.toLowerCase()} "${label}"`;
    if (r.height < 43.5) out.push(`${what}: ${Math.round(r.height)}px high`);
    if (/^(INPUT|SELECT|TEXTAREA)$/.test(el.tagName) && parseFloat(cs.fontSize) < 16) out.push(`${what}: text ${cs.fontSize}`);
  }
  return out;
};

const inViewport = (box, w, h) => box && box.x >= -0.5 && box.y >= -0.5 && box.x + box.width <= w + 0.5 && box.y + box.height <= h + 0.5;

// R1: no page scrolls sideways. Also R8: a screenshot of each page in each width.
for (const p of PAGES) {
  for (const w of WIDTHS) {
    test(`R1 ${p.name} at ${w}px has no horizontal scroll`, { timeout: 60000 }, withPage(w, p.path, { auth: p.auth !== false }, async (page) => {
      const m = await page.evaluate(() => ({ doc: document.documentElement.scrollWidth, body: document.body.scrollWidth, view: window.innerWidth }));
      await page.screenshot({ path: join(shots, `${p.name}-${w}.png`), fullPage: true });
      assert.ok(m.doc <= m.view && m.body <= m.view, `${p.name} @${w}: the page is ${Math.max(m.doc, m.body)}px wide in a ${m.view}px screen`);
    }));
  }
}

// R2: ingredients and quotations are cards on a phone.
for (const w of WIDTHS) {
  test(`R2 ingredients at ${w}px`, { timeout: 60000 }, withPage(w, "/ingredients", {}, async (page) => {
    const list = page.locator('[data-testid="ingredients-list"]');
    await list.waitFor({ state: "visible" });
    const view = await list.getAttribute("data-view");
    if (isPhone(w)) {
      assert.equal(view, "cards", `at ${w}px the ingredients are shown as ${view}`);
      const cards = page.locator('[data-testid="ingredients-list"] [data-card]');
      assert.ok((await cards.count()) >= 10, "each ingredient is a card");
      const first = await cards.first().boundingBox();
      assert.ok(first.x >= 0 && first.x + first.width <= w + 0.5, "a card sticks out of the screen");
      for (const name of ["Ajustar stock", "Editar", "Eliminar"]) {
        const b = await cards.first().getByRole("button", { name }).boundingBox();
        assert.ok(b && b.height >= 43.5, `the "${name}" button of a card is missing or too small`);
      }
    } else {
      assert.ok(view === "table" || view === "cards", "the list must say how it is shown (data-view)");
      if (view === "table") assert.ok(await list.locator("table").first().isVisible(), "data-view=table without a table");
    }
  }));

  test(`R2 quotations at ${w}px`, { timeout: 60000 }, withPage(w, "/sales", {}, async (page) => {
    const list = page.locator('[data-testid="quotations-list"]');
    await list.waitFor({ state: "visible" });
    const view = await list.getAttribute("data-view");
    if (isPhone(w)) {
      assert.equal(view, "cards", `at ${w}px the quotations are shown as ${view}`);
      assert.equal(await list.locator("table:visible").count(), 0, "a table is still shown on a phone");
      const cards = list.locator("[data-card]");
      assert.ok((await cards.count()) >= 5, "each quotation is a card");
      const text = await cards.first().innerText();
      assert.match(text, /María González|Cumpleaños|Evento/, "the card should show the client");
      assert.match(text, /\$/, "the card should show the price");
      const box = await cards.first().boundingBox();
      assert.ok(box.x >= 0 && box.x + box.width <= w + 0.5, "a card sticks out of the screen");
      const actions = cards.first().getByRole("button");
      assert.ok((await actions.count()) >= 1, "the card has its actions at hand");
    } else {
      assert.ok(view === "table" || view === "cards", "the list must say how it is shown (data-view)");
    }
  }));
}

// R3: on a phone, what is tapped is big enough and the fields do not make the phone zoom.
for (const p of PAGES) {
  for (const w of PHONES) {
    test(`R3 ${p.name} at ${w}px: touch targets of 44px and fields of 16px`, { timeout: 60000 }, withPage(w, p.path, { auth: p.auth !== false }, async (page) => {
      const bad = await page.evaluate(interactive);
      assert.deepEqual(bad.slice(0, 8), [], `${bad.length} things are too small (first 8 shown)`);
    }));
  }
}

// R4: navigation. A phone reaches every section with taps; nothing is hidden behind the bar.
async function reach(page, to, w) {
  const bar = page.locator('[data-testid="bottom-nav"]');
  const link = bar.locator(`a[href="${to}"]`);
  const box = (await link.count()) ? await link.first().boundingBox() : null;
  if (box && box.x >= -0.5 && box.x + box.width <= w + 0.5 && box.width >= 44 - 0.5) {
    await link.first().click();
  } else {
    await page.locator('[data-testid="nav-more"]').click();
    await page.locator(`[data-testid="nav-more-menu"] a[href="${to}"]`).click();
  }
  await page.waitForURL(`**${to}`);
}

for (const w of PHONES) {
  test(`R4 navigation at ${w}px reaches every section`, { timeout: 90000 }, withPage(w, "/dashboard", {}, async (page) => {
    assert.ok(await page.locator('[data-testid="bottom-nav"]').isVisible(), "the bottom bar is not shown on a phone");
    for (const to of SECTIONS) await reach(page, to, w);
  }));

  test(`R4 the last thing of a page is not hidden by the bar at ${w}px`, { timeout: 60000 }, withPage(w, "/ingredients", {}, async (page) => {
    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
    await page.waitForTimeout(200);
    const m = await page.evaluate(() => {
      const nav = document.querySelector('[data-testid="bottom-nav"]');
      const last = document.querySelector("main").lastElementChild;
      return { navTop: nav.getBoundingClientRect().top, lastBottom: last.getBoundingClientRect().bottom };
    });
    assert.ok(m.lastBottom <= m.navTop + 1, `the end of the page (${Math.round(m.lastBottom)}) is under the bar (${Math.round(m.navTop)})`);
  }));
}

for (const w of [768, 1280]) {
  test(`R4 at ${w}px the side bar is shown and the bottom bar is not`, { timeout: 60000 }, withPage(w, "/dashboard", {}, async (page) => {
    assert.ok(await page.locator('[data-testid="sidebar"]').isVisible(), "the side bar is not shown");
    assert.equal(await page.locator('[data-testid="bottom-nav"]').isVisible(), false, "the bottom bar is shown on a computer");
  }));
}

// R5: windows fit the screen and their main button is always within reach.
const WINDOWS = [
  { name: "new ingredient", open: (page) => page.getByRole("button", { name: "+ Nuevo" }).click() },
  { name: "import Excel", open: (page) => page.getByRole("button", { name: /Importar Excel/ }).click() },
  { name: "adjust stock", open: (page) => page.getByRole("button", { name: "Ajustar stock" }).first().click() },
];
const SCREENS = [{ w: 360, h: 640 }, { w: 390, h: 700 }, { w: 768, h: 900 }, { w: 1280, h: 800 }];
for (const win of WINDOWS) {
  for (const { w, h } of SCREENS) {
    test(`R5 window "${win.name}" at ${w}x${h}`, { timeout: 60000 }, withPage(w, "/ingredients", { height: h }, async (page) => {
      await win.open(page);
      const dialog = page.locator('[role="dialog"]').first();
      await dialog.waitFor({ state: "visible", timeout: 5000 });
      assert.ok(inViewport(await dialog.boundingBox(), w, h), "the window does not fit the screen");
      const primary = dialog.locator("[data-primary-action]").first();
      assert.ok(await primary.isVisible(), "the window has no main button (data-primary-action)");
      assert.ok(inViewport(await primary.boundingBox(), w, h), "the main button is out of reach without scrolling");
      if (isPhone(w)) {
        const bad = await page.evaluate(interactive);
        const inDialog = bad.filter((s) => !/^a "/.test(s));
        assert.deepEqual(inDialog.slice(0, 8), [], "things in the window are too small");
      }
    }));
  }
}

// R6: the quotation generator and the PDF work from a phone.
for (const w of PHONES) {
  test(`R6 quotation generator and PDF at ${w}px`, { timeout: 90000 }, withPage(w, "/costs", {}, async (page) => {
    await page.locator("select").first().selectOption(recipes[0].id);
    const download = page.getByRole("button", { name: /Descargar PDF/ });
    await download.waitFor({ state: "visible" });
    await page.getByPlaceholder("ej. María González").fill("María González");

    const m = await page.evaluate(() => ({ doc: document.documentElement.scrollWidth, view: window.innerWidth }));
    assert.ok(m.doc <= m.view, `with the generator open the page is ${m.doc}px wide in a ${m.view}px screen`);
    const bad = await page.evaluate(interactive);
    assert.deepEqual(bad.slice(0, 8), [], "things in the generator are too small");

    await download.scrollIntoViewIfNeeded();
    assert.ok(inViewport(await download.boundingBox(), w, HEIGHT), "the download button is out of the screen");
    const [file] = await Promise.all([page.waitForEvent("download", { timeout: 10000 }), download.click()]);
    assert.match(file.suggestedFilename(), /^cotizacion_.*\.pdf$/);
  }));
}

// R7: big figures and the chart fit.
for (const name of ["dashboard", "profitability"]) {
  for (const w of WIDTHS) {
    test(`R7 figures and chart of ${name} at ${w}px`, { timeout: 60000 }, withPage(w, `/${name}`, {}, async (page) => {
      const kpis = page.locator("[data-kpi]");
      const n = await kpis.count();
      if (name === "profitability") assert.ok(n >= 6, `profitability should mark its figures with data-kpi (found ${n})`);
      const clipped = await page.evaluate(() => [...document.querySelectorAll("[data-kpi]")]
        .filter((el) => el.scrollWidth > el.clientWidth + 1 || getComputedStyle(el).textOverflow === "ellipsis")
        .map((el) => `${el.textContent.trim()} (${el.scrollWidth} > ${el.clientWidth})`));
      assert.deepEqual(clipped, [], "figures that are cut off");
      if (name === "profitability") {
        const chart = page.locator('[data-testid="trend-chart"]');
        await chart.waitFor({ state: "visible" });
        assert.ok(inViewport(await chart.boundingBox(), w, 100000), "the chart is wider than the screen");
      }
    }));
  }
}

// O10 (sprint 10): orders are born responsive.
for (const w of WIDTHS) {
  test(`O10 orders list at ${w}px`, { timeout: 60000 }, withPage(w, "/sales", {}, async (page) => {
    await page.getByRole("button", { name: "Pedidos", exact: true }).click();
    const list = page.locator('[data-testid="orders-list"]');
    await list.waitFor({ state: "visible" });
    const view = await list.getAttribute("data-view");
    const m = await page.evaluate(() => ({ doc: document.documentElement.scrollWidth, view: window.innerWidth }));
    assert.ok(m.doc <= m.view, `the orders make the page ${m.doc}px wide in a ${m.view}px screen`);
    if (isPhone(w)) {
      assert.equal(view, "cards", `at ${w}px the orders are shown as ${view}`);
      assert.equal(await list.locator("table:visible").count(), 0, "a table is still shown on a phone");
      const cards = list.locator("[data-card]");
      assert.ok((await cards.count()) >= 5, "each order is a card");
      const text = await cards.first().innerText();
      assert.match(text, /María González|Cumpleaños|Evento/, "the card should show the customer");
      assert.match(text, /\$/, "the card should show the total");
      assert.match(text, /Anular pedido/, "the card should offer to cancel the order");
      const bad = await page.evaluate(interactive);
      assert.deepEqual(bad.slice(0, 8), [], "things in the orders are too small");
    } else {
      assert.ok(view === "table" || view === "cards", "the list must say how it is shown (data-view)");
    }
  }));
}

for (const { w, h } of SCREENS) {
  test(`O10 new order window at ${w}x${h}`, { timeout: 60000 }, withPage(w, "/sales", { height: h }, async (page) => {
    await page.getByRole("button", { name: "Nuevo pedido" }).click();
    const dialog = page.locator('[role="dialog"]').first();
    await dialog.waitFor({ state: "visible", timeout: 5000 });
    for (let i = 0; i < 3; i++) await dialog.getByRole("button", { name: "Agregar producto" }).click();
    assert.ok(inViewport(await dialog.boundingBox(), w, h), "the window does not fit the screen");
    const primary = dialog.getByRole("button", { name: "Registrar pedido" });
    assert.ok(await primary.isVisible(), "there is no 'Registrar pedido' button");
    assert.ok(inViewport(await primary.boundingBox(), w, h), "'Registrar pedido' is out of reach without scrolling");
    const m = await page.evaluate(() => ({ doc: document.documentElement.scrollWidth, view: window.innerWidth }));
    assert.ok(m.doc <= m.view, "the page scrolls sideways with the order window open");
    if (isPhone(w)) {
      const bad = await page.evaluate(interactive);
      assert.deepEqual(bad.filter((s) => !/^a "/.test(s)).slice(0, 8), [], "things in the order window are too small");
    }
  }));
}

// R8: the run leaves a screenshot of every page in every width.
test("R8 every page left its screenshots", () => {
  const missing = [];
  for (const p of PAGES) for (const w of WIDTHS) if (!existsSync(join(shots, `${p.name}-${w}.png`))) missing.push(`${p.name}-${w}.png`);
  assert.deepEqual(missing, []);
});
