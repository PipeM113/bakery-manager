export type AdjustmentReason = "merma" | "correccion" | "otro";

export interface IStockAdjustment {
  id: string;
  ingredient_id: string;
  ingredient_name: string;
  reason: AdjustmentReason;
  /** Signed, in the unit of the ingredient. */
  quantity_delta: number;
  unit: string;
  stock_before: number;
  stock_after: number;
  /** Whole pesos; only a merma has one. */
  loss_amount: number;
  note: string;
  adjusted_on: string;
  /** Who registered it; empty if that user was deleted. */
  created_by: string | null;
  created_at: string;
}

/** Quantity (merma), counted_quantity (correccion) or quantity_delta (otro), according to the reason. */
export interface StockAdjustmentForm {
  reason: AdjustmentReason;
  quantity?: number;
  counted_quantity?: number;
  quantity_delta?: number;
  unit?: string;
  note?: string;
  adjusted_on?: string;
}
