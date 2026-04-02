import type { IRecipe } from "../types/recipe";

interface Props {
  recipes: IRecipe[];
  onView: (r: IRecipe) => void;
  onEdit: (r: IRecipe) => void;
  onDelete: (id: string) => void;
}

export default function RecipeList({ recipes, onView, onEdit, onDelete }: Props) {
  if (recipes.length === 0) {
    return (
      <p className="text-center text-stone-400 text-sm font-light py-16">
        No hay recetas aún
      </p>
    );
  }

  return (
    <div className="space-y-2">
      {recipes.map((r) => (
        <div
          key={r.id}
          className="bg-white border border-stone-200 hover:border-gold transition-all duration-200"
        >
          <div className="flex items-center justify-between p-4">
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <p className="text-stone-800 text-sm font-light truncate">{r.name}</p>
                <span
                  className={`text-xs px-2 py-0.5 border shrink-0 ${
                    r.is_base
                      ? "text-gold border-gold border-opacity-40"
                      : "text-stone-400 border-stone-200"
                  }`}
                >
                  {r.is_base ? "Base" : "Versión"}
                </span>
              </div>
              <p className="text-stone-400 text-xs mt-0.5 font-light">
                {r.yield} {r.yield_unit} · {r.ingredients?.length || 0} ingredientes
              </p>
            </div>
            <div className="flex items-center gap-3 ml-4">
              <button
                onClick={() => onView(r)}
                className="text-xs text-stone-400 hover:text-gold tracking-wider transition-colors"
              >
                Ver
              </button>
              <button
                onClick={() => onEdit(r)}
                className="text-xs text-stone-400 hover:text-gold tracking-wider transition-colors"
              >
                Editar
              </button>
              <button
                onClick={() => onDelete(r.id)}
                className="text-xs text-stone-400 hover:text-terracota-500 tracking-wider transition-colors"
              >
                Eliminar
              </button>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
