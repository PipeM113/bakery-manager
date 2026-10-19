// The one way pesos are written in the app: whole pesos, a dot every three digits, and the minus
// before the $ (-$271.000). The grouping is done by hand so that a number of four digits is
// grouped too (the Spanish locale of some browsers leaves 1234 as it is). Docs: docs/glosario-dinero.md.

/** Pesos as text. A value that is not a number is shown as "—" instead of hiding it as $0. */
export function formatCLP(value: number | null | undefined): string {
  if (value == null || !Number.isFinite(value)) return "—";
  const whole = Math.round(Math.abs(value));
  const grouped = String(whole).replace(/\B(?=(\d{3})+(?!\d))/g, ".");
  return `${value < 0 && whole > 0 ? "-" : ""}$${grouped}`;
}
