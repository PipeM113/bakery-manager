// A fake backend for the responsive suite: the screens get realistic, deliberately awkward data
// (long names, many rows, big numbers) without needing the real API.
const iso = "2026-03-10T15:00:00.000Z";

const ingredientNames = [
  "Manjar Colun Receta Tradicional Balde Grande", "Harina sin polvos de hornear marca Selecta", "Azúcar flor",
  "Mantequilla sin sal Soprole 250 gr", "Huevos de campo extra grandes bandeja", "Leche entera líquida",
  "Esencia de vainilla concentrada", "Chocolate cobertura semiamargo", "Crema de leche vegetal para montar",
  "Frambuesas congeladas enteras", "Limón de Pica", "Cacao amargo en polvo",
];
const supplyNames = ["Caja de cartón para torta 30x30", "Base de cartón dorada 28 cm"];

export const ingredients = [
  ...ingredientNames.map((name, i) => ({
    id: `00000000-0000-4000-8000-0000000001${String(i).padStart(2, "0")}`, kind: "ingredient", name,
    default_unit: ["kg", "gr", "gr", "gr", "und", "lt", "ml", "gr", "ml", "gr", "und", "gr"][i],
    package_size: i % 2 ? 1000 : 1, package_price: 1990 + i * 733, price_per_unit: 1.99 + i,
    stock_quantity: i === 3 ? 20 : 500 + i * 37.5, alert_threshold: i === 3 ? 100 : 0,
    brand: ["Colun", "Selecta", "", "Soprole", "Campo", "Loncoleche"][i % 6], created_at: iso, updated_at: iso,
  })),
  ...supplyNames.map((name, i) => ({
    id: `00000000-0000-4000-8000-0000000002${String(i).padStart(2, "0")}`, kind: "supply", name,
    default_unit: "und", package_size: 10, package_price: 8500, price_per_unit: 850,
    stock_quantity: 40, alert_threshold: 10, brand: "Cartonera", created_at: iso, updated_at: iso,
  })),
];

const recipeLine = (recipeId, ing, quantity, unit) => ({
  id: `${recipeId}-${ing.id.slice(-4)}`, recipe_id: recipeId, ingredient_id: ing.id, name: ing.name,
  kind: ing.kind, quantity, unit,
});

const recipe = (n, name, yieldN) => {
  const id = `00000000-0000-4000-8000-0000000003${String(n).padStart(2, "0")}`;
  return {
    id, user_id: "u1", parent_id: null, name, description: "Bizcocho húmedo relleno y cubierto, con decoración.",
    yield: yieldN, yield_unit: "porciones", photo_url: null, is_base: true, scale_factor: 1,
    indirect_cost_pct: 0.3, labor_cost_pct: 0.3, margin_pct: 0.3, extra_charge: 0, delivery_cost: 0,
    ingredients: [
      recipeLine(id, ingredients[0], 200, "gr"), recipeLine(id, ingredients[1], 0.5, "kg"),
      recipeLine(id, ingredients[3], 250, "gr"), recipeLine(id, ingredients[12], 1, "und"),
    ],
    created_at: iso, updated_at: iso,
  };
};

export const recipes = [
  recipe(1, "Torta de milhojas con manjar y nueces", 20), recipe(2, "Pie de limón", 8), recipe(3, "Brownie", 12),
];

export const cost = (r) => ({
  recipe_id: r.id, recipe_name: r.name, yield: r.yield, yield_unit: r.yield_unit,
  ingredients: r.ingredients.map((l) => ({
    ingredient_id: l.ingredient_id, name: l.name, kind: l.kind, quantity: l.quantity, unit: l.unit,
    price_per_unit: 12.5, subtotal: 12.5 * l.quantity,
  })),
  ingredients_total: 12345678, supplies_total: 8500, indirect_costs: 3703703, labor_costs: 3703703,
  total_cost: 19761584, margin_pct: 0.3, cost_per_portion: 988079, base_price: 25690059,
  extra_charge: 0, delivery_cost: 0, suggested_price: 25690500,
  indirect_cost_pct: 0.3, labor_cost_pct: 0.3,
});

const clientNames = ["María González Pérez de la Fuente", "Cumpleaños de Sofía", "Evento corporativo Banco del Sur Ltda."];
export const quotations = Array.from({ length: 8 }, (_, i) => {
  const lines = [0, 1].slice(0, 1 + (i % 2)).map((k) => ({
    id: `00000000-0000-4000-8000-0000000006${i}${k}`, position: k + 1, recipe_id: recipes[(i + k) % 3].id,
    recipe_name: recipes[(i + k) % 3].name, sold_as: "recipe", quantity: 1 + k, unit_price: 25000, line_total: 25000 * (1 + k),
    margin_pct: 0.3, indirect_cost_pct: 0.3, labor_cost_pct: 0.3, extra_charge: 0,
  }));
  const delivery = i % 2 ? 2500 : 0;
  return {
    id: `00000000-0000-4000-8000-0000000004${String(i).padStart(2, "0")}`, created_by: "u1",
    client_name: clientNames[i % 3], delivery_cost: delivery, lines,
    total: lines.reduce((a, l) => a + l.line_total, 0) + delivery,
    status: ["pending", "pending", "confirmed", "cancelled"][i % 4], created_at: iso, updated_at: iso,
  };
});

export const sales = Array.from({ length: 6 }, (_, i) => ({
  id: `00000000-0000-4000-8000-0000000005${String(i).padStart(2, "0")}`, user_id: "u1",
  recipe_id: recipes[i % 3].id, recipe_name: recipes[i % 3].name, quantity_sold: 2 + i, unit_price: 18000,
  total_price: (2 + i) * 18000, sold_as: i % 2 ? "portion" : "recipe", sale_date: "2026-03-0" + (i + 1),
  notes: "Entrega a domicilio", ingredients: [], stock_shortfalls: [], created_at: iso,
}));

export const orders = Array.from({ length: 5 }, (_, i) => ({
  id: `00000000-0000-4000-8000-0000000008${String(i).padStart(2, "0")}`, user_id: "u1",
  customer_name: ["María González Pérez de la Fuente", "Cumpleaños de Sofía", "Evento corporativo Banco del Sur Ltda."][i % 3],
  notes: "Entrega el sábado en la mañana", order_date: "2026-03-1" + i, total_price: 45000 + i * 5000,
  lines: [
    { sale_id: `s-${i}-a`, recipe_id: recipes[0].id, recipe_name: recipes[0].name, quantity_sold: 1, sold_as: "recipe", unit_price: 30000, line_total: 30000 },
    { sale_id: `s-${i}-b`, recipe_id: recipes[1].id, recipe_name: recipes[1].name, quantity_sold: 6, sold_as: "portion", unit_price: 3000, line_total: 15000 + i * 5000 },
  ],
  stock_shortfalls: [], created_at: iso,
}));

export const fixedCosts = ["Arriendo del local", "Luz", "Gas licuado", "Contador"].map((name, i) => ({
  id: `00000000-0000-4000-8000-0000000006${String(i).padStart(2, "0")}`, user_id: "u1", name,
  monthly_amount: 150000 * (i + 1), is_active: i !== 3, valid_from: "2026-01-01", ended_from: null,
  created_at: iso, updated_at: iso,
}));

export const categories = ["Gas", "Envases", "Transporte"].map((name, i) => ({ id: `cat-${i}`, name }));
export const expenses = Array.from({ length: 6 }, (_, i) => ({
  id: `00000000-0000-4000-8000-0000000007${String(i).padStart(2, "0")}`, user_id: "u1",
  description: "Compra de gas para el horno industrial del local", amount: 15000 + i * 2500,
  category_id: categories[i % 3].id, category: categories[i % 3].name, expense_date: "2026-03-0" + (i + 1),
  created_at: iso, updated_at: iso,
}));

export const monthly = {
  month: 3, year: 2026, total_revenue: 12345678, total_ingredients_cost: 3456789, total_fixed_costs: 1200000,
  total_operational_expenses: 456789, total_waste_cost: 25000, gross_profit: 8888889, net_profit: -271000,
  profit_margin: -2.2, expenses_by_category: [{ category_id: "cat-0", category: "Gas", total: 456789 }],
};
export const trends = ["2025-10", "2025-11", "2025-12", "2026-01", "2026-02", "2026-03"].map((date, i) => ({
  date, revenue: 900000 + i * 350000, net_profit: 300000 + i * 120000 - (i === 2 ? 450000 : 0),
}));
export const recipeMetrics = recipes.map((r, i) => ({
  recipe_id: r.id, recipe_name: r.name, units_sold: 10 + i, total_revenue: 250000 + i * 90000,
  cost_per_unit: 9000 + i * 1000, total_cost: 90000 + i * 20000, unit_profit: 16000, gross_margin: 64.5,
}));

// A tiny but valid PDF.
export const pdfBytes = Buffer.from("%PDF-1.1\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj\n3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 200 200]>>endobj\ntrailer<</Root 1 0 R>>\n%%EOF");

export const businessProfile = {
  name: "Panadería de Prueba", phone: "+56 9 1111 2222", instagram: "@prueba", bank: "Banco Prueba", account_type: "Cuenta Corriente", holder: "Titular Prueba", email: "pagos@prueba.test", missing: [],
};

/** Answers every call the screens make; returns undefined for what it does not know. */
export function answer(method, pathname) {
  const m = (re) => pathname.match(re);
  if (method === "POST" && pathname === "/quotations") return { body: quotations[0] };
  if (method === "PUT" && pathname === "/business") return { body: businessProfile };
  if (method === "GET" && m(/^\/quotations\/[^/]+\/pdf$/)) return { pdf: true };
  if (method === "POST" && pathname === "/auth/login") return { body: { token: "x" } };
  if (method === "POST" && pathname === "/orders") return { body: orders[0] };
  if (method !== "GET") return { body: {} };
  if (pathname === "/ingredients") return { body: ingredients };
  if (m(/^\/ingredients\/[^/]+\/(history|stock-adjustments)$/)) return { body: [] };
  if (pathname === "/recipes") return { body: recipes };
  if (m(/^\/recipes\/[^/]+\/cost$/)) return { body: cost(recipes.find((r) => pathname.includes(r.id)) ?? recipes[0]) };
  if (m(/^\/recipes\/[^/]+$/)) return { body: recipes.find((r) => pathname.endsWith(r.id)) ?? recipes[0] };
  if (pathname === "/quotations") return { body: quotations };
  if (pathname === "/business") return { body: businessProfile };
  if (pathname === "/sales") return { body: sales };
  if (pathname === "/orders") return { body: orders };
  if (pathname === "/fixed-costs") return { body: fixedCosts };
  if (pathname === "/expense-categories") return { body: categories };
  if (pathname === "/expenses") return { body: expenses };
  if (pathname === "/stock-adjustments") return { body: [] };
  if (pathname === "/analytics/monthly") return { body: monthly };
  if (pathname === "/analytics/trends") return { body: trends };
  if (pathname === "/analytics/recipes") return { body: recipeMetrics };
  return undefined;
}
