// Sprint 14: the rules, the deploy step and the backlog are written down.
import { test } from "node:test";
import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const docs = join(dirname(fileURLToPath(import.meta.url)), "..", "..", "docs");
const doc = (f) => {
  assert.ok(existsSync(join(docs, f)), `docs/${f} does not exist`);
  return readFileSync(join(docs, f), "utf8");
};

test("docs/ingredientes.md explains the key, the order, the quick stock and the equivalences", () => {
  const s = doc("ingredientes.md");
  for (const word of ["Proveedor", "marca", "orden alfabético", "compra", "Peso de 1 litro", "Peso de 1 unidad", "densidad"]) {
    assert.ok(s.toLowerCase().includes(word.toLowerCase()), `ingredientes.md does not mention "${word}"`);
  }
  for (const rule of ["E1", "E2", "E3", "E4", "E5", "E6", "E7", "E8", "E9"]) assert.ok(s.includes(rule), `rule ${rule} is not listed`);
  assert.match(s, /no se inventa|sin fuente|no hay densidad/i, "it must say that no density is preloaded");
});

test("docs/despliegue.md says migration 28 goes before the deploy", () => {
  const s = doc("despliegue.md");
  assert.match(s, /28/);
  assert.match(s, /antes del despliegue|antes de desplegar|antes de mergear/i);
});

test("docs/backlog.md has sprint 14 in the table", () => {
  assert.match(doc("backlog.md"), /\|\s*14\s*\|/);
});
