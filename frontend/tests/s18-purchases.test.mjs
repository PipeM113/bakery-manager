// ADR-005: "+ Stock" becomes "Registrar compra": quantity, package size, price paid and supplier.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

const src = (p) => readFileSync(new URL(`../src/${p}`, import.meta.url), "utf8");

test("R1 the window registers a purchase with its quantity, package, price and supplier", () => {
  const s = src("components/QuickStockModal.tsx");
  assert.match(s, /Registrar compra/);
  for (const id of ["purchase-quantity", "purchase-package-size", "purchase-package-price", "purchase-supplier"]) {
    assert.match(s, new RegExp(`data-testid="${id}"`), id);
  }
  assert.match(s, /aria-label="Unidad de la compra"/);
  assert.match(s, /aria-label="Unidad del paquete"/);
});

test("R2 the package and the supplier start as they are now, and the new price per unit is shown against the old one", () => {
  const s = src("components/QuickStockModal.tsx");
  assert.match(s, /ingredient\.package_size/);
  assert.match(s, /ingredient\.package_price/);
  assert.match(s, /ingredient\.supplier/);
  assert.match(s, /data-testid="purchase-price-preview"/);
});

test("R3 the supplier is suggested from the ones already used", () => {
  const s = src("components/QuickStockModal.tsx");
  assert.match(s, /<datalist/);
  assert.match(s, /suppliers/);
});

test("R4 the request carries the detail of the purchase", () => {
  const types = src("types/stockAdjustment.ts");
  for (const f of ["supplier", "package_size", "package_unit", "package_price"]) {
    assert.match(types, new RegExp(`${f}\\??:`), f);
  }
  const s = src("components/QuickStockModal.tsx");
  assert.match(s, /package_price:/);
  assert.match(s, /reason: "compra"/);
});

test("R5 the list offers \"+ Compra\" where it said \"+ Stock\", and the form calls the supplier the last one", () => {
  const page = src("pages/IngredientsPage.tsx");
  assert.match(page, /\+ Compra/);
  assert.doesNotMatch(page, />\s*\+ Stock\s*</);
  assert.match(page, /Proveedor \(última compra\)/);
});

test("R6 the page passes the known suppliers to the window", () => {
  const page = src("pages/IngredientsPage.tsx");
  assert.match(page, /suppliers=\{/);
});
