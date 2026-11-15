import type { IRecipe } from "../types/recipe";
import type { RecipeGroup } from "../utils/recipes";
import { formatCLP } from "../utils/money";

interface Props {
  groups: RecipeGroup<IRecipe>[];
  /** True when some recipe has a type: each group gets a heading. */
  hasTypes: boolean;
  /** What to say when there is nothing to list. */
  emptyText: string;
  onView: (r: IRecipe) => void;
  onEdit: (r: IRecipe) => void;
  onDelete: (id: string) => void;
}

function Thumbnail({ recipe }: { recipe: IRecipe }) {
  const box = "w-14 h-14 shrink-0 border border-linea bg-lino overflow-hidden";
  if (!recipe.photo_url) {
    // No photo: a neutral frame with a line drawing of a cake (no text, so the name stays the first thing of the card).
    return (
      <div aria-hidden="true" className={`${box} flex items-center justify-center text-oro/60`}>
        <svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" strokeWidth="1.2">
          <path d="M4 20h16M5 20v-6h14v6M5 14c0-2 3-2 3-4m4 4c0-2 3-2 3-4M12 6V4" />
        </svg>
      </div>
    );
  }
  return <img src={recipe.photo_url} alt="" loading="lazy" className={`${box} object-cover`} />;
}

function RecipeRow({ r, onView, onEdit, onDelete }: { r: IRecipe } & Pick<Props, "onView" | "onEdit" | "onDelete">) {
  return (
    <div
      data-testid="recipe-card"
      onClick={() => onView(r)}
      className="bg-white border border-linea hover:border-oro transition-all duration-200 cursor-pointer"
    >
      <div className="flex gap-3 p-4">
        <Thumbnail recipe={r} />
        <div className="flex-1 min-w-0 md:flex md:items-center md:justify-between md:gap-4">
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <button onClick={(e) => { e.stopPropagation(); onView(r); }}
                className="text-negro text-sm text-left break-words md:truncate hover:text-oro transition-colors">
                {r.name}
              </button>
              <span
                className={`text-xs px-2 py-0.5 border shrink-0 ${
                  r.is_base ? "text-oro border-oro border-opacity-40" : "text-negro/70 border-linea"
                }`}
              >
                {r.is_base ? "Base" : "Versión"}
              </span>
            </div>
            <p className="text-negro/70 text-xs mt-0.5 font-light">
              {r.yield} {r.yield_unit} · {r.ingredients?.length || 0} ingredientes
            </p>
            {r.total_cost === null || r.suggested_price === null ? (
              <p className="text-negro/70 text-xs mt-1 font-light">Sin costo: falta un dato de algún ingrediente</p>
            ) : (
              <p className="text-negro text-xs mt-1 font-light">
                <span className="text-negro/70">Costo total</span> {formatCLP(r.total_cost)}
                <span className="text-negro/70"> · Precio sugerido</span> {formatCLP(r.suggested_price)}
              </p>
            )}
          </div>
          <div className="flex items-center gap-3 mt-2 md:mt-0 md:ml-4 shrink-0">
            <button onClick={(e) => { e.stopPropagation(); onEdit(r); }}
              className="text-xs text-negro/70 hover:text-oro tracking-wider transition-colors">
              Editar
            </button>
            <button onClick={(e) => { e.stopPropagation(); onDelete(r.id); }}
              className="text-xs text-negro/70 hover:text-peligro tracking-wider transition-colors">
              Eliminar
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

export default function RecipeList({ groups, hasTypes, emptyText, onView, onEdit, onDelete }: Props) {
  if (groups.every((g) => g.recipes.length === 0)) {
    return <p className="text-center text-negro/70 text-sm font-light py-16">{emptyText}</p>;
  }
  return (
    <div className="space-y-6">
      {groups.filter((g) => g.recipes.length > 0).map((g) => (
        <section key={g.id ?? "none"} className="space-y-2">
          {hasTypes && (
            <h2 data-testid="type-group" className="font-display text-xl text-negro flex items-baseline gap-2">
              {g.name}
              <span className="font-sans text-xs text-negro/70 font-light">{g.recipes.length}</span>
            </h2>
          )}
          {g.recipes.map((r) => <RecipeRow key={r.id} r={r} onView={onView} onEdit={onEdit} onDelete={onDelete} />)}
        </section>
      ))}
    </div>
  );
}
