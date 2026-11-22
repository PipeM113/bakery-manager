// Review of 2026-11-22: every screen puts its own name in the title of the tab, so a person with several tabs open
// (and a screen reader) knows where they are.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { titleFor, ROUTE_TITLES } from "../src/utils/titles.ts";

const src = (p) => readFileSync(new URL(`../src/${p}`, import.meta.url), "utf8");

test("T1 every section of the navigation has a title that starts with its name and ends with the brand", () => {
  const sections = ["/dashboard", "/ingredients", "/recipes", "/costs", "/fixed-costs", "/sales", "/expenses", "/profitability", "/business", "/account"];
  const seen = new Set();
  for (const path of sections) {
    const t = titleFor(path);
    assert.match(t, / · Angeles Bakery$/, path);
    assert.ok(!t.startsWith("Angeles"), `${path} has only the brand`);
    seen.add(t);
  }
  assert.equal(seen.size, sections.length, "two screens share a title");
});

test("T2 an unknown path falls back to the brand title", () => {
  assert.equal(titleFor("/nope"), "Angeles Bakery · Gestión");
  assert.equal(titleFor("/recipes/"), titleFor("/recipes"), "a trailing slash is the same screen");
});

test("T3 the layout sets the title when the route changes", () => {
  const layout = src("components/layout/AppLayout.tsx");
  assert.match(layout, /useLocation\(\)/);
  assert.match(layout, /document\.title = titleFor\(/);
  assert.ok(Object.keys(ROUTE_TITLES).length >= 10);
});
