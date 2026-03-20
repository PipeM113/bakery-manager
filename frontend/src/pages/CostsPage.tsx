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

const clp = (n: number) => `$${Math.round(n).toLocaleString("es-CL")}`;

export default function CostsPage() {
  const [recipes, setRecipes]       = useState<Recipe[]>([]);
  const [selected, setSelected]     = useState<string>("");
  const [breakdown, setBreakdown]   = useState<CostBreakdown | null>(null);
  const [margin, setMargin]         = useState("");
  const [suggestion, setSuggestion] = useState<PriceSuggestion | null>(null);
  const [loading, setLoading]       = useState(false);
  const [saleMode, setSaleMode]     = useState<"portion" | "full">("full");

  useEffect(() => { api.get("/recipes").then((r) => setRecipes(r.data)); }, []);

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

  const suggestedFinal = suggestion
    ? saleMode === "full"
      ? suggestion.suggested_price * suggestion.yield
      : suggestion.suggested_price
    : null;

  return (
    <div className="px-6 py-10 max-w-2xl mx-auto md:max-w-none md:px-10">

      <div className="mb-8">
        <div className="flex items-center gap-3 mb-2">
          <div className="h-px w-8 bg-gold opacity-40" />
          <span className="text-gold text-xs tracking-[0.3em] uppercase font-light">Análisis</span>
        </div>
        <h1 className="font-display text-4xl text-stone-800">Costos</h1>
      </div>

      <div className="mb-8">
        <label className="block text-xs tracking-widest uppercase text-stone-400 mb-2 font-light">
          Selecciona una receta
        </label>
        <select
          value={selected}
          onChange={(e) => { setSelected(e.target.value); setBreakdown(null); setSuggestion(null); loadCost(e.target.value); }}
          className="w-full bg-white border border-stone-200 text-stone-800
                     px-4 py-3 text-sm font-light focus:outline-none focus:border-gold transition-all"
        >
          <option value="">— Elige una receta —</option>
          {recipes.map((r) => <option key={r.id} value={r.id}>{r.name}</option>)}
        </select>
      </div>

      {loading && (
        <p className="text-gold text-sm tracking-widest uppercase animate-pulse text-center py-8">
          Calculando...
        </p>
      )}

      {breakdown && !loading && (
        <div className="space-y-6">

          <div className="bg-white border border-stone-200 p-6">
            <div className="flex items-center gap-4 mb-5">
              <h2 className="font-display text-xl text-stone-800">Desglose</h2>
              <div className="flex-1 h-px bg-stone-200" />
            </div>

            <div className="space-y-2 mb-5">
              {breakdown.ingredients.map((ing) => (
                <div key={ing.ingredient_id}
                  className="flex justify-between items-center py-2
                             border-b border-stone-100 last:border-0">
                  <div>
                    <p className="text-stone-800 text-sm font-light">{ing.name}</p>
                    <p className="text-stone-400 text-xs mt-0.5">
                      {ing.quantity} {ing.unit} × {clp(ing.price_per_unit)}
                    </p>
                  </div>
                  <span className="text-stone-800 text-sm font-light">{clp(ing.subtotal)}</span>
                </div>
              ))}
            </div>

            <div className="space-y-2 pt-2 border-t border-stone-200">
              {[
                { label: "Ingredientes",      value: breakdown.ingredients_total },
                { label: "Costos indirectos", value: breakdown.indirect_costs    },
                { label: "Mano de obra",      value: breakdown.labor_costs       },
              ].map(({ label, value }) => (
                <div key={label} className="flex justify-between items-center">
                  <span className="text-stone-400 text-xs tracking-wide font-light">{label}</span>
                  <span className="text-stone-800 text-sm font-light">{clp(value)}</span>
                </div>
              ))}
              <div className="flex justify-between items-center">
                <span className="text-stone-400 text-xs tracking-wide font-light">Costo por porción</span>
                <span className="text-stone-800 text-sm">{clp(breakdown.cost_per_portion)}</span>
              </div>
              <div className="flex justify-between items-center bg-vanilla-100 border border-gold border-opacity-20 px-4 py-3 mt-2">
                <span className="text-gold text-xs tracking-widest uppercase font-light">Costo torta completa</span>
                <span className="text-gold font-display text-2xl">{clp(breakdown.total_cost)}</span>
              </div>
            </div>
          </div>

          <div className="bg-white border border-stone-200 p-6">
            <div className="flex items-center gap-4 mb-5">
              <h2 className="font-display text-xl text-stone-800">Precio sugerido</h2>
              <div className="flex-1 h-px bg-stone-200" />
            </div>

            <div className="mb-5">
              <label className="block text-xs tracking-widest uppercase text-stone-400 mb-2 font-light">
                Vender por
              </label>
              <div className="flex gap-2">
                {(["full", "portion"] as const).map((mode) => (
                  <button key={mode}
                    onClick={() => { setSaleMode(mode); setSuggestion(null); }}
                    className={`flex-1 py-2.5 text-xs tracking-widest uppercase font-medium
                      transition-all duration-200 border
                      ${saleMode === mode
                        ? "bg-gold text-white border-gold"
                        : "bg-white text-stone-400 border-stone-200 hover:border-gold"}`}>
                    {mode === "full" ? "Torta completa" : "Por porción"}
                  </button>
                ))}
              </div>
            </div>

            <div className="flex gap-3 mb-4">
              <div className="flex-1">
                <label className="block text-xs tracking-widest uppercase text-stone-400 mb-1.5 font-light">
                  % de ganancia
                </label>
                <input type="number" value={margin} placeholder="ej. 50"
                  onChange={(e) => { setMargin(e.target.value); setSuggestion(null); }}
                  className="w-full bg-vanilla-100 border border-stone-200 text-stone-800
                             px-4 py-2.5 text-sm font-light focus:outline-none focus:border-gold transition-all" />
              </div>
              <div className="flex items-end">
                <button onClick={simulate}
                  className="bg-gold text-white text-xs tracking-widest uppercase font-medium
                             px-6 py-2.5 hover:bg-gold-light transition-all">
                  Calcular
                </button>
              </div>
            </div>

            {suggestion && suggestedFinal !== null && (
              <div className="space-y-3 pt-4 border-t border-stone-200">
                <div className="flex justify-between items-center">
                  <span className="text-stone-400 text-xs tracking-wide font-light">
                    Costo base ({saleMode === "full" ? "torta completa" : "por porción"})
                  </span>
                  <span className="text-stone-800 text-sm">
                    {saleMode === "full" ? clp(suggestion.total_cost) : clp(suggestion.cost_per_portion)}
                  </span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-stone-400 text-xs tracking-wide font-light">
                    Ganancia ({(suggestion.margin_pct * 100).toFixed(0)}%)
                  </span>
                  <span className="text-stone-800 text-sm">
                    {saleMode === "full"
                      ? clp(suggestion.profit_amount * suggestion.yield)
                      : clp(suggestion.profit_amount)}
                  </span>
                </div>
                <div className="flex justify-between items-center bg-vanilla-100 border border-gold border-opacity-20 px-4 py-4 mt-2">
                  <span className="text-gold text-xs tracking-widest uppercase font-light">
                    {saleMode === "full"
                      ? `Precio torta completa (${suggestion.yield} ${suggestion.yield_unit})`
                      : "Precio por porción"}
                  </span>
                  <span className="text-gold font-display text-3xl">{clp(suggestedFinal)}</span>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}