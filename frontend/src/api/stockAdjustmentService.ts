import api from "../services/api";
import type { IStockAdjustment, StockAdjustmentForm } from "../types/stockAdjustment";

export const stockAdjustmentService = {
  /** Merma: `quantity`; correccion: `counted_quantity`; otro: `quantity_delta` (signed) with a note. */
  create: async (ingredientId: string, form: StockAdjustmentForm): Promise<IStockAdjustment> => {
    const { data } = await api.post<IStockAdjustment>(`/ingredients/${ingredientId}/stock-adjustments`, form);
    return data;
  },

  /** The whole history of one ingredient, newest first. */
  forIngredient: async (ingredientId: string): Promise<IStockAdjustment[]> => {
    const { data } = await api.get<IStockAdjustment[]>(`/ingredients/${ingredientId}/stock-adjustments`);
    return data;
  },

  /** What was adjusted in a month (YYYY-MM; the current one by default). */
  forMonth: async (month?: string): Promise<IStockAdjustment[]> => {
    const { data } = await api.get<IStockAdjustment[]>("/stock-adjustments", { params: month ? { month } : undefined });
    return data;
  },
};
