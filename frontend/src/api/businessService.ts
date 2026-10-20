import api from "../services/api";

/** What the quotation PDF prints and can change. The RUT and the account number are not here (Vercel); the name and the Instagram of the brand are fixed in the backend. */
export interface IBusinessProfile {
  phone: string;
  bank: string;
  account_type: string;
  holder: string;
  email: string;
  /** What keeps the PDF from being made: empty fields (their names) and unset BIZ_* variables. */
  missing: string[];
}

export type BusinessForm = Omit<IBusinessProfile, "missing">;

export const ACCOUNT_TYPES = ["Cuenta Corriente", "Cuenta Vista", "Cuenta RUT", "Cuenta de Ahorro"];

export const businessService = {
  async get(): Promise<IBusinessProfile> {
    const { data } = await api.get<IBusinessProfile>("/business");
    return data;
  },

  async save(form: BusinessForm): Promise<IBusinessProfile> {
    const { data } = await api.put<IBusinessProfile>("/business", form);
    return data;
  },
};
