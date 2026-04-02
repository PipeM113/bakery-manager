import api from "../services/api";
import type { ISale, SaleFormData } from "../types/sale";

export const saleService = {
  getAll: async (from?: string, to?: string): Promise<ISale[]> => {
    const params: Record<string, string> = {};
    if (from) params.from = from;
    if (to) params.to = to;
    const { data } = await api.get<ISale[]>("/sales", { params });
    return data;
  },

  create: async (form: SaleFormData): Promise<ISale> => {
    const { data } = await api.post<ISale>("/sales", form);
    return data;
  },

  delete: async (id: string): Promise<void> => {
    await api.delete(`/sales/${id}`);
  },
};
