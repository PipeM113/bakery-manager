import { groupRecipes, type TypedRecipe } from "../utils/recipes";

/** The <option>s of a recipe selector: grouped by type when some recipe has one, flat otherwise. */
export default function RecipeOptions({ recipes }: { recipes: TypedRecipe[] }) {
  const { hasTypes, groups } = groupRecipes(recipes);
  if (!hasTypes) {
    return <>{recipes.map((r) => <option key={r.id} value={r.id}>{r.name}</option>)}</>;
  }
  return (
    <>
      {groups.map((g) => (
        <optgroup key={g.id ?? "none"} label={g.name}>
          {g.recipes.map((r) => <option key={r.id} value={r.id}>{r.name}</option>)}
        </optgroup>
      ))}
    </>
  );
}
