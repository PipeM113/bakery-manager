// Sprint 2a, rules P6 and P9: the price is calculated in one place (the Go backend) and the
// recipe markup is called "Recargo sobre costo" on screen. Static checks over the sources.
//
// Run with: npm test
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
const read = (rel) => readFileSync(join(frontendDir, rel), "utf8");

test("P6: no price formula lives in the frontend", () => {
  for (const f of files) {
    assert.doesNotMatch(f.text, /ceilTo500/, `${f.path} rounds prices itself`);
    assert.doesNotMatch(f.text, /Math\.ceil\([^)]*500/, `${f.path} rounds prices to 500 itself`);
    assert.doesNotMatch(f.text, /\(1\s*\+\s*marginNum\)/, `${f.path} applies the markup itself`);
  }
});

test("P9: the duplicated subtotal_sin_margen field is not used any more", () => {
  for (const f of files) {
    assert.doesNotMatch(f.text, /subtotal_sin_margen/, f.path);
  }
});

test("P9: the recipe markup is called 'Recargo sobre costo' where prices are set", () => {
  const quotation = read("src/components/QuotationGenerator.tsx");
  assert.match(quotation, /Recargo sobre costo/);
  assert.doesNotMatch(quotation, /margen/i, "QuotationGenerator still says 'margen' for the recipe markup");
  assert.doesNotMatch(read("src/pages/SalesPage.tsx"), /margen/i, "SalesPage still says 'margen' for the recipe markup");
});

test("P9: 'Margen' stays for the dashboard figure (profit over sales)", () => {
  assert.match(read("src/pages/ProfitabilityPage.tsx"), /Margen de Ganancia/);
});

test("E4: the price for another number of portions is asked of the backend", () => {
  const quotation = read("src/components/QuotationGenerator.tsx");
  assert.match(quotation, /portions:\s*String\(/, "the cost request should carry the portions");
  assert.match(quotation, /Precio final \(\{portionsNum\} porciones\)/);
  assert.match(quotation, /breakdown\.suggested_price/);
});
