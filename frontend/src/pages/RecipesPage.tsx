import { useEffect, useState } from "react";
import api from "../services/api";

interface Ingredient {
  id: string;
  name: string;
  default_unit: string;
}

interface RecipeIngredient {
  ingredient_id: string;
  quantity: number;
  unit: string;
  name?: string;
}

interface Recipe {
  id: string;
  name: string;
  yield: number;
  yield_unit: string;
  is_base: boolean;
  parent_id: string | null;
  indirect_cost_pct: number;
  labor_cost_pct: number;
  ingredients: RecipeIngredient[];
}

const emptyForm = {
  name: "", yield: 1, yield_unit: "porciones",
  indirect_cost_pct: 0.15, labor_cost_pct: 0.30,
  ingredients: [] as RecipeIngredient[],
};

export default function RecipesPage() {
  const [recipes, setRecipes]         = useState<Recipe[]>([]);
  const [ingredients, setIngredients] = useState<Ingredient[]>([]);
  const [loading, setLoading]         = useState(true);
  const [showForm, setShowForm]       = useState(false);
  const [form, setForm]               = useState(emptyForm);
  const [saving, setSaving]           = useState(false);
  const [error, setError]             = useState("");
  const [selected, setSelected]       = useState<Recipe | null>(null);
  const [scalePortions, setScalePortions] = useState("");
  const [scaled, setScaled]           = useState<Recipe | null>(null);

  const load = () => {
    Promise.all([api.get("/recipes"), api.get("/ingredients")])
      .then(([r, i]) => { setRecipes(r.data); setIngredients(i.data); })
      .finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, []);

  const openCreate = () => {
    setForm({ ...emptyForm, ingredients: [] });
    setError("");
    setShowForm(true);
  };

  const addIngredient = () => {
    if (ingredients.length === 0) return;
    setForm((f) => ({
      ...f,
      ingredients: [
        ...f.ingredients,
        { ingredient_id: ingredients[0].id, quantity: 0, unit: ingredients[0].default_unit },
      ],
    }));
  };

  const updateIngredient = (idx: number, field: string, value: string | number) => {
    setForm((f) => {
      const updated = [...f.ingredients];
      if (field === "ingredient_id") {
        const ing = ingredients.find((i) => i.id === value);
        updated[idx] = { ...updated[idx], ingredient_id: value as string, unit: ing?.default_unit || "gr" };
      } else {
        updated[idx] = { ...updated[idx], [field]: value };
      }
      return { ...f, ingredients: updated };
    });
  };

  const removeIngredient = (idx: number) => {
    setForm((f) => ({ ...f, ingredients: f.ingredients.filter((_, i) => i !== idx) }));
  };

  const handleSave = async () => {
    setError("");
    if (!form.name.trim()) { setError("El nombre es requerido"); return; }
    if (form.ingredients.length === 0) { setError("Agrega al menos un ingrediente"); return; }
    setSaving(true);
    try {
      await api.post("/recipes", form);
      setShowForm(false);
      load();
    } catch (e: any) {
      setError(e.response?.data?.error || "Error al guardar");
    } finally {
      setSaving(false);
    }
  };

  const handleScale = async () => {
    if (!selected || !scalePortions) return;
    const portions = parseFloat(scalePortions);
    if (portions <= 0) return;
    const { data } = await api.post(`/recipes/${selected.id}/scale`, { portions });
    setScaled(data);
  };

  const handleDelete = async (id: string) => {
    if (!confirm("¿Eliminar esta receta?")) return;
    await api.delete(`/recipes/${id}`);
    load();
  };

  if (loading) return (
    <div className="min-h-screen flex items-center justify-center">
      <span className="text-gold text-sm tracking-widest uppercase animate-pulse">Cargando...</span>
    </div>
  );

  return (
    <div className="px-6 py-10 max-w-2xl mx-auto md:max-w-none md:px-10">

      {/* Header */}
      <div className="flex items-end justify-between mb-8">
        <div>
          <div className="flex items-center gap-3 mb-2">
            <div className="h-px w-8 bg-gold opacity-50" />
            <span className="text-gold text-xs tracking-[0.3em] uppercase font-light">Recetario</span>
          </div>
          <h1 className="font-display text-4xl text-cream">Recetas</h1>
        </div>
        <button onClick={openCreate}
          className="bg-gold text-noir text-xs tracking-widest uppercase font-medium px-5 py-3
                     hover:bg-gold-light transition-all duration-200">
          + Nueva
        </button>
      </div>

      {/* Lista de recetas */}
      {recipes.length === 0 ? (
        <p className="text-center text-cream-muted text-sm font-light py-16">No hay recetas aún</p>
      ) : (
        <div className="space-y-2">
          {recipes.map((r) => (
            <div key={r.id}
              className="bg-noir-700 border border-gold border-opacity-10
                         hover:border-opacity-30 transition-all duration-200">
              <div className="flex items-center justify-between p-4">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <p className="text-cream text-sm font-light truncate">{r.name}</p>
                    <span className={`text-xs px-2 py-0.5 border shrink-0
                      ${r.is_base ? "text-gold border-gold border-opacity-40" : "text-cream-muted border-cream-muted border-opacity-20"}`}>
                      {r.is_base ? "Base" : "Versión"}
                    </span>
                  </div>
                  <p className="text-cream-muted text-xs mt-0.5 font-light">
                    {r.yield} {r.yield_unit} · {r.ingredients?.length || 0} ingredientes
                  </p>
                </div>
                <div className="flex items-center gap-3 ml-4">
                  <button onClick={() => { setSelected(r); setScaled(null); setScalePortions(""); }}
                    className="text-xs text-cream-muted hover:text-gold tracking-wider transition-colors">
                    Escalar
                  </button>
                  <button onClick={() => handleDelete(r.id)}
                    className="text-xs text-cream-muted hover:text-terracota-400 tracking-wider transition-colors">
                    Eliminar
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Modal formulario nueva receta */}
      {showForm && (
        <div className="fixed inset-0 z-50 bg-black bg-opacity-75 flex items-end md:items-center justify-center p-4">
          <div className="bg-noir-800 border border-gold border-opacity-30 w-full max-w-lg
                          max-h-[90vh] overflow-y-auto p-6 space-y-5">
            <div className="flex items-center justify-between">
              <h2 className="font-display text-2xl text-gold">Nueva receta</h2>
              <button onClick={() => setShowForm(false)} className="text-cream-muted hover:text-cream text-xl">×</button>
            </div>

            {/* Nombre */}
            <div>
              <label className="block text-xs tracking-widest uppercase text-cream-muted mb-1.5 font-light">Nombre</label>
              <input type="text" value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                className="w-full bg-noir-700 border border-gold border-opacity-20 text-cream
                           px-4 py-2.5 text-sm font-light focus:outline-none focus:border-gold
                           focus:border-opacity-60 transition-all" />
            </div>

            {/* Yield */}
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs tracking-widest uppercase text-cream-muted mb-1.5 font-light">Rendimiento</label>
                <input type="number" value={form.yield}
                  onChange={(e) => setForm({ ...form, yield: parseFloat(e.target.value) || 1 })}
                  className="w-full bg-noir-700 border border-gold border-opacity-20 text-cream
                             px-4 py-2.5 text-sm font-light focus:outline-none focus:border-gold
                             focus:border-opacity-60 transition-all" />
              </div>
              <div>
                <label className="block text-xs tracking-widest uppercase text-cream-muted mb-1.5 font-light">Unidad</label>
                <input type="text" value={form.yield_unit}
                  onChange={(e) => setForm({ ...form, yield_unit: e.target.value })}
                  className="w-full bg-noir-700 border border-gold border-opacity-20 text-cream
                             px-4 py-2.5 text-sm font-light focus:outline-none focus:border-gold
                             focus:border-opacity-60 transition-all" />
              </div>
            </div>

            {/* Porcentajes */}
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs tracking-widest uppercase text-cream-muted mb-1.5 font-light">% Costos indirectos</label>
                <input type="number" step="0.01" value={form.indirect_cost_pct}
                  onChange={(e) => setForm({ ...form, indirect_cost_pct: parseFloat(e.target.value) || 0 })}
                  className="w-full bg-noir-700 border border-gold border-opacity-20 text-cream
                             px-4 py-2.5 text-sm font-light focus:outline-none focus:border-gold
                             focus:border-opacity-60 transition-all" />
              </div>
              <div>
                <label className="block text-xs tracking-widest uppercase text-cream-muted mb-1.5 font-light">% Mano de obra</label>
                <input type="number" step="0.01" value={form.labor_cost_pct}
                  onChange={(e) => setForm({ ...form, labor_cost_pct: parseFloat(e.target.value) || 0 })}
                  className="w-full bg-noir-700 border border-gold border-opacity-20 text-cream
                             px-4 py-2.5 text-sm font-light focus:outline-none focus:border-gold
                             focus:border-opacity-60 transition-all" />
              </div>
            </div>

            {/* Ingredientes */}
            <div>
              <div className="flex items-center justify-between mb-3">
                <label className="text-xs tracking-widest uppercase text-cream-muted font-light">Ingredientes</label>
                <button onClick={addIngredient}
                  className="text-xs text-gold hover:text-gold-light tracking-wider transition-colors">
                  + Agregar
                </button>
              </div>
              <div className="space-y-2">
                {form.ingredients.map((ing, idx) => (
                  <div key={idx} className="flex gap-2 items-center">
                    <select value={ing.ingredient_id}
                      onChange={(e) => updateIngredient(idx, "ingredient_id", e.target.value)}
                      className="flex-1 bg-noir-700 border border-gold border-opacity-20 text-cream
                                 px-3 py-2 text-xs font-light focus:outline-none focus:border-gold
                                 focus:border-opacity-60 transition-all min-w-0">
                      {ingredients.map((i) => (
                        <option key={i.id} value={i.id}>{i.name}</option>
                      ))}
                    </select>
                    <input type="number" value={ing.quantity} placeholder="Qty"
                      onChange={(e) => updateIngredient(idx, "quantity", parseFloat(e.target.value) || 0)}
                      className="w-20 bg-noir-700 border border-gold border-opacity-20 text-cream
                                 px-3 py-2 text-xs font-light focus:outline-none focus:border-gold
                                 focus:border-opacity-60 transition-all" />
                    <input type="text" value={ing.unit} placeholder="Und"
                      onChange={(e) => updateIngredient(idx, "unit", e.target.value)}
                      className="w-16 bg-noir-700 border border-gold border-opacity-20 text-cream
                                 px-3 py-2 text-xs font-light focus:outline-none focus:border-gold
                                 focus:border-opacity-60 transition-all" />
                    <button onClick={() => removeIngredient(idx)}
                      className="text-cream-muted hover:text-terracota-400 text-lg shrink-0 transition-colors">×</button>
                  </div>
                ))}
              </div>
            </div>

            {error && <p className="text-red-400 text-xs tracking-wide">{error}</p>}

            <button onClick={handleSave} disabled={saving}
              className="w-full bg-gold text-noir text-xs tracking-widest uppercase font-medium
                         py-3 hover:bg-gold-light transition-all disabled:opacity-50">
              {saving ? "Guardando..." : "Guardar receta"}
            </button>
          </div>
        </div>
      )}

      {/* Modal escalado */}
      {selected && (
        <div className="fixed inset-0 z-50 bg-black bg-opacity-75 flex items-end md:items-center justify-center p-4">
          <div className="bg-noir-800 border border-gold border-opacity-30 w-full max-w-md p-6 space-y-5">
            <div className="flex items-center justify-between">
              <h2 className="font-display text-2xl text-gold">Escalar receta</h2>
              <button onClick={() => setSelected(null)} className="text-cream-muted hover:text-cream text-xl">×</button>
            </div>

            <p className="text-cream text-sm font-light">{selected.name}</p>
            <p className="text-cream-muted text-xs">Base: {selected.yield} {selected.yield_unit}</p>

            <div>
              <label className="block text-xs tracking-widest uppercase text-cream-muted mb-1.5 font-light">
                Número de porciones
              </label>
              <div className="flex gap-3">
                <input type="number" value={scalePortions}
                  onChange={(e) => { setScalePortions(e.target.value); setScaled(null); }}
                  className="flex-1 bg-noir-700 border border-gold border-opacity-20 text-cream
                             px-4 py-2.5 text-sm font-light focus:outline-none focus:border-gold
                             focus:border-opacity-60 transition-all" />
                <button onClick={handleScale}
                  className="bg-gold text-noir text-xs tracking-widest uppercase font-medium
                             px-5 hover:bg-gold-light transition-all">
                  Calcular
                </button>
              </div>
            </div>

            {scaled && (
              <div className="space-y-2 pt-2 border-t border-gold border-opacity-20">
                <p className="text-xs tracking-widest uppercase text-cream-muted font-light mb-3">
                  Ingredientes para {scaled.yield} {scaled.yield_unit}
                </p>
                {scaled.ingredients.map((ing) => (
                  <div key={ing.ingredient_id} className="flex justify-between items-center">
                    <span className="text-cream text-sm font-light">{ing.name}</span>
                    <span className="text-gold text-sm">
                      {ing.quantity.toFixed(2)} {ing.unit}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}