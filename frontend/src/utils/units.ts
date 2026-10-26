export const UNITS = ["gr", "kg", "ml", "lt", "und"] as const;

export interface Equivalences {
  grams_per_liter?: number;
  grams_per_unit?: number;
}

type Family = "mass" | "volume" | "count";

const familyOf = (unit: string): Family | null => {
  switch (unit) {
    case "gr": case "kg": return "mass";
    case "ml": case "lt": return "volume";
    case "und": return "count";
    default: return null;
  }
};

/** Units of the same family as `unit` first, then the rest: what a select offers. */
export const unitsFor = (unit: string): string[] => {
  const family = familyOf(unit);
  const same = UNITS.filter((u) => familyOf(u) === family);
  return [...same, ...UNITS.filter((u) => !same.includes(u))];
};

/**
 * Which equivalence is missing to use `from` where the ingredient is kept in `to`:
 * "grams_per_liter" (gr <-> ml), "grams_per_unit" (und <-> gr) or null when nothing is needed or
 * it is already known. Only asked when the unit changes family.
 */
export const equivalenceNeeded = (from: string, to: string, eq: Equivalences): "grams_per_liter" | "grams_per_unit" | null => {
  const a = familyOf(from);
  const b = familyOf(to);
  if (!a || !b || a === b) return null;
  for (const f of [a, b]) {
    if (f === "volume" && !(eq.grams_per_liter && eq.grams_per_liter > 0)) return "grams_per_liter";
    if (f === "count" && !(eq.grams_per_unit && eq.grams_per_unit > 0)) return "grams_per_unit";
  }
  return null;
};

export const equivalenceLabel = (need: "grams_per_liter" | "grams_per_unit"): string =>
  need === "grams_per_liter" ? "Peso de 1 litro (gr)" : "Peso de 1 unidad (gr)";
