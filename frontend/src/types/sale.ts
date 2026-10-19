export interface ISaleIngredient {
  id: string;
  sale_id: string;
  ingredient_id: string;
  name: string;
  quantity_used: number;
  unit: string;
  price_at_time: number;
}

export type SoldAs = "recipe" | "portion";

/** What a sale needed and the stock did not have. */
export interface IStockShortfall {
  ingredient: string;
  missing: number;
  unit: string;
}

export interface ISale {
  id: string;
  user_id: string;
  recipe_id: string;
  recipe_name: string;
  quantity_sold: number;
  unit_price: number;
  total_price: number;
  sold_as: SoldAs;
  /** The order this sale is a line of, if any. */
  order_id: string | null;
  sale_date: string;
  notes: string;
  ingredients: ISaleIngredient[];
  /** Empty when the stock was enough. */
  stock_shortfalls: IStockShortfall[];
  created_at: string;
}

export interface SaleFormData {
  recipe_id: string;
  quantity_sold: number;
  unit_price: number;
  sold_as: SoldAs;
  sale_date: string;
  notes: string;
}
