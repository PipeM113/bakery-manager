package service

import (
	"context"
	"fmt"

	"github.com/PipeM113/bakery-manager/internal/costs/domain"
	"github.com/PipeM113/bakery-manager/internal/shared/kernel"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CostService struct {
	db *pgxpool.Pool
}

func NewCostService(db *pgxpool.Pool) *CostService {
	return &CostService{db: db}
}

// CostParams permite sobrescribir los porcentajes almacenados en la receta.
// Si un valor es 0, se usa el valor guardado en la receta (indirect/labor)
// o se usa 0 como margen (margin_pct).
type CostParams struct {
	IndirectCostPct float64
	LaborCostPct    float64
	MarginPct       float64
}

func (s *CostService) GetCostBreakdown(ctx context.Context, recipeID string, params CostParams) (domain.CostBreakdown, error) {
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
		SELECT ri.ingredient_id, i.name, ri.quantity, ri.unit,
		       i.price_per_unit, i.default_unit
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
		var recipeUnit, baseUnit string
		if err := rows.Scan(
			&ing.IngredientID, &ing.Name,
			&ing.Quantity, &recipeUnit,
			&ing.PricePerUnit, &baseUnit,
		); err != nil {
			return domain.CostBreakdown{}, fmt.Errorf("scan ingredient: %w", err)
		}

		// Convertir cantidad a la unidad base del insumo para calcular correctamente
		converted, err := kernel.ConvertToBase(ing.Quantity, recipeUnit, baseUnit)
		if err != nil {
			return domain.CostBreakdown{}, fmt.Errorf("ingrediente %q: %w", ing.Name, err)
		}
		ing.Quantity = converted
		ing.Unit = baseUnit
		ingredients = append(ingredients, ing)
	}

	indirectPct := recipe.IndirectCostPct
	if params.IndirectCostPct > 0 {
		indirectPct = params.IndirectCostPct
	}
	laborPct := recipe.LaborCostPct
	if params.LaborCostPct > 0 {
		laborPct = params.LaborCostPct
	}

	return domain.Calculate(
		recipeID, recipe.Name,
		recipe.Yield, recipe.YieldUnit,
		indirectPct, laborPct, params.MarginPct,
		ingredients,
	), nil
}
