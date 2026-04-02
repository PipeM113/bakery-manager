import { useEffect, useState } from "react";
import { recipeService } from "../api/recipeService";
import type { IRecipe } from "../types/recipe";
import QuotationGenerator from "../components/QuotationGenerator";

export default function CostsPage() {
  const [recipes, setRecipes]   = useState<IRecipe[]>([]);
  const [selected, setSelected] = useState<string>("");

  useEffect(() => { recipeService.getAll().then(setRecipes); }, []);

  const recipe = recipes.find((r) => r.id === selected) ?? null;

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
          onChange={(e) => setSelected(e.target.value)}
          className="w-full bg-white border border-stone-200 text-stone-800
                     px-4 py-3 text-sm font-light focus:outline-none focus:border-gold transition-all"
        >
          <option value="">— Elige una receta —</option>
          {recipes.map((r) => <option key={r.id} value={r.id}>{r.name}</option>)}
        </select>
      </div>

      {recipe && (
        <div className="bg-white border border-stone-200 p-6">
          <div className="flex items-center gap-4 mb-5">
            <h2 className="font-display text-xl text-stone-800">Cotización</h2>
            <div className="flex-1 h-px bg-stone-200" />
          </div>
          <QuotationGenerator key={recipe.id} recipe={recipe} />
        </div>
      )}
    </div>
  );
}
