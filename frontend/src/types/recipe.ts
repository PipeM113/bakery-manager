export interface IRecipeIngredient {
  id: string;
  recipe_id: string;
  ingredient_id: string;
  name: string;
  quantity: number;
  unit: string;
}

export interface IRecipe {
  id: string;
  user_id: string;
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
  ingredients: IRecipeIngredient[];
  created_at: string;
  updated_at: string;
}

export interface IIngredientOption {
  id: string;
  name: string;
  default_unit: string;
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
