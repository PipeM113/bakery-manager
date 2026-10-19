// Sprint 13 (C7): one way of writing pesos, everywhere.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");

const walk = (dir) =>
  readdirSync(dir).flatMap((f) => {
    const p = join(dir, f);
    return statSync(p).isDirectory() ? walk(p) : [p];
  });

test("formatCLP writes whole pesos with dots, and the minus before the $", async () => {
  const { formatCLP } = await import("../src/utils/money.ts");
  const cases = [
    [0, "$0"], [5, "$5"], [999, "$999"], [1234, "$1.234"], [12345, "$12.345"], [1234567, "$1.234.567"],
    [-271000, "-$271.000"], [-500, "-$500"], [1499.6, "$1.500"], [1499.4, "$1.499"], [-0.4, "$0"],
  ];
  for (const [n, want] of cases) assert.equal(formatCLP(n), want, `formatCLP(${n})`);
});

test("formatCLP does not hide a value that is not a number", async () => {
  const { formatCLP } = await import("../src/utils/money.ts");
  for (const bad of [NaN, Infinity, undefined, null]) assert.equal(formatCLP(bad), "—", String(bad));
});

test("no screen keeps its own copy of the formatter", () => {
  const offenders = [];
  for (const file of walk(join(root, "src")).filter((f) => /\.(ts|tsx)$/.test(f))) {
    const text = readFileSync(file, "utf8");
    if (/const clp\s*=/.test(text)) offenders.push(`${file}: defines its own clp`);
    if (/toLocaleString\(\s*["']es-CL["']/.test(text) && !file.endsWith(join("utils", "money.ts"))) offenders.push(`${file}: formats numbers by itself`);
  }
  assert.deepEqual(offenders, []);
});
