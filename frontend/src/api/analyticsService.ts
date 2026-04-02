import api from "../services/api";

export interface MonthlyMetrics {
  month: number;
  year: number;
  total_revenue: number;
  total_ingredients_cost: number;
  total_fixed_costs: number;
  total_operational_expenses: number;
  gross_profit: number;
  net_profit: number;
  profit_margin: number;
}

export interface RecipeMetrics {
  recipe_id: string;
  recipe_name: string;
  units_sold: number;
  total_revenue: number;
  cost_per_unit: number;
  total_cost: number;
  unit_profit: number;
  gross_margin: number;
}

export interface TrendPoint {
  date: string; // "YYYY-MM"
  revenue: number;
  net_profit: number;
}

export const analyticsService = {
  getMonthlyMetrics: async (month: number, year: number): Promise<MonthlyMetrics> => {
    const { data } = await api.get<MonthlyMetrics>("/analytics/monthly", {
      params: { month, year },
    });
    return data;
  },

  getRecipeMetrics: async (month: number, year: number): Promise<RecipeMetrics[]> => {
    const { data } = await api.get<RecipeMetrics[]>("/analytics/recipes", {
      params: { month, year },
    });
    return data;
  },

  getTrends: async (months = 6): Promise<TrendPoint[]> => {
    const { data } = await api.get<TrendPoint[]>("/analytics/trends", {
      params: { months },
    });
    return data;
  },
};
