// Sprint 11 (N7): the business is one; the screens know records by who created them, and the
// home screen greets the person who logged in.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, dirname, relative } from "node:path";
import { fileURLToPath } from "node:url";

const frontendDir = join(dirname(fileURLToPath(import.meta.url)), "..");
function sources(dir) {
  return readdirSync(dir).flatMap((n) => {
    const f = join(dir, n);
    return statSync(f).isDirectory() ? sources(f) : /\.(ts|tsx)$/.test(n) ? [f] : [];
  });
}

test("no screen or type still talks about user_id", () => {
  for (const f of sources(join(frontendDir, "src"))) {
    assert.doesNotMatch(readFileSync(f, "utf8"), /\buser_id\b/, `${relative(frontendDir, f)} still mentions user_id`);
  }
});

test("the main records carry created_by in their types", () => {
  const types = ["sale", "order", "expense", "fixedCost", "recipe", "stockAdjustment"]
    .map((n) => readFileSync(join(frontendDir, "src/types", `${n}.ts`), "utf8")).join("\n");
  assert.ok((types.match(/created_by/g) ?? []).length >= 6, "each record type should carry created_by");
});

test("the home screen says Hola, not Bienvenida", () => {
  const page = readFileSync(join(frontendDir, "src/pages/DashboardPage.tsx"), "utf8");
  assert.match(page, /Hola/);
  assert.doesNotMatch(page, /Bienvenida/);
});
