import type { IStockShortfall, SoldAs } from "./sale";

export interface IOrderLine {
  sale_id: string;
  recipe_id: string;
  recipe_name: string;
  quantity_sold: number;
  sold_as: SoldAs;
  /** The normal price of one unit (a recipe or a portion). */
  unit_price: number;
  /** What this line really brought in: its share of the order total. */
  line_total: number;
}

export interface IOrder {
  id: string;
  customer_name: string;
  notes: string;
  order_date: string;
  total_price: number;
  lines: IOrderLine[];
  /** Empty when the stock was enough. */
  stock_shortfalls: IStockShortfall[];
  created_at: string;
}

export interface OrderLineForm {
  recipe_id: string;
  quantity: number;
  unit_price: number;
  sold_as: SoldAs;
}

export interface OrderFormData {
  customer_name?: string;
  notes?: string;
  order_date?: string;
  /** The order's own total; when it is missing the total is the sum of the lines. */
  total_price?: number;
  lines: OrderLineForm[];
}
