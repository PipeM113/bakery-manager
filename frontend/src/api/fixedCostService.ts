import api from "../services/api";
import type { IFixedCost, FixedCostFormData } from "../types/fixedCost";

export const fixedCostService = {
  getAll: async (): Promise<IFixedCost[]> => {
    const { data } = await api.get<IFixedCost[]>("/fixed-costs");
    return data;
  },

  create: async (form: Pick<FixedCostFormData, "name" | "monthly_amount">): Promise<IFixedCost> => {
    const { data } = await api.post<IFixedCost>("/fixed-costs", form);
    return data;
  },

  update: async (id: string, form: FixedCostFormData): Promise<IFixedCost> => {
    const { data } = await api.put<IFixedCost>(`/fixed-costs/${id}`, form);
    return data;
  },

  delete: async (id: string): Promise<void> => {
    await api.delete(`/fixed-costs/${id}`);
  },
};
