package domain

// MonthlyMetrics aggregates profitability for a given month/year.
type MonthlyMetrics struct {
	Month                    int     `json:"month"`
	Year                     int     `json:"year"`
	TotalRevenue             float64 `json:"total_revenue"`
	TotalIngredientsCost     float64 `json:"total_ingredients_cost"`
	TotalFixedCosts          float64 `json:"total_fixed_costs"`
	TotalOperationalExpenses float64 `json:"total_operational_expenses"`
	GrossProfit              float64 `json:"gross_profit"`
	NetProfit                float64 `json:"net_profit"`
	ProfitMargin             float64 `json:"profit_margin"`
}

// RecipeMetrics holds per-recipe profitability for a given month/year.
type RecipeMetrics struct {
	RecipeID     string  `json:"recipe_id"`
	RecipeName   string  `json:"recipe_name"`
	UnitsSold    int     `json:"units_sold"`
	TotalRevenue float64 `json:"total_revenue"`
	CostPerUnit  float64 `json:"cost_per_unit"`
	TotalCost    float64 `json:"total_cost"`
	UnitProfit   float64 `json:"unit_profit"`
	GrossMargin  float64 `json:"gross_margin"`
}

// TrendPoint represents aggregated metrics for a single month in a time series.
type TrendPoint struct {
	Date      string  `json:"date"` // "YYYY-MM"
	Revenue   float64 `json:"revenue"`
	NetProfit float64 `json:"net_profit"`
}
