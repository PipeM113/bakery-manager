import { useState } from "react";
import { stockAdjustmentService } from "../api/stockAdjustmentService";
import type { IIngredient } from "../types/ingredient";
import type { StockAdjustmentForm } from "../types/stockAdjustment";
import { todayInSantiago } from "../utils/businessDate";
import { apiErrorMessage } from "../utils/apiError";
import { formatCLP } from "../utils/money";
import { ingredientLabel } from "../utils/ingredients";
import Modal from "./Modal";

const num = (n: number) => (Number.isInteger(n) ? String(n) : n.toFixed(3).replace(/0+$/, "").replace(/\.$/, ""));
const UNITS_OF: Record<string, string[]> = { gr: ["gr", "kg"], kg: ["kg", "gr"], ml: ["ml", "lt"], lt: ["lt", "ml"], und: ["und"] };
/** How many of the base unit (gr, ml) one of each unit is: only inside a family. */
const FACTOR: Record<string, number> = { gr: 1, kg: 1000, ml: 1, lt: 1000, und: 1 };

/** The price of one unit of the ingredient, from a package of `size` `from` that cost `price`. */
const perUnit = (price: number, size: number, from: string, to: string): number | null => {
  if (!(price > 0) || !(size > 0)) return null;
  return price / ((size * FACTOR[from]) / FACTOR[to]);
};

interface Props {
  ingredient: IIngredient;
  /** The suppliers already used, to suggest them. */
  suppliers: string[];
  onClose: () => void;
  /** Called after it is saved, so the list shows the new stock and price. */
  onSaved: (what: "purchase" | "count") => void;
}

/** Registers a purchase (quantity, package, price paid and supplier) or counts the stock. */
export default function QuickStockModal({ ingredient, suppliers, onClose, onSaved }: Props) {
  const [mode, setMode]       = useState<"purchase" | "count">("purchase");
  const [amount, setAmount]   = useState("");
  const [unit, setUnit]       = useState(ingredient.default_unit);
  const [pkgSize, setPkgSize] = useState(ingredient.package_size ? num(ingredient.package_size) : "");
  const [pkgUnit, setPkgUnit] = useState(ingredient.default_unit);
  const [pkgPrice, setPkgPrice] = useState(ingredient.package_price ? num(ingredient.package_price) : "");
  const [supplier, setSupplier] = useState(ingredient.supplier ?? "");
  const [saving, setSaving]   = useState(false);
  const [error, setError]     = useState("");

  const newPrice = perUnit(parseFloat(pkgPrice), parseFloat(pkgSize), pkgUnit, ingredient.default_unit);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    const value = parseFloat(amount);
    if (isNaN(value) || (mode === "purchase" && value <= 0) || value < 0) {
      setError(mode === "purchase" ? "Indica cuánto compraste (más de 0)" : "Indica la cantidad contada (0 o más)");
      return;
    }
    let form: StockAdjustmentForm;
    if (mode === "purchase") {
      const size = parseFloat(pkgSize);
      const price = parseFloat(pkgPrice);
      const hasPackage = pkgSize.trim() !== "" || pkgPrice.trim() !== "";
      if (hasPackage && !(size > 0 && price > 0)) {
        setError("El tamaño del paquete y su precio van juntos y deben ser más de 0 (o deja los dos vacíos)");
        return;
      }
      form = {
        reason: "compra", quantity: value, unit, adjusted_on: todayInSantiago(),
        ...(hasPackage ? { package_size: size, package_unit: pkgUnit, package_price: price, supplier: supplier.trim() } : {}),
      };
    } else {
      form = { reason: "correccion", counted_quantity: value, unit, adjusted_on: todayInSantiago() };
    }
    setSaving(true);
    try {
      await stockAdjustmentService.create(ingredient.id, form);
      onSaved(mode);
      onClose();
    } catch (err) {
      setError(apiErrorMessage(err, mode === "purchase" ? "Error registrando la compra" : "Error guardando el stock"));
    } finally {
      setSaving(false);
    }
  };

  const field = "w-full bg-crema border border-linea text-negro px-3 py-1.5 text-sm font-light focus:outline-none focus:border-oro transition-colors";
  const label = "block text-xs tracking-widest uppercase text-negro/70 mb-1 font-light";
  const units = UNITS_OF[ingredient.default_unit] ?? [ingredient.default_unit];
  const oldPrice = ingredient.price_per_unit;

  return (
    <Modal
      title={`${mode === "purchase" ? "Compra" : "Stock"} · ${ingredientLabel(ingredient)}`}
      onClose={onClose}
      footer={
        <div className="space-y-2">
          {error && <p className="text-peligro text-xs">{error}</p>}
          <button type="submit" form="quick-stock-form" disabled={saving} data-primary-action
            className="w-full bg-negro text-crema text-xs tracking-widest uppercase font-medium px-6 py-2.5 hover:bg-black transition-[color,background-color,border-color,opacity] disabled:opacity-50">
            {saving ? "Guardando..." : mode === "purchase" ? "Registrar compra" : "Guardar conteo"}
          </button>
        </div>
      }
    >
      <p className="text-xs text-negro/70 font-light">
        Stock actual: <span className="text-negro font-medium">{num(ingredient.stock_quantity)} {ingredient.default_unit}</span>
      </p>
      <div className="grid grid-cols-2 gap-2">
        {([["purchase", "Compra"], ["count", "Contar"]] as const).map(([m, text]) => (
          <button key={m} type="button" onClick={() => setMode(m)} aria-pressed={mode === m}
            className={`text-xs tracking-widest uppercase font-medium px-4 py-2 border transition-colors
              ${mode === m ? "bg-negro text-crema border-oro" : "border-linea text-negro/70 hover:border-oro hover:text-oro"}`}>
            {text}
          </button>
        ))}
      </div>
      <p className="text-xs text-negro/70 font-light">
        {mode === "purchase"
          ? "Suma lo que compraste al stock y deja este precio como el último pagado: el costo de las recetas lo usa."
          : "Queda el stock que cuentes, sin pérdida."}
      </p>
      <form id="quick-stock-form" onSubmit={submit} className="space-y-3">
        <div className="flex gap-2">
          <div className="flex-1">
            <label className={label}>{mode === "purchase" ? "Cantidad comprada" : "Cantidad contada"}</label>
            <input data-testid="purchase-quantity" type="number" step="any" min={0} value={amount} onChange={(e) => setAmount(e.target.value)}
              placeholder="ej. 500" autoFocus className={field} />
          </div>
          <div className="w-24">
            <label className={label}>Unidad</label>
            <select aria-label="Unidad de la compra" value={unit} onChange={(e) => setUnit(e.target.value)} className={field}>
              {units.map((u) => <option key={u} value={u}>{u}</option>)}
            </select>
          </div>
        </div>

        {mode === "purchase" && (
          <>
            <div className="flex gap-2">
              <div className="flex-1">
                <label className={label}>Tamaño del paquete</label>
                <input data-testid="purchase-package-size" type="number" step="any" min={0} value={pkgSize}
                  onChange={(e) => setPkgSize(e.target.value)} placeholder="ej. 25" className={field} />
              </div>
              <div className="w-24">
                <label className={label}>Unidad</label>
                <select aria-label="Unidad del paquete" value={pkgUnit} onChange={(e) => setPkgUnit(e.target.value)} className={field}>
                  {units.map((u) => <option key={u} value={u}>{u}</option>)}
                </select>
              </div>
            </div>
            <div>
              <label className={label}>Precio pagado por el paquete</label>
              <input data-testid="purchase-package-price" type="number" step="any" min={0} value={pkgPrice}
                onChange={(e) => setPkgPrice(e.target.value)} placeholder="ej. 20000" className={field} />
            </div>
            <div>
              <label className={label}>Proveedor (dónde compraste)</label>
              <input data-testid="purchase-supplier" type="text" list="purchase-suppliers" value={supplier} maxLength={100}
                onChange={(e) => setSupplier(e.target.value)} placeholder="ej. Mayorista, Líder" className={field} />
              <datalist id="purchase-suppliers">
                {suppliers.map((s) => <option key={s} value={s} />)}
              </datalist>
            </div>
            {newPrice !== null && (
              <p data-testid="purchase-price-preview" className="text-xs text-negro font-light">
                Precio por {ingredient.default_unit}: antes {formatCLP(Math.round(oldPrice * 100) / 100)} · ahora {formatCLP(Math.round(newPrice * 100) / 100)}
              </p>
            )}
          </>
        )}
      </form>
    </Modal>
  );
}
