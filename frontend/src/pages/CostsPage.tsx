import { useEffect, useState } from "react";
import RecipeOptions from "../components/RecipeOptions";
import { recipeService } from "../api/recipeService";
import type { IRecipe } from "../types/recipe";
import { useToast } from "../hooks/useToast";
import QuotationGenerator from "../components/QuotationGenerator";

export default function CostsPage() {
  const [recipes, setRecipes]   = useState<IRecipe[]>([]);
  const [selected, setSelected] = useState<string>("");
  const toast = useToast();


  useEffect(() => { recipeService.getAll().then(setRecipes); }, []);

  const recipe = recipes.find((r) => r.id === selected) ?? null;

  return (
    <div className="px-6 py-10 max-w-2xl mx-auto md:max-w-none md:px-10">



      <div className="mb-8">
        <div className="flex items-center gap-3 mb-2">
          <div className="h-px w-8 bg-oro/40" />
          <span className="text-oro text-xs tracking-[0.3em] uppercase font-light">Análisis</span>
        </div>
        <h1 className="font-display text-4xl text-negro">Costos</h1>
      </div>

      <div className="mb-8">
        <label className="block text-xs tracking-widest uppercase text-negro/70 mb-2 font-light">
          Selecciona una receta
        </label>
        <select
          value={selected}
          onChange={(e) => setSelected(e.target.value)}
          className="w-full bg-white border border-linea text-negro
                     px-4 py-3 text-sm font-light focus:outline-none focus:border-oro transition-colors"
        >
          <option value="">— Elige una receta —</option>
          <RecipeOptions recipes={recipes} />
        </select>
      </div>

      {recipe && (
        <div className="bg-white border border-linea p-6">
          <div className="flex items-center gap-4 mb-5">
            <h2 className="font-display text-xl text-negro">Cotización</h2>
            <div className="flex-1 h-px bg-lino" />
          </div>
          <QuotationGenerator
            key={recipe.id}
            recipe={recipe}
            deferredSave
            onDeferredSaveDone={(ok) => (ok ? toast.success("Costos guardados") : toast.error("No se pudieron guardar los costos"))}
          />
        </div>
      )}
    </div>
  );
}
