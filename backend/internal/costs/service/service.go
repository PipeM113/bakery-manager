package service

import (
	"context"
	"fmt"

	"github.com/PipeM113/bakery-manager/internal/costs/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CostService struct {
	db *pgxpool.Pool
}

func NewCostService(db *pgxpool.Pool) *CostService {
	return &CostService{db: db}
}

func (s *CostService) GetCostBreakdown(ctx context.Context, recipeID string) (domain.CostBreakdown, error) {
	var recipe struct {
		Name            string
		Yield           float64
		YieldUnit       string
		IndirectCostPct float64
		LaborCostPct    float64
	}

	err := s.db.QueryRow(ctx, `
		SELECT name, yield, yield_unit, indirect_cost_pct, labor_cost_pct
		FROM recipes WHERE id = $1`, recipeID,
	).Scan(
		&recipe.Name, &recipe.Yield, &recipe.YieldUnit,
		&recipe.IndirectCostPct, &recipe.LaborCostPct,
	)
	if err != nil {
		return domain.CostBreakdown{}, fmt.Errorf("recipe not found: %w", err)
	}

	rows, err := s.db.Query(ctx, `
		SELECT ri.ingredient_id, i.name, ri.quantity, ri.unit, i.price_per_unit
		FROM recipe_ingredients ri
		JOIN ingredients i ON i.id = ri.ingredient_id
		WHERE ri.recipe_id = $1`, recipeID)
	if err != nil {
		return domain.CostBreakdown{}, fmt.Errorf("get ingredients: %w", err)
	}
	defer rows.Close()

	var ingredients []domain.IngredientCost
	for rows.Next() {
		var ing domain.IngredientCost
		if err := rows.Scan(
			&ing.IngredientID, &ing.Name,
			&ing.Quantity, &ing.Unit, &ing.PricePerUnit,
		); err != nil {
			return domain.CostBreakdown{}, fmt.Errorf("scan ingredient: %w", err)
		}
		ingredients = append(ingredients, ing)
	}

	return domain.Calculate(
		recipeID, recipe.Name,
		recipe.Yield, recipe.YieldUnit,
		recipe.IndirectCostPct, recipe.LaborCostPct,
		ingredients,
	), nil
}
