import api from "../services/api";
import type { IIngredient, IngredientFormData } from "../types/ingredient";

export interface ImportRowError {
  row: number;
  name: string;
  reason: string;
}

export interface ImportResult {
  imported: number;
  skipped: number;
  errors: ImportRowError[];
}

export const ingredientService = {
  getAll: async (): Promise<IIngredient[]> => {
    const { data } = await api.get<IIngredient[]>("/ingredients");
    return data;
  },

  create: async (form: IngredientFormData): Promise<IIngredient> => {
    const pkgSize = Number(form.package_size) || 1;
    const pkgPrice = Number(form.package_price) || 0;
    const { data } = await api.post<IIngredient>("/ingredients", {
      ...form,
      package_size: pkgSize,
      package_price: pkgPrice,
      price_per_unit: pkgPrice / pkgSize,
      stock_quantity: Number(form.stock_quantity) || 0,
      alert_threshold: Number(form.alert_threshold) || 0,
    });
    return data;
  },

  update: async (id: string, form: IngredientFormData): Promise<IIngredient> => {
    const pkgSize = Number(form.package_size) || 1;
    const pkgPrice = Number(form.package_price) || 0;
    const { data } = await api.put<IIngredient>(`/ingredients/${id}`, {
      ...form,
      package_size: pkgSize,
      package_price: pkgPrice,
      price_per_unit: pkgPrice / pkgSize,
      stock_quantity: Number(form.stock_quantity) || 0,
      alert_threshold: Number(form.alert_threshold) || 0,
    });
    return data;
  },

  delete: async (id: string): Promise<void> => {
    await api.delete(`/ingredients/${id}`);
  },

  exportExcel: async (): Promise<Blob> => {
    const { data } = await api.get("/ingredients/export", { responseType: "blob" });
    return data;
  },

  import: async (file: File): Promise<ImportResult> => {
    const formData = new FormData();
    formData.append("file", file);
    const { data } = await api.post<ImportResult>("/ingredients/import", formData);
    return data;
  },
};
