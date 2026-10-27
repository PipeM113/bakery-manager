interface Named {
  name: string;
  brand?: string;
  supplier?: string;
}

/** "Mantequilla · Soprole · La Vega": name, brand and supplier, leaving out what is empty. */
export const ingredientLabel = (i: Named): string =>
  [i.name, i.brand, i.supplier].map((x) => (x ?? "").trim()).filter(Boolean).join(" · ");

/** The text as it is compared when sorting: lowercase and without accents. */
export const sortKey = (s: string): string =>
  s.trim().normalize("NFD").replace(/\p{Mn}/gu, "").toLowerCase();

/** Alphabetical by name without case or accents, then by brand, then by supplier. */
export const compareIngredients = (a: Named, b: Named): number =>
  sortKey(a.name).localeCompare(sortKey(b.name)) ||
  sortKey(a.brand ?? "").localeCompare(sortKey(b.brand ?? "")) ||
  sortKey(a.supplier ?? "").localeCompare(sortKey(b.supplier ?? ""));
