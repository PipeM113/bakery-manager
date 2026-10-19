import { useState } from "react";
import api from "../services/api";
import { orderService } from "../api/orderService";
import Modal from "./Modal";
import type { IOrder } from "../types/order";
import type { IRecipe } from "../types/recipe";
import type { SoldAs } from "../types/sale";
import { todayInSantiago } from "../utils/businessDate";
import { formatCLP } from "../utils/money";


interface FormLine {
  key: number;
  recipe_id: string;
  quantity: string;
  sold_as: SoldAs;
  unit_price: string;
  /** Suggested price of the whole recipe, to price a portion by its share. */
  suggested: number | null;
}

let nextKey = 1;
const newLine = (): FormLine => ({ key: nextKey++, recipe_id: "", quantity: "1", sold_as: "recipe", unit_price: "", suggested: null });

interface Props {
  recipes: IRecipe[];
  onClose: () => void;
  onCreated: (order: IOrder) => void;
}

export default function OrderForm({ recipes, onClose, onCreated }: Props) {
  const [lines, setLines]       = useState<FormLine[]>([newLine()]);
  const [customer, setCustomer] = useState("");
  const [notes, setNotes]       = useState("");
  const [date, setDate]         = useState(todayInSantiago());
  const [total, setTotal]       = useState("");
  const [saving, setSaving]     = useState(false);
  const [error, setError]       = useState("");

  const patch = (key: number, change: Partial<FormLine>) =>
    setLines((ls) => ls.map((l) => (l.key === key ? { ...l, ...change } : l)));

  const priceFor = (recipeId: string, soldAs: SoldAs, whole: number) => {
    const recipe = recipes.find((r) => r.id === recipeId);
    return soldAs === "portion" && recipe && recipe.yield > 0 ? Math.round(whole / recipe.yield) : whole;
  };

  const chooseRecipe = async (key: number, recipeId: string) => {
    patch(key, { recipe_id: recipeId, suggested: null, unit_price: "" });
    if (!recipeId) return;
    try {
      const { data } = await api.get<{ suggested_price: number }>(`/recipes/${recipeId}/cost?margin_pct=0.30`);
      setLines((ls) => ls.map((l) => l.key === key
        ? { ...l, suggested: data.suggested_price, unit_price: String(priceFor(recipeId, l.sold_as, data.suggested_price)) }
        : l));
    } catch { /* the price can be typed */ }
  };

  const chooseSoldAs = (l: FormLine, soldAs: SoldAs) =>
    patch(l.key, {
      sold_as: soldAs,
      ...(l.suggested !== null ? { unit_price: String(priceFor(l.recipe_id, soldAs, l.suggested)) } : {}),
    });

  const normalTotal = lines.reduce((sum, l) => sum + (parseInt(l.quantity, 10) || 0) * (parseFloat(l.unit_price) || 0), 0);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    for (const l of lines) {
      if (!l.recipe_id) { setError("Elige la receta de cada producto"); return; }
      const q = parseInt(l.quantity, 10);
      if (!q || q <= 0) { setError("La cantidad de cada producto debe ser mayor a 0"); return; }
      const p = parseFloat(l.unit_price);
      if (isNaN(p) || p < 0) { setError("Indica el precio de cada producto"); return; }
    }
    if (total !== "" && (isNaN(parseFloat(total)) || parseFloat(total) < 0)) { setError("El precio total debe ser 0 o más"); return; }
    setSaving(true);
    try {
      const order = await orderService.create({
        customer_name: customer.trim(),
        notes: notes.trim(),
        order_date: date,
        ...(total !== "" ? { total_price: Math.round(parseFloat(total)) } : {}),
        lines: lines.map((l) => ({
          recipe_id: l.recipe_id, quantity: parseInt(l.quantity, 10), unit_price: Math.round(parseFloat(l.unit_price)), sold_as: l.sold_as,
        })),
      });
      onCreated(order);
    } catch (err: unknown) {
      const resp = (err as { response?: { data?: { error?: string } } }).response;
      setError(resp?.data?.error ?? "Error registrando el pedido");
    } finally {
      setSaving(false);
    }
  };

  const field = "w-full bg-crema border border-linea text-negro px-3 py-1.5 text-sm font-light focus:outline-none focus:border-oro transition-all";
  const label = "block text-xs tracking-widest uppercase text-negro/70 mb-1 font-light";

  return (
    <Modal
      title="Nuevo pedido"
      onClose={onClose}
      maxWidth="max-w-xl"
      footer={
        <div className="space-y-2">
          {error && <p className="text-peligro text-xs">{error}</p>}
          <button type="submit" form="order-form" disabled={saving} data-primary-action
            className="w-full bg-negro text-crema text-xs tracking-widest uppercase font-medium px-6 py-2.5 hover:bg-black transition-all disabled:opacity-50">
            {saving ? "Registrando..." : "Registrar pedido"}
          </button>
        </div>
      }
    >
      <form id="order-form" onSubmit={submit} className="space-y-3">
        <div>
          <label className={label}>Cliente</label>
          <input type="text" value={customer} onChange={(e) => setCustomer(e.target.value)} placeholder="Opcional" className={field} />
        </div>

        <div className="space-y-3">
          {lines.map((l, i) => (
            <div key={l.key} className="border border-linea p-3 space-y-2">
              <div className="flex items-center justify-between gap-2">
                <span className="text-xs tracking-widest uppercase text-negro/70 font-light">Producto {i + 1}</span>
                {lines.length > 1 && (
                  <button type="button" onClick={() => setLines((ls) => ls.filter((x) => x.key !== l.key))}
                    className="text-xs text-negro/70 hover:text-peligro tracking-wider">Quitar</button>
                )}
              </div>
              <select value={l.recipe_id} onChange={(e) => chooseRecipe(l.key, e.target.value)} className={field} aria-label="Receta">
                <option value="">Seleccionar receta...</option>
                {recipes.map((r) => <option key={r.id} value={r.id}>{r.name}</option>)}
              </select>
              <div className="grid grid-cols-2 gap-2">
                <div>
                  <label className={label}>Cantidad</label>
                  <input type="number" min={1} step={1} value={l.quantity} onChange={(e) => patch(l.key, { quantity: e.target.value })} className={field} />
                </div>
                <div>
                  <label className={label}>Vendida como</label>
                  <select value={l.sold_as} onChange={(e) => chooseSoldAs(l, e.target.value as SoldAs)} className={field}>
                    <option value="recipe">Receta completa</option>
                    <option value="portion">Porción</option>
                  </select>
                </div>
              </div>
              <div>
                <label className={label}>Precio unitario (CLP)</label>
                <input type="number" min={0} value={l.unit_price} onChange={(e) => patch(l.key, { unit_price: e.target.value })} placeholder="ej. 2500" className={field} />
              </div>
            </div>
          ))}
          <button type="button" onClick={() => setLines((ls) => [...ls, newLine()])}
            className="w-full border border-dashed border-linea text-negro/70 text-xs tracking-widest uppercase hover:border-oro hover:text-oro transition-all">
            Agregar producto
          </button>
        </div>

        <div>
          <label className={label}>Precio total del pedido (opcional)</label>
          <input type="number" min={0} value={total} onChange={(e) => setTotal(e.target.value)}
            placeholder={normalTotal > 0 ? String(Math.round(normalTotal)) : "Suma de los productos"} className={field} />
          <p className="text-xs text-negro/70 font-light mt-1">
            Suma de los productos: <span className="text-negro">{formatCLP(normalTotal)}</span>. Si indicas otro total, se reparte entre los productos.
          </p>
        </div>

        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className={label}>Fecha</label>
            <input type="date" value={date} max={todayInSantiago()} onChange={(e) => setDate(e.target.value)} className={field} />
          </div>
          <div>
            <label className={label}>Notas</label>
            <input type="text" value={notes} onChange={(e) => setNotes(e.target.value)} placeholder="Opcional" className={field} />
          </div>
        </div>
      </form>
    </Modal>
  );
}
