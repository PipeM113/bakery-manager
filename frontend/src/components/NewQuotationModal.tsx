import { useEffect, useState } from "react";
import RecipeOptions from "./RecipeOptions";
import Modal from "./Modal";
import { recipeService } from "../api/recipeService";
import { quotationService, type IQuotation, type QuotationKind, type SoldAs } from "../api/quotationService";
import type { IRecipe } from "../types/recipe";
import { formatCLP } from "../utils/money";
import { todayInSantiago } from "../utils/businessDate";

const MAX_LINES = 30;

interface Line {
  key: number;
  recipe_id: string;
  quantity: string;
  sold_as: SoldAs;
}

interface Props {
  onClose: () => void;
  /** Called as soon as the quotation is saved, so the list can refresh even if the PDF fails. */
  onCreated: (q: IQuotation) => void;
}

const fieldClass = "w-full border border-linea bg-white px-3 py-2 text-sm font-light text-negro focus:outline-none focus:border-oro";
const labelClass = "block text-xs tracking-widest uppercase text-negro/70 mb-1.5 font-light";

let nextKey = 1;
const newLine = (): Line => ({ key: nextKey++, recipe_id: "", quantity: "1", sold_as: "recipe" });

/** A quotation with several products: the client, the lines and one delivery. It is saved, then its PDF downloads. */
export default function NewQuotationModal({ onClose, onCreated }: Props) {
  const [recipes, setRecipes]   = useState<IRecipe[]>([]);
  const [client, setClient]     = useState("");
  const [kind, setKind]         = useState<QuotationKind>("person");
  const [deliveryDate, setDeliveryDate] = useState("");
  const [paymentTerms, setPaymentTerms] = useState("");
  const [validDays, setValidDays]       = useState("7");
  const [delivery, setDelivery] = useState("0");
  const [lines, setLines]       = useState<Line[]>(() => [newLine()]);
  const [error, setError]       = useState("");
  const [busy, setBusy]         = useState(false);
  const [created, setCreated]   = useState<IQuotation | null>(null);

  useEffect(() => {
    recipeService.getAll().then(setRecipes).catch(() => setError("Error cargando las recetas"));
  }, []);

  const update = (key: number, patch: Partial<Line>) =>
    setLines((ls) => ls.map((l) => (l.key === key ? { ...l, ...patch } : l)));

  const download = async (q: IQuotation) => {
    try {
      await quotationService.downloadPDF(q);
      onClose();
    } catch (e) {
      setError(`La cotización se guardó, pero ${(e as Error).message.replace(/^no se puede/i, "no se pudo")}`);
    }
  };

  const submit = async () => {
    setError("");
    if (created) { setBusy(true); await download(created); setBusy(false); return; }
    if (!client.trim()) { setError("Escribe el nombre del cliente"); return; }
    for (const [i, l] of lines.entries()) {
      const qty = Number(l.quantity);
      if (!l.recipe_id) { setError(`Elige el producto de la línea ${i + 1}`); return; }
      if (!Number.isInteger(qty) || qty < 1) { setError(`La cantidad de la línea ${i + 1} debe ser un entero desde 1`); return; }
    }
    const deliveryNum = Math.max(0, Math.round(Number(delivery) || 0));
    const days = Number(validDays);
    if (kind === "company" && (!Number.isInteger(days) || days < 1 || days > 90)) { setError("La validez debe ser un entero entre 1 y 90 días"); return; }
    setBusy(true);
    try {
      const q = await quotationService.create({
        client_name: client.trim(),
        delivery_cost: deliveryNum,
        ...(kind === "company"
          ? {
              kind,
              valid_days: days,
              ...(deliveryDate ? { delivery_date: deliveryDate } : {}),
              ...(paymentTerms.trim() ? { payment_terms: paymentTerms.trim() } : {}),
            }
          : { kind }),
        lines: lines.map((l) => ({ recipe_id: l.recipe_id, quantity: Number(l.quantity), sold_as: l.sold_as })),
      });
      setCreated(q);
      onCreated(q);
      await download(q);
    } catch (e) {
      const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error;
      setError(msg ?? "Error creando la cotización");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal
      title="Nueva cotización"
      onClose={onClose}
      maxWidth="max-w-2xl"
      footer={
        <div className="flex gap-2 justify-end">
          <button onClick={onClose} className="px-4 text-xs tracking-widest uppercase border border-linea text-negro/70 hover:border-oro">
            {created ? "Cerrar" : "Cancelar"}
          </button>
          <button
            data-primary-action
            onClick={submit}
            disabled={busy}
            className="px-4 text-xs tracking-widest uppercase font-medium bg-negro text-crema hover:bg-black disabled:opacity-50"
          >
            {busy ? "Generando…" : created ? "Reintentar PDF" : "Crear y descargar PDF"}
          </button>
        </div>
      }
    >
      <div>
        <label htmlFor="quote-client" className={labelClass}>Cliente</label>
        <input id="quote-client" className={fieldClass} value={client} maxLength={100}
          onChange={(e) => setClient(e.target.value)} placeholder="Nombre del cliente" disabled={!!created} />
      </div>

      <div>
        <label htmlFor="quote-kind" className={labelClass}>Para</label>
        <select id="quote-kind" aria-label="Para" className={fieldClass} value={kind} disabled={!!created}
          onChange={(e) => setKind(e.target.value as QuotationKind)}>
          <option value="person">Persona (solo precios finales)</option>
          <option value="company">Empresa (con detalle y condiciones)</option>
        </select>
      </div>

      {kind === "company" && (
        <div className="border border-linea p-3 space-y-3">
          <span className="text-xs tracking-widest uppercase text-negro/70 font-light">Condiciones del pedido</span>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label htmlFor="quote-delivery-date" className={labelClass}>Fecha de entrega</label>
              <input id="quote-delivery-date" data-testid="quote-delivery-date" type="date" min={todayInSantiago()} className={fieldClass}
                value={deliveryDate} disabled={!!created} onChange={(e) => setDeliveryDate(e.target.value)} />
            </div>
            <div>
              <label htmlFor="quote-valid-days" className={labelClass}>Validez (días)</label>
              <input id="quote-valid-days" data-testid="quote-valid-days" type="number" inputMode="numeric" min={1} max={90} step={1} className={fieldClass}
                value={validDays} disabled={!!created} onChange={(e) => setValidDays(e.target.value)} />
            </div>
          </div>
          <div>
            <label htmlFor="quote-payment-terms" className={labelClass}>Forma de pago</label>
            <input id="quote-payment-terms" data-testid="quote-payment-terms" className={fieldClass} maxLength={120}
              value={paymentTerms} disabled={!!created} onChange={(e) => setPaymentTerms(e.target.value)} placeholder="Ej. Transferencia a 30 días" />
          </div>
        </div>
      )}

      <div className="space-y-3">
        {lines.map((l, i) => (
          <div key={l.key} data-testid="quote-line" className="border border-linea p-3 space-y-3">
            <div className="flex items-center justify-between gap-3">
              <span className="text-xs tracking-widest uppercase text-negro/70 font-light">Producto {i + 1}</span>
              {lines.length > 1 && !created && (
                <button type="button" aria-label={`Quitar el producto ${i + 1}`} onClick={() => setLines((ls) => ls.filter((x) => x.key !== l.key))}
                  className="text-xs tracking-widest uppercase text-negro/70 hover:text-peligro px-2">
                  Quitar
                </button>
              )}
            </div>
            <select aria-label="Producto" className={fieldClass} value={l.recipe_id} disabled={!!created}
              onChange={(e) => update(l.key, { recipe_id: e.target.value })}>
              <option value="">Elige una receta…</option>
              <RecipeOptions recipes={recipes} />
            </select>
            <div className="grid grid-cols-2 gap-3">
              <input aria-label="Cantidad" type="number" inputMode="numeric" min={1} step={1} className={fieldClass}
                value={l.quantity} disabled={!!created} onChange={(e) => update(l.key, { quantity: e.target.value })} />
              <select aria-label="Vender como" className={fieldClass} value={l.sold_as} disabled={!!created}
                onChange={(e) => update(l.key, { sold_as: e.target.value as SoldAs })}>
                <option value="recipe">Recetas completas</option>
                <option value="portion">Porciones</option>
              </select>
            </div>
          </div>
        ))}
        {lines.length < MAX_LINES && !created && (
          <button type="button" data-testid="quote-add-line" onClick={() => setLines((ls) => [...ls, newLine()])}
            className="w-full border border-dashed border-oro text-oro text-xs tracking-widest uppercase hover:bg-lino">
            + Agregar producto
          </button>
        )}
      </div>

      <div>
        <label htmlFor="quote-delivery" className={labelClass}>Delivery ($)</label>
        <input id="quote-delivery" data-testid="quote-delivery" type="number" inputMode="numeric" min={0} step={500} className={fieldClass}
          value={delivery} disabled={!!created} onChange={(e) => setDelivery(e.target.value)} />
      </div>

      {created && (
        <p className="text-sm font-light text-negro bg-salvia/40 border border-salvia px-3 py-2">
          Cotización guardada por {formatCLP(created.total)}.
        </p>
      )}
      {error && <p role="alert" className="text-peligro text-xs">{error}</p>}
    </Modal>
  );
}
