export type IngredientKind = "ingredient" | "supply";

export interface IIngredient {
  id: string;
  kind: IngredientKind;
  name: string;
  default_unit: string;
  package_size: number;
  package_price: number;
  price_per_unit: number;
  stock_quantity: number;
  alert_threshold: number;
  brand: string;
  supplier: string;
  /** Weight of 1 litre / of 1 unit in gr; 0 when it is not known. */
  grams_per_liter: number;
  grams_per_unit: number;
  created_at: string;
  updated_at: string;
}

export type IngredientFormData = {
  kind: IngredientKind;
  name: string;
  default_unit: string;
  package_size: number | "";
  package_price: number | "";
  stock_quantity: number | "";
  alert_threshold: number | "";
  brand: string;
  supplier: string;
  grams_per_liter: number | "";
  grams_per_unit: number | "";
};

export const emptyIngredientForm: IngredientFormData = {
  kind: "ingredient",
  name: "",
  default_unit: "gr",
  package_size: "",
  package_price: "",
  stock_quantity: "",
  alert_threshold: "",
  brand: "",
  supplier: "",
  grams_per_liter: "",
  grams_per_unit: "",
};
