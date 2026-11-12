import { sortKey } from "./ingredients.ts";

export interface TypedRecipe {
  id: string;
  name: string;
  category_id?: string | null;
  category?: string;
}

export interface RecipeGroup<T> {
  /** The id of the type, or null for the recipes without type (and for the single flat group). */
  id: string | null;
  name: string;
  recipes: T[];
}

/** The recipe is found by its name or by its type, without case or accents; an empty search finds everything. */
export const recipeMatches = (r: TypedRecipe, query: string): boolean => {
  const q = sortKey(query);
  if (!q) return true;
  return sortKey(r.name).includes(q) || sortKey(r.category ?? "").includes(q);
};

const byName = (a: TypedRecipe, b: TypedRecipe) => sortKey(a.name).localeCompare(sortKey(b.name)) || a.name.localeCompare(b.name);

export const NO_TYPE = "Sin tipo";

/**
 * Groups the recipes by type. With no type in use there is one flat group in alphabetical order (the list looks as
 * it did before there were types). Otherwise: a group per type in alphabetical order without accents, the recipes
 * without type last under "Sin tipo", and the recipes of each group in alphabetical order.
 */
export function groupRecipes<T extends TypedRecipe>(recipes: T[]): { hasTypes: boolean; groups: RecipeGroup<T>[] } {
  const typed = new Map<string, RecipeGroup<T>>();
  const untyped: T[] = [];
  for (const r of recipes) {
    if (!r.category_id) {
      untyped.push(r);
      continue;
    }
    const group = typed.get(r.category_id) ?? { id: r.category_id, name: r.category ?? "", recipes: [] };
    group.recipes.push(r);
    typed.set(r.category_id, group);
  }
  if (typed.size === 0) return { hasTypes: false, groups: [{ id: null, name: "", recipes: [...untyped].sort(byName) }] };
  const groups = [...typed.values()]
    .map((g) => ({ ...g, recipes: g.recipes.sort(byName) }))
    .sort((a, b) => sortKey(a.name).localeCompare(sortKey(b.name)) || a.name.localeCompare(b.name));
  if (untyped.length > 0) groups.push({ id: null, name: NO_TYPE, recipes: untyped.sort(byName) });
  return { hasTypes: true, groups };
}
