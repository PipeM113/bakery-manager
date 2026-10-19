import api from "../services/api";
import type { IFixedCost, FixedCostFormData, FixedCostCreateData } from "../types/fixedCost";

export const fixedCostService = {
  /** The costs of a month (YYYY-MM): each with the amount in force then. */
  getAll: async (month?: string): Promise<IFixedCost[]> => {
    const { data } = await api.get<IFixedCost[]>("/fixed-costs", { params: month ? { month } : undefined });
    return data;
  },

  /** `valid_from` (YYYY-MM) is the first month the cost counts in. */
  create: async (form: FixedCostCreateData): Promise<IFixedCost> => {
    const { data } = await api.post<IFixedCost>("/fixed-costs", form);
    return data;
  },

  /** `effective_from` (YYYY-MM) is the month the new amount, or the end, applies from. */
  update: async (id: string, form: FixedCostFormData): Promise<IFixedCost> => {
    const { data } = await api.put<IFixedCost>(`/fixed-costs/${id}`, form);
    return data;
  },

  /** Ends the cost from this month; the months before keep it. */
  delete: async (id: string): Promise<void> => {
    await api.delete(`/fixed-costs/${id}`);
  },
};
