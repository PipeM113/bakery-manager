// Sprint 16: types of recipe, search and cost in the list (U1 to U5) and the documents (docs).
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
const r = (name, category_id = null, category = "") => ({ id: name, name, category_id, category });

test("groupRecipes: with no type in use there is one flat group in alphabetical order", async () => {
  const { groupRecipes } = await import("../src/utils/recipes.ts");
  const out = groupRecipes([r("Pan"), r("brownie"), r("Álamo")]);
  assert.equal(out.hasTypes, false);
  assert.equal(out.groups.length, 1);
  assert.deepEqual(out.groups[0].recipes.map((x) => x.name), ["Álamo", "brownie", "Pan"]);
});

test("groupRecipes: types in alphabetical order without accents, 'Sin tipo' last, recipes sorted inside", async () => {
  const { groupRecipes } = await import("../src/utils/recipes.ts");
  const out = groupRecipes([r("Pavlova", "m", "Merengue"), r("Alfajor", "h", "Hojarasca"), r("Beso", "m", "Merengue"), r("Pan"), r("Éclair", "b", "Álamo")]);
  assert.equal(out.hasTypes, true);
  assert.deepEqual(out.groups.map((g) => g.name), ["Álamo", "Hojarasca", "Merengue", "Sin tipo"]);
  assert.deepEqual(out.groups[2].recipes.map((x) => x.name), ["Beso", "Pavlova"]);
  assert.equal(out.groups[3].id, null);
});

test("groupRecipes: when every recipe has a type there is no 'Sin tipo' group", async () => {
  const { groupRecipes } = await import("../src/utils/recipes.ts");
  const out = groupRecipes([r("Alfajor", "h", "Hojarasca"), r("Beso", "m", "Merengue")]);
  assert.deepEqual(out.groups.map((g) => g.name), ["Hojarasca", "Merengue"]);
});

test("recipeMatches: by name or by type, without case or accents, ignoring the spaces around", async () => {
  const { recipeMatches } = await import("../src/utils/recipes.ts");
  assert.equal(recipeMatches(r("Pie de limón"), "LIMON"), true);
  assert.equal(recipeMatches(r("Pie de limón"), "  pie "), true);
  assert.equal(recipeMatches(r("Alfajor", "h", "Hojarasca"), "hoja"), true);
  assert.equal(recipeMatches(r("Alfajor", "h", "Hojarasca"), "merengue"), false);
  assert.equal(recipeMatches(r("Alfajor"), ""), true);
  assert.equal(recipeMatches(r("Alfajor"), "zzz"), false);
});

test("the service of the types talks to /recipe-categories", () => {
  const s = read("src/api/recipeCategoryService.ts");
  assert.match(s, /\/recipe-categories/);
  for (const verb of ["get", "post", "put", "delete"]) assert.match(s, new RegExp(`api\\.${verb}`));
});

test("the recipe has its type, its cost and its price in the types", () => {
  const s = read("src/types/recipe.ts");
  for (const field of ["category_id", "category", "total_cost", "suggested_price"]) assert.ok(s.includes(field), `IRecipe has no ${field}`);
});

test("U1, U2: the list has the search, the cards, the groups and the chips", () => {
  const s = read("src/components/RecipeList.tsx") + read("src/pages/RecipesPage.tsx");
  for (const id of ["recipe-search", "recipe-card", "type-group", "type-chips"]) assert.ok(s.includes(`data-testid="${id}"`), `no data-testid="${id}"`);
  assert.ok(s.includes("Costo total") && s.includes("Precio sugerido") && s.includes("Sin costo"));
  assert.match(s, /groupRecipes/);
});

test("U3: the form has the optional type with the way to create one", () => {
  const s = read("src/components/RecipeForm.tsx");
  assert.match(s, /<select\s+aria-label="Tipo"/);
  for (const text of ["Sin tipo", "+ Crear tipo…", "Nombre del tipo"]) assert.ok(s.includes(text), `the form has no "${text}"`);
});

test("U4: the window that manages the types uses Modal and shows its errors in an alert", () => {
  const s = read("src/components/RecipeCategoriesModal.tsx");
  assert.match(s, /<Modal/);
  assert.match(s, /role="alert"/);
  assert.match(s, /data-testid="type-row"/);
  assert.match(s, /data-primary-action/);
  for (const text of ["Renombrar", "Eliminar", "sin tipo"]) assert.ok(s.includes(text), `no "${text}"`);
  assert.match(read("src/pages/RecipesPage.tsx"), /RecipeCategoriesModal/);
});

test("U5: the four selectors of recipes use the grouped options", () => {
  assert.ok(existsSync(join(root, "src/components/RecipeOptions.tsx")), "src/components/RecipeOptions.tsx does not exist");
  for (const file of ["src/pages/CostsPage.tsx", "src/pages/SalesPage.tsx", "src/components/OrderForm.tsx", "src/components/NewQuotationModal.tsx"]) {
    assert.match(read(file), /RecipeOptions/, `${file} does not use RecipeOptions`);
  }
});

test("the rules, the deploy step and the sprint are written down", () => {
  const doc = read("../docs/recetario.md");
  for (const word of ["T1", "T3", "L1", "U2", "Sin tipo", "recipe-categories", "quedan sin tipo"]) assert.ok(doc.includes(word), `docs/recetario.md does not mention "${word}"`);
  const deploy = read("../docs/despliegue.md");
  assert.match(deploy, /29/);
  assert.match(deploy, /recipe_categories/);
  assert.match(read("../docs/backlog.md"), /\|\s*16\s*\|/);
});
