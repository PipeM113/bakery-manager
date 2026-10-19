import api from "../services/api";
import type { IExpenseCategory } from "../types/expense";

export const expenseCategoryService = {
  list: async (): Promise<IExpenseCategory[]> => {
    const { data } = await api.get<IExpenseCategory[]>("/expense-categories");
    return data;
  },

  create: async (name: string): Promise<IExpenseCategory> => {
    const { data } = await api.post<IExpenseCategory>("/expense-categories", { name });
    return data;
  },

  rename: async (id: string, name: string): Promise<IExpenseCategory> => {
    const { data } = await api.put<IExpenseCategory>(`/expense-categories/${id}`, { name });
    return data;
  },

  remove: async (id: string): Promise<void> => {
    await api.delete(`/expense-categories/${id}`);
  },
};
