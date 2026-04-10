import api from "../services/api";
import type { IRecipe, RecipeFormData } from "../types/recipe";

export const recipeService = {
  getAll: async (): Promise<IRecipe[]> => {
    const { data } = await api.get<IRecipe[]>("/recipes");
    return data;
  },

  getById: async (id: string): Promise<IRecipe> => {
    const { data } = await api.get<IRecipe>(`/recipes/${id}`);
    return data;
  },

  create: async (form: RecipeFormData): Promise<IRecipe> => {
    const { data } = await api.post<IRecipe>("/recipes", {
      ...form,
      yield: Number(form.yield),
    });
    return data;
  },

  update: async (id: string, form: RecipeFormData): Promise<IRecipe> => {
    const { data } = await api.put<IRecipe>(`/recipes/${id}`, {
      ...form,
      yield: Number(form.yield),
    });
    return data;
  },

  delete: async (id: string): Promise<void> => {
    await api.delete(`/recipes/${id}`);
  },

  createVersion: async (parentId: string, versionName: string, form: RecipeFormData): Promise<IRecipe> => {
    const { data } = await api.post<IRecipe>("/recipes", {
      ...form,
      name: versionName,
      yield: Number(form.yield),
      parent_id: parentId,
    });
    return data;
  },

  saveScaled: async (id: string, scaleFactor: number, newName: string): Promise<IRecipe> => {
    const { data } = await api.post<IRecipe>(`/recipes/${id}/save-scaled`, {
      scale_factor: scaleFactor,
      new_name: newName,
    });
    return data;
  },

  uploadPhoto: async (id: string, file: File): Promise<{ photo_url: string }> => {
    const form = new FormData();
    form.append("photo", file);
    // No fijar Content-Type: axios lo setea automáticamente con el boundary correcto al detectar FormData
    const { data } = await api.post<{ photo_url: string }>(`/recipes/${id}/photo`, form);
    return data;
  },
};
