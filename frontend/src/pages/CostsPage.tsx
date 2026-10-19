import { useEffect, useState } from "react";
import { recipeService } from "../api/recipeService";
import type { IRecipe } from "../types/recipe";
import QuotationGenerator from "../components/QuotationGenerator";

type ToastState = { msg: string; ok: boolean } | null;

export default function CostsPage() {
  const [recipes, setRecipes]   = useState<IRecipe[]>([]);
  const [selected, setSelected] = useState<string>("");
  const [toast, setToast]       = useState<ToastState>(null);

  const showToast = (msg: string, ok: boolean) => {
    setToast({ msg, ok });
    setTimeout(() => setToast(null), ok ? 1500 : 4000);
  };

  useEffect(() => { recipeService.getAll().then(setRecipes); }, []);

  const recipe = recipes.find((r) => r.id === selected) ?? null;

  return (
    <div className="px-6 py-10 max-w-2xl mx-auto md:max-w-none md:px-10">

      {toast && (
        <div className={`fixed top-4 right-4 z-50 px-4 py-2 text-xs tracking-widest uppercase font-medium text-crema transition-all
          ${toast.ok ? "bg-negro" : "bg-peligro"}`}>
          {toast.msg}
        </div>
      )}

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
                     px-4 py-3 text-sm font-light focus:outline-none focus:border-oro transition-all"
        >
          <option value="">— Elige una receta —</option>
          {recipes.map((r) => <option key={r.id} value={r.id}>{r.name}</option>)}
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
            onDeferredSaveDone={(ok) => showToast(ok ? "Costos guardados" : "Error al guardar costos", ok)}
          />
        </div>
      )}
    </div>
  );
}
