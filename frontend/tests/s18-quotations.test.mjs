// Sprint 18: a quotation is made for a person or for a company, chosen when it is created.
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

const src = (p) => readFileSync(new URL(`../src/${p}`, import.meta.url), "utf8");

test("F1 the new quotation asks who it is for, a person by default, and says so in Spanish", () => {
  const s = src("components/NewQuotationModal.tsx");
  assert.match(s, /aria-label="Para"/);
  assert.match(s, /Persona/);
  assert.match(s, /Empresa/);
  assert.match(s, /useState<QuotationKind>\("person"\)/);
});

test("F2 the conditions are asked only for a company: delivery date, way of payment and days of validity", () => {
  const s = src("components/NewQuotationModal.tsx");
  assert.match(s, /data-testid="quote-delivery-date"/);
  assert.match(s, /data-testid="quote-payment-terms"/);
  assert.match(s, /data-testid="quote-valid-days"/);
  assert.match(s, /kind === "company"/);
});

test("F3 the request carries the kind and, for a company only, its conditions", () => {
  const s = src("api/quotationService.ts");
  assert.match(s, /export type QuotationKind = "person" \| "company"/);
  assert.match(s, /kind\??: QuotationKind/);
  assert.match(s, /delivery_date\??: string/);
  assert.match(s, /payment_terms\??: string/);
  assert.match(s, /valid_days\??: number/);
});

test("F4 the list tells which quotations are for a company", () => {
  const s = src("pages/SalesPage.tsx");
  assert.match(s, /data-testid="quote-kind"/);
  assert.match(s, /q\.kind === "company"/);
});
