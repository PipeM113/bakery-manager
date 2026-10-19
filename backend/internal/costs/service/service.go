package service

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/PipeM113/bakery-manager/internal/costs/domain"
	"github.com/PipeM113/bakery-manager/internal/shared/kernel"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Limits of the percentages, as fractions: 1 is 100%.
const (
	MaxIndirectPct = 1.0
	MaxLaborPct    = 1.0
	MaxMarkupPct   = 10.0
)

// DefaultPct is what a new recipe starts with in the three percentages.
const DefaultPct = 0.30

// ErrInvalidYield means the recipe cannot be priced because its yield is not positive.
var ErrInvalidYield = errors.New("el rendimiento de la receta debe ser mayor a 0")

// ParamError is a cost parameter that is out of range or not a number.
type ParamError struct{ Message string }

func (e *ParamError) Error() string { return e.Message }

type CostService struct {
	db *pgxpool.Pool
}

func NewCostService(db *pgxpool.Pool) *CostService {
	return &CostService{db: db}
}

// CostParams overrides what is stored in the recipe. A nil percentage means "not sent": the
// recipe's stored value is used. A pointer to 0 is a real 0%.
// ExtraCharge and DeliveryCost are always used as given (0 when not sent).
type CostParams struct {
	IndirectCostPct *float64
	LaborCostPct    *float64
	MarginPct       *float64
	ExtraCharge     int
	DeliveryCost    int
	// Portions, when set, prices the recipe made for that many portions (at least 1).
	Portions *float64
}

// CheckPct validates one percentage (a fraction) against 0..max.
func CheckPct(name string, v float64, max float64) error {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > max {
		return &ParamError{Message: fmt.Sprintf("%s debe estar entre 0 y %g", name, max)}
	}
	return nil
}

// Validate checks the percentages that were sent.
func (p CostParams) Validate() error {
	if p.IndirectCostPct != nil {
		if err := CheckPct("indirect_cost_pct", *p.IndirectCostPct, MaxIndirectPct); err != nil {
			return err
		}
	}
	if p.LaborCostPct != nil {
		if err := CheckPct("labor_cost_pct", *p.LaborCostPct, MaxLaborPct); err != nil {
			return err
		}
	}
	if p.MarginPct != nil {
		if err := CheckPct("margin_pct", *p.MarginPct, MaxMarkupPct); err != nil {
			return err
		}
	}
	if p.Portions != nil {
		if v := *p.Portions; math.IsNaN(v) || math.IsInf(v, 0) || v < 1 {
			return &ParamError{Message: "portions debe ser un número mayor o igual a 1"}
		}
	}
	return nil
}

func (s *CostService) GetCostBreakdown(ctx context.Context, recipeID string, params CostParams) (domain.CostBreakdown, error) {
	var recipe struct {
		Name            string
		Yield           float64
		YieldUnit       string
		IndirectCostPct float64
		LaborCostPct    float64
		MarginPct       float64
	}

	err := s.db.QueryRow(ctx, `
		SELECT name, yield, yield_unit, indirect_cost_pct, labor_cost_pct, margin_pct
		FROM recipes WHERE id = $1`, recipeID,
	).Scan(
		&recipe.Name, &recipe.Yield, &recipe.YieldUnit,
		&recipe.IndirectCostPct, &recipe.LaborCostPct, &recipe.MarginPct,
	)
	if err != nil {
		return domain.CostBreakdown{}, fmt.Errorf("recipe not found: %w", err)
	}
	if recipe.Yield <= 0 {
		return domain.CostBreakdown{}, ErrInvalidYield
	}

	rows, err := s.db.Query(ctx, `
		SELECT ri.ingredient_id, i.name, ri.quantity, ri.unit,
		       i.price_per_unit, i.default_unit, i.package_price, i.package_size
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
			&ing.PackagePrice, &ing.PackageSize,
		); err != nil {
			return domain.CostBreakdown{}, fmt.Errorf("scan ingredient: %w", err)
		}

		converted, err := kernel.ConvertToBase(ing.Quantity, recipeUnit, baseUnit)
		if err != nil {
			return domain.CostBreakdown{}, fmt.Errorf("ingrediente %q: %w", ing.Name, err)
		}
		ing.Quantity = converted
		ing.Unit = baseUnit
		ingredients = append(ingredients, ing)
	}
	if err := rows.Err(); err != nil {
		return domain.CostBreakdown{}, fmt.Errorf("read ingredients: %w", err)
	}

	indirectPct, laborPct, markupPct := recipe.IndirectCostPct, recipe.LaborCostPct, recipe.MarginPct
	if params.IndirectCostPct != nil {
		indirectPct = *params.IndirectCostPct
	}
	if params.LaborCostPct != nil {
		laborPct = *params.LaborCostPct
	}
	if params.MarginPct != nil {
		markupPct = *params.MarginPct
	}

	if params.Portions != nil {
		return domain.CalculateScaled(
			recipeID, recipe.Name,
			recipe.Yield, *params.Portions, recipe.YieldUnit,
			indirectPct, laborPct, markupPct,
			params.ExtraCharge, params.DeliveryCost,
			ingredients,
		), nil
	}
	return domain.Calculate(
		recipeID, recipe.Name,
		recipe.Yield, recipe.YieldUnit,
		indirectPct, laborPct, markupPct,
		params.ExtraCharge, params.DeliveryCost,
		ingredients,
	), nil
}
