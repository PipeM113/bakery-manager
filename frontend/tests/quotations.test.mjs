// Sprint 12 (Q1–Q12, front side): quotations with several products and a PDF made from what was saved.
import { test } from "node:test";
import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const src = (p) => readFileSync(join(root, "src", p), "utf8");
const docs = (p) => readFileSync(join(root, "..", "docs", p), "utf8");

test("the service creates, reads and downloads by id; the old one-shot endpoint is gone", () => {
  const s = src("api/quotationService.ts");
  assert.match(s, /async create\(/);
  assert.match(s, /api\.post\(\s*["']\/quotations["']/);
  assert.match(s, /async downloadPDF\(/);
  assert.match(s, /\/quotations\/\$\{id\}\/pdf/);
  assert.doesNotMatch(s, /quotations\/generate/);
  assert.doesNotMatch(s, /generatePDF/);
});

test("a quotation has lines, a delivery and a total", () => {
  const s = src("api/quotationService.ts");
  assert.match(s, /interface IQuotationLine/);
  for (const field of ["lines", "delivery_cost", "total", "recipe_name", "quantity", "unit_price", "line_total", "sold_as"]) {
    assert.match(s, new RegExp(`\\b${field}\\b`), `IQuotation misses ${field}`);
  }
  assert.doesNotMatch(s, /final_price/);
});

test("the new-quotation window has lines, add-line, delivery and the main button", () => {
  const path = join(root, "src/components/NewQuotationModal.tsx");
  assert.ok(existsSync(path), "components/NewQuotationModal.tsx does not exist");
  const s = readFileSync(path, "utf8");
  assert.match(s, /MAX_LINES = 30/);
  assert.match(s, /data-testid="quote-line"/);
  assert.match(s, /data-testid="quote-add-line"/);
  assert.match(s, /data-testid="quote-delivery"/);
  assert.match(s, /aria-label="Producto"/);
  assert.match(s, /aria-label="Cantidad"/);
  assert.match(s, /Nombre del cliente/);
  assert.match(s, /Crear y descargar PDF/);
  assert.match(s, /Modal/);
  assert.match(s, /quotationService\.create/);
  assert.match(s, /quotationService\.downloadPDF/);
});

test("Ventas offers 'Nueva cotización' and a download button on every quotation", () => {
  const s = src("pages/SalesPage.tsx");
  assert.match(s, /Nueva cotización/);
  assert.match(s, /NewQuotationModal/);
  assert.match(s, /data-testid="quote-download"/);
  assert.match(s, /quotationService\.downloadPDF/);
  assert.match(s, /\.total/);
  assert.doesNotMatch(s, /final_price/);
});

test("the generator of the costs screen quotes through the saved quotation", () => {
  const s = src("components/QuotationGenerator.tsx");
  assert.match(s, /quotationService\.create/);
  assert.match(s, /quotationService\.downloadPDF/);
  assert.doesNotMatch(s, /generatePDF/);
});

test("the docs explain the quotation and the business data to configure", () => {
  const q = docs("cotizaciones.md");
  for (const word of ["BIZ_NAME", "BIZ_PHONE", "BIZ_INSTAGRAM", "BIZ_BANK", "BIZ_ACCOUNT", "BIZ_HOLDER", "BIZ_RUT"]) {
    assert.ok(q.includes(word), `docs/cotizaciones.md does not mention ${word}`);
    assert.ok(docs("despliegue.md").includes(word), `docs/despliegue.md does not mention ${word}`);
  }
  assert.doesNotMatch(docs("pedidos.md"), /Cotizaciones con varios productos \(van con el PDF/);
});
