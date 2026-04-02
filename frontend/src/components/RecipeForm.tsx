import { useState, useEffect, useCallback } from "react";
import type { IRecipe, IIngredientOption, RecipeFormData } from "../types/recipe";
import { emptyRecipeForm } from "../types/recipe";
import RecipePhotoUpload from "./RecipePhotoUpload";

interface Props {
  editing: IRecipe | null;
  ingredients: IIngredientOption[];
  onSave: (data: RecipeFormData) => Promise<void>;
  onClose: () => void;
}

export default function RecipeForm({ editing, ingredients, onSave, onClose }: Props) {
  const [form, setForm] = useState<RecipeFormData>(emptyRecipeForm);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [descriptionTouched, setDescriptionTouched] = useState(false);

  const handleClose = useCallback(() => onClose(), [onClose]);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => { if (e.key === "Escape") handleClose(); };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [handleClose]);

  useEffect(() => {
    if (editing) {
      setForm({
        name: editing.name,
        description: editing.description || "",
        yield: editing.yield,
        yield_unit: editing.yield_unit,
        ingredients: editing.ingredients.map((i) => ({
          ingredient_id: i.ingredient_id,
          quantity: i.quantity,
          unit: i.unit,
        })),
      });
    } else {
      setForm(emptyRecipeForm);
    }
    setError("");
    setDescriptionTouched(false);
  }, [editing]);

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
    if (!form.description.trim()) { setDescriptionTouched(true); setError("La descripción es obligatoria para generar cotización"); return; }
    if (!form.yield || form.yield <= 0) { setError("El rendimiento debe ser mayor a 0"); return; }
    if (form.ingredients.length === 0) { setError("Agrega al menos un ingrediente"); return; }
    setSaving(true);
    try {
      await onSave(form);
    } catch (e: any) {
      setError(e.response?.data?.error || "Error al guardar");
    } finally {
      setSaving(false);
    }
  };

  const inputCls =
    "w-full bg-vanilla-100 border border-stone-200 text-stone-800 px-3 py-2 text-sm font-light focus:outline-none focus:border-gold transition-all";

  return (
    <div
      className="fixed inset-0 z-50 bg-stone-800 bg-opacity-40 flex items-end md:items-center justify-center p-4"
      onClick={(e) => { if (e.target === e.currentTarget) handleClose(); }}
    >
      <div className="bg-white border border-stone-200 w-full max-w-lg max-h-[90vh] overflow-y-auto p-6 space-y-5 shadow-lg">
        <div className="flex items-center justify-between">
          <h2 className="font-display text-2xl text-stone-800">
            {editing ? "Editar receta" : "Nueva receta"}
          </h2>
          <button onClick={onClose} className="text-stone-400 hover:text-stone-800 text-xl">
            ×
          </button>
        </div>

        <div>
          <label className="block text-xs tracking-widest uppercase text-stone-400 mb-1.5 font-light">
            Nombre
          </label>
          <input
            type="text"
            value={form.name}
            placeholder="ej. Torta Hojarasca 15 personas"
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            className={inputCls}
          />
        </div>

        <div>
          <label className="block text-xs tracking-widest uppercase text-stone-400 mb-1.5 font-light">
            Descripción <span className="text-terracota-500">*</span>
          </label>
          <textarea
            value={form.description}
            placeholder="ej. Torta de hojarasca con manjar y crema, rinde 15 porciones"
            onChange={(e) => { setForm({ ...form, description: e.target.value }); if (descriptionTouched && e.target.value.trim()) setError(""); }}
            onBlur={() => { setDescriptionTouched(true); if (!form.description.trim()) setError("La descripción es obligatoria para generar cotización"); }}
            rows={2}
            className={`w-full bg-vanilla-100 border text-stone-800 px-3 py-2 text-sm font-light focus:outline-none transition-all resize-none ${descriptionTouched && !form.description.trim() ? "border-terracota-500 focus:border-terracota-500" : "border-stone-200 focus:border-gold"}`}
          />
          {descriptionTouched && !form.description.trim() && (
            <p className="text-terracota-500 text-xs mt-1 tracking-wide">La descripción es obligatoria para generar cotización</p>
          )}
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-xs tracking-widest uppercase text-stone-400 mb-1.5 font-light">
              Rendimiento
            </label>
            <input
              type="number"
              value={form.yield === "" ? "" : form.yield}
              placeholder="ej. 15"
              onChange={(e) => setForm({ ...form, yield: parseFloat(e.target.value) || "" })}
              className={inputCls}
            />
          </div>
          <div>
            <label className="block text-xs tracking-widest uppercase text-stone-400 mb-1.5 font-light">
              Unidad
            </label>
            <input
              type="text"
              value={form.yield_unit}
              onChange={(e) => setForm({ ...form, yield_unit: e.target.value })}
              className={inputCls}
            />
          </div>
        </div>


        <div>
          <div className="flex items-center justify-between mb-3">
            <label className="text-xs tracking-widest uppercase text-stone-400 font-light">
              Ingredientes
            </label>
            <button
              onClick={addIngredient}
              className="text-xs text-gold hover:text-gold-light tracking-wider transition-colors"
            >
              + Agregar
            </button>
          </div>
          <div className="space-y-2">
            {form.ingredients.map((ing, idx) => (
              <div key={idx} className="flex gap-2 items-center">
                <select
                  value={ing.ingredient_id}
                  onChange={(e) => updateIngredient(idx, "ingredient_id", e.target.value)}
                  className="flex-1 bg-vanilla-100 border border-stone-200 text-stone-800 px-3 py-2 text-xs font-light focus:outline-none focus:border-gold transition-all min-w-0"
                >
                  {ingredients.map((i) => (
                    <option key={i.id} value={i.id}>
                      {i.name}
                    </option>
                  ))}
                </select>
                <input
                  type="number"
                  value={ing.quantity === 0 ? "" : ing.quantity}
                  placeholder="Qty"
                  onChange={(e) => updateIngredient(idx, "quantity", parseFloat(e.target.value) || 0)}
                  className="w-20 bg-vanilla-100 border border-stone-200 text-stone-800 px-3 py-2 text-xs font-light focus:outline-none focus:border-gold transition-all"
                />
                <input
                  type="text"
                  value={ing.unit}
                  placeholder="Und"
                  onChange={(e) => updateIngredient(idx, "unit", e.target.value)}
                  className="w-16 bg-vanilla-100 border border-stone-200 text-stone-800 px-3 py-2 text-xs font-light focus:outline-none focus:border-gold transition-all"
                />
                <button
                  onClick={() => removeIngredient(idx)}
                  className="text-stone-400 hover:text-terracota-500 text-lg shrink-0 transition-colors"
                >
                  ×
                </button>
              </div>
            ))}
          </div>
        </div>

        {editing && (
          <RecipePhotoUpload
            recipeId={editing.id}
            currentPhotoUrl={editing.photo_url}
            onUploaded={() => {}}
          />
        )}

        {error && <p className="text-terracota-500 text-xs tracking-wide">{error}</p>}

        <button
          onClick={handleSave}
          disabled={saving}
          className="w-full bg-gold text-white text-xs tracking-widest uppercase font-medium py-3 hover:bg-gold-light transition-all disabled:opacity-50"
        >
          {saving ? "Guardando..." : editing ? "Guardar cambios" : "Guardar receta"}
        </button>
      </div>
    </div>
  );
}
