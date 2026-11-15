import api from "../services/api";
import type { IRecipeType } from "../types/recipe";

export const recipeCategoryService = {
  getAll: async (): Promise<IRecipeType[]> => {
    const { data } = await api.get<IRecipeType[]>("/recipe-categories");
    return data;
  },

  create: async (name: string): Promise<IRecipeType> => {
    const { data } = await api.post<IRecipeType>("/recipe-categories", { name });
    return data;
  },

  rename: async (id: string, name: string): Promise<IRecipeType> => {
    const { data } = await api.put<IRecipeType>(`/recipe-categories/${id}`, { name });
    return data;
  },

  delete: async (id: string): Promise<void> => {
    await api.delete(`/recipe-categories/${id}`);
  },
};
