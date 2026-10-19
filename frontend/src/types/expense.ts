export interface IExpense {
  id: string;
  user_id: string;
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
