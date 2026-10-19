import api from "../services/api";
import { dateInSantiago } from "../utils/businessDate";

export type SoldAs = "recipe" | "portion";

export interface IQuotationLine {
  id: string;
  position: number;
  recipe_id: string | null;
  /** The name the recipe had when it was quoted. */
  recipe_name: string;
  sold_as: SoldAs;
  quantity: number;
  unit_price: number;
  line_total: number;
  margin_pct: number | null;
  indirect_cost_pct: number | null;
  labor_cost_pct: number | null;
  extra_charge: number;
}

export interface IQuotation {
  id: string;
  /** Who registered it; empty if that user was deleted. */
  created_by: string | null;
  client_name: string;
  delivery_cost: number;
  /** The lines plus the delivery, in whole pesos. */
  total: number;
  status: "pending" | "confirmed" | "cancelled";
  lines: IQuotationLine[];
  created_at: string;
  updated_at: string;
}

/** What is sent for each line; what is left out comes from the recipe. */
export interface QuotationLineRequest {
  recipe_id: string;
  quantity: number;
  sold_as?: SoldAs;
  margin_pct?: number;
  indirect_cost_pct?: number;
  labor_cost_pct?: number;
  extra_charge?: number;
}

export interface QuotationRequest {
  client_name: string;
  delivery_cost: number;
  lines: QuotationLineRequest[];
}

const fold = (s: string) =>
  s.replace(/[áÁ]/g, "a").replace(/[éÉ]/g, "e").replace(/[íÍ]/g, "i").replace(/[óÓ]/g, "o")
    .replace(/[úÚüÜ]/g, "u").replace(/[ñÑ]/g, "n");

/** The name of the file, as the backend names it: cotizacion_<client>_<date in Santiago>.pdf */
export const quotationFileName = (q: Pick<IQuotation, "client_name" | "created_at">): string => {
  const client = fold(q.client_name).replace(/[^A-Za-z0-9]+/g, "_").replace(/^_+|_+$/g, "") || "cliente";
  return `cotizacion_${client}_${dateInSantiago(q.created_at)}.pdf`;
};

/** The message of a failed PDF request (the body is a blob because the answer was asked as one). */
const pdfError = async (err: unknown): Promise<Error> => {
  const data = (err as { response?: { data?: unknown } })?.response?.data;
  if (data instanceof Blob) {
    try {
      const msg = (JSON.parse(await data.text()) as { error?: string }).error;
      if (msg) return new Error(msg);
    } catch { /* not JSON */ }
  }
  return new Error("No se pudo generar el PDF");
};

export const quotationService = {
  /** Saves the quotation (every line priced and checked) and answers it; the PDF is a separate step. */
  async create(req: QuotationRequest): Promise<IQuotation> {
    const { data } = await api.post("/quotations", req);
    return data as IQuotation;
  },

  async getById(id: string): Promise<IQuotation> {
    const { data } = await api.get<IQuotation>(`/quotations/${id}`);
    return data;
  },

  async getAll(status?: string): Promise<IQuotation[]> {
    const params = status ? { status } : {};
    const { data } = await api.get<IQuotation[]>("/quotations", { params });
    return data;
  },

  /** Makes the PDF of a saved quotation and hands it to the browser as a download. */
  async downloadPDF(quotation: Pick<IQuotation, "id" | "client_name" | "created_at">): Promise<void> {
    const id = quotation.id;
    let blob: Blob;
    try {
      const response = await api.get(`/quotations/${id}/pdf`, { responseType: "blob" });
      blob = response.data as Blob;
    } catch (err) {
      throw await pdfError(err);
    }
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = quotationFileName(quotation);
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
  },

  async confirm(id: string): Promise<void> {
    await api.put(`/quotations/${id}/confirm`);
  },

  async cancel(id: string): Promise<void> {
    await api.put(`/quotations/${id}/cancel`);
  },
};
