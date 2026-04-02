export interface IFixedCost {
  id: string;
  user_id: string;
  name: string;
  monthly_amount: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface FixedCostFormData {
  name: string;
  monthly_amount: number;
  is_active: boolean;
}
