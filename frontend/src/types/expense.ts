export interface IExpense {
  id: string;
  user_id: string;
  description: string;
  amount: number;
  category: "ingredientes" | "servicios" | "otros";
  expense_date: string;
  created_at: string;
  updated_at: string;
}

export interface ExpenseFormData {
  description: string;
  amount: string;
  category: "ingredientes" | "servicios" | "otros";
  expense_date: string;
}
