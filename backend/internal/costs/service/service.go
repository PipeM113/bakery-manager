package service

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/PipeM113/bakery-manager/internal/costs/domain"
	"github.com/PipeM113/bakery-manager/internal/shared/kernel"
	"github.com/jackc/pgx/v5"
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

// ErrRecipeNotFound means the recipe does not exist.
var ErrRecipeNotFound = errors.New("la receta no existe")

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

// Recipe is a recipe with what is needed to price it, read once. Compute can price it many times
// (with different percentages, portions...) without going back to the database.
type Recipe struct {
	ID              string
	Name            string
	PhotoURL        *string
	Yield           float64
	YieldUnit       string
	IndirectCostPct float64
	LaborCostPct    float64
	MarginPct       float64
	ingredients     []domain.IngredientCost
}

// LoadRecipe reads a recipe and its ingredients (2 queries). A recipe that does not exist gives
// ErrRecipeNotFound; one whose yield is not positive gives ErrInvalidYield.
func (s *CostService) LoadRecipe(ctx context.Context, recipeID string) (Recipe, error) {
	recipe := Recipe{ID: recipeID}
	err := s.db.QueryRow(ctx, `
		SELECT name, photo_url, yield, yield_unit, indirect_cost_pct, labor_cost_pct, margin_pct
		FROM recipes WHERE id = $1`, recipeID,
	).Scan(&recipe.Name, &recipe.PhotoURL, &recipe.Yield, &recipe.YieldUnit,
		&recipe.IndirectCostPct, &recipe.LaborCostPct, &recipe.MarginPct)
	if errors.Is(err, pgx.ErrNoRows) {
		return Recipe{}, fmt.Errorf("recipe not found: %w", ErrRecipeNotFound)
	}
	if err != nil {
		return Recipe{}, fmt.Errorf("recipe not found: %w", err)
	}
	if recipe.Yield <= 0 {
		return Recipe{}, ErrInvalidYield
	}

	rows, err := s.db.Query(ctx, `
		SELECT ri.ingredient_id, i.name, i.kind, ri.quantity, ri.unit,
		       i.price_per_unit, i.default_unit, i.package_price, i.package_size
		FROM recipe_ingredients ri
		JOIN ingredients i ON i.id = ri.ingredient_id
		WHERE ri.recipe_id = $1`, recipeID)
	if err != nil {
		return Recipe{}, fmt.Errorf("get ingredients: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ing domain.IngredientCost
		var recipeUnit, baseUnit string
		if err := rows.Scan(
			&ing.IngredientID, &ing.Name, &ing.Kind,
			&ing.Quantity, &recipeUnit,
			&ing.PricePerUnit, &baseUnit,
			&ing.PackagePrice, &ing.PackageSize,
		); err != nil {
			return Recipe{}, fmt.Errorf("scan ingredient: %w", err)
		}

		converted, err := kernel.ConvertToBase(ing.Quantity, recipeUnit, baseUnit)
		if err != nil {
			return Recipe{}, fmt.Errorf("ingrediente %q: %w", ing.Name, err)
		}
		ing.Quantity = converted
		ing.Unit = baseUnit
		recipe.ingredients = append(recipe.ingredients, ing)
	}
	if err := rows.Err(); err != nil {
		return Recipe{}, fmt.Errorf("read ingredients: %w", err)
	}
	return recipe, nil
}

// Compute prices the recipe with the given parameters; the recipe itself is not changed.
func (r Recipe) Compute(params CostParams) domain.CostBreakdown {
	ingredients := append([]domain.IngredientCost(nil), r.ingredients...) // the calculation writes into them

	indirectPct, laborPct, markupPct := r.IndirectCostPct, r.LaborCostPct, r.MarginPct
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
			r.ID, r.Name,
			r.Yield, *params.Portions, r.YieldUnit,
			indirectPct, laborPct, markupPct,
			params.ExtraCharge, params.DeliveryCost,
			ingredients,
		)
	}
	return domain.Calculate(
		r.ID, r.Name,
		r.Yield, r.YieldUnit,
		indirectPct, laborPct, markupPct,
		params.ExtraCharge, params.DeliveryCost,
		ingredients,
	)
}

func (s *CostService) GetCostBreakdown(ctx context.Context, recipeID string, params CostParams) (domain.CostBreakdown, error) {
	recipe, err := s.LoadRecipe(ctx, recipeID)
	if err != nil {
		return domain.CostBreakdown{}, err
	}
	return recipe.Compute(params), nil
}
