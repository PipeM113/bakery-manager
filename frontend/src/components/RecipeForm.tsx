import { useState, useEffect } from "react";
import type { IRecipe, IIngredientOption, RecipeFormData } from "../types/recipe";
import { emptyRecipeForm } from "../types/recipe";
import RecipePhotoUpload from "./RecipePhotoUpload";
import { apiErrorMessage } from "../utils/apiError";
import { equivalenceNeeded, unitsFor } from "../utils/units";
import { compareIngredients, ingredientLabel, sortKey } from "../utils/ingredients";

interface Props {
  editing: IRecipe | null;
  ingredients: IIngredientOption[];
  onSave: (data: RecipeFormData) => Promise<void>;
  onSaveAs?: (data: RecipeFormData, versionName: string) => Promise<void>;
  onClose: () => void;
}

export default function RecipeForm({ editing, ingredients, onSave, onSaveAs, onClose }: Props) {
  const [form, setForm] = useState<RecipeFormData>(emptyRecipeForm);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [descriptionTouched, setDescriptionTouched] = useState(false);
  const [showVersionModal, setShowVersionModal] = useState(false);
  const [versionName, setVersionName] = useState("");
  const [savingAs, setSavingAs] = useState(false);
  const [versionError, setVersionError] = useState("");

  // Fix 2: dirty-state tracking
  const [isDirty, setIsDirty] = useState(false);
  const [showCloseConfirm, setShowCloseConfirm] = useState(false);

  // Fix 1: ingredient picker modal
  const [showIngPicker, setShowIngPicker] = useState(false);
  const [ingSearch, setIngSearch] = useState("");
  const [pickerSelected, setPickerSelected] = useState<Set<string>>(new Set());

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
    setIsDirty(false);
  }, [editing]);

  // Fix 2: gate close behind dirty check
  const requestClose = () => {
    if (isDirty) {
      setShowCloseConfirm(true);
    } else {
      onClose();
    }
  };

  const markDirty = () => setIsDirty(true);

  // Fix 1: ingredient picker helpers
  const openIngPicker = () => {
    setIngSearch("");
    setPickerSelected(new Set());
    setShowIngPicker(true);
  };

  const togglePicker = (id: string) => {
    setPickerSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  };

  const confirmIngPicker = () => {
    if (pickerSelected.size > 0) {
      const toAdd = Array.from(pickerSelected).map((id) => {
        const ing = ingredients.find((i) => i.id === id)!;
        return { ingredient_id: id, quantity: 0, unit: ing.default_unit };
      });
      setForm((f) => ({ ...f, ingredients: [...f.ingredients, ...toAdd] }));
      markDirty();
    }
    setShowIngPicker(false);
  };

  // Ingredients first, then supplies (boxes, cardboard); each group stays in name order.
  const filteredForPicker = ingredients
    .filter((i) => sortKey(ingredientLabel(i)).includes(sortKey(ingSearch)))
    .sort((a, b) => Number((a.kind ?? "ingredient") === "supply") - Number((b.kind ?? "ingredient") === "supply") || compareIngredients(a, b));

  // Using a unit of another family than the one the product is kept in needs its equivalence.
  const missingEquivalence = (line: { ingredient_id: string; unit: string }) => {
    const source = ingredients.find((i) => i.id === line.ingredient_id);
    return source ? equivalenceNeeded(line.unit, source.default_unit, source) : null;
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
    markDirty();
  };

  const removeIngredient = (idx: number) => {
    setForm((f) => ({ ...f, ingredients: f.ingredients.filter((_, i) => i !== idx) }));
    markDirty();
  };

  const validateForm = (): boolean => {
    if (!form.name.trim()) { setError("El nombre es requerido"); return false; }
    if (!form.description.trim()) { setDescriptionTouched(true); setError("La descripción es obligatoria para generar cotización"); return false; }
    if (!form.yield || form.yield <= 0) { setError("El rendimiento debe ser mayor a 0"); return false; }
    if (form.ingredients.length === 0) { setError("Agrega al menos un ingrediente"); return false; }
    return true;
  };

  const handleSave = async () => {
    setError("");
    if (!validateForm()) return;
    setSaving(true);
    try {
      await onSave(form);
    } catch (e) {
      setError(apiErrorMessage(e, "Error al guardar"));
    } finally {
      setSaving(false);
    }
  };

  const openVersionModal = () => {
    setError("");
    if (!validateForm()) return;
    setVersionName("");
    setVersionError("");
    setShowVersionModal(true);
  };

  const handleConfirmSaveAs = async () => {
    const name = versionName.trim();
    if (!name) { setVersionError("El nombre de la versión es requerido"); return; }
    setSavingAs(true);
    setVersionError("");
    try {
      await onSaveAs!(form, name);
      setShowVersionModal(false);
    } catch (e) {
      setVersionError(apiErrorMessage(e, "Error al crear versión"));
    } finally {
      setSavingAs(false);
    }
  };

  const inputCls =
    "w-full bg-crema border border-linea text-negro px-3 py-2 text-sm font-light focus:outline-none focus:border-oro transition-all";

  return (
    // Fix 3: no onClick on backdrop (removed backdrop-click-to-close)
    <div className="fixed inset-0 z-[60] bg-negro bg-opacity-40 flex items-end md:items-center justify-center p-4">
      <div className="bg-white border border-linea w-full max-w-lg max-h-[90vh] overflow-y-auto p-6 space-y-5 shadow-lg">
        <div className="flex items-center justify-between">
          <h2 className="font-display text-2xl text-negro">
            {editing ? "Editar receta" : "Nueva receta"}
          </h2>
          {/* Fix 2: X uses requestClose */}
          <button onClick={requestClose} className="text-negro/70 hover:text-negro text-xl">
            ×
          </button>
        </div>

        <div>
          <label className="block text-xs tracking-widest uppercase text-negro/70 mb-1.5 font-light">
            Nombre
          </label>
          <input
            type="text"
            value={form.name}
            placeholder="ej. Torta Hojarasca 15 personas"
            onChange={(e) => { setForm({ ...form, name: e.target.value }); markDirty(); }}
            className={inputCls}
          />
        </div>

        <div>
          <label className="block text-xs tracking-widest uppercase text-negro/70 mb-1.5 font-light">
            Descripción <span className="text-peligro">*</span>
          </label>
          <textarea
            value={form.description}
            placeholder="ej. Torta de hojarasca con manjar y crema, rinde 15 porciones"
            maxLength={500}
            onChange={(e) => {
              setForm({ ...form, description: e.target.value });
              markDirty();
              if (descriptionTouched && e.target.value.trim()) setError("");
            }}
            onBlur={() => { setDescriptionTouched(true); if (!form.description.trim()) setError("La descripción es obligatoria para generar cotización"); }}
            rows={2}
            className={`w-full bg-crema border text-negro px-3 py-2 text-sm font-light focus:outline-none transition-all resize-none ${descriptionTouched && !form.description.trim() ? "border-peligro focus:border-peligro" : "border-linea focus:border-oro"}`}
          />
          <div className="flex justify-between items-start mt-1">
            {descriptionTouched && !form.description.trim() ? (
              <p className="text-peligro text-xs tracking-wide">La descripción es obligatoria para generar cotización</p>
            ) : (
              <span />
            )}
            <p className={`text-xs ${form.description.length >= 500 ? "text-peligro" : form.description.length > 450 ? "text-peligro" : "text-negro/70"}`}>
              {form.description.length}/500
            </p>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-xs tracking-widest uppercase text-negro/70 mb-1.5 font-light">
              Rendimiento
            </label>
            <input
              type="number"
              value={form.yield === "" ? "" : form.yield}
              placeholder="ej. 15"
              onChange={(e) => { setForm({ ...form, yield: parseFloat(e.target.value) || "" }); markDirty(); }}
              className={inputCls}
            />
          </div>
          <div>
            <label className="block text-xs tracking-widest uppercase text-negro/70 mb-1.5 font-light">
              Unidad
            </label>
            <input
              type="text"
              value={form.yield_unit}
              onChange={(e) => { setForm({ ...form, yield_unit: e.target.value }); markDirty(); }}
              className={inputCls}
            />
          </div>
        </div>

        {/* Ingredients section */}
        <div>
          <label className="block text-xs tracking-widest uppercase text-negro/70 mb-3 font-light">
            Ingredientes
          </label>

          {/* Fix 4: ingredient list BEFORE the add button */}
          <div className="space-y-2">
            {form.ingredients.map((ing, idx) => (
              <div key={idx} className="space-y-1">
              <div className="flex gap-2 items-center">
                <select
                  value={ing.ingredient_id}
                  onChange={(e) => updateIngredient(idx, "ingredient_id", e.target.value)}
                  className="flex-1 bg-crema border border-linea text-negro px-3 py-2 text-xs font-light focus:outline-none focus:border-oro transition-all min-w-0"
                >
                  {([["ingredient", "Ingredientes"], ["supply", "Insumos (cajas, cartones)"]] as const).map(([kind, label]) => (
                    <optgroup key={kind} label={label}>
                      {ingredients.filter((i) => (i.kind ?? "ingredient") === kind).sort(compareIngredients).map((i) => (
                        <option key={i.id} value={i.id}>
                          {ingredientLabel(i)}
                        </option>
                      ))}
                    </optgroup>
                  ))}
                </select>
                <input
                  type="number"
                  value={ing.quantity === 0 ? "" : ing.quantity}
                  placeholder="Qty"
                  onChange={(e) => updateIngredient(idx, "quantity", parseFloat(e.target.value) || 0)}
                  className="w-20 bg-crema border border-linea text-negro px-3 py-2 text-xs font-light focus:outline-none focus:border-oro transition-all"
                />
                <select
                  aria-label="Unidad"
                  value={ing.unit}
                  onChange={(e) => updateIngredient(idx, "unit", e.target.value)}
                  className="w-20 bg-crema border border-linea text-negro px-2 py-2 text-xs font-light focus:outline-none focus:border-oro transition-all"
                >
                  {unitsFor(ingredients.find((i) => i.id === ing.ingredient_id)?.default_unit ?? ing.unit).concat(
                    unitsFor(ing.unit).includes(ing.unit) ? [] : [ing.unit]).map((u) => (
                    <option key={u} value={u}>{u}</option>
                  ))}
                </select>
                <button
                  onClick={() => removeIngredient(idx)}
                  className="text-negro/70 hover:text-peligro text-lg shrink-0 transition-colors"
                >
                  ×
                </button>
              </div>
              {missingEquivalence(ing) && (
                <p className="text-peligro text-xs font-light -mt-1">
                  Para usar {ing.unit} aquí falta el dato «{missingEquivalence(ing) === "grams_per_liter" ? "Peso de 1 litro" : "Peso de 1 unidad"}» del producto. Complétalo en Ingredientes (Editar → Equivalencias).
                </p>
              )}
              </div>
            ))}
          </div>

          {/* Fix 4: button AFTER the list; Fix 1: opens picker modal */}
          <button
            onClick={openIngPicker}
            className="mt-3 text-xs text-oro hover:text-negro tracking-wider transition-colors"
          >
            + Agregar ingrediente
          </button>
        </div>

        {editing && (
          <RecipePhotoUpload
            recipeId={editing.id}
            currentPhotoUrl={editing.photo_url}
            onUploaded={() => {}}
          />
        )}

        {error && <p className="text-peligro text-xs tracking-wide">{error}</p>}

        {editing && onSaveAs ? (
          <div className="flex gap-2">
            <button
              onClick={openVersionModal}
              disabled={saving || savingAs}
              className="flex-1 border border-linea text-negro/70 text-xs tracking-widest uppercase font-medium py-3 hover:border-oro hover:text-oro transition-all disabled:opacity-50"
            >
              Guardar como...
            </button>
            <button
              onClick={handleSave}
              disabled={saving || savingAs}
              className="flex-1 bg-negro text-crema text-xs tracking-widest uppercase font-medium py-3 hover:bg-black transition-all disabled:opacity-50"
            >
              {saving ? "Guardando..." : "Guardar"}
            </button>
          </div>
        ) : (
          <button
            onClick={handleSave}
            disabled={saving}
            className="w-full bg-negro text-crema text-xs tracking-widest uppercase font-medium py-3 hover:bg-black transition-all disabled:opacity-50"
          >
            {saving ? "Guardando..." : editing ? "Guardar cambios" : "Guardar receta"}
          </button>
        )}
      </div>

      {/* Fix 1: Ingredient picker modal */}
      {showIngPicker && (
        // Fix 3: no backdrop click, no ESC
        <div className="fixed inset-0 z-[60] bg-negro bg-opacity-60 flex items-center justify-center p-4">
          <div className="bg-white border border-linea w-full max-w-sm shadow-xl flex flex-col max-h-[80vh]">
            <div className="flex items-center justify-between p-4 border-b border-linea">
              <h3 className="font-display text-lg text-negro">Seleccionar ingredientes</h3>
              <button
                onClick={() => setShowIngPicker(false)}
                className="text-negro/70 hover:text-negro text-xl leading-none"
              >
                ×
              </button>
            </div>

            {/* Search input */}
            <div className="p-3 border-b border-linea">
              <input
                type="text"
                value={ingSearch}
                onChange={(e) => setIngSearch(e.target.value)}
                placeholder="Buscar ingrediente..."
                autoFocus
                className="w-full bg-crema border border-linea text-negro px-3 py-2 text-sm font-light focus:outline-none focus:border-oro transition-all"
              />
            </div>

            {/* Ingredient list */}
            <div className="overflow-y-auto flex-1 p-2">
              {filteredForPicker.length === 0 ? (
                <p className="text-center text-negro/70 text-sm font-light py-6">
                  No se encontraron ingredientes
                </p>
              ) : (
                <div className="space-y-0.5">
                  {filteredForPicker.map((i, idx, all) => (
                    <div key={i.id}>
                    {(idx === 0 || (all[idx - 1].kind ?? "ingredient") !== (i.kind ?? "ingredient")) && (
                      <p className="px-3 pt-2 pb-1 text-xs tracking-widest uppercase text-negro/70 font-light">
                        {(i.kind ?? "ingredient") === "supply" ? "Insumos (cajas, cartones)" : "Ingredientes"}
                      </p>
                    )}
                    <label
                      key={i.id}
                      className="flex items-center gap-3 px-3 py-2 hover:bg-lino cursor-pointer transition-colors"
                    >
                      <input
                        type="checkbox"
                        checked={pickerSelected.has(i.id)}
                        onChange={() => togglePicker(i.id)}
                        className="accent-negro w-4 h-4"
                      />
                      <span className="text-negro/70 text-sm font-light">{ingredientLabel(i)}</span>
                    </label>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Actions */}
            <div className="flex gap-2 p-4 border-t border-linea">
              <button
                onClick={() => setShowIngPicker(false)}
                className="flex-1 border border-linea text-negro/70 text-xs tracking-widest uppercase font-medium py-2.5 hover:bg-lino transition-all"
              >
                Cancelar
              </button>
              <button
                onClick={confirmIngPicker}
                disabled={pickerSelected.size === 0}
                className="flex-1 bg-negro text-crema text-xs tracking-widest uppercase font-medium py-2.5 hover:bg-black transition-all disabled:opacity-40 disabled:cursor-not-allowed"
              >
                Confirmar ({pickerSelected.size})
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Save-as version modal */}
      {showVersionModal && (
        <div className="fixed inset-0 z-[60] bg-negro bg-opacity-60 flex items-center justify-center p-4">
          <div className="bg-white border border-linea w-full max-w-sm p-6 space-y-4 shadow-xl">
            <div className="flex items-center justify-between">
              <h3 className="font-display text-xl text-negro">Guardar como nueva versión</h3>
              <button
                onClick={() => setShowVersionModal(false)}
                className="text-negro/70 hover:text-negro text-xl"
              >
                ×
              </button>
            </div>

            <p className="text-negro/70 text-xs font-light leading-relaxed">
              Se creará una copia de la receta con los cambios actuales. La receta original no se modificará.
            </p>

            <div>
              <label className="block text-xs tracking-widest uppercase text-negro/70 mb-1.5 font-light">
                Nombre de la versión
              </label>
              <input
                type="text"
                value={versionName}
                onChange={(e) => { setVersionName(e.target.value); if (versionError) setVersionError(""); }}
                placeholder="ej. 12 porciones, sin gluten, versión verano"
                className="w-full bg-crema border border-linea text-negro px-3 py-2 text-sm font-light focus:outline-none focus:border-oro transition-all"
                autoFocus
                onKeyDown={(e) => { if (e.key === "Enter") handleConfirmSaveAs(); }}
              />
            </div>

            {versionError && (
              <p className="text-peligro text-xs tracking-wide">{versionError}</p>
            )}

            <button
              onClick={handleConfirmSaveAs}
              disabled={savingAs}
              className="w-full bg-negro text-crema text-xs tracking-widest uppercase font-medium py-3 hover:bg-black transition-all disabled:opacity-50"
            >
              {savingAs ? "Creando versión..." : "Crear versión"}
            </button>
          </div>
        </div>
      )}

      {/* Fix 2: Unsaved changes confirmation dialog */}
      {showCloseConfirm && (
        <div className="fixed inset-0 z-[70] bg-negro bg-opacity-70 flex items-center justify-center p-4">
          <div className="bg-white border border-linea w-full max-w-xs p-6 space-y-4 shadow-xl">
            <h3 className="font-display text-xl text-negro">¿Salir sin guardar?</h3>
            <p className="text-negro/70 text-sm font-light leading-relaxed">
              Los cambios realizados se perderán.
            </p>
            <div className="flex gap-2">
              <button
                onClick={() => setShowCloseConfirm(false)}
                className="flex-1 border border-linea text-negro/70 text-xs tracking-widest uppercase font-medium py-2.5 hover:border-oro hover:text-oro transition-all"
              >
                Continuar editando
              </button>
              <button
                onClick={() => { setShowCloseConfirm(false); onClose(); }}
                className="flex-1 bg-peligro text-crema text-xs tracking-widest uppercase font-medium py-2.5 hover:opacity-90 transition-all"
              >
                Salir sin guardar
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
