// Sprint 14 (E1-E9, screen side): supplier, order, quick stock, equivalences.
import { test } from "node:test";
import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const src = (p) => {
  const path = join(root, "src", p);
  assert.ok(existsSync(path), `${p} does not exist`);
  return readFileSync(path, "utf8");
};

test("ingredientLabel writes name, brand and supplier, leaving out what is empty", async () => {
  const { ingredientLabel } = await import("../src/utils/ingredients.ts");
  assert.equal(ingredientLabel({ name: "Mantequilla", brand: "Soprole", supplier: "La Vega" }), "Mantequilla · Soprole · La Vega");
  assert.equal(ingredientLabel({ name: "Mantequilla", brand: "Soprole", supplier: "" }), "Mantequilla · Soprole");
  assert.equal(ingredientLabel({ name: "Mantequilla", brand: "", supplier: "La Vega" }), "Mantequilla · La Vega");
  assert.equal(ingredientLabel({ name: "Harina" }), "Harina");
});

test("compareIngredients orders by name without case or accents, then brand, then supplier", async () => {
  const { compareIngredients } = await import("../src/utils/ingredients.ts");
  const mk = (name, brand = "", supplier = "") => ({ name, brand, supplier });
  const sorted = [mk("Zanahoria"), mk("éter"), mk("Azúcar"), mk("álamo"), mk("Leche", "Soprole", "Líder"), mk("Leche", "Soprole", "Jumbo"), mk("Leche", "Colún")]
    .sort(compareIngredients)
    .map((i) => [i.name, i.brand, i.supplier].join("|"));
  assert.deepEqual(sorted, ["álamo||", "Azúcar||", "éter||", "Leche|Colún|", "Leche|Soprole|Jumbo", "Leche|Soprole|Líder", "Zanahoria||"]);
});

test("equivalenceNeeded tells what is asked only when the unit changes family", async () => {
  const { equivalenceNeeded } = await import("../src/utils/units.ts");
  const none = { grams_per_liter: 0, grams_per_unit: 0 };
  assert.equal(equivalenceNeeded("kg", "gr", none), null);
  assert.equal(equivalenceNeeded("lt", "ml", none), null);
  assert.equal(equivalenceNeeded("lt", "gr", none), "grams_per_liter");
  assert.equal(equivalenceNeeded("kg", "ml", none), "grams_per_liter");
  assert.equal(equivalenceNeeded("lt", "gr", { grams_per_liter: 1030, grams_per_unit: 0 }), null);
  assert.equal(equivalenceNeeded("und", "gr", none), "grams_per_unit");
  assert.equal(equivalenceNeeded("gr", "und", none), "grams_per_unit");
  assert.equal(equivalenceNeeded("und", "gr", { grams_per_liter: 0, grams_per_unit: 60 }), null);
  assert.equal(equivalenceNeeded("lt", "und", { grams_per_liter: 1000, grams_per_unit: 0 }), "grams_per_unit");
});

test("the ingredient screen has the supplier and the two equivalences", () => {
  const s = src("pages/IngredientsPage.tsx");
  assert.match(s, /Proveedor/);
  assert.match(s, /supplier/);
  assert.ok(s.includes("Peso de 1 litro (gr)") && s.includes("Peso de 1 unidad (gr)"));
  assert.match(s, /ingredientLabel|compareIngredients/);
  assert.match(s, /data-testid="ingredient-sort"/);
});

test("the list has a quick stock button that opens a window with Agregar and Contar", () => {
  const s = src("pages/IngredientsPage.tsx");
  assert.match(s, /data-testid="stock-quick"/);
  assert.ok(s.includes("+ Stock"));
  const files = ["components/QuickStockModal.tsx", "components/StockAdjustmentModal.tsx", "pages/IngredientsPage.tsx"].filter((f) => existsSync(join(root, "src", f)));
  const all = files.map((f) => readFileSync(join(root, "src", f), "utf8")).join("\n");
  assert.ok(all.includes("Agregar") && all.includes("Contar"));
  assert.match(all, /compra/, "the quick add records the reason compra");
});

test("the recipe form picks the unit from a list and shows the label with brand and supplier", () => {
  const s = src("components/RecipeForm.tsx");
  assert.match(s, /<select[^>]*aria-label="Unidad"/);
  assert.doesNotMatch(s, /placeholder="Und"/);
  assert.match(s, /ingredientLabel/);
  assert.match(s, /equivalenceNeeded/);
  assert.ok(s.includes("Peso de 1 litro") && s.includes("Peso de 1 unidad"));
});

test("the import template and the types carry the supplier", () => {
  assert.match(src("types/ingredient.ts") , /supplier/);
  assert.match(src("types/ingredient.ts"), /grams_per_liter/);
});
