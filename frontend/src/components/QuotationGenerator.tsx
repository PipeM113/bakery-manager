import { useEffect, useState } from "react";
import api from "../services/api";
import { quotationService } from "../api/quotationService";
import type { IRecipe } from "../types/recipe";

interface Props {
  recipe: IRecipe;
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
  total_cost: number;
  cost_per_portion: number;
}

const clp = (n: number) => `$${Math.round(n).toLocaleString("es-CL")}`;

export default function QuotationGenerator({ recipe }: Props) {
  const [indirectPct, setIndirectPct] = useState(String(Math.round(recipe.indirect_cost_pct * 100)));
  const [laborPct, setLaborPct]       = useState(String(Math.round(recipe.labor_cost_pct * 100)));
  const [margin, setMargin]           = useState("30");
  const [breakdown, setBreakdown]     = useState<CostBreakdown | null>(null);
  const [loading, setLoading]         = useState(false);
  const [downloading, setDownloading] = useState(false);
  const [noDescError, setNoDescError] = useState(false);

  const indirectNum = parseFloat(indirectPct || "0") / 100;
  const laborNum    = parseFloat(laborPct    || "0") / 100;
  const marginNum   = parseFloat(margin      || "0") / 100;
  const finalPrice  = breakdown
    ? Math.ceil(breakdown.total_cost * (1 + marginNum) / 500) * 500
    : null;

  const fetchBreakdown = () => {
    setBreakdown(null);
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
    setIndirectPct(String(Math.round(recipe.indirect_cost_pct * 100)));
    setLaborPct(String(Math.round(recipe.labor_cost_pct * 100)));
    setMargin("30");
    fetchBreakdown();
  }, [recipe.id]);

  const handleDownload = async () => {
    if (!margin) return;
    if (!recipe.description?.trim()) { setNoDescError(true); return; }
    setNoDescError(false);
    setDownloading(true);
    try {
      const blob = await quotationService.generatePDF({
        recipe_id:        recipe.id,
        margin_pct:       marginNum,
        indirect_cost_pct: indirectNum,
        labor_cost_pct:   laborNum,
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

  if (loading) {
    return (
      <p className="text-gold text-xs tracking-widest uppercase animate-pulse py-4 text-center">
        Calculando...
      </p>
    );
  }

  if (!breakdown) return null;

  return (
    <div className="space-y-6">

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
            { label: "Ingredientes",      value: breakdown.ingredients_total },
            { label: "Costos indirectos", value: breakdown.indirect_costs    },
            { label: "Mano de obra",      value: breakdown.labor_costs       },
          ].map(({ label, value }) => (
            <div key={label} className="flex justify-between items-center">
              <span className="text-stone-400 text-xs tracking-wide font-light">{label}</span>
              <span className="text-stone-700 text-sm font-light">{clp(value)}</span>
            </div>
          ))}
          <div className="flex justify-between items-center">
            <span className="text-stone-400 text-xs tracking-wide font-light">Costo por porción</span>
            <span className="text-stone-700 text-sm">{clp(breakdown.cost_per_portion)}</span>
          </div>
          <div className="flex justify-between items-center bg-vanilla-100 border border-stone-200 px-3 py-2 mt-1">
            <span className="text-stone-500 text-xs tracking-wide font-light">Costo total receta</span>
            <span className="text-stone-800 text-sm font-medium">{clp(breakdown.total_cost)}</span>
          </div>
        </div>
      </div>

      {/* Parámetros de costo + margen */}
      <div className="border-t border-stone-200 pt-4 space-y-4">
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
              onChange={(e) => setIndirectPct(e.target.value)}
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
              onChange={(e) => setLaborPct(e.target.value)}
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
              onChange={(e) => { setMargin(e.target.value); setNoDescError(false); }}
              className="w-full bg-vanilla-100 border border-stone-200 text-stone-800
                         px-3 py-2.5 text-sm font-light focus:outline-none focus:border-gold transition-all"
            />
          </div>
        </div>
        <button
          onClick={fetchBreakdown}
          className="w-full border border-stone-200 text-stone-500 text-xs tracking-widest uppercase
                     font-medium py-2 hover:border-gold hover:text-gold transition-all"
        >
          Recalcular
        </button>

        {finalPrice !== null && (
          <div className="flex justify-between items-center bg-vanilla-100 border border-gold border-opacity-20 px-4 py-4">
            <span className="text-gold text-xs tracking-widest uppercase font-light">
              Precio sugerido (torta completa)
            </span>
            <span className="text-gold font-display text-3xl">{clp(finalPrice)}</span>
          </div>
        )}

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
