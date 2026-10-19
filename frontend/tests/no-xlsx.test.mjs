// Sprint 13 (C12, decided by the owner): the file preview of the ingredient import is gone, and with it
// the xlsx package (two high advisories without a fixed version on npm, docs/fuentes.md). The server
// reads and validates the file.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const read = (p) => readFileSync(join(root, p), "utf8");

test("the xlsx package is not a dependency any more", () => {
  const pkg = JSON.parse(read("package.json"));
  assert.equal(pkg.dependencies?.xlsx, undefined);
  assert.equal(pkg.devDependencies?.xlsx, undefined);
  assert.doesNotMatch(read("package-lock.json"), /"node_modules\/xlsx"/);
});

test("the import window sends the file to the server without reading it", () => {
  const s = read("src/components/IngredientImportModal.tsx");
  assert.doesNotMatch(s, /from ["']xlsx["']/);
  assert.doesNotMatch(s, /XLSX\./);
  assert.doesNotMatch(s, /FileReader/);
  assert.match(s, /\/ingredients\/import/);
  assert.match(s, /FormData/);
});

test("the docs say the preview is gone", () => {
  const f = read("../docs/fuentes.md");
  assert.match(f, /vista previa/i);
  assert.doesNotMatch(read("../docs/backlog.md"), /Decisión del propietario\):\*\* dos avisos/);
});
