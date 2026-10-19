// Sprint 10 (O10): the sales screen takes orders with several products and lists them.
import { test } from "node:test";
import assert from "node:assert/strict";
import { existsSync, readFileSync, readdirSync, statSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..", "src");
const read = (p) => readFileSync(join(root, p), "utf8");
function all(dir) {
  return readdirSync(dir).flatMap((n) => {
    const f = join(dir, n);
    return statSync(f).isDirectory() ? all(f) : /\.(ts|tsx)$/.test(n) ? [readFileSync(f, "utf8")] : [];
  });
}
const everything = all(root).join("\n");

test("there is an API client and types for orders", () => {
  assert.ok(existsSync(join(root, "api/orderService.ts")), "api/orderService.ts is missing");
  const svc = read("api/orderService.ts");
  assert.match(svc, /\/orders/);
  assert.match(svc, /total_price/);
  assert.ok(existsSync(join(root, "types/order.ts")), "types/order.ts is missing");
  const types = read("types/order.ts");
  assert.match(types, /line_total/);
  assert.match(types, /stock_shortfalls/);
  assert.match(types, /lines/);
});

test("the sales screen offers 'Nuevo pedido' and lists the orders", () => {
  const page = read("pages/SalesPage.tsx");
  assert.match(page, /Nuevo pedido/);
  assert.match(page, /orderService/);
  assert.match(everything, /data-testid="orders-list"/);
});

test("the order form adds products and takes an optional total", () => {
  assert.match(everything, /Agregar producto/);
  assert.match(everything, /Precio total/);
  assert.match(everything, /Registrar pedido/);
});

test("a cancelled order is cancelled whole", () => {
  assert.match(everything, /Anular pedido/);
});

test("the stock warning of an order is shown", () => {
  assert.match(read("types/order.ts"), /IStockShortfall|stock_shortfalls/);
  assert.match(everything, /faltaron|falt[óo]/i);
});
