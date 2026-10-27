export interface IRecipeIngredient {
  id: string;
  recipe_id: string;
  ingredient_id: string;
  name: string;
  kind: "ingredient" | "supply";
  brand?: string;
  supplier?: string;
  quantity: number;
  unit: string;
}

export interface IRecipe {
  id: string;
  /** Who registered it; empty if that user was deleted. */
  created_by: string | null;
  parent_id: string | null;
  name: string;
  description: string;
  yield: number;
  yield_unit: string;
  photo_url: string | null;
  is_base: boolean;
  scale_factor: number;
  indirect_cost_pct: number;
  labor_cost_pct: number;
  margin_pct: number;
  extra_charge: number;
  delivery_cost: number;
  ingredients: IRecipeIngredient[];
  created_at: string;
  updated_at: string;
}

export interface IIngredientOption {
  id: string;
  kind?: "ingredient" | "supply";
  name: string;
  default_unit: string;
  brand?: string;
  supplier?: string;
  grams_per_liter?: number;
  grams_per_unit?: number;
}

export type RecipeFormData = {
  name: string;
  description: string;
  yield: number | "";
  yield_unit: string;
  ingredients: Array<{ ingredient_id: string; quantity: number; unit: string }>;
};

export const emptyRecipeForm: RecipeFormData = {
  name: "",
  description: "",
  yield: "",
  yield_unit: "porciones",
  ingredients: [],
};
