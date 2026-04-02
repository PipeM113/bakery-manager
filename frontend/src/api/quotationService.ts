import api from "../services/api";

export interface QuotationRequest {
  recipe_id:         string;
  margin_pct:        number;
  indirect_cost_pct: number;
  labor_cost_pct:    number;
}

export const quotationService = {
  async generatePDF(req: QuotationRequest): Promise<Blob> {
    const response = await api.post("/quotations/generate", req, {
      responseType: "blob",
    });
    return response.data as Blob;
  },
};
