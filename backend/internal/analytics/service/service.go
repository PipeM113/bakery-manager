package service

import (
	"context"
	"fmt"
	"math"
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

// GetMonthlyMetrics computes profitability aggregates for the given month and year, in whole
// pesos. The cost of ingredients is summed exactly and rounded once at the end of the month.
func (s *AnalyticsService) GetMonthlyMetrics(ctx context.Context, userID string, month, year int) (domain.MonthlyMetrics, error) {
	m := domain.MonthlyMetrics{Month: month, Year: year, ExpensesByCategory: []domain.CategoryExpense{}}

	// Total revenue
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(quantity_sold::bigint * unit_price), 0)::bigint
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
		SELECT COALESCE(ROUND(SUM(si.price_at_time * si.quantity_used)), 0)::bigint
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

	// Fixed costs in force that month, each with the amount of that month
	err = s.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(v.monthly_amount), 0)::bigint
		FROM fixed_costs fc
		JOIN LATERAL (
			SELECT monthly_amount FROM fixed_cost_versions
			WHERE fixed_cost_id = fc.id AND valid_from <= $2::date
			ORDER BY valid_from DESC LIMIT 1) v ON TRUE
		WHERE fc.user_id = $1 AND (fc.ended_from IS NULL OR $2::date < fc.ended_from)`,
		userID, fmt.Sprintf("%04d-%02d-01", year, month),
	).Scan(&m.TotalFixedCosts)
	if err != nil {
		return m, fmt.Errorf("total fixed costs: %w", err)
	}

	// Operational expenses for the month, by category (largest first)
	rows, err := s.db.Query(ctx, `
		SELECT c.id, c.name, SUM(e.amount)::bigint AS total
		FROM operational_expenses e
		JOIN expense_categories c ON c.id = e.category_id
		WHERE e.user_id = $1
		  AND EXTRACT(MONTH FROM e.expense_date) = $2
		  AND EXTRACT(YEAR  FROM e.expense_date) = $3
		GROUP BY c.id, c.name
		ORDER BY total DESC, lower(c.name)`,
		userID, month, year,
	)
	if err != nil {
		return m, fmt.Errorf("expenses by category: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var ce domain.CategoryExpense
		if err := rows.Scan(&ce.CategoryID, &ce.Category, &ce.Total); err != nil {
			return m, fmt.Errorf("scan expenses by category: %w", err)
		}
		m.TotalOperationalExpenses += ce.Total
		m.ExpensesByCategory = append(m.ExpensesByCategory, ce)
	}
	if err := rows.Err(); err != nil {
		return m, fmt.Errorf("read expenses by category: %w", err)
	}

	m.GrossProfit = m.TotalRevenue - m.TotalIngredientsCost
	m.NetProfit = m.GrossProfit - m.TotalFixedCosts - m.TotalOperationalExpenses
	if m.TotalRevenue > 0 {
		m.ProfitMargin = percentOneDecimal(m.NetProfit, m.TotalRevenue)
	}
	return m, nil
}

// percentOneDecimal is part over whole in percent, rounded to one decimal.
func percentOneDecimal(part, whole int64) float64 {
	return math.Round(float64(part)/float64(whole)*1000) / 10
}

// GetRecipeMetrics returns per-recipe profitability for the given month/year, sorted by revenue desc.
func (s *AnalyticsService) GetRecipeMetrics(ctx context.Context, userID string, month, year int) ([]domain.RecipeMetrics, error) {
	rows, err := s.db.Query(ctx, `
		WITH per_recipe AS (
			SELECT
				r.id,
				r.name,
				SUM(s.quantity_sold)::bigint                        AS units_sold,
				SUM(s.quantity_sold::bigint * s.unit_price)         AS revenue,
				COALESCE(SUM(si_agg.ingredient_cost), 0)            AS cost
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
		)
		SELECT id, name, units_sold::int, revenue::bigint,
		       ROUND(cost)::bigint,
		       ROUND(cost / units_sold)::bigint,
		       ROUND(revenue::numeric / units_sold)::bigint,
		       COALESCE(ROUND((revenue - cost) * 100 / NULLIF(revenue, 0), 1), 0)::float8
		FROM per_recipe
		ORDER BY revenue DESC`,
		userID, month, year,
	)
	if err != nil {
		return nil, fmt.Errorf("recipe metrics query: %w", err)
	}
	defer rows.Close()

	var result []domain.RecipeMetrics
	for rows.Next() {
		var rm domain.RecipeMetrics
		var avgPrice int64
		if err := rows.Scan(&rm.RecipeID, &rm.RecipeName, &rm.UnitsSold, &rm.TotalRevenue,
			&rm.TotalCost, &rm.CostPerUnit, &avgPrice, &rm.GrossMargin); err != nil {
			return nil, fmt.Errorf("scan recipe metrics: %w", err)
		}
		rm.UnitProfit = avgPrice - rm.CostPerUnit
		result = append(result, rm)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read recipe metrics: %w", err)
	}
	return result, nil
}

// GetTrendData returns the revenue and the net profit of each month of `months` ("YYYY-MM",
// consecutive, oldest first). It asks the database four questions in all, whatever the number of
// months, and each figure is calculated as in GetMonthlyMetrics.
func (s *AnalyticsService) GetTrendData(ctx context.Context, userID string, months []string) ([]domain.TrendPoint, error) {
	if len(months) == 0 {
		return []domain.TrendPoint{}, nil
	}
	first, err := time.Parse("2006-01", months[0])
	if err != nil {
		return nil, fmt.Errorf("trend months: %w", err)
	}
	from := first.Format("2006-01-02")
	until := first.AddDate(0, len(months), 0).Format("2006-01-02")

	perMonth := func(name, query string) (map[string]int64, error) {
		rows, err := s.db.Query(ctx, query, userID, from, until)
		if err != nil {
			return nil, fmt.Errorf("trend %s: %w", name, err)
		}
		defer rows.Close()
		out := map[string]int64{}
		for rows.Next() {
			var ym string
			var total int64
			if err := rows.Scan(&ym, &total); err != nil {
				return nil, fmt.Errorf("scan trend %s: %w", name, err)
			}
			out[ym] = total
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("read trend %s: %w", name, err)
		}
		return out, nil
	}

	revenue, err := perMonth("revenue", `
		SELECT to_char(sale_date, 'YYYY-MM'), SUM(quantity_sold::bigint * unit_price)::bigint
		FROM sales
		WHERE user_id = $1 AND sale_date >= $2::date AND sale_date < $3::date
		GROUP BY 1`)
	if err != nil {
		return nil, err
	}
	ingredients, err := perMonth("ingredients", `
		SELECT to_char(s.sale_date, 'YYYY-MM'), ROUND(SUM(si.price_at_time * si.quantity_used))::bigint
		FROM sale_ingredients si
		JOIN sales s ON s.id = si.sale_id
		WHERE s.user_id = $1 AND s.sale_date >= $2::date AND s.sale_date < $3::date
		GROUP BY 1`)
	if err != nil {
		return nil, err
	}
	expenses, err := perMonth("expenses", `
		SELECT to_char(expense_date, 'YYYY-MM'), SUM(amount)::bigint
		FROM operational_expenses
		WHERE user_id = $1 AND expense_date >= $2::date AND expense_date < $3::date
		GROUP BY 1`)
	if err != nil {
		return nil, err
	}

	// Fixed costs in force in each month, with that month's amount (as in GetMonthlyMetrics).
	fixed, err := perMonth("fixed costs", `
		SELECT to_char(m, 'YYYY-MM'), COALESCE(SUM(v.monthly_amount), 0)::bigint
		FROM generate_series($2::date, $3::date - 1, interval '1 month') AS m
		JOIN fixed_costs fc ON fc.user_id = $1 AND (fc.ended_from IS NULL OR m::date < fc.ended_from)
		JOIN LATERAL (
			SELECT monthly_amount FROM fixed_cost_versions
			WHERE fixed_cost_id = fc.id AND valid_from <= m::date
			ORDER BY valid_from DESC LIMIT 1) v ON TRUE
		GROUP BY 1`)
	if err != nil {
		return nil, err
	}

	result := make([]domain.TrendPoint, 0, len(months))
	for _, ym := range months {
		result = append(result, domain.TrendPoint{
			Date:      ym,
			Revenue:   revenue[ym],
			NetProfit: revenue[ym] - ingredients[ym] - fixed[ym] - expenses[ym],
		})
	}
	return result, nil
}
