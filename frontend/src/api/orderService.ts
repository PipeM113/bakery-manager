import api from "../services/api";
import type { IOrder, OrderFormData } from "../types/order";

export const orderService = {
  /** Registers an order: each line is a sale; `total_price` (optional) is shared between the lines. */
  create: async (form: OrderFormData): Promise<IOrder> => {
    const { data } = await api.post<IOrder>("/orders", form);
    return data;
  },

  /** The orders of a month (YYYY-MM; the current one by default), newest first. */
  getAll: async (month?: string): Promise<IOrder[]> => {
    const { data } = await api.get<IOrder[]>("/orders", { params: month ? { month } : undefined });
    return data;
  },

  /** Cancels the whole order and gives the stock back. */
  delete: async (id: string): Promise<void> => {
    await api.delete(`/orders/${id}`);
  },
};
