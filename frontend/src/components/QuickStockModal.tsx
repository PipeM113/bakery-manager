import { useState } from "react";
import { stockAdjustmentService } from "../api/stockAdjustmentService";
import type { IIngredient } from "../types/ingredient";
import type { StockAdjustmentForm } from "../types/stockAdjustment";
import { todayInSantiago } from "../utils/businessDate";
import { apiErrorMessage } from "../utils/apiError";
import { ingredientLabel } from "../utils/ingredients";
import Modal from "./Modal";

const num = (n: number) => (Number.isInteger(n) ? String(n) : n.toFixed(3).replace(/0+$/, "").replace(/\.$/, ""));
const UNITS_OF: Record<string, string[]> = { gr: ["gr", "kg"], kg: ["kg", "gr"], ml: ["ml", "lt"], lt: ["lt", "ml"], und: ["und"] };

interface Props {
  ingredient: IIngredient;
  onClose: () => void;
  /** Called after the stock is saved, so the list shows the new stock. */
  onSaved: () => void;
}

/** The fast way to change the stock: add what was bought, or type what is counted now. */
export default function QuickStockModal({ ingredient, onClose, onSaved }: Props) {
  const [mode, setMode]     = useState<"add" | "count">("add");
  const [amount, setAmount] = useState("");
  const [unit, setUnit]     = useState(ingredient.default_unit);
  const [saving, setSaving] = useState(false);
  const [error, setError]   = useState("");

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    const value = parseFloat(amount);
    if (isNaN(value) || (mode === "add" && value <= 0) || value < 0) {
      setError(mode === "add" ? "Indica cuánto compraste (más de 0)" : "Indica la cantidad contada (0 o más)");
      return;
    }
    const form: StockAdjustmentForm = mode === "add"
      ? { reason: "compra", quantity: value, unit, adjusted_on: todayInSantiago() }
      : { reason: "correccion", counted_quantity: value, unit, adjusted_on: todayInSantiago() };
    setSaving(true);
    try {
      await stockAdjustmentService.create(ingredient.id, form);
      onSaved();
      onClose();
    } catch (err) {
      setError(apiErrorMessage(err, "Error guardando el stock"));
    } finally {
      setSaving(false);
    }
  };

  const field = "w-full bg-crema border border-linea text-negro px-3 py-1.5 text-sm font-light focus:outline-none focus:border-oro transition-all";
  const label = "block text-xs tracking-widest uppercase text-negro/70 mb-1 font-light";
  const units = UNITS_OF[ingredient.default_unit] ?? [ingredient.default_unit];

  return (
    <Modal
      title={`Stock · ${ingredientLabel(ingredient)}`}
      onClose={onClose}
      footer={
        <div className="space-y-2">
          {error && <p className="text-peligro text-xs">{error}</p>}
          <button type="submit" form="quick-stock-form" disabled={saving} data-primary-action
            className="w-full bg-negro text-crema text-xs tracking-widest uppercase font-medium px-6 py-2.5 hover:bg-black transition-all disabled:opacity-50">
            {saving ? "Guardando..." : mode === "add" ? "Agregar al stock" : "Guardar conteo"}
          </button>
        </div>
      }
    >
      <p className="text-xs text-negro/70 font-light">
        Stock actual: <span className="text-negro font-medium">{num(ingredient.stock_quantity)} {ingredient.default_unit}</span>
      </p>
      <div className="grid grid-cols-2 gap-2">
        {([["add", "Agregar"], ["count", "Contar"]] as const).map(([m, text]) => (
          <button key={m} type="button" onClick={() => setMode(m)} aria-pressed={mode === m}
            className={`text-xs tracking-widest uppercase font-medium px-4 py-2 border transition-all
              ${mode === m ? "bg-negro text-crema border-oro" : "border-linea text-negro/70 hover:border-oro hover:text-oro"}`}>
            {text}
          </button>
        ))}
      </div>
      <p className="text-xs text-negro/70 font-light">
        {mode === "add" ? "Suma lo que compraste al stock que hay." : "Queda el stock que cuentes, sin pérdida."}
      </p>
      <form id="quick-stock-form" onSubmit={submit} className="flex gap-2">
        <div className="flex-1">
          <label className={label}>{mode === "add" ? "Cantidad comprada" : "Cantidad contada"}</label>
          <input type="number" step="any" min={0} value={amount} onChange={(e) => setAmount(e.target.value)}
            placeholder="ej. 500" autoFocus className={field} />
        </div>
        <div className="w-24">
          <label className={label}>Unidad</label>
          <select value={unit} onChange={(e) => setUnit(e.target.value)} className={field}>
            {units.map((u) => <option key={u} value={u}>{u}</option>)}
          </select>
        </div>
      </form>
    </Modal>
  );
}
