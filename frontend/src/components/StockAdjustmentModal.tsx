import { useEffect, useState } from "react";
import { stockAdjustmentService } from "../api/stockAdjustmentService";
import type { IIngredient } from "../types/ingredient";
import type { AdjustmentReason, IStockAdjustment, StockAdjustmentForm } from "../types/stockAdjustment";
import { todayInSantiago } from "../utils/businessDate";
import Modal from "./Modal";
import { formatCLP } from "../utils/money";

const num = (n: number) => (Number.isInteger(n) ? String(n) : n.toFixed(3).replace(/0+$/, "").replace(/\.$/, ""));

const REASONS: { value: AdjustmentReason; label: string; help: string }[] = [
  { value: "merma", label: "Merma", help: "Se perdió o se echó a perder. Baja el stock y cuenta como pérdida." },
  { value: "correccion", label: "Corrección", help: "Contaste el stock real. Queda lo que contaste, sin pérdida." },
  { value: "otro", label: "Otro", help: "Regalo, muestra u otro caso. Sube o baja el stock, sin pérdida. Pide una nota." },
];

const UNITS_OF: Record<string, string[]> = { gr: ["gr", "kg"], kg: ["kg", "gr"], ml: ["ml", "lt"], lt: ["lt", "ml"], und: ["und"] };

interface Props {
  ingredient: IIngredient;
  onClose: () => void;
  /** Called after an adjustment is saved, so the list shows the new stock. */
  onSaved: () => void;
}

export default function StockAdjustmentModal({ ingredient, onClose, onSaved }: Props) {
  const [reason, setReason]     = useState<AdjustmentReason>("merma");
  const [amount, setAmount]     = useState("");
  const [unit, setUnit]         = useState(ingredient.default_unit);
  const [direction, setDirection] = useState<"down" | "up">("down");
  const [note, setNote]         = useState("");
  const [date, setDate]         = useState(todayInSantiago());
  const [saving, setSaving]     = useState(false);
  const [error, setError]       = useState("");
  const [history, setHistory]   = useState<IStockAdjustment[]>([]);

  const loadHistory = () =>
    stockAdjustmentService.forIngredient(ingredient.id).then(setHistory).catch(() => setError("Error cargando el historial"));

  useEffect(() => { loadHistory(); }, [ingredient.id]);

  const units = UNITS_OF[ingredient.default_unit] ?? [ingredient.default_unit];

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    const value = parseFloat(amount);
    if (isNaN(value) || (reason !== "correccion" && value <= 0) || (reason === "correccion" && value < 0)) {
      setError(reason === "correccion" ? "Indica la cantidad contada (0 o más)" : "Indica una cantidad mayor a 0");
      return;
    }
    if (reason === "otro" && !note.trim()) { setError("Escribe una nota"); return; }
    const form: StockAdjustmentForm = { reason, unit, note: note.trim(), adjusted_on: date };
    if (reason === "merma") form.quantity = value;
    if (reason === "correccion") form.counted_quantity = value;
    if (reason === "otro") form.quantity_delta = direction === "down" ? -value : value;
    setSaving(true);
    try {
      await stockAdjustmentService.create(ingredient.id, form);
      setAmount("");
      setNote("");
      await loadHistory();
      onSaved();
    } catch (err: unknown) {
      let msg = "Error guardando el ajuste";
      const resp = (err as { response?: { data?: { error?: string } } }).response;
      if (resp?.data?.error) msg = resp.data.error;
      setError(msg);
    } finally {
      setSaving(false);
    }
  };

  const help = REASONS.find((r) => r.value === reason)!.help;
  const field = "w-full bg-crema border border-linea text-negro px-3 py-1.5 text-sm font-light focus:outline-none focus:border-oro transition-all";
  const label = "block text-xs tracking-widest uppercase text-negro/70 mb-1 font-light";

  return (
    <Modal
      title={`Ajustar stock · ${ingredient.name}`}
      onClose={onClose}
      footer={
        <div className="space-y-2">
          {error && <p className="text-peligro text-xs">{error}</p>}
          <button type="submit" form="stock-adjustment-form" disabled={saving} data-primary-action
            className="w-full bg-negro text-crema text-xs tracking-widest uppercase font-medium px-6 py-2.5 hover:bg-black transition-all disabled:opacity-50">
            {saving ? "Guardando..." : "Guardar ajuste"}
          </button>
        </div>
      }
    >
        <p className="text-xs text-negro/70 font-light">
          Stock actual: <span className="text-negro font-medium">{num(ingredient.stock_quantity)} {ingredient.default_unit}</span>
        </p>

        <form id="stock-adjustment-form" onSubmit={submit} className="space-y-3">
          <div>
            <label className={label}>Motivo</label>
            <select value={reason} onChange={(e) => setReason(e.target.value as AdjustmentReason)} className={field}>
              {REASONS.map((r) => <option key={r.value} value={r.value}>{r.label}</option>)}
            </select>
            <p className="text-xs text-negro/70 font-light mt-1">{help}</p>
          </div>

          {reason === "otro" && (
            <div>
              <label className={label}>Sentido</label>
              <select value={direction} onChange={(e) => setDirection(e.target.value as "down" | "up")} className={field}>
                <option value="down">Baja el stock</option>
                <option value="up">Sube el stock</option>
              </select>
            </div>
          )}

          <div className="flex gap-2">
            <div className="flex-1">
              <label className={label}>{reason === "correccion" ? "Cantidad contada" : "Cantidad"}</label>
              <input type="number" step="any" min={0} value={amount} onChange={(e) => setAmount(e.target.value)}
                placeholder="ej. 200" className={field} />
            </div>
            <div className="w-24">
              <label className={label}>Unidad</label>
              <select value={unit} onChange={(e) => setUnit(e.target.value)} className={field}>
                {units.map((u) => <option key={u} value={u}>{u}</option>)}
              </select>
            </div>
          </div>

          <div>
            <label className={label}>Nota{reason === "otro" ? " *" : ""}</label>
            <input type="text" value={note} onChange={(e) => setNote(e.target.value)} placeholder="Opcional" className={field} />
          </div>

          <div>
            <label className={label}>Fecha</label>
            <input type="date" value={date} max={todayInSantiago()} onChange={(e) => setDate(e.target.value)} className={field} />
          </div>
        </form>

        <div className="pt-2 border-t border-linea">
          <h3 className="font-display text-base text-negro mb-2">Historial</h3>
          {history.length === 0 ? (
            <p className="text-xs text-negro/70 font-light">Sin ajustes todavía</p>
          ) : (
            <ul className="space-y-2">
              {history.map((a) => (
                <li key={a.id} className="text-xs text-negro/70 font-light border-b border-linea pb-2 last:border-0">
                  <span className="text-negro font-medium">{REASONS.find((r) => r.value === a.reason)?.label}</span>
                  {" · "}{a.adjusted_on}{" · "}
                  {a.quantity_delta > 0 ? "+" : ""}{num(a.quantity_delta)} {a.unit}
                  {" "}({num(a.stock_before)} → {num(a.stock_after)})
                  {a.loss_amount > 0 && <span className="text-peligro"> · pérdida {formatCLP(a.loss_amount)}</span>}
                  {a.note && <span> · {a.note}</span>}
                </li>
              ))}
            </ul>
          )}
        </div>
    </Modal>
  );
}
