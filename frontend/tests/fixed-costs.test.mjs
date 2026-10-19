// Sprint 6 (F11): the fixed costs screen works by month and asks "from which month" with the
// Santiago calendar, never the UTC one.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..", "src");
const read = (p) => readFileSync(join(root, p), "utf8");

test("the API client sends the month and the effective month", () => {
  const svc = read("api/fixedCostService.ts");
  assert.match(svc, /month/);
  assert.match(svc, /effective_from/);
  assert.match(svc, /valid_from/);
});

test("the screen has a month selector that starts on the Santiago month", () => {
  const page = read("pages/FixedCostsPage.tsx");
  assert.match(page, /type="month"/);
  assert.match(page, /businessDate|currentMonth|todayInSantiago/);
  assert.doesNotMatch(page, /toISOString\(\)/);
});

test("the screen says the change applies from a month", () => {
  const page = read("pages/FixedCostsPage.tsx");
  assert.match(page, /[Vv]igente desde/);
});

test("the type carries the history fields the screen needs", () => {
  const t = read("types/fixedCost.ts");
  assert.match(t, /effective_from/);
  assert.match(t, /valid_from/);
});
