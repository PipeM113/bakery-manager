import api from "../services/api";

export interface QuotationRequest {
  recipe_id:         string;
  client_name:       string;
  margin_pct:        number;
  indirect_cost_pct: number;
  labor_cost_pct:    number;
}

export interface IQuotation {
  id:          string;
  user_id:     string;
  recipe_id:   string;
  recipe_name: string;
  client_name: string;
  margin_pct:  number;
  final_price: number;
  status:      "pending" | "confirmed" | "cancelled";
  created_at:  string;
  updated_at:  string;
}

export const quotationService = {
  async generatePDF(req: QuotationRequest): Promise<Blob> {
    const response = await api.post("/quotations/generate", req, {
      responseType: "blob",
    });
    return response.data as Blob;
  },

  async getAll(status?: string): Promise<IQuotation[]> {
    const params = status ? { status } : {};
    const { data } = await api.get<IQuotation[]>("/quotations", { params });
    return data;
  },

  async confirm(id: string): Promise<void> {
    await api.put(`/quotations/${id}/confirm`);
  },

  async cancel(id: string): Promise<void> {
    await api.put(`/quotations/${id}/cancel`);
  },
};
