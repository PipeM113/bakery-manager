package service

import (
	"context"
	"fmt"
	"time"

	"github.com/PipeM113/bakery-manager/internal/analytics/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AnalyticsService struct {
	db *pgxpool.Pool
}

func NewAnalyticsService(db *pgxpool.Pool) *AnalyticsService {
	return &AnalyticsService{db: db}
}

// GetMonthlyMetrics computes profitability aggregates for the given month and year.
func (s *AnalyticsService) GetMonthlyMetrics(ctx context.Context, userID string, month, year int) (domain.MonthlyMetrics, error) {
	m := domain.MonthlyMetrics{Month: month, Year: year}

	// Total revenue
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(quantity_sold * unit_price), 0)
		FROM sales
		WHERE user_id = $1
		  AND EXTRACT(MONTH FROM sale_date) = $2
		  AND EXTRACT(YEAR  FROM sale_date) = $3`,
		userID, month, year,
	).Scan(&m.TotalRevenue)
	if err != nil {
		return m, fmt.Errorf("total revenue: %w", err)
	}

	// Total ingredient cost (using sale_ingredients price snapshots)
	err = s.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(si.price_at_time * si.quantity_used), 0)
		FROM sale_ingredients si
		JOIN sales s ON s.id = si.sale_id
		WHERE s.user_id = $1
		  AND EXTRACT(MONTH FROM s.sale_date) = $2
		  AND EXTRACT(YEAR  FROM s.sale_date) = $3`,
		userID, month, year,
	).Scan(&m.TotalIngredientsCost)
	if err != nil {
		return m, fmt.Errorf("total ingredients cost: %w", err)
	}

	// Total active fixed costs (monthly amount, applied in full)
	err = s.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(monthly_amount), 0)
		FROM fixed_costs
		WHERE user_id = $1 AND is_active = TRUE`,
		userID,
	).Scan(&m.TotalFixedCosts)
	if err != nil {
		return m, fmt.Errorf("total fixed costs: %w", err)
	}

	// Total operational expenses for the month
	err = s.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0)
		FROM operational_expenses
		WHERE user_id = $1
		  AND EXTRACT(MONTH FROM expense_date) = $2
		  AND EXTRACT(YEAR  FROM expense_date) = $3`,
		userID, month, year,
	).Scan(&m.TotalOperationalExpenses)
	if err != nil {
		return m, fmt.Errorf("total operational expenses: %w", err)
	}

	m.GrossProfit = m.TotalRevenue - m.TotalIngredientsCost
	m.NetProfit = m.GrossProfit - m.TotalFixedCosts - m.TotalOperationalExpenses
	if m.TotalRevenue > 0 {
		m.ProfitMargin = m.NetProfit / m.TotalRevenue * 100
	}
	return m, nil
}

// GetRecipeMetrics returns per-recipe profitability for the given month/year, sorted by revenue desc.
func (s *AnalyticsService) GetRecipeMetrics(ctx context.Context, userID string, month, year int) ([]domain.RecipeMetrics, error) {
	rows, err := s.db.Query(ctx, `
		SELECT
			r.id,
			r.name,
			SUM(s.quantity_sold)::int                           AS units_sold,
			SUM(s.quantity_sold * s.unit_price)                 AS total_revenue,
			COALESCE(SUM(si_agg.ingredient_cost), 0)            AS total_cost
		FROM sales s
		JOIN recipes r ON r.id = s.recipe_id
		LEFT JOIN (
			SELECT si.sale_id, SUM(si.price_at_time * si.quantity_used) AS ingredient_cost
			FROM sale_ingredients si
			GROUP BY si.sale_id
		) si_agg ON si_agg.sale_id = s.id
		WHERE s.user_id = $1
		  AND EXTRACT(MONTH FROM s.sale_date) = $2
		  AND EXTRACT(YEAR  FROM s.sale_date) = $3
		GROUP BY r.id, r.name
		ORDER BY total_revenue DESC`,
		userID, month, year,
	)
	if err != nil {
		return nil, fmt.Errorf("recipe metrics query: %w", err)
	}
	defer rows.Close()

	var result []domain.RecipeMetrics
	for rows.Next() {
		var rm domain.RecipeMetrics
		if err := rows.Scan(&rm.RecipeID, &rm.RecipeName, &rm.UnitsSold, &rm.TotalRevenue, &rm.TotalCost); err != nil {
			return nil, fmt.Errorf("scan recipe metrics: %w", err)
		}
		if rm.UnitsSold > 0 {
			rm.CostPerUnit = rm.TotalCost / float64(rm.UnitsSold)
			avgPrice := rm.TotalRevenue / float64(rm.UnitsSold)
			rm.UnitProfit = avgPrice - rm.CostPerUnit
		}
		if rm.TotalRevenue > 0 {
			rm.GrossMargin = (rm.TotalRevenue - rm.TotalCost) / rm.TotalRevenue * 100
		}
		result = append(result, rm)
	}
	return result, nil
}

// GetTrendData returns monthly aggregates for the last N months (oldest first).
func (s *AnalyticsService) GetTrendData(ctx context.Context, userID string, months int) ([]domain.TrendPoint, error) {
	if months <= 0 {
		months = 6
	}
	now := time.Now()
	result := make([]domain.TrendPoint, 0, months)

	for i := months - 1; i >= 0; i-- {
		t := now.AddDate(0, -i, 0)
		m, y := int(t.Month()), t.Year()

		metrics, err := s.GetMonthlyMetrics(ctx, userID, m, y)
		if err != nil {
			return nil, fmt.Errorf("trend month %d/%d: %w", m, y, err)
		}
		result = append(result, domain.TrendPoint{
			Date:      fmt.Sprintf("%04d-%02d", y, m),
			Revenue:   metrics.TotalRevenue,
			NetProfit: metrics.NetProfit,
		})
	}
	return result, nil
}
