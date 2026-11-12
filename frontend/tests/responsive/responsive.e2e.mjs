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
//   button "Nueva cotización" -> window (role=dialog) with [data-testid="quote-line"] (select aria-label "Producto",
//     input aria-label "Cantidad"), [data-testid="quote-add-line"], [data-testid="quote-delivery"], placeholder
//     "Nombre del cliente" and the main button "Crear y descargar PDF"
//   /business "Datos del negocio": [data-testid="business-form"] with the fields "Teléfono", "Banco", "Tipo de cuenta", "Titular de la cuenta", "Correo electrónico" and the button "Guardar"; reachable from the navigation
//   /account "Mi cuenta": [data-testid="account-form"] with "Contraseña actual", "Nueva contraseña",
//     "Repite la nueva contraseña" and the button "Cambiar contraseña"; reachable from the navigation
//   [data-testid="quote-download"]              "Descargar PDF" button of each quotation card / row
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
  { name: "business", path: "/business" },
  { name: "account", path: "/account" },
];
const SECTIONS = ["/dashboard", "/ingredients", "/recipes", "/costs", "/fixed-costs", "/sales", "/expenses", "/profitability", "/business", "/account"];

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

// `respond(method, pathname, body, calls)` may answer a call before the mock does: it returns { status, body } or undefined.
// Every call the screens make is kept in `context.calls` so a flow can check what was sent.
async function open(width, path, { auth = true, height = HEIGHT, respond } = {}) {
  const context = await browser.newContext({ viewport: { width, height }, acceptDownloads: true });
  context.calls = [];
  if (auth) await context.addInitScript(() => localStorage.setItem("token", "test.token.value"));
  await context.route(`${API}/**`, (route) => {
    const req = route.request();
    const url = new URL(req.url());
    if (req.method() === "OPTIONS") return route.fulfill({ status: 204, headers: CORS });
    let body;
    try { body = req.postDataJSON(); } catch { body = undefined; }
    const call = { method: req.method(), path: url.pathname, body };
    context.calls.push(call);
    const custom = respond?.(call.method, call.path, call.body, context.calls);
    if (custom) return route.fulfill({ status: custom.status ?? 200, headers: CORS, contentType: "application/json", body: JSON.stringify(custom.body ?? {}) });
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

  test(`E4 ingredients at ${w}px can be sorted and show the supplier`, { timeout: 60000 }, withPage(w, "/ingredients", {}, async (page) => {
    await page.locator('[data-testid="ingredients-list"]').waitFor({ state: "visible" });
    const sort = page.locator('[data-testid="ingredient-sort"]');
    assert.ok(await sort.isVisible(), "there is no way to choose the order");
    const text = await page.locator('[data-testid="ingredients-list"]').innerText();
    assert.ok(text.includes("La Vega"), "the supplier is not shown");
    const m = await page.evaluate(() => ({ doc: document.documentElement.scrollWidth, view: window.innerWidth }));
    assert.ok(m.doc <= m.view + 1, "horizontal scroll");
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
  { name: "quick stock", open: (page) => page.locator('[data-testid="stock-quick"]').first().click() },
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
      assert.match(text, /anular pedido/i, "the card should offer to cancel the order");
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

// Q13 (sprint 12): a quotation with several products is made and downloaded from any screen size.
for (const w of WIDTHS) {
  test(`Q13 new quotation window at ${w}px`, { timeout: 90000 }, withPage(w, "/sales", {}, async (page) => {
    await page.getByRole("button", { name: "Nueva cotización" }).click();
    const dialog = page.getByRole("dialog");
    await dialog.waitFor({ state: "visible" });
    await dialog.getByPlaceholder("Nombre del cliente").fill("María González");
    await dialog.locator('[data-testid="quote-add-line"]').click();
    const lines = dialog.locator('[data-testid="quote-line"]');
    assert.ok((await lines.count()) >= 2, "the window should let you add a second product");
    for (let i = 0; i < 2; i++) {
      await lines.nth(i).locator('select[aria-label="Producto"]').selectOption(recipes[i].id);
      await lines.nth(i).locator('input[aria-label="Cantidad"]').fill("2");
    }
    await dialog.locator('[data-testid="quote-delivery"]').fill("2500");

    const m = await page.evaluate(() => ({ doc: document.documentElement.scrollWidth, view: window.innerWidth }));
    assert.ok(m.doc <= m.view, `with the window open the page is ${m.doc}px wide in a ${m.view}px screen`);
    if (isPhone(w)) {
      const bad = (await page.evaluate(interactive)).filter((s) => !/^a "/.test(s));
      assert.deepEqual(bad.slice(0, 8), [], "things in the window are too small");
    }
    const create = dialog.locator("[data-primary-action]");
    await create.scrollIntoViewIfNeeded();
    assert.match(await create.innerText(), /Crear y descargar PDF/i);
    assert.ok(inViewport(await create.boundingBox(), w, HEIGHT), "the main button is out of the screen");
    const [file] = await Promise.all([page.waitForEvent("download", { timeout: 10000 }), create.click()]);
    assert.match(file.suggestedFilename(), /^cotizacion_.*\.pdf$/);
  }));
}

// Q14 (sprint 12): any quotation of the list can be downloaded again.
for (const w of WIDTHS) {
  test(`Q14 download again from the list at ${w}px`, { timeout: 60000 }, withPage(w, "/sales", {}, async (page) => {
    const list = page.locator('[data-testid="quotations-list"]');
    await list.waitFor({ state: "visible" });
    const button = list.locator('[data-testid="quote-download"]:visible').first();
    await button.waitFor({ state: "visible" });
    if (isPhone(w)) assert.ok((await button.boundingBox()).height >= 43.5, "the download button is too small to tap");
    const [file] = await Promise.all([page.waitForEvent("download", { timeout: 10000 }), button.click()]);
    assert.match(file.suggestedFilename(), /^cotizacion_.*\.pdf$/);
    if (isPhone(w)) {
      const text = await list.locator("[data-card]").first().innerText();
      assert.match(text, /\$/, "the card shows the total");
    }
  }));
}

// Q15 (sprint 12): the business data screen works on a phone and on a computer.
for (const w of WIDTHS) {
  test(`Q15 business data form at ${w}px`, { timeout: 60000 }, withPage(w, "/business", {}, async (page) => {
    const form = page.locator('[data-testid="business-form"]');
    await form.waitFor({ state: "visible" });
    for (const label of ["Teléfono", "Banco", "Tipo de cuenta", "Titular de la cuenta", "Correo electrónico"]) {
      await form.getByLabel(label).waitFor({ state: "visible" });
    }
    assert.equal(await form.getByLabel("Teléfono").inputValue(), "+56 9 1111 2222", "the form shows what is saved");
    const m = await page.evaluate(() => ({ doc: document.documentElement.scrollWidth, view: window.innerWidth }));
    assert.ok(m.doc <= m.view, `the form makes the page ${m.doc}px wide in a ${m.view}px screen`);
    if (isPhone(w)) {
      const bad = (await page.evaluate(interactive)).filter((s) => !/^a "/.test(s));
      assert.deepEqual(bad.slice(0, 8), [], "things in the form are too small");
    }
    const save = form.getByRole("button", { name: "Guardar" });
    await save.scrollIntoViewIfNeeded();
    assert.ok(inViewport(await save.boundingBox(), w, HEIGHT), "the save button is out of the screen");
    await form.getByLabel("Teléfono").fill("+56 9 3333 4444");
    await save.click();
    await page.getByText(/guardado/i).first().waitFor({ state: "visible", timeout: 5000 });
  }));
}

// Q16 (sprint 12): on a phone the bottom bar never covers the main button of a window (found when a tap
// on "Crear y descargar PDF" went to the bar instead).
const coveredPrimary = () => {
  const btn = document.querySelector('[role="dialog"] [data-primary-action]');
  if (!btn) return "no main button";
  btn.scrollIntoView({ block: "nearest" });
  const r = btn.getBoundingClientRect();
  const top = document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2);
  return btn.contains(top) ? "" : `covered by ${top?.tagName} ${top?.closest("nav")?.getAttribute("data-testid") ?? ""}`;
};
for (const w of PHONES) {
  test(`Q16 windows are not covered by the bottom bar at ${w}px`, { timeout: 60000 }, withPage(w, "/sales", {}, async (page) => {
    await page.getByRole("button", { name: "Nueva cotización" }).click();
    await page.getByRole("dialog").waitFor({ state: "visible" });
    assert.equal(await page.evaluate(coveredPrimary), "", "the quotation window's main button is covered");
    await page.getByRole("button", { name: "Cancelar" }).first().click();
    await page.getByRole("button", { name: "Pedidos", exact: true }).click();
    await page.getByRole("button", { name: "Nuevo pedido" }).click();
    await page.getByRole("dialog").waitFor({ state: "visible" });
    assert.equal(await page.evaluate(coveredPrimary), "", "the order window's main button is covered");
  }));
}

// C2 (sprint 13): the account screen works on a phone and on a computer.
for (const w of WIDTHS) {
  test(`C2 account form at ${w}px`, { timeout: 60000 }, withPage(w, "/account", {}, async (page) => {
    const form = page.locator('[data-testid="account-form"]');
    await form.waitFor({ state: "visible" });
    for (const label of ["Contraseña actual", "Nueva contraseña", "Repite la nueva contraseña"]) {
      await form.getByLabel(label, { exact: true }).waitFor({ state: "visible" });
    }
    const m = await page.evaluate(() => ({ doc: document.documentElement.scrollWidth, view: window.innerWidth }));
    assert.ok(m.doc <= m.view, `the form makes the page ${m.doc}px wide in a ${m.view}px screen`);
    if (isPhone(w)) {
      const bad = (await page.evaluate(interactive)).filter((s) => !/^a "/.test(s));
      assert.deepEqual(bad.slice(0, 8), [], "things in the form are too small");
    }
    const save = form.getByRole("button", { name: "Cambiar contraseña" });
    await save.scrollIntoViewIfNeeded();
    assert.ok(inViewport(await save.boundingBox(), w, HEIGHT), "the button is out of the screen");

    await form.getByLabel("Contraseña actual", { exact: true }).fill("la-clave-de-hoy");
    await form.getByLabel("Nueva contraseña", { exact: true }).fill("una-clave-nueva-123");
    await form.getByLabel("Repite la nueva contraseña", { exact: true }).fill("otra-cosa-distinta");
    await save.click();
    await page.getByText(/no coinciden/i).first().waitFor({ state: "visible", timeout: 5000 });

    await form.getByLabel("Repite la nueva contraseña", { exact: true }).fill("una-clave-nueva-123");
    await save.click();
    await page.getByText(/contraseña cambiada/i).first().waitFor({ state: "visible", timeout: 5000 });
  }));
}

// C7 (sprint 13): a negative amount is written -$271.000, not $-271.000, in every screen.
test("C7 no screen writes the minus after the $", withPage(1280, "/profitability", {}, async (page) => {
  const text = await page.locator("main, body").first().innerText();
  assert.doesNotMatch(text, /\$-\d/, "found an amount written as $-...");
  assert.match(text, /-\$\d/, "the trend of the mock has a negative amount and it should read -$...");
}));

// ---- Sprint 15: flows (what the screens send and show), not only how they look ----

const flow = (width, path, opts, fn) => async () => {
  const { page, context } = await open(width, path, opts);
  try { await fn(page, context); } finally { await context.close(); }
};
const FLOW_WIDTHS = [390, 1280];
const field = (scope, labelText, tag) => scope.locator("label", { hasText: labelText }).locator("..").locator(tag);

for (const w of FLOW_WIDTHS) {
  // F1: "+ Stock" adds a purchase without going through the edit window.
  test(`F1 quick stock sends a purchase at ${w}px`, { timeout: 60000 }, flow(w, "/ingredients", {}, async (page, context) => {
    await page.locator('[data-testid="stock-quick"]').first().click();
    const dialog = page.locator('[role="dialog"]').first();
    await dialog.waitFor({ state: "visible" });
    await dialog.getByRole("button", { name: "Agregar", exact: true }).click();
    await dialog.locator("input[type=number]").fill("500");
    await dialog.getByRole("button", { name: /Agregar al stock/ }).click();
    await dialog.waitFor({ state: "hidden" });
    const call = context.calls.find((c) => c.method === "POST" && /^\/ingredients\/[^/]+\/stock-adjustments$/.test(c.path));
    assert.ok(call, "no stock adjustment was sent");
    assert.equal(call.body.reason, "compra");
    assert.equal(call.body.quantity, 500);
  }));

  // F2: changing the unit of an ingredient that a recipe uses asks for the equivalence, there, and then saves.
  test(`F2 a unit change that needs the weight of 1 litre asks for it and saves at ${w}px`, { timeout: 60000 }, flow(w, "/ingredients", {
    respond: (method, path, body, calls) => {
      if (method !== "PUT" || !/^\/ingredients\/[^/]+$/.test(path)) return undefined;
      const earlier = calls.filter((c) => c.method === "PUT" && c.path === path).length;
      if (earlier > 1) return undefined;
      return { status: 409, body: { error: 'la receta "Flan" usa este insumo en gr: para pasar de gr a ml falta el peso de 1 litro del insumo' } };
    },
  }, async (page, context) => {
    await page.getByPlaceholder(/Buscar/).fill("Harina sin polvos");
    await page.getByRole("button", { name: "Editar" }).first().click();
    const dialog = page.locator('[role="dialog"]').first();
    await dialog.waitFor({ state: "visible" });
    await field(dialog, "Unidad base", "select").selectOption("ml");
    await dialog.getByRole("button", { name: "Guardar" }).click();
    await dialog.getByText(/peso de 1 litro/i).first().waitFor({ state: "visible" });
    const weight = field(dialog, "Peso de 1 litro (gr)", "input");
    assert.ok(await weight.isVisible(), "the weight of 1 litre is not offered after the refusal");
    await weight.fill("1250");
    await dialog.getByRole("button", { name: "Guardar" }).click();
    await dialog.waitFor({ state: "hidden" });
    const puts = context.calls.filter((c) => c.method === "PUT" && /^\/ingredients\/[^/]+$/.test(c.path));
    assert.equal(puts.length, 2, "the second save was not sent");
    assert.equal(puts[1].body.default_unit, "ml");
    assert.equal(puts[1].body.grams_per_liter, 1250);
  }));

  // F3: in a recipe, a unit of another family than the product's warns about the missing equivalence.
  test(`F3 a recipe line in another family warns about the missing equivalence at ${w}px`, { timeout: 60000 }, flow(w, "/recipes", {}, async (page) => {
    await page.getByRole("button", { name: "Editar" }).first().click();
    await page.getByRole("heading", { name: "Editar receta" }).waitFor({ state: "visible" });
    await page.getByLabel("Ingrediente", { exact: true }).first().selectOption({ label: "Esencia de vainilla concentrada · Colun · Líder" });
    await page.getByLabel("Unidad", { exact: true }).first().selectOption("gr");
    await page.getByText(/Peso de 1 litro/).first().waitFor({ state: "visible", timeout: 5000 });
  }));

  // D5: a refused delete shows why, in the screen.
  test(`D5 a refused delete of an ingredient shows the reason at ${w}px`, { timeout: 60000 }, flow(w, "/ingredients", {
    respond: (method, path) => (method === "DELETE" && path.startsWith("/ingredients/") ? { status: 409, body: { error: "No se puede eliminar: se usa en las recetas Pan y Torta" } } : undefined),
  }, async (page) => {
    page.on("dialog", (d) => d.accept());
    await page.getByRole("button", { name: "Eliminar" }).first().click();
    const alert = page.getByRole("alert").filter({ hasText: "se usa en las recetas Pan y Torta" });
    await alert.waitFor({ state: "visible", timeout: 5000 });
  }));

  test(`D5 a refused delete of a recipe shows the reason at ${w}px`, { timeout: 60000 }, flow(w, "/recipes", {
    respond: (method, path) => (method === "DELETE" && path.startsWith("/recipes/") ? { status: 409, body: { error: "No se puede eliminar: tiene 2 ventas registradas" } } : undefined),
  }, async (page) => {
    page.on("dialog", (d) => d.accept());
    await page.getByRole("button", { name: "Eliminar" }).first().click();
    const alert = page.getByRole("alert").filter({ hasText: "tiene 2 ventas registradas" });
    await alert.waitFor({ state: "visible", timeout: 5000 });
  }));
}
