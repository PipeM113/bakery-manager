import { useEffect, useState } from "react";
import { ingredientService } from "../api/ingredientService";
import { recipeService } from "../api/recipeService";
import { useAuth } from "../context/AuthContext";
import type { IIngredient } from "../types/ingredient";
import type { IRecipe } from "../types/recipe";

export default function DashboardPage() {
  const { userName } = useAuth();
  const [ingredients, setIngredients] = useState<IIngredient[]>([]);
  const [recipes, setRecipes]         = useState<IRecipe[]>([]);
  const [loading, setLoading]         = useState(true);

  useEffect(() => {
    Promise.all([ingredientService.getAll(), recipeService.getAll()])
      .then(([ings, recs]) => { setIngredients(ings); setRecipes(recs); })
      .finally(() => setLoading(false));
  }, []);

  const alerts      = ingredients.filter((i) => i.alert_threshold > 0 && i.stock_quantity < i.alert_threshold);
  const baseRecipes = recipes.filter((r) => r.is_base);
  const words     = userName.split(" ");
  const firstName = words.length > 2 ? words.slice(0, -2).join(" ") : words[0] ?? "";

  if (loading) return (
    <div className="min-h-screen flex items-center justify-center">
      <span className="text-gold text-sm tracking-widest uppercase font-light animate-pulse">Cargando...</span>
    </div>
  );

  return (
    <div className="px-6 py-10 max-w-2xl mx-auto md:max-w-none md:px-10">

      <div className="mb-10">
        <div className="flex items-center gap-3 mb-3">
          <div className="h-px w-8 bg-gold opacity-40" />
          <span className="text-gold text-xs tracking-[0.3em] uppercase font-light">Panel principal</span>
        </div>
        <h1 className="font-display text-4xl text-stone-800">Bienvenida{firstName ? `, ${firstName}` : ""}</h1>
      </div>

      {alerts.length > 0 && (
        <div className="mb-8 border border-terracota-400 bg-terracota-400 bg-opacity-5 p-5">
          <p className="text-xs tracking-widest uppercase text-terracota-500 font-light mb-3">◆ Stock bajo</p>
          <div className="space-y-2">
            {alerts.map((a) => (
              <div key={a.id} className="flex justify-between items-center">
                <span className="text-stone-800 text-sm font-light">{a.name}</span>
                <span className="text-terracota-500 text-xs">{a.stock_quantity} {a.default_unit} restantes</span>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="grid grid-cols-2 gap-4 mb-10 md:grid-cols-4">
        {[
          { label: "Insumos",       value: ingredients.length },
          { label: "Recetas base",  value: baseRecipes.length },
          { label: "Total recetas", value: recipes.length     },
          { label: "Alertas",       value: alerts.length      },
        ].map(({ label, value }) => (
          <div key={label} className="bg-white border border-stone-200 p-5 hover:border-gold transition-all">
            <p className="font-display text-3xl text-gold mb-1">{value}</p>
            <p className="text-xs tracking-widest uppercase text-stone-400 font-light">{label}</p>
          </div>
        ))}
      </div>

      <div>
        <div className="flex items-center gap-4 mb-5">
          <h2 className="font-display text-xl text-stone-800">Recetas recientes</h2>
          <div className="flex-1 h-px bg-stone-200" />
        </div>

        {recipes.length === 0 ? (
          <p className="text-stone-400 text-sm font-light text-center py-8">No hay recetas aún</p>
        ) : (
          <div className="space-y-2">
            {recipes.slice(0, 5).map((r) => (
              <div key={r.id}
                className="flex items-center justify-between p-4 bg-white border border-stone-200
                           hover:border-gold transition-all duration-200">
                <div>
                  <p className="text-stone-800 text-sm font-light">{r.name}</p>
                  <p className="text-stone-400 text-xs mt-0.5">{r.yield} {r.yield_unit}</p>
                </div>
                <span className={`text-xs tracking-widest uppercase px-2 py-1 border
                  ${r.is_base
                    ? "text-gold border-gold border-opacity-40"
                    : "text-stone-400 border-stone-200"}`}>
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