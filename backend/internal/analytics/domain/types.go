package domain

// All money is in whole pesos. Percentages carry one decimal.

// CategoryExpense is what was spent in one expense category during a month.
type CategoryExpense struct {
	CategoryID string `json:"category_id"`
	Category   string `json:"category"`
	Total      int64  `json:"total"`
}

// MonthlyMetrics aggregates profitability for a given month/year.
type MonthlyMetrics struct {
	Month                    int               `json:"month"`
	Year                     int               `json:"year"`
	TotalRevenue             int64             `json:"total_revenue"`
	TotalIngredientsCost     int64             `json:"total_ingredients_cost"`
	TotalFixedCosts          int64             `json:"total_fixed_costs"`
	TotalOperationalExpenses int64             `json:"total_operational_expenses"`
	GrossProfit              int64             `json:"gross_profit"`
	NetProfit                int64             `json:"net_profit"`
	ProfitMargin             float64           `json:"profit_margin"` // net profit over revenue, in percent
	ExpensesByCategory       []CategoryExpense `json:"expenses_by_category"`
}

// RecipeMetrics holds per-recipe profitability for a given month/year.
type RecipeMetrics struct {
	RecipeID     string  `json:"recipe_id"`
	RecipeName   string  `json:"recipe_name"`
	UnitsSold    int     `json:"units_sold"`
	TotalRevenue int64   `json:"total_revenue"`
	CostPerUnit  int64   `json:"cost_per_unit"`
	TotalCost    int64   `json:"total_cost"`
	UnitProfit   int64   `json:"unit_profit"`
	GrossMargin  float64 `json:"gross_margin"` // (revenue - ingredients) over revenue, in percent
}

// TrendPoint represents aggregated metrics for a single month in a time series.
type TrendPoint struct {
	Date      string `json:"date"` // "YYYY-MM"
	Revenue   int64  `json:"revenue"`
	NetProfit int64  `json:"net_profit"`
}
