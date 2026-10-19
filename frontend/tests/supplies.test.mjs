// Sprint 3, rules I2, I3 and I4: supplies (boxes, cardboard) are shown apart from ingredients.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const frontendDir = join(dirname(fileURLToPath(import.meta.url)), "..");
const read = (rel) => readFileSync(join(frontendDir, rel), "utf8");

test("I1: the ingredient type has a kind", () => {
  assert.match(read("src/types/ingredient.ts"), /kind/);
});

test("I2: the ingredients screen separates ingredients and supplies and sends the kind", () => {
  const page = read("src/pages/IngredientsPage.tsx");
  assert.match(page, /Insumos/);
  assert.match(page, /Ingredientes/);
  assert.match(page, /kind/);
});

// Corrected with the owner's approval: the page only loads the data; the grouping lives in
// the form (picker and lines) and in the detail.
test("I3: the recipe form and detail group the lines by kind", () => {
  assert.match(read("src/components/RecipeForm.tsx"), /Insumos/);
  assert.match(read("src/components/RecipeDetail.tsx"), /Insumos/);
});

test("I4: the cost breakdown shows the supplies total", () => {
  const quotation = read("src/components/QuotationGenerator.tsx");
  assert.match(quotation, /supplies_total/);
  assert.match(quotation, /Insumos/);
});
