// Loading states and shared UI rules: screens that fetch show the shape of the page while they wait
// (never a bare "Cargando..." on an empty screen), every page keeps the same margin, and the two
// navigations use one icon set.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync, readdirSync } from "node:fs";

const read = (p) => readFileSync(new URL(`../src/${p}`, import.meta.url), "utf8");
const pages = readdirSync(new URL("../src/pages/", import.meta.url)).filter((f) => f.endsWith(".tsx"));

test("no page shows a bare 'Cargando...' text on an empty screen", () => {
  for (const f of pages) {
    assert.doesNotMatch(read(`pages/${f}`), />\s*(Cargando|Calculando)\.\.\.\s*</, f);
  }
});

test("the screens that fetch on open use skeletons", () => {
  for (const [page, name] of [["DashboardPage", "CardsSkeleton"], ["IngredientsPage", "ListSkeleton"], ["RecipesPage", "ListSkeleton"],
    ["ProfitabilityPage", "CardsSkeleton"], ["BusinessPage", "FormSkeleton"], ["ExpensesPage", "RowsSkeleton"], ["FixedCostsPage", "RowsSkeleton"], ["SalesPage", "RowsSkeleton"]]) {
    assert.match(read(`pages/${page}.tsx`), new RegExp(name), `${page} uses ${name}`);
  }
});

test("a skeleton tells screen readers it is loading and respects reduced motion", () => {
  const s = read("components/Skeleton.tsx");
  assert.match(s, /role="status"/);
  assert.match(s, /aria-busy="true"/);
  assert.match(s, /sr-only/);
  assert.match(s, /motion-safe:animate-pulse/);
});

test("business data and account have the same page margin as the other screens", () => {
  for (const f of ["BusinessPage", "AccountPage"]) {
    assert.match(read(`pages/${f}.tsx`), /className="px-6 py-10 md:px-10"/, f);
  }
});

test("both navigations draw their icons from one set (no text symbols, no icon used twice)", () => {
  for (const f of ["components/layout/Sidebar.tsx", "components/layout/BottomNav.tsx"]) {
    const src = read(f);
    assert.match(src, /from "\.\/icons"/, f);
    assert.doesNotMatch(src, /icon: "[^\x00-\x7F]/, `${f} has a text symbol as icon`);
    const names = [...src.matchAll(/icon: "(\w+)"/g)].map((m) => m[1]);
    assert.equal(new Set(names).size, names.length, `${f} repeats an icon`);
  }
});

test("the labels of the bottom bar are at least 11 px and are not cut", () => {
  const src = read("components/layout/BottomNav.tsx");
  assert.doesNotMatch(src, /text-\[10px\]/);
});

test("deleting a recipe or an ingredient names it in the question", () => {
  assert.match(read("pages/RecipesPage.tsx"), /¿Eliminar la receta "\$\{name\}"\?/);
  assert.match(read("pages/IngredientsPage.tsx"), /¿Eliminar "\$\{name\}"\?/);
});
