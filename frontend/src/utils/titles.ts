/** The name of each screen, for the title of the tab (a person with several tabs open, and a screen reader, need it). */
export const ROUTE_TITLES: Record<string, string> = {
  "/dashboard": "Panel",
  "/ingredients": "Ingredientes e insumos",
  "/recipes": "Recetas",
  "/costs": "Costos",
  "/fixed-costs": "Costos fijos",
  "/sales": "Ventas",
  "/expenses": "Gastos",
  "/profitability": "Rentabilidad",
  "/business": "Datos del negocio",
  "/account": "Mi cuenta",
};

const BRAND = "Angeles Bakery";

/** "Recetas · Angeles Bakery"; the brand title for a path that is not a screen. */
export function titleFor(pathname: string): string {
  const path = pathname.length > 1 ? pathname.replace(/\/+$/, "") : pathname;
  const name = ROUTE_TITLES[path];
  return name ? `${name} · ${BRAND}` : `${BRAND} · Gestión`;
}
