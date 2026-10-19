// Sprint 9 (R8): the responsive suite is one command away and has what it needs.
import { test } from "node:test";
import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const frontendDir = join(dirname(fileURLToPath(import.meta.url)), "..");
const pkg = JSON.parse(readFileSync(join(frontendDir, "package.json"), "utf8"));

test("npm run test:responsive runs the suite", () => {
  assert.match(pkg.scripts["test:responsive"] ?? "", /responsive\.e2e\.mjs/);
  assert.ok(existsSync(join(frontendDir, "tests/responsive/responsive.e2e.mjs")));
});

test("Playwright is a pinned dev dependency", () => {
  assert.match(pkg.devDependencies?.playwright ?? "", /^\d+\.\d+\.\d+$/);
});

test("the unit test command does not run the browser suite", () => {
  assert.doesNotMatch(pkg.scripts.test, /e2e/);
  assert.match(pkg.scripts.test, /\.test\.mjs/);
});

test("the screenshots are not committed", () => {
  assert.match(readFileSync(join(frontendDir, ".gitignore"), "utf8"), /^test-results$/m);
});
