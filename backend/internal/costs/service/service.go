package service

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/PipeM113/bakery-manager/internal/costs/domain"
	"github.com/PipeM113/bakery-manager/internal/recipeneeds"
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

	lines, err := recipeneeds.Load(ctx, s.db, recipeID, 1)
	var conv *recipeneeds.ConversionError
	if errors.As(err, &conv) {
		return Recipe{}, fmt.Errorf("ingrediente %q: %w", conv.Ingredient, conv.Err)
	}
	if err != nil {
		return Recipe{}, err
	}
	for _, l := range lines {
		recipe.ingredients = append(recipe.ingredients, domain.IngredientCost{
			IngredientID: l.IngredientID, Name: l.Name, Kind: l.Kind, Quantity: l.Quantity, Unit: l.Unit,
			PricePerUnit: l.PricePerUnit, PackagePrice: l.PackagePrice, PackageSize: l.PackageSize,
		})
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

// Price is what the list of recipes shows for a recipe: the total cost and the suggested price, in whole
// pesos, as the cost screen gives them with its default values.
type Price struct {
	TotalCost      float64
	SuggestedPrice float64
}

// PriceInput is what PriceMany needs to know of a recipe besides its lines (the list already has it read).
type PriceInput struct {
	ID              string
	Name            string
	Yield           float64
	YieldUnit       string
	IndirectCostPct float64
	LaborCostPct    float64
	MarginPct       float64
}

// PriceMany prices several recipes with one query (their lines). A recipe that cannot be priced (a line
// that cannot be converted, a yield that is not positive) is left out of the result.
func (s *CostService) PriceMany(ctx context.Context, in []PriceInput) (map[string]Price, error) {
	out := map[string]Price{}
	if len(in) == 0 {
		return out, nil
	}
	ids := make([]string, 0, len(in))
	for _, p := range in {
		ids = append(ids, p.ID)
	}
	lines, failed, err := recipeneeds.LoadMany(ctx, s.db, ids, 1)
	if err != nil {
		return nil, err
	}
	for _, p := range in {
		recipeLines := lines[p.ID] // a recipe without lines costs 0, as in the cost screen
		if p.Yield <= 0 || failed[p.ID] != nil {
			continue // without yield, or with a line that could not be converted
		}
		r := Recipe{ID: p.ID, Name: p.Name, Yield: p.Yield, YieldUnit: p.YieldUnit,
			IndirectCostPct: p.IndirectCostPct, LaborCostPct: p.LaborCostPct, MarginPct: p.MarginPct}
		for _, l := range recipeLines {
			r.ingredients = append(r.ingredients, domain.IngredientCost{
				IngredientID: l.IngredientID, Name: l.Name, Kind: l.Kind, Quantity: l.Quantity, Unit: l.Unit,
				PricePerUnit: l.PricePerUnit, PackagePrice: l.PackagePrice, PackageSize: l.PackageSize,
			})
		}
		b := r.Compute(CostParams{})
		out[p.ID] = Price{TotalCost: b.TotalCost, SuggestedPrice: b.SuggestedPrice}
	}
	return out, nil
}
