import { useEffect, useRef, useState } from "react";
import api from "../services/api";
import { quotationService } from "../api/quotationService";
import type { IRecipe } from "../types/recipe";

interface Props {
  recipe: IRecipe;
  deferredSave?: boolean;
  onDeferredSaveDone?: (ok: boolean) => void;
}

interface IngredientCost {
  ingredient_id: string;
  name: string;
  quantity: number;
  unit: string;
  price_per_unit: number;
  subtotal: number;
}

interface CostBreakdown {
  yield: number;
  yield_unit: string;
  ingredients: IngredientCost[];
  ingredients_total: number;
  indirect_costs: number;
  labor_costs: number;
  subtotal_sin_margen: number;
  total_cost: number;
  margin_pct: number;
  cost_per_portion: number;
  suggested_price: number;
}

type ToastState = { msg: string; ok: boolean } | null;

const clp = (n: number) => `$${Math.round(n).toLocaleString("es-CL")}`;

export default function QuotationGenerator({ recipe, deferredSave = false, onDeferredSaveDone }: Props) {
  const [indirectPct, setIndirectPct] = useState(String(Math.round(recipe.indirect_cost_pct * 100)));
  const [laborPct, setLaborPct]       = useState(String(Math.round(recipe.labor_cost_pct * 100)));
  const [margin, setMargin]           = useState(String(Math.round(recipe.margin_pct * 100)));
  const [portions, setPortions]       = useState(String(Math.round(recipe.yield)));
  const [breakdown, setBreakdown]     = useState<CostBreakdown | null>(null);
  const [, setLoading]                = useState(false);
  const [clientName, setClientName]   = useState("");
  const [downloading, setDownloading] = useState(false);
  const [noDescError, setNoDescError] = useState(false);
  const [clientError, setClientError] = useState(false);
  const [toast, setToast]             = useState<ToastState>(null);

  // Save-as modal state
  const [showSaveModal, setShowSaveModal] = useState(false);
  const [saveAsName, setSaveAsName]       = useState("");
  const [saving, setSaving]               = useState(false);

  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Refs for deferred save (used when deferredSave=true)
  const hasChangesRef   = useRef(false);
  const recipeIdRef     = useRef(recipe.id);
  const onSaveDoneRef   = useRef(onDeferredSaveDone);
  const deferredCostsRef = useRef({
    indirect: recipe.indirect_cost_pct,
    labor:    recipe.labor_cost_pct,
    margin:   recipe.margin_pct,
  });
  // Keep callback ref current (safe since key= remounts on recipe change)
  onSaveDoneRef.current = onDeferredSaveDone;

  // Save on unmount when deferredSave=true (handles: recipe change + tab navigation)
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => {
    if (!deferredSave) return;
    return () => {
      if (!hasChangesRef.current) return;
      api.put(`/recipes/${recipeIdRef.current}/costs`, {
        indirect_cost_pct: deferredCostsRef.current.indirect,
        labor_cost_pct:    deferredCostsRef.current.labor,
        margin_pct:        deferredCostsRef.current.margin,
      })
        .then(() => onSaveDoneRef.current?.(true))
        .catch(() => onSaveDoneRef.current?.(false));
    };
  }, []); // intentional empty deps — relies on refs to read latest values

  const indirectNum = parseFloat(indirectPct || "0") / 100;
  const laborNum    = parseFloat(laborPct    || "0") / 100;
  const marginNum   = parseFloat(margin      || "0") / 100;
  const portionsNum = Math.max(1, Math.round(parseFloat(portions || "1")));

  // Adjusted per-portion cost and final price based on temporary portions
  const adjustedCostPerPortion = breakdown
    ? (breakdown.subtotal_sin_margen / portionsNum) * (1 + marginNum)
    : null;
  const adjustedFinalPrice = adjustedCostPerPortion !== null
    ? adjustedCostPerPortion * portionsNum
    : null;

  const portionsChanged = breakdown && portionsNum !== Math.round(breakdown.yield);

  const showToast = (msg: string, ok: boolean) => {
    setToast({ msg, ok });
    setTimeout(() => setToast(null), 2500);
  };

  const saveCosts = (indirect: number, labor: number, marginVal: number) => {
    api.put(`/recipes/${recipe.id}/costs`, {
      indirect_cost_pct: indirect,
      labor_cost_pct:    labor,
      margin_pct:        marginVal,
    })
      .then(() => showToast("Guardado", true))
      .catch(() => showToast("Error al guardar", false));
  };

  const scheduleSave = (indirect: number, labor: number, marginVal: number) => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      saveCosts(indirect, labor, marginVal);
    }, 500);
  };

  const fetchBreakdown = () => {
    setLoading(true);
    const params = new URLSearchParams({
      indirect_cost_pct: String(indirectNum),
      labor_cost_pct:    String(laborNum),
      margin_pct:        String(marginNum),
    });
    api.get<CostBreakdown>(`/recipes/${recipe.id}/cost?${params}`)
      .then(({ data }) => setBreakdown(data))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    const newIndirect = String(Math.round(recipe.indirect_cost_pct * 100));
    const newLabor    = String(Math.round(recipe.labor_cost_pct * 100));
    const newMargin   = String(Math.round(recipe.margin_pct * 100));
    setIndirectPct(newIndirect);
    setLaborPct(newLabor);
    setMargin(newMargin);
    setPortions(String(Math.round(recipe.yield)));
    fetchBreakdown();
  }, [recipe.id]);

  const recalcDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const scheduleRecalc = (indirect: number, labor: number, marginVal: number) => {
    if (recalcDebounceRef.current) clearTimeout(recalcDebounceRef.current);
    recalcDebounceRef.current = setTimeout(() => {
      setLoading(true);
      const params = new URLSearchParams({
        indirect_cost_pct: String(indirect),
        labor_cost_pct:    String(labor),
        margin_pct:        String(marginVal),
      });
      api.get<CostBreakdown>(`/recipes/${recipe.id}/cost?${params}`)
        .then(({ data }) => setBreakdown(data))
        .finally(() => setLoading(false));
    }, 400);
  };

  const handleIndirectChange = (val: string) => {
    setIndirectPct(val);
    const n = parseFloat(val || "0") / 100;
    if (n >= 0) {
      if (deferredSave) {
        deferredCostsRef.current.indirect = n;
        hasChangesRef.current = true;
      } else {
        scheduleSave(n, laborNum, marginNum);
      }
      scheduleRecalc(n, laborNum, marginNum);
    }
  };

  const handleLaborChange = (val: string) => {
    setLaborPct(val);
    const n = parseFloat(val || "0") / 100;
    if (n >= 0) {
      if (deferredSave) {
        deferredCostsRef.current.labor = n;
        hasChangesRef.current = true;
      } else {
        scheduleSave(indirectNum, n, marginNum);
      }
      scheduleRecalc(indirectNum, n, marginNum);
    }
  };

  const handleMarginChange = (val: string) => {
    setMargin(val);
    setNoDescError(false);
    const n = parseFloat(val || "0") / 100;
    if (n >= 0) {
      if (deferredSave) {
        deferredCostsRef.current.margin = n;
        hasChangesRef.current = true;
      } else {
        scheduleSave(indirectNum, laborNum, n);
      }
      scheduleRecalc(indirectNum, laborNum, n);
    }
  };

  const handlePortionsChange = (val: string) => {
    setPortions(val);
  };

  const openSaveModal = () => {
    setSaveAsName(`${recipe.name} - ${portionsNum} porciones`);
    setShowSaveModal(true);
  };

  const handleSaveAs = async () => {
    if (!saveAsName.trim()) return;
    setSaving(true);
    try {
      await api.post(`/recipes/${recipe.id}/save-as`, {
        portions: portionsNum,
        new_name: saveAsName.trim(),
      });
      setShowSaveModal(false);
      showToast("Versión guardada correctamente", true);
    } catch {
      showToast("Error al guardar la versión", false);
    } finally {
      setSaving(false);
    }
  };

  const handleDownload = async () => {
    if (!margin) return;
    const trimmedClient = clientName.trim();
    if (!trimmedClient) { setClientError(true); return; }
    setClientError(false);
    if (!recipe.description?.trim()) { setNoDescError(true); return; }
    setNoDescError(false);
    setDownloading(true);
    try {
      const blob = await quotationService.generatePDF({
        recipe_id:         recipe.id,
        client_name:       trimmedClient,
        margin_pct:        marginNum,
        indirect_cost_pct: indirectNum,
        labor_cost_pct:    laborNum,
      });
      const url  = URL.createObjectURL(blob);
      const a    = document.createElement("a");
      const date = new Date().toISOString().slice(0, 10);
      const name = recipe.name.replace(/\s+/g, "_");
      a.href     = url;
      a.download = `cotizacion_${name}_${date}.pdf`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    } finally {
      setDownloading(false);
    }
  };

  if (!breakdown) return null;

  return (
    <div className="space-y-6">

      {/* Toast */}
      {toast && (
        <div className={`fixed top-4 right-4 z-50 px-4 py-2 text-xs tracking-widest uppercase font-medium text-white transition-all
          ${toast.ok ? "bg-stone-700" : "bg-terracota-500"}`}>
          {toast.msg}
        </div>
      )}

      {/* Save-as modal */}
      {showSaveModal && (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-black bg-opacity-40">
          <div className="bg-white border border-stone-200 shadow-lg p-6 w-full max-w-sm space-y-4">
            <div className="flex items-center justify-between">
              <p className="text-xs tracking-widest uppercase text-stone-400 font-light">
                Guardar escalado como nueva versión
              </p>
              <button onClick={() => setShowSaveModal(false)} className="text-stone-400 hover:text-stone-800 text-xl leading-none">×</button>
            </div>
            <p className="text-stone-500 text-sm font-light">
              Se creará una copia de <strong className="font-medium text-stone-700">{recipe.name}</strong> con {portionsNum} porciones.
            </p>
            <div>
              <label className="block text-xs tracking-widest uppercase text-stone-400 mb-1.5 font-light">
                Nombre
              </label>
              <input
                type="text"
                value={saveAsName}
                onChange={(e) => setSaveAsName(e.target.value)}
                className="w-full bg-vanilla-100 border border-stone-200 text-stone-800
                           px-3 py-2.5 text-sm font-light focus:outline-none focus:border-gold transition-all"
              />
            </div>
            <div className="flex gap-2 pt-1">
              <button
                onClick={() => setShowSaveModal(false)}
                className="flex-1 border border-stone-200 text-stone-500 text-xs tracking-widest uppercase
                           font-medium py-2.5 hover:bg-stone-50 transition-all"
              >
                Cancelar
              </button>
              <button
                onClick={handleSaveAs}
                disabled={!saveAsName.trim() || saving}
                className="flex-1 bg-gold text-white text-xs tracking-widest uppercase font-medium
                           py-2.5 hover:bg-gold-light transition-all
                           disabled:opacity-40 disabled:cursor-not-allowed"
              >
                {saving ? "Guardando..." : "Crear versión"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Porciones */}
      <div className="border border-stone-200 bg-vanilla-100 p-4 space-y-3">
        <p className="text-xs tracking-widest uppercase text-stone-400 font-light">
          Ajuste de porciones
        </p>
        <div className="flex items-end gap-4">
          <div className="flex-1">
            <label className="block text-xs tracking-widest uppercase text-stone-400 mb-1.5 font-light">
              Número de porciones
            </label>
            <input
              type="number"
              value={portions}
              min="1"
              step="1"
              onChange={(e) => handlePortionsChange(e.target.value)}
              className="w-full bg-white border border-stone-200 text-stone-800
                         px-3 py-2.5 text-sm font-light focus:outline-none focus:border-gold transition-all"
            />
            <p className="text-stone-400 text-xs mt-1 font-light">
              Original: {Math.round(recipe.yield)} {recipe.yield_unit}
            </p>
          </div>
          {portionsChanged && (
            <button
              onClick={openSaveModal}
              className="bg-stone-800 text-white text-xs tracking-widest uppercase font-medium
                         px-4 py-2.5 hover:bg-stone-700 transition-all whitespace-nowrap"
            >
              Guardar escalado
            </button>
          )}
        </div>

        {portionsChanged && adjustedCostPerPortion !== null && adjustedFinalPrice !== null && (
          <div className="space-y-1.5 pt-2 border-t border-stone-200">
            <div className="flex justify-between items-center">
              <span className="text-stone-400 text-xs tracking-wide font-light">
                Costo por porción (ajustado)
              </span>
              <span className="text-stone-700 text-sm font-medium">{clp(adjustedCostPerPortion)}</span>
            </div>
            <div className="flex justify-between items-center bg-white border border-gold border-opacity-30 px-3 py-2">
              <span className="text-gold text-xs tracking-wide font-medium uppercase tracking-widest">
                Precio final ({portionsNum} porciones)
              </span>
              <span className="text-gold font-display text-xl">{clp(adjustedFinalPrice)}</span>
            </div>
            <p className="text-stone-400 text-xs font-light">
              Este ajuste es temporal. Usa "Guardar escalado" para crear una versión permanente.
            </p>
          </div>
        )}
      </div>

      {/* Desglose de costos */}
      <div>
        <p className="text-xs tracking-widest uppercase text-stone-400 font-light mb-3">Desglose</p>
        <div className="space-y-1 mb-4">
          {breakdown.ingredients.map((ing) => (
            <div key={ing.ingredient_id}
              className="flex justify-between items-center py-1.5 border-b border-stone-100 last:border-0">
              <div>
                <p className="text-stone-800 text-sm font-light">{ing.name}</p>
                <p className="text-stone-400 text-xs mt-0.5">
                  {ing.quantity % 1 === 0 ? ing.quantity : ing.quantity.toFixed(2)} {ing.unit} × {clp(ing.price_per_unit)}
                </p>
              </div>
              <span className="text-stone-700 text-sm font-light">{clp(ing.subtotal)}</span>
            </div>
          ))}
        </div>
        <div className="space-y-1.5 pt-2 border-t border-stone-200">
          {[
            { label: "Costo de ingredientes", value: breakdown.ingredients_total },
            { label: "Costos fijos",          value: breakdown.indirect_costs    },
            { label: "Mano de obra",          value: breakdown.labor_costs       },
          ].map(({ label, value }) => (
            <div key={label} className="flex justify-between items-center">
              <span className="text-stone-400 text-xs tracking-wide font-light">{label}</span>
              <span className="text-stone-700 text-sm font-light">{clp(value)}</span>
            </div>
          ))}
          <div className="flex justify-between items-center bg-vanilla-100 border border-stone-200 px-3 py-2 mt-1">
            <span className="text-stone-600 text-xs tracking-wide font-medium">Subtotal (sin margen)</span>
            <span className="text-stone-800 text-sm font-medium">{clp(breakdown.subtotal_sin_margen)}</span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-stone-400 text-xs tracking-wide font-light">Margen (%)</span>
            <span className="text-stone-700 text-sm font-light">{Math.round(breakdown.margin_pct * 100)}%</span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-stone-400 text-xs tracking-wide font-light">
              Costo por porción (con margen)
            </span>
            <span className="text-stone-700 text-sm font-medium">{clp(breakdown.cost_per_portion)}</span>
          </div>
          <div className="flex justify-between items-center bg-vanilla-100 border border-gold border-opacity-30 px-3 py-2 mt-1">
            <span className="text-gold text-xs tracking-wide font-medium uppercase tracking-widest">
              Precio final torta completa
            </span>
            <span className="text-gold font-display text-2xl">{clp(breakdown.suggested_price)}</span>
          </div>
        </div>
      </div>

      {/* Datos del cliente */}
      <div className="border-t border-stone-200 pt-4">
        <label className="block text-xs tracking-widest uppercase text-stone-400 mb-1.5 font-light">
          Nombre del cliente
        </label>
        <input
          type="text"
          value={clientName}
          maxLength={100}
          placeholder="ej. María González"
          onChange={(e) => { setClientName(e.target.value); if (clientError) setClientError(false); }}
          className={`w-full bg-vanilla-100 border text-stone-800 px-3 py-2.5 text-sm font-light
                      focus:outline-none transition-all
                      ${clientError ? "border-terracota-500 focus:border-terracota-500" : "border-stone-200 focus:border-gold"}`}
        />
        {clientError && (
          <p className="text-terracota-500 text-xs tracking-wide mt-1">
            El nombre del cliente es requerido para generar la cotización.
          </p>
        )}
      </div>

      {/* Parámetros de costo + margen */}
      <div className="space-y-4">
        <div className="grid grid-cols-3 gap-3">
          <div>
            <label className="block text-xs tracking-widest uppercase text-stone-400 mb-1.5 font-light">
              Costos indirectos (%)
            </label>
            <input
              type="number"
              value={indirectPct}
              min="0"
              placeholder="ej. 15"
              onChange={(e) => handleIndirectChange(e.target.value)}
              className="w-full bg-vanilla-100 border border-stone-200 text-stone-800
                         px-3 py-2.5 text-sm font-light focus:outline-none focus:border-gold transition-all"
            />
          </div>
          <div>
            <label className="block text-xs tracking-widest uppercase text-stone-400 mb-1.5 font-light">
              Mano de obra (%)
            </label>
            <input
              type="number"
              value={laborPct}
              min="0"
              placeholder="ej. 30"
              onChange={(e) => handleLaborChange(e.target.value)}
              className="w-full bg-vanilla-100 border border-stone-200 text-stone-800
                         px-3 py-2.5 text-sm font-light focus:outline-none focus:border-gold transition-all"
            />
          </div>
          <div>
            <label className="block text-xs tracking-widest uppercase text-stone-400 mb-1.5 font-light">
              Margen ganancia (%)
            </label>
            <input
              type="number"
              value={margin}
              min="0"
              placeholder="ej. 30"
              onChange={(e) => handleMarginChange(e.target.value)}
              className="w-full bg-vanilla-100 border border-stone-200 text-stone-800
                         px-3 py-2.5 text-sm font-light focus:outline-none focus:border-gold transition-all"
            />
          </div>
        </div>
        {noDescError && (
          <p className="text-terracota-500 text-xs tracking-wide">
            La receta no tiene descripción. Agrégala antes de generar la cotización.
          </p>
        )}

        <button
          onClick={handleDownload}
          disabled={!margin || downloading}
          className="w-full bg-gold text-white text-xs tracking-widest uppercase font-medium
                     py-3 hover:bg-gold-light transition-all
                     disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {downloading ? "Generando PDF..." : "Descargar PDF"}
        </button>
      </div>
    </div>
  );
}
