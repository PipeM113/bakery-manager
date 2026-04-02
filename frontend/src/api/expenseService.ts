import api from "../services/api";
import type { IExpense } from "../types/expense";

export const expenseService = {
  getByMonth: async (month: number, year: number): Promise<IExpense[]> => {
    const { data } = await api.get<IExpense[]>("/expenses", {
      params: { month, year },
    });
    return data;
  },

  create: async (form: {
    description: string;
    amount: number;
    category: string;
    expense_date: string;
  }): Promise<IExpense> => {
    const { data } = await api.post<IExpense>("/expenses", form);
    return data;
  },

  update: async (
    id: string,
    form: { description: string; amount: number; category: string; expense_date: string }
  ): Promise<IExpense> => {
    const { data } = await api.put<IExpense>(`/expenses/${id}`, form);
    return data;
  },

  delete: async (id: string): Promise<void> => {
    await api.delete(`/expenses/${id}`);
  },
};
