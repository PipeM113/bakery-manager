// Sprint 5 (T1/T2, support test): the screens take "today" and "this month" from Santiago, not
// from toISOString(), which is UTC (after 21:00 in Chile it is already tomorrow there).
import { test } from "node:test";
import assert from "node:assert/strict";
import { existsSync, readdirSync, readFileSync, statSync } from "node:fs";
import { join, dirname, relative } from "node:path";
import { fileURLToPath } from "node:url";

const frontendDir = join(dirname(fileURLToPath(import.meta.url)), "..");
function sources(dir) {
  return readdirSync(dir).flatMap((name) => {
    const full = join(dir, name);
    if (statSync(full).isDirectory()) return sources(full);
    return /\.(ts|tsx)$/.test(name) ? [full] : [];
  });
}

test("the date helper uses the America/Santiago time zone", () => {
  const file = join(frontendDir, "src/utils/businessDate.ts");
  assert.ok(existsSync(file), "src/utils/businessDate.ts is missing");
  assert.match(readFileSync(file, "utf8"), /America\/Santiago/);
});

test("no screen builds a date from toISOString()", () => {
  for (const f of sources(join(frontendDir, "src"))) {
    assert.doesNotMatch(readFileSync(f, "utf8"), /toISOString\(\)\s*\.slice\(0,\s*10\)/, `${relative(frontendDir, f)} uses the UTC date`);
  }
});
