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
};
