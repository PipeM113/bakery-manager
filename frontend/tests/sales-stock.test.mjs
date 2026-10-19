// Sprint 7 (V8): the sales screen lets a sale be of a whole recipe or of portions, and shows what
// was missing from the stock in a way anyone can read.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..", "src");
const read = (p) => readFileSync(join(root, p), "utf8");

test("the sale types carry how it was sold and what was missing", () => {
  const t = read("types/sale.ts");
  assert.match(t, /sold_as/);
  assert.match(t, /stock_shortfalls/);
});

test("the sales screen offers whole recipe or portion", () => {
  const page = read("pages/SalesPage.tsx");
  assert.match(page, /sold_as/);
  assert.match(page, /Receta completa/);
  assert.match(page, /Porci[óo]n/);
});

test("the sales screen warns when the stock was not enough", () => {
  const page = read("pages/SalesPage.tsx");
  assert.match(page, /stock_shortfalls/);
  assert.match(page, /faltaron|falt[óo]|sin stock suficiente/i);
});
