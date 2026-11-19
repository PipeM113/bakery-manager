export type AdjustmentReason = "merma" | "correccion" | "otro" | "compra";

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
  /** Only a purchase: where it was bought and the package paid for (size in the unit of the ingredient). */
  supplier?: string;
  package_size?: number;
  package_price?: number;
  /** Who registered it; empty if that user was deleted. */
  created_by: string | null;
  created_at: string;
}

/** Quantity (merma, compra), counted_quantity (correccion) or quantity_delta (otro), according to the reason. */
export interface StockAdjustmentForm {
  reason: AdjustmentReason;
  quantity?: number;
  counted_quantity?: number;
  quantity_delta?: number;
  unit?: string;
  note?: string;
  adjusted_on?: string;
  /** Purchase only: the package size and its price come together; the supplier is free text. */
  supplier?: string;
  package_size?: number;
  package_unit?: string;
  package_price?: number;
}
