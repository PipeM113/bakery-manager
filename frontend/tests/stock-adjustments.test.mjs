// Sprint 8 (A10): the ingredients screen adjusts stock with a reason and shows the history, and the
// profitability screen shows the month's waste.
import { test } from "node:test";
import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..", "src");
const read = (p) => readFileSync(join(root, p), "utf8");

test("there is an API client for stock adjustments", () => {
  assert.ok(existsSync(join(root, "api/stockAdjustmentService.ts")), "api/stockAdjustmentService.ts is missing");
  const svc = read("api/stockAdjustmentService.ts");
  assert.match(svc, /stock-adjustments/);
  assert.match(svc, /counted_quantity/);
  assert.match(svc, /quantity_delta/);
});

test("the ingredients screen offers 'Ajustar stock', and the window has the three reasons", () => {
  const page = read("pages/IngredientsPage.tsx");
  assert.match(page, /Ajustar stock/);
  assert.match(page, /StockAdjustmentModal/);
  const modal = read("components/StockAdjustmentModal.tsx");
  assert.match(modal, /merma/i);
  assert.match(modal, /correcci[óo]n/i);
  assert.match(modal, /otro/i);
  assert.match(modal, /stockAdjustmentService/);
});

test("the profitability screen shows the waste of the month", () => {
  assert.match(read("api/analyticsService.ts"), /total_waste_cost/);
  const page = read("pages/ProfitabilityPage.tsx");
  assert.match(page, /total_waste_cost/);
  assert.match(page, /Mermas/);
});
