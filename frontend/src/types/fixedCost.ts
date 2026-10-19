export interface IFixedCost {
  id: string;
  user_id: string;
  name: string;
  /** The amount in force in the month that was asked for. */
  monthly_amount: number;
  /** Whether the cost counts in the month that was asked for. */
  is_active: boolean;
  /** First day (YYYY-MM-DD) of the version in force. */
  valid_from: string;
  /** First day (YYYY-MM-DD) of the month the cost ends in, if it does. */
  ended_from: string | null;
  created_at: string;
  updated_at: string;
}

export interface FixedCostFormData {
  name: string;
  monthly_amount: number;
  is_active: boolean;
  /** YYYY-MM; the change applies from this month on (default: the current one). */
  effective_from?: string;
}

export interface FixedCostCreateData {
  name: string;
  monthly_amount: number;
  /** YYYY-MM; the cost starts in this month (default: the current one). */
  valid_from?: string;
}
