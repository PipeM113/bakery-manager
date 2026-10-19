// Sprint 2b, rules D3 and D4: the expense categories come from the backend, they are not a
// fixed list in the screen.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, dirname, relative } from "node:path";
import { fileURLToPath } from "node:url";

const frontendDir = join(dirname(fileURLToPath(import.meta.url)), "..");
const srcDir = join(frontendDir, "src");

function sources(dir) {
  return readdirSync(dir).flatMap((name) => {
    const full = join(dir, name);
    if (statSync(full).isDirectory()) return sources(full);
    return /\.(ts|tsx)$/.test(name) ? [full] : [];
  });
}
const files = sources(srcDir).map((f) => ({ path: relative(frontendDir, f), text: readFileSync(f, "utf8") }));

test("D3: the screens read the categories from /expense-categories", () => {
  assert.ok(files.some((f) => f.text.includes("/expense-categories")), "nothing calls /expense-categories");
});

test("D3: the expenses screen has no fixed list of categories", () => {
  const page = files.find((f) => f.path.endsWith("pages/ExpensesPage.tsx"));
  assert.ok(page, "ExpensesPage.tsx not found");
  assert.doesNotMatch(page.text, /["'`]ingredientes["'`]/, "'ingredientes' is still hardcoded");
  assert.doesNotMatch(page.text, /["'`]servicios["'`]/, "'servicios' is still hardcoded");
  assert.match(page.text, /category_id/, "the expense is not sent with category_id");
});
