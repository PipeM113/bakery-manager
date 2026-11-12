import { useEffect, useState } from "react";
import { recipeService } from "../api/recipeService";
import { ingredientService } from "../api/ingredientService";
import type { IRecipe, IRecipeType, IIngredientOption, RecipeFormData } from "../types/recipe";
import RecipeList from "../components/RecipeList";
import RecipeForm from "../components/RecipeForm";
import RecipeDetail from "../components/RecipeDetail";
import RecipeCategoriesModal from "../components/RecipeCategoriesModal";
import { recipeCategoryService } from "../api/recipeCategoryService";
import { groupRecipes, recipeMatches } from "../utils/recipes";
import { sortKey } from "../utils/ingredients";
import { apiErrorMessage } from "../utils/apiError";

export default function RecipesPage() {
  const [recipes, setRecipes]         = useState<IRecipe[]>([]);
  const [ingredients, setIngredients] = useState<IIngredientOption[]>([]);
  const [loading, setLoading]         = useState(true);
  const [showForm, setShowForm]       = useState(false);
  const [editing, setEditing]         = useState<IRecipe | null>(null);
  const [detail, setDetail]           = useState<IRecipe | null>(null);
  const [deleteError, setDeleteError] = useState("");
  const [types, setTypes]             = useState<IRecipeType[]>([]);
  const [showTypes, setShowTypes]     = useState(false);
  const [search, setSearch]           = useState("");
  const [activeType, setActiveType]   = useState<string | null>(null);

  const load = () => {
    Promise.all([
      recipeService.getAll(),
      ingredientService.getAll(),
      // The types are optional: without them the screen still works.
      recipeCategoryService.getAll().catch(() => [] as IRecipeType[]),
    ])
      .then(([recs, ings, kinds]) => {
        setRecipes(recs);
        // A filter whose chip is gone (its type was deleted, or only one type is left) is forgotten, so it cannot come back by itself.
        const inUseNow = groupRecipes(recs).groups.filter((g) => g.id !== null);
        setActiveType((current) => (inUseNow.length >= 2 && inUseNow.some((g) => g.id === current) ? current : null));
        setIngredients(ings);
        setTypes([...kinds].sort((a, b) => sortKey(a.name).localeCompare(sortKey(b.name))));
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

  const handleCreateType = async (name: string): Promise<IRecipeType> => {
    const created = await recipeCategoryService.create(name);
    setTypes((all) => [...all, created].sort((a, b) => sortKey(a.name).localeCompare(sortKey(b.name))));
    return created;
  };

  const handleSaveAs = async (data: RecipeFormData, versionName: string) => {
    await recipeService.createVersion(editing!.id, versionName, data);
    setShowForm(false);
    load();
  };

  const handleDelete = async (id: string) => {
    if (!confirm("¿Eliminar esta receta?")) return;
    setDeleteError("");
    try {
      await recipeService.delete(id);
      setDeleteError("");
      load();
    } catch (e) {
      // 409: it has sales or quotations; the message says how many.
      setDeleteError(apiErrorMessage(e, "No se pudo eliminar"));
    }
  };

  // The chips come from the types in use; the list shows what the search and the chip leave.
  const inUse = groupRecipes(recipes);
  const typesInUse = inUse.groups.filter((g) => g.id !== null);
  const chipsShown = typesInUse.length >= 2;
  const chip = chipsShown && typesInUse.some((g) => g.id === activeType) ? activeType : null;
  const shown = groupRecipes(recipes.filter((r) => recipeMatches(r, search) && (chip === null || r.category_id === chip)));

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <span className="text-oro text-sm tracking-widest uppercase animate-pulse">Cargando...</span>
      </div>
    );
  }

  return (
    <div className="px-6 py-10 max-w-2xl mx-auto md:max-w-none md:px-10">
      <div className="flex items-end justify-between mb-8">
        <div>
          <div className="flex items-center gap-3 mb-2">
            <div className="h-px w-8 bg-oro/40" />
            <span className="text-oro text-xs tracking-[0.3em] uppercase font-light">Recetario</span>
          </div>
          <h1 className="font-display text-4xl text-negro">Recetas</h1>
        </div>
        <div className="flex gap-2">
          <button
            onClick={() => setShowTypes(true)}
            className="border border-linea text-negro/70 text-xs tracking-widest uppercase font-medium px-4 py-3 hover:border-oro hover:text-oro transition-all duration-200"
          >
            Tipos
          </button>
          <button
            onClick={openCreate}
            className="bg-negro text-crema text-xs tracking-widest uppercase font-medium px-5 py-3 hover:bg-black transition-all duration-200"
          >
            + Nueva
          </button>
        </div>
      </div>

      <input
        type="search"
        data-testid="recipe-search"
        aria-label="Buscar receta"
        placeholder="Buscar por nombre o tipo..."
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        className="w-full bg-white border border-linea text-negro text-sm px-4 py-3 mb-4 font-light tracking-wide
                   placeholder-negro/70 focus:outline-none focus:border-oro transition-all"
      />

      {chipsShown && (
        <div data-testid="type-chips" className="flex flex-wrap gap-2 mb-6">
          {[{ id: null as string | null, name: "Todos", count: recipes.length }, ...inUse.groups.filter((g) => g.id !== null).map((g) => ({ id: g.id, name: g.name, count: g.recipes.length }))].map((c) => (
            <button
              key={c.id ?? "all"}
              onClick={() => setActiveType(c.id)}
              aria-pressed={activeType === c.id}
              className={`text-xs tracking-wide font-medium px-4 py-2 border transition-all duration-200
                ${activeType === c.id ? "bg-negro text-crema border-oro" : "border-linea text-negro/70 hover:border-oro hover:text-oro"}`}
            >
              {c.name} <span className="opacity-70">{c.count}</span>
            </button>
          ))}
        </div>
      )}

      {deleteError && (
        <div role="alert" className="mb-4 bg-rosa text-negro px-4 py-3 text-sm font-light flex items-start justify-between gap-3">
          <span>{deleteError}</span>
          <button onClick={() => setDeleteError("")} aria-label="Cerrar aviso" className="text-negro/70 hover:text-negro shrink-0">×</button>
        </div>
      )}

      <RecipeList
        groups={shown.groups}
        hasTypes={shown.hasTypes}
        emptyText={recipes.length === 0 ? "No hay recetas aún" : "No se encontró ninguna receta con esa búsqueda"}
        onView={setDetail}
        onEdit={openEdit}
        onDelete={handleDelete}
      />

      {showTypes && (
        <RecipeCategoriesModal
          onClose={() => setShowTypes(false)}
          onChanged={load}
        />
      )}

      {showForm && (
        <RecipeForm
          editing={editing}
          ingredients={ingredients}
          types={types}
          onCreateType={handleCreateType}
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
