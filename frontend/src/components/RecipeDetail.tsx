import { useState } from "react";
import type { IRecipe } from "../types/recipe";
import { recipeService } from "../api/recipeService";
import RecipePhotoUpload from "./RecipePhotoUpload";
import QuotationGenerator from "./QuotationGenerator";

interface Props {
  recipe: IRecipe;
  onClose: () => void;
  onEdit: (r: IRecipe) => void;
  onSaved: () => void;
}

export default function RecipeDetail({ recipe, onClose, onEdit, onSaved }: Props) {
  const [targetYield, setTargetYield] = useState(recipe.yield);
  const [showSaveModal, setShowSaveModal] = useState(false);
  const [saveName, setSaveName] = useState("");
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState("");
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [photoUrl, setPhotoUrl] = useState<string | null>(recipe.photo_url);
  const [showPhotoUpload, setShowPhotoUpload] = useState(false);

  const factor = recipe.yield > 0 ? targetYield / recipe.yield : 1;
  const isScaled = Math.abs(factor - 1) > 0.001;
  const maxSlider = Math.max(recipe.yield * 5, 10);

  const openSaveModal = () => {
    setSaveName(`${recipe.name} ×${factor.toFixed(1)}`);
    setSaveError("");
    setSaveSuccess(false);
    setShowSaveModal(true);
  };

  const handleSaveScaled = async () => {
    const name = saveName.trim();
    if (!name) { setSaveError("El nombre es requerido"); return; }
    setSaving(true);
    setSaveError("");
    try {
      await recipeService.saveScaled(recipe.id, factor, name);
      setSaveSuccess(true);
      setTimeout(() => {
        setShowSaveModal(false);
        onSaved();
      }, 1200);
    } catch (e: any) {
      setSaveError(e.response?.data?.error || "Error al guardar");
    } finally {
      setSaving(false);
    }
  };

  return (
    <>
      <div className="fixed inset-0 z-50 bg-stone-800 bg-opacity-40 flex items-end md:items-center justify-center p-4">
        <div className="bg-white border border-stone-200 w-full max-w-lg max-h-[90vh] overflow-y-auto p-6 space-y-5 shadow-lg">
          <div className="flex items-center justify-between">
            <h2 className="font-display text-2xl text-stone-800 truncate pr-4">{recipe.name}</h2>
            <button onClick={onClose} className="text-stone-400 hover:text-stone-800 text-xl shrink-0">
              ×
            </button>
          </div>

          <div className="flex items-center gap-3 flex-wrap">
            <span
              className={`text-xs px-2 py-0.5 border ${
                recipe.is_base
                  ? "text-gold border-gold border-opacity-40"
                  : "text-stone-400 border-stone-200"
              }`}
            >
              {recipe.is_base ? "Base" : "Versión"}
            </span>
            <span className="text-stone-400 text-xs font-light">
              Rendimiento base: {recipe.yield} {recipe.yield_unit}
            </span>
            <span className="text-stone-400 text-xs font-light">
              {recipe.ingredients.length} ingredientes
            </span>
          </div>

          {recipe.description && (
            <p className="text-stone-500 text-sm font-light leading-relaxed">{recipe.description}</p>
          )}

          {/* Foto */}
          {photoUrl && !showPhotoUpload && (
            <div className="relative">
              <img
                src={photoUrl}
                alt={recipe.name}
                className="w-full h-44 object-cover border border-stone-200"
              />
              <button
                onClick={() => setShowPhotoUpload(true)}
                className="absolute bottom-2 right-2 bg-white border border-stone-200 text-stone-600 text-xs px-2 py-1 hover:border-gold hover:text-gold transition-all"
              >
                Reemplazar foto
              </button>
            </div>
          )}

          {(!photoUrl || showPhotoUpload) && (
            <RecipePhotoUpload
              recipeId={recipe.id}
              currentPhotoUrl={photoUrl}
              onUploaded={(url) => {
                setPhotoUrl(url);
                setShowPhotoUpload(false);
              }}
            />
          )}

          {/* Escalado dinámico */}
          <div className="bg-vanilla-100 p-4 space-y-3">
            <div className="flex items-center justify-between">
              <label className="text-xs tracking-widest uppercase text-stone-400 font-light">
                Escalar rendimiento
              </label>
              {isScaled && (
                <button
                  onClick={() => setTargetYield(recipe.yield)}
                  className="text-xs text-stone-400 hover:text-gold transition-colors tracking-wide"
                >
                  Restablecer
                </button>
              )}
            </div>
            <div className="flex items-center gap-3">
              <input
                type="range"
                min={0.5}
                max={maxSlider}
                step={0.5}
                value={targetYield}
                onChange={(e) => setTargetYield(parseFloat(e.target.value))}
                className="flex-1 accent-gold"
              />
              <input
                type="number"
                min={0.1}
                step={0.5}
                value={targetYield}
                onChange={(e) => {
                  const v = parseFloat(e.target.value);
                  if (v > 0) setTargetYield(v);
                }}
                className="w-20 bg-white border border-stone-200 text-stone-800 px-3 py-1.5 text-sm font-light focus:outline-none focus:border-gold transition-all text-center"
              />
              <span className="text-stone-400 text-xs shrink-0 w-16 truncate">
                {recipe.yield_unit}
              </span>
            </div>
            {isScaled && (
              <div className="flex items-center justify-between">
                <p className="text-xs text-gold font-light">Factor ×{factor.toFixed(2)}</p>
                <button
                  onClick={openSaveModal}
                  className="text-xs bg-gold text-white px-3 py-1.5 tracking-widest uppercase font-medium hover:bg-gold-light transition-all"
                >
                  Guardar escalada
                </button>
              </div>
            )}
          </div>

          {/* Lista de ingredientes */}
          <div>
            <p className="text-xs tracking-widest uppercase text-stone-400 font-light mb-3">
              Ingredientes{isScaled ? ` · para ${targetYield} ${recipe.yield_unit}` : ""}
            </p>
            {recipe.ingredients.length === 0 ? (
              <p className="text-stone-400 text-xs font-light">Sin ingredientes registrados</p>
            ) : (
              <div className="space-y-1">
                {recipe.ingredients.map((ing) => {
                  const qty = ing.quantity * factor;
                  return (
                    <div
                      key={ing.ingredient_id}
                      className="flex justify-between items-center py-2 border-b border-stone-100 last:border-0"
                    >
                      <span className="text-stone-800 text-sm font-light">{ing.name}</span>
                      <span className={`text-sm tabular-nums ${isScaled ? "text-gold" : "text-stone-600"}`}>
                        {qty.toFixed(2)} {ing.unit}
                      </span>
                    </div>
                  );
                })}
              </div>
            )}
          </div>

          {/* Cotización */}
          <div className="bg-white border border-stone-200 p-5 space-y-1">
            <div className="flex items-center gap-4 mb-4">
              <p className="text-xs tracking-widest uppercase text-stone-400 font-light">
                Cotización
              </p>
              <div className="flex-1 h-px bg-stone-100" />
            </div>
            <QuotationGenerator recipe={recipe} />
          </div>

          <div className="flex gap-3 pt-2">
            <button
              onClick={() => onEdit(recipe)}
              className="flex-1 border border-stone-200 text-stone-600 text-xs tracking-widest uppercase font-medium py-2.5 hover:border-gold hover:text-gold transition-all"
            >
              Editar
            </button>
            <button
              onClick={onClose}
              className="flex-1 bg-gold text-white text-xs tracking-widest uppercase font-medium py-2.5 hover:bg-gold-light transition-all"
            >
              Cerrar
            </button>
          </div>
        </div>
      </div>

      {/* Modal guardar receta escalada */}
      {showSaveModal && (
        <div className="fixed inset-0 z-[60] bg-stone-800 bg-opacity-60 flex items-center justify-center p-4">
          <div className="bg-white border border-stone-200 w-full max-w-sm p-6 space-y-4 shadow-xl">
            <div className="flex items-center justify-between">
              <h3 className="font-display text-xl text-stone-800">Guardar receta escalada</h3>
              <button
                onClick={() => setShowSaveModal(false)}
                className="text-stone-400 hover:text-stone-800 text-xl"
              >
                ×
              </button>
            </div>

            <div className="bg-vanilla-100 px-4 py-3 text-xs font-light text-stone-500 space-y-0.5">
              <p>Base: {recipe.yield} {recipe.yield_unit}</p>
              <p className="text-gold font-medium">
                Nueva: {targetYield} {recipe.yield_unit} (×{factor.toFixed(2)})
              </p>
            </div>

            <div>
              <label className="block text-xs tracking-widest uppercase text-stone-400 mb-1.5 font-light">
                Nombre de la nueva receta
              </label>
              <input
                type="text"
                value={saveName}
                onChange={(e) => setSaveName(e.target.value)}
                className="w-full bg-vanilla-100 border border-stone-200 text-stone-800 px-3 py-2 text-sm font-light focus:outline-none focus:border-gold transition-all"
                autoFocus
              />
            </div>

            {saveError && (
              <p className="text-terracota-500 text-xs tracking-wide">{saveError}</p>
            )}

            {saveSuccess ? (
              <p className="text-center text-sm font-light text-gold py-2">
                Receta guardada correctamente
              </p>
            ) : (
              <button
                onClick={handleSaveScaled}
                disabled={saving}
                className="w-full bg-gold text-white text-xs tracking-widest uppercase font-medium py-3 hover:bg-gold-light transition-all disabled:opacity-50"
              >
                {saving ? "Guardando..." : "Guardar"}
              </button>
            )}
          </div>
        </div>
      )}
    </>
  );
}
