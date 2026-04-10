import { useEffect, useState } from "react";
import { recipeService } from "../api/recipeService";
import { ingredientService } from "../api/ingredientService";
import type { IRecipe, IIngredientOption, RecipeFormData } from "../types/recipe";
import RecipeList from "../components/RecipeList";
import RecipeForm from "../components/RecipeForm";
import RecipeDetail from "../components/RecipeDetail";

export default function RecipesPage() {
  const [recipes, setRecipes]         = useState<IRecipe[]>([]);
  const [ingredients, setIngredients] = useState<IIngredientOption[]>([]);
  const [loading, setLoading]         = useState(true);
  const [showForm, setShowForm]       = useState(false);
  const [editing, setEditing]         = useState<IRecipe | null>(null);
  const [detail, setDetail]           = useState<IRecipe | null>(null);

  const load = () => {
    Promise.all([recipeService.getAll(), ingredientService.getAll()])
      .then(([recs, ings]) => {
        setRecipes(recs);
        setIngredients(ings);
      })
      .finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, []);

  const openCreate = () => {
    setEditing(null);
    setShowForm(true);
  };

  const openEdit = (r: IRecipe) => {
    setDetail(null);
    setEditing(r);
    setShowForm(true);
  };

  const handleSave = async (data: RecipeFormData) => {
    if (editing) {
      await recipeService.update(editing.id, data);
    } else {
      await recipeService.create(data);
    }
    setShowForm(false);
    load();
  };

  const handleSaveAs = async (data: RecipeFormData, versionName: string) => {
    await recipeService.createVersion(editing!.id, versionName, data);
    setShowForm(false);
    load();
  };

  const handleDelete = async (id: string) => {
    if (!confirm("¿Eliminar esta receta?")) return;
    await recipeService.delete(id);
    load();
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <span className="text-gold text-sm tracking-widest uppercase animate-pulse">Cargando...</span>
      </div>
    );
  }

  return (
    <div className="px-6 py-10 max-w-2xl mx-auto md:max-w-none md:px-10">
      <div className="flex items-end justify-between mb-8">
        <div>
          <div className="flex items-center gap-3 mb-2">
            <div className="h-px w-8 bg-gold opacity-40" />
            <span className="text-gold text-xs tracking-[0.3em] uppercase font-light">Recetario</span>
          </div>
          <h1 className="font-display text-4xl text-stone-800">Recetas</h1>
        </div>
        <button
          onClick={openCreate}
          className="bg-gold text-white text-xs tracking-widest uppercase font-medium px-5 py-3 hover:bg-gold-light transition-all duration-200"
        >
          + Nueva
        </button>
      </div>

      <RecipeList
        recipes={recipes}
        onView={setDetail}
        onEdit={openEdit}
        onDelete={handleDelete}
      />

      {showForm && (
        <RecipeForm
          editing={editing}
          ingredients={ingredients}
          onSave={handleSave}
          onSaveAs={editing ? handleSaveAs : undefined}
          onClose={() => setShowForm(false)}
        />
      )}

      {detail && (
        <RecipeDetail
          recipe={detail}
          onClose={() => setDetail(null)}
          onEdit={openEdit}
          onSaved={() => { setDetail(null); load(); }}
        />
      )}
    </div>
  );
}
