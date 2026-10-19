export interface IExpense {
  id: string;
  /** Who registered it; empty if that user was deleted. */
  created_by: string | null;
  description: string;
  amount: number;
  category_id: string;
  category: string;
  expense_date: string;
  created_at: string;
  updated_at: string;
}

export interface IExpenseCategory {
  id: string;
  name: string;
}

export interface ExpenseFormData {
  description: string;
  amount: string;
  category_id: string;
  expense_date: string;
}
