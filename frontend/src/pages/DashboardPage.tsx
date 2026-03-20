import { useEffect, useState } from "react";
import api from "../services/api";

interface Ingredient {
  id: string;
  name: string;
  stock_quantity: number;
  alert_threshold: number;
  default_unit: string;
}

interface Recipe {
  id: string;
  name: string;
  yield: number;
  yield_unit: string;
  is_base: boolean;
}

export default function DashboardPage() {
  const [ingredients, setIngredients] = useState<Ingredient[]>([]);
  const [recipes, setRecipes]         = useState<Recipe[]>([]);
  const [loading, setLoading]         = useState(true);

  useEffect(() => {
    Promise.all([
      api.get("/ingredients"),
      api.get("/recipes"),
    ]).then(([ing, rec]) => {
      setIngredients(ing.data);
      setRecipes(rec.data);
    }).finally(() => setLoading(false));
  }, []);

  const alerts = ingredients.filter(
    (i) => i.alert_threshold > 0 && i.stock_quantity < i.alert_threshold
  );

  const baseRecipes = recipes.filter((r) => r.is_base);

  if (loading) return (
    <div className="min-h-screen flex items-center justify-center">
      <span className="text-gold text-sm tracking-widest uppercase font-light animate-pulse">
        Cargando...
      </span>
    </div>
  );

  return (
    <div className="px-6 py-10 max-w-2xl mx-auto md:max-w-none md:px-10">

      {/* Header */}
      <div className="mb-10">
        <div className="flex items-center gap-3 mb-3">
          <div className="h-px w-8 bg-gold opacity-50" />
          <span className="text-gold text-xs tracking-[0.3em] uppercase font-light">
            Panel principal
          </span>
        </div>
        <h1 className="font-display text-4xl text-cream">
          Bienvenida, Angeles
        </h1>
      </div>

      {/* Alertas de stock */}
      {alerts.length > 0 && (
        <div className="mb-8 border border-terracota-400 border-opacity-50 bg-terracota-400 bg-opacity-10 p-5">
          <p className="text-xs tracking-widest uppercase text-terracota-400 font-light mb-3">
            ◆ Stock bajo
          </p>
          <div className="space-y-2">
            {alerts.map((a) => (
              <div key={a.id} className="flex justify-between items-center">
                <span className="text-cream text-sm font-light">{a.name}</span>
                <span className="text-terracota-400 text-xs">
                  {a.stock_quantity} {a.default_unit} restantes
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Stats */}
      <div className="grid grid-cols-2 gap-4 mb-10 md:grid-cols-4">
        {[
          { label: "Insumos",       value: ingredients.length },
          { label: "Recetas base",  value: baseRecipes.length },
          { label: "Total recetas", value: recipes.length },
          { label: "Alertas",       value: alerts.length },
        ].map(({ label, value }) => (
          <div
            key={label}
            className="bg-noir-700 border border-gold border-opacity-20 p-5"
          >
            <p className="font-display text-3xl text-gold mb-1">{value}</p>
            <p className="text-xs tracking-widest uppercase text-cream-muted font-light">
              {label}
            </p>
          </div>
        ))}
      </div>

      {/* Recetas recientes */}
      <div>
        <div className="flex items-center gap-4 mb-5">
          <h2 className="font-display text-xl text-cream">Recetas recientes</h2>
          <div className="flex-1 h-px bg-gold opacity-20" />
        </div>

        {recipes.length === 0 ? (
          <p className="text-cream-muted text-sm font-light text-center py-8">
            No hay recetas aún
          </p>
        ) : (
          <div className="space-y-2">
            {recipes.slice(0, 5).map((r) => (
              <div
                key={r.id}
                className="flex items-center justify-between p-4 bg-noir-700
                           border border-gold border-opacity-10
                           hover:border-opacity-30 transition-all duration-200"
              >
                <div>
                  <p className="text-cream text-sm font-light">{r.name}</p>
                  <p className="text-cream-muted text-xs mt-0.5">
                    {r.yield} {r.yield_unit}
                  </p>
                </div>
                <span className={`text-xs tracking-widest uppercase px-2 py-1
                  ${r.is_base
                    ? "text-gold border border-gold border-opacity-40"
                    : "text-cream-muted border border-cream-muted border-opacity-20"
                  }`}>
                  {r.is_base ? "Base" : "Versión"}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}