import { useEffect, useState } from "react";
import api from "../services/api";

interface Recipe {
  id: string;
  name: string;
  yield: number;
  yield_unit: string;
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
  recipe_id: string;
  recipe_name: string;
  yield: number;
  yield_unit: string;
  ingredients: IngredientCost[];
  ingredients_total: number;
  indirect_costs: number;
  labor_costs: number;
  total_cost: number;
  cost_per_portion: number;
}

interface PriceSuggestion extends CostBreakdown {
  margin_pct: number;
  profit_amount: number;
  suggested_price: number;
}

const fmt = (n: number) => `$${n.toFixed(0).replace(/\B(?=(\d{3})+(?!\d))/g, ".")}`;

export default function CostsPage() {
  const [recipes, setRecipes]       = useState<Recipe[]>([]);
  const [selected, setSelected]     = useState<string>("");
  const [breakdown, setBreakdown]   = useState<CostBreakdown | null>(null);
  const [margin, setMargin]         = useState("50");
  const [suggestion, setSuggestion] = useState<PriceSuggestion | null>(null);
  const [loading, setLoading]       = useState(false);

  useEffect(() => {
    api.get("/recipes").then((r) => setRecipes(r.data));
  }, []);

  const loadCost = async (id: string) => {
    if (!id) return;
    setLoading(true);
    setSuggestion(null);
    try {
      const { data } = await api.get(`/recipes/${id}/cost`);
      setBreakdown(data);
    } finally {
      setLoading(false);
    }
  };

  const simulate = async () => {
    if (!selected || !margin) return;
    const { data } = await api.post(`/recipes/${selected}/cost/simulate`, {
      margin_pct: parseFloat(margin) / 100,
    });
    setSuggestion(data);
  };

  return (
    <div className="px-6 py-10 max-w-2xl mx-auto md:max-w-none md:px-10">

      {/* Header */}
      <div className="mb-8">
        <div className="flex items-center gap-3 mb-2">
          <div className="h-px w-8 bg-gold opacity-50" />
          <span className="text-gold text-xs tracking-[0.3em] uppercase font-light">Análisis</span>
        </div>
        <h1 className="font-display text-4xl text-cream">Costos</h1>
      </div>

      {/* Selector de receta */}
      <div className="mb-8">
        <label className="block text-xs tracking-widest uppercase text-cream-muted mb-2 font-light">
          Selecciona una receta
        </label>
        <select
          value={selected}
          onChange={(e) => { setSelected(e.target.value); setBreakdown(null); setSuggestion(null); loadCost(e.target.value); }}
          className="w-full bg-noir-700 border border-gold border-opacity-20 text-cream
                     px-4 py-3 text-sm font-light focus:outline-none focus:border-gold
                     focus:border-opacity-60 transition-all"
        >
          <option value="">— Elige una receta —</option>
          {recipes.map((r) => (
            <option key={r.id} value={r.id}>{r.name}</option>
          ))}
        </select>
      </div>

      {loading && (
        <p className="text-gold text-sm tracking-widest uppercase animate-pulse text-center py-8">
          Calculando...
        </p>
      )}

      {breakdown && !loading && (
        <div className="space-y-6">

          {/* Desglose de ingredientes */}
          <div className="bg-noir-700 border border-gold border-opacity-20 p-6">
            <div className="flex items-center gap-4 mb-5">
              <h2 className="font-display text-xl text-gold">Desglose</h2>
              <div className="flex-1 h-px bg-gold opacity-20" />
            </div>

            <div className="space-y-2 mb-5">
              {breakdown.ingredients.map((ing) => (
                <div key={ing.ingredient_id} className="flex justify-between items-center py-2
                  border-b border-gold border-opacity-10 last:border-0">
                  <div>
                    <p className="text-cream text-sm font-light">{ing.name}</p>
                    <p className="text-cream-muted text-xs mt-0.5">
                      {ing.quantity} {ing.unit} × {fmt(ing.price_per_unit)}
                    </p>
                  </div>
                  <span className="text-cream text-sm font-light">{fmt(ing.subtotal)}</span>
                </div>
              ))}
            </div>

            {/* Totales */}
            <div className="space-y-2 pt-2 border-t border-gold border-opacity-20">
              {[
                { label: "Ingredientes",      value: breakdown.ingredients_total },
                { label: "Costos indirectos", value: breakdown.indirect_costs },
                { label: "Mano de obra",      value: breakdown.labor_costs },
              ].map(({ label, value }) => (
                <div key={label} className="flex justify-between items-center">
                  <span className="text-cream-muted text-xs tracking-wide font-light">{label}</span>
                  <span className="text-cream text-sm font-light">{fmt(value)}</span>
                </div>
              ))}
              <div className="flex justify-between items-center pt-3 border-t border-gold border-opacity-20">
                <span className="text-gold text-xs tracking-widest uppercase font-light">Total</span>
                <span className="text-gold font-display text-2xl">{fmt(breakdown.total_cost)}</span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-cream-muted text-xs tracking-wide font-light">
                  Por porción ({breakdown.yield} {breakdown.yield_unit})
                </span>
                <span className="text-cream text-sm">{fmt(breakdown.cost_per_portion)}</span>
              </div>
            </div>
          </div>

          {/* Simulador de precio */}
          <div className="bg-noir-700 border border-gold border-opacity-20 p-6">
            <div className="flex items-center gap-4 mb-5">
              <h2 className="font-display text-xl text-gold">Precio sugerido</h2>
              <div className="flex-1 h-px bg-gold opacity-20" />
            </div>

            <div className="flex gap-3 mb-4">
              <div className="flex-1">
                <label className="block text-xs tracking-widest uppercase text-cream-muted mb-1.5 font-light">
                  % de ganancia
                </label>
                <input
                  type="number"
                  value={margin}
                  onChange={(e) => { setMargin(e.target.value); setSuggestion(null); }}
                  className="w-full bg-noir-600 border border-gold border-opacity-20 text-cream
                             px-4 py-2.5 text-sm font-light focus:outline-none focus:border-gold
                             focus:border-opacity-60 transition-all"
                />
              </div>
              <div className="flex items-end">
                <button
                  onClick={simulate}
                  className="bg-gold text-noir text-xs tracking-widest uppercase font-medium
                             px-6 py-2.5 hover:bg-gold-light transition-all"
                >
                  Calcular
                </button>
              </div>
            </div>

            {suggestion && (
              <div className="space-y-3 pt-4 border-t border-gold border-opacity-20">
                <div className="flex justify-between items-center">
                  <span className="text-cream-muted text-xs tracking-wide font-light">Costo por porción</span>
                  <span className="text-cream text-sm">{fmt(suggestion.cost_per_portion)}</span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-cream-muted text-xs tracking-wide font-light">
                    Ganancia ({(suggestion.margin_pct * 100).toFixed(0)}%)
                  </span>
                  <span className="text-cream text-sm">{fmt(suggestion.profit_amount)}</span>
                </div>
                <div className="flex justify-between items-center pt-3 border-t border-gold border-opacity-20">
                  <span className="text-gold text-xs tracking-widest uppercase font-light">
                    Precio sugerido
                  </span>
                  <span className="text-gold font-display text-3xl">
                    {fmt(suggestion.suggested_price)}
                  </span>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}