// Sprint 15 (D5, F3, docs): refused deletes are shown, the recipe line is reachable by its label, and the rules are written down.
import { test } from "node:test";
import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const read = (p) => {
  const path = join(root, p);
  assert.ok(existsSync(path), `${p} does not exist`);
  return readFileSync(path, "utf8");
};

test("D5: the ingredient and recipe screens show a refused delete in an alert", () => {
  for (const page of ["src/pages/IngredientsPage.tsx", "src/pages/RecipesPage.tsx"]) {
    const s = read(page);
    assert.match(s, /role="alert"/, `${page} has no role="alert" for the refusal`);
    assert.match(s, /apiErrorMessage/, `${page} does not read the message of the error`);
    assert.doesNotMatch(s, /await (ingredientService|recipeService)\.delete\([^)]*\);\s*\n\s*load\(\);/, `${page} still ignores a failed delete`);
  }
});

test("F3: the ingredient and the unit of a recipe line have a label", () => {
  const s = read("src/components/RecipeForm.tsx");
  assert.match(s, /<select\s+aria-label="Ingrediente"/);
  assert.match(s, /<select\s+aria-label="Unidad"/);
});

test("the delete rules and the sprint are written down", () => {
  const rules = read("../docs/ingredientes.md");
  for (const word of ["409", "ventas", "cotizaciones", "recetas", "Eliminar"]) {
    assert.ok(rules.includes(word), `docs/ingredientes.md does not mention "${word}"`);
  }
  assert.match(rules, /D1/);
  assert.match(rules, /P2|bucket/i, "the photos rule (P2) is not written down");
  assert.match(read("../docs/backlog.md"), /\|\s*15\s*\|/);
});
