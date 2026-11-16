// Sprint 17: notices and confirmations of the app's own (not the browser's), at the top right,
// stacked and long enough to read; and a quotation can be deleted from the list.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync, readdirSync, existsSync } from "node:fs";
import { execFileSync } from "node:child_process";
import { MAX_VISIBLE, durationFor, pushToast, dismissToast } from "../src/utils/toasts.ts";

const src = (p) => readFileSync(new URL(`../src/${p}`, import.meta.url), "utf8");
const all = (dir) => readdirSync(new URL(`../src/${dir}/`, import.meta.url)).filter((f) => f.endsWith(".tsx")).map((f) => [`${dir}/${f}`, src(`${dir}/${f}`)]);
const sources = [...all("pages"), ...all("components")];

// --- N: the queue of notices (pure logic) ---

test("N1 a success notice lasts long enough to read and an error stays longer", () => {
  assert.ok(durationFor("success") >= 5000, "success >= 5 s");
  assert.ok(durationFor("error") >= 8000, "error >= 8 s");
  assert.ok(durationFor("error") > durationFor("success"));
});

test("N2 notices pile up, the newest on top, and never more than the maximum at once", () => {
  let list = [];
  for (let id = 1; id <= MAX_VISIBLE + 3; id++) list = pushToast(list, { id, kind: "success", title: `t${id}` });
  assert.equal(list.length, MAX_VISIBLE);
  assert.equal(list[0].id, MAX_VISIBLE + 3, "the newest is first");
  assert.ok(!list.some((t) => t.id === 1), "the oldest was dropped");
  assert.ok(MAX_VISIBLE >= 3 && MAX_VISIBLE <= 6);
});

test("N3 a notice is removed by its id and the others stay", () => {
  const list = [{ id: 3, kind: "info", title: "c" }, { id: 2, kind: "error", title: "b" }, { id: 1, kind: "success", title: "a" }];
  assert.deepEqual(dismissToast(list, 2).map((t) => t.id), [3, 1]);
  assert.deepEqual(dismissToast(list, 99).map((t) => t.id), [3, 2, 1]);
});

// --- N: how they look and where ---

test("N4 the notices are fixed at the top right, announced to screen readers, closable, paused on hover", () => {
  const s = src("components/Toasts.tsx");
  assert.match(s, /fixed top-4 right-4/);
  assert.match(s, /role=\{t\.kind === "error" \? "alert" : "status"\}/);
  assert.match(s, /aria-label="Cerrar aviso"/);
  assert.match(s, /onMouseEnter/);
  assert.match(s, /onMouseLeave/);
  assert.match(s, /pointer-events-none/, "the container lets clicks through between notices");
  assert.match(s, /z-\[70\]/, "above the dialogs (z-[60])");
});

test("N5 success is shown on salvia, an error on rosa with the danger color, both with text and an icon (not color alone)", () => {
  const s = src("components/Toasts.tsx");
  assert.match(s, /bg-salvia/);
  assert.match(s, /bg-rosa/);
  assert.match(s, /text-peligro/);
  assert.match(s, /<svg/);
});

test("N6 the app mounts the notices and the confirmations once, for every page", () => {
  const main = src("main.tsx");
  assert.match(main, /ToastProvider/);
  assert.match(main, /ConfirmProvider/);
});

test("N7 no screen builds its own notice any more", () => {
  for (const [f, s] of sources) {
    if (f === "components/Toasts.tsx") continue; // the one that owns the queue
    assert.doesNotMatch(s, /setToasts?\(/, `${f} keeps its own toast state`);
    assert.doesNotMatch(s, /useState<ToastState/, `${f} keeps its own toast state`);
  }
});

test("N8 the screens that create, change or delete say so with a notice", () => {
  for (const f of ["RecipesPage", "IngredientsPage", "SalesPage", "ExpensesPage", "FixedCostsPage", "BusinessPage", "AccountPage"]) {
    assert.match(src(`pages/${f}.tsx`), /useToast\(\)/, `${f} uses the notices`);
  }
  const recipes = src("pages/RecipesPage.tsx");
  for (const t of ["Receta creada", "Receta actualizada", "Receta eliminada"]) assert.ok(recipes.includes(t), t);
  const sales = src("pages/SalesPage.tsx");
  for (const t of ["Cotización eliminada", "Cotización cancelada", "Cotización confirmada"]) assert.ok(sales.includes(t), t);
});

test("N9 a refused delete (409) shows the reason of the server in an error notice", () => {
  assert.match(src("pages/RecipesPage.tsx"), /toast\.error\([^)]*apiErrorMessage/s);
  assert.match(src("pages/IngredientsPage.tsx"), /toast\.error\([^)]*apiErrorMessage/s);
});

// --- D: confirmations ---

test("D1 no screen uses the confirm or alert of the browser", () => {
  for (const [f, s] of sources) {
    assert.doesNotMatch(s, /\bwindow\.(confirm|alert|prompt)\(/, f);
    assert.doesNotMatch(s, /(?<![.\w])(confirm|alert|prompt)\(/, f);
  }
});

test("D2 the confirmation names the thing and the danger button says what it does", () => {
  const s = src("components/ConfirmDialog.tsx");
  assert.match(s, /role="alertdialog"/);
  assert.match(s, /aria-describedby/);
  assert.match(s, /bg-peligro/);
  assert.match(s, /autoFocus/, "the focus starts on Cancelar, not on the destructive button");
  assert.match(s, /Escape|onClose/, "Esc closes it");
});

test("D3 every delete asks with its own words", () => {
  const checks = [
    ["pages/RecipesPage.tsx", /Eliminar receta/],
    ["pages/IngredientsPage.tsx", /Eliminar/],
    ["pages/SalesPage.tsx", /Eliminar cotización/],
    ["pages/ExpensesPage.tsx", /Eliminar gasto/],
    ["pages/FixedCostsPage.tsx", /Terminar costo fijo/],
  ];
  for (const [f, re] of checks) assert.match(src(f), re, f);
});

// --- Q: deleting a quotation ---

test("Q1 the quotation service can delete and the list has an Eliminar action", () => {
  assert.match(src("api/quotationService.ts"), /delete/);
  const sales = src("pages/SalesPage.tsx");
  assert.match(sales, /quotationService\.delete\(/);
  assert.match(sales, /data-testid="quote-delete"/);
});

test("Q2 a confirmed quotation does not offer Eliminar (it is cancelled first)", () => {
  assert.match(src("pages/SalesPage.tsx"), /q\.status !== "confirmed"/);
});

test("Q3 the components exist", () => {
  for (const f of ["components/Toasts.tsx", "components/ConfirmDialog.tsx", "utils/toasts.ts", "hooks/useToast.ts", "hooks/useConfirm.ts"]) assert.ok(existsSync(new URL(`../src/${f}`, import.meta.url)), f);
});

// The CI builds with `tsc -b`; `tsc -p .` alone does not look at src, so a leftover name slipped through once
// (showSuccess in the order handler) and the browser tests did not notice because they do not type-check.
test("the whole project type-checks like the CI build does (tsc -b)", () => {
  execFileSync("npx", ["tsc", "-b"], { cwd: new URL("..", import.meta.url), stdio: "pipe" });
});
