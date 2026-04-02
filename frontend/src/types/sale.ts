export interface ISaleIngredient {
  id: string;
  sale_id: string;
  ingredient_id: string;
  name: string;
  quantity_used: number;
  unit: string;
  price_at_time: number;
}

export interface ISale {
  id: string;
  user_id: string;
  recipe_id: string;
  recipe_name: string;
  quantity_sold: number;
  unit_price: number;
  total_price: number;
  sale_date: string;
  notes: string;
  ingredients: ISaleIngredient[];
  created_at: string;
}

export interface SaleFormData {
  recipe_id: string;
  quantity_sold: number;
  unit_price: number;
  sale_date: string;
  notes: string;
}
