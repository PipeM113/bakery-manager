// Sprint 12 (Q11): the data of the business that the quotation PDF prints is edited in the app.
import { test } from "node:test";
import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const src = (p) => readFileSync(join(root, "src", p), "utf8");

test("the service reads and saves the profile", () => {
  const path = join(root, "src/api/businessService.ts");
  assert.ok(existsSync(path), "api/businessService.ts does not exist");
  const s = readFileSync(path, "utf8");
  assert.match(s, /api\.get[^(]*\(\s*["']\/business["']/);
  assert.match(s, /api\.put[^(]*\(\s*["']\/business["']/);
  assert.match(s, /missing/);
});

test("the screen has the five fields and a save button", () => {
  const path = join(root, "src/pages/BusinessPage.tsx");
  assert.ok(existsSync(path), "pages/BusinessPage.tsx does not exist");
  const s = readFileSync(path, "utf8");
  assert.match(s, /data-testid="business-form"/);
  for (const label of ["Nombre del negocio", "Teléfono", "Instagram", "Banco", "Titular de la cuenta"]) {
    assert.ok(s.includes(label), `no field "${label}"`);
  }
  assert.match(s, /Guardar/);
  assert.match(s, /BIZ_RUT|BIZ_ACCOUNT/, "it should tell that the RUT and the account are set in Vercel");
});

test("the route and both navigations lead to it", () => {
  assert.match(src("App.tsx"), /path="\/business"/);
  assert.match(src("components/layout/Sidebar.tsx"), /\/business/);
  assert.match(src("components/layout/Sidebar.tsx"), /Datos del negocio/);
  assert.match(src("components/layout/BottomNav.tsx"), /\/business/);
  assert.match(src("components/layout/BottomNav.tsx"), /Datos del negocio/);
});

test("the quotations screen warns when the PDF cannot be made yet", () => {
  const s = src("pages/SalesPage.tsx");
  assert.match(s, /data-testid="business-missing-notice"/);
  assert.match(s, /businessService/);
  assert.match(s, /to="\/business"|href="\/business"/);
});
