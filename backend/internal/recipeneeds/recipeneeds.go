// Package recipeneeds says how much of each ingredient a recipe needs, in the unit the ingredient is kept
// in, using the equivalences of the ingredient. Costs and sales both read it from here, so the rule is
// written once (docs/ingredientes.md).
package recipeneeds

import (
	"context"
	"fmt"

	"github.com/PipeM113/bakery-manager/internal/shared/kernel"
	"github.com/jackc/pgx/v5"
)

// DB is what Load needs: a pool or a transaction.
type DB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Line is one ingredient of a recipe: the quantity already converted to Unit, the unit of the ingredient,
// and its prices.
type Line struct {
	IngredientID string
	Name         string
	Kind         string
	Quantity     float64
	Unit         string
	PricePerUnit float64
	PackagePrice float64
	PackageSize  float64
}

// ConversionError says which ingredient of the recipe cannot be converted to its unit, and why (it wraps
// the *kernel.EquivalenceError when an equivalence is missing).
type ConversionError struct {
	Ingredient string
	Err        error
}

func (e *ConversionError) Error() string { return fmt.Sprintf("%s: %v", e.Ingredient, e.Err) }
func (e *ConversionError) Unwrap() error { return e.Err }

// Load reads the lines of a recipe and multiplies every quantity by factor (1 for the recipe as it is;
// units sold, or a share of the recipe, for a sale).
func Load(ctx context.Context, db DB, recipeID string, factor float64) ([]Line, error) {
	rows, err := db.Query(ctx, `
		SELECT ri.ingredient_id, i.name, i.kind, ri.quantity, ri.unit,
		       i.price_per_unit, i.default_unit, i.package_price, i.package_size,
		       i.grams_per_liter, i.grams_per_unit
		FROM recipe_ingredients ri
		JOIN ingredients i ON i.id = ri.ingredient_id
		WHERE ri.recipe_id = $1`, recipeID)
	if err != nil {
		return nil, fmt.Errorf("get ingredients: %w", err)
	}
	defer rows.Close()

	lines := []Line{}
	for rows.Next() {
		var l Line
		var quantity float64
		var recipeUnit string
		var eq kernel.Equivalence
		if err := rows.Scan(&l.IngredientID, &l.Name, &l.Kind, &quantity, &recipeUnit,
			&l.PricePerUnit, &l.Unit, &l.PackagePrice, &l.PackageSize, &eq.GramsPerLiter, &eq.GramsPerUnit); err != nil {
			return nil, fmt.Errorf("scan ingredient: %w", err)
		}
		converted, err := kernel.Convert(quantity*factor, recipeUnit, l.Unit, eq)
		if err != nil {
			return nil, &ConversionError{Ingredient: l.Name, Err: err}
		}
		l.Quantity = converted
		lines = append(lines, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read ingredients: %w", err)
	}
	return lines, nil
}
