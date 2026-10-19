// Sprint 13 (C8, C9): no dead files and a lint that passes.
import { test } from "node:test";
import assert from "node:assert/strict";
import { existsSync, readFileSync, readdirSync, statSync } from "node:fs";
import { spawnSync } from "node:child_process";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const pkg = JSON.parse(readFileSync(join(root, "package.json"), "utf8"));

const walk = (dir) =>
  readdirSync(dir).flatMap((f) => {
    const p = join(dir, f);
    return statSync(p).isDirectory() ? walk(p) : [p];
  });

test("src/App.css is gone and nothing imports it", () => {
  assert.equal(existsSync(join(root, "src/App.css")), false);
  for (const file of walk(join(root, "src")).filter((f) => /\.(ts|tsx|css)$/.test(f))) {
    assert.doesNotMatch(readFileSync(file, "utf8"), /App\.css/, file);
  }
});

test("npm run lint exists and finds no errors", { timeout: 180000 }, () => {
  assert.ok(pkg.scripts.lint, "package.json has no lint script");
  const run = spawnSync("npx", ["eslint", "."], { cwd: root, encoding: "utf8" });
  const errors = (run.stdout.match(/\berror\b/g) ?? []).length;
  assert.equal(run.status, 0, `eslint found errors:\n${run.stdout.split("\n").filter((l) => /error/.test(l)).slice(0, 12).join("\n")}`);
  assert.equal(errors, 0);
});
