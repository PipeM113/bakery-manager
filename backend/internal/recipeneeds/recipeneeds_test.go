package recipeneeds_test

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/PipeM113/bakery-manager/internal/recipeneeds"
	"github.com/PipeM113/bakery-manager/internal/shared/kernel"
	"github.com/PipeM113/bakery-manager/internal/testsupport/pgtest"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Sprint 15, N1: one place says how much of each ingredient a recipe needs, in the unit the ingredient is
// kept in, using the equivalences. Costs and sales both read it from here.

type fixture struct {
	pool   *pgxpool.Pool
	recipe string
}

func setup(t *testing.T) fixture {
	t.Helper()
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, "../../db/migrations", "up")
	ctx := context.Background()
	var recipe string
	if err := pool.QueryRow(ctx, `insert into recipes (name, description, yield, yield_unit) values ('Flan', 'x', 4, 'porciones') returning id`).Scan(&recipe); err != nil {
		t.Fatal(err)
	}
	return fixture{pool: pool, recipe: recipe}
}

func (f fixture) line(t *testing.T, name, unit string, perLiter, perUnit float64, packageSize, packagePrice, quantity float64, lineUnit string) {
	t.Helper()
	ctx := context.Background()
	var id string
	if err := f.pool.QueryRow(ctx, `insert into ingredients (name, brand, default_unit, price_per_unit, package_size, package_price, grams_per_liter, grams_per_unit)
		values ($1, 'M', $2, $3, $4, $5, $6, $7) returning id`, name, unit, packagePrice/packageSize, packageSize, packagePrice, perLiter, perUnit).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `insert into recipe_ingredients (recipe_id, ingredient_id, quantity, unit) values ($1, $2, $3, $4)`, f.recipe, id, quantity, lineUnit); err != nil {
		t.Fatal(err)
	}
}

func byName(lines []recipeneeds.Line) map[string]recipeneeds.Line {
	out := map[string]recipeneeds.Line{}
	for _, l := range lines {
		out[l.Name] = l
	}
	return out
}

func TestN1_QuantitiesComeInTheUnitOfTheIngredient(t *testing.T) {
	f := setup(t)
	f.line(t, "Harina", "gr", 0, 0, 1000, 2000, 0.5, "kg") // 0.5 kg -> 500 gr
	f.line(t, "Leche", "lt", 1250, 0, 1, 1000, 625, "gr")  // 625 gr at 1250 gr/lt -> 0.5 lt
	f.line(t, "Huevo", "und", 0, 60, 12, 3600, 120, "gr")  // 120 gr at 60 gr/und -> 2 und
	lines, err := recipeneeds.Load(context.Background(), f.pool, f.recipe, 1)
	if err != nil {
		t.Fatal(err)
	}
	got := byName(lines)
	for name, want := range map[string]struct {
		qty  float64
		unit string
	}{"Harina": {500, "gr"}, "Leche": {0.5, "lt"}, "Huevo": {2, "und"}} {
		l, ok := got[name]
		if !ok || math.Abs(l.Quantity-want.qty) > 1e-9 || l.Unit != want.unit {
			t.Errorf("%s = %+v, want %v %s", name, l, want.qty, want.unit)
		}
	}
	if l := got["Harina"]; l.PackagePrice != 2000 || l.PackageSize != 1000 || l.PricePerUnit != 2 || l.IngredientID == "" {
		t.Errorf("the prices of the ingredient travel with the line: %+v", l)
	}
}

func TestN1_TheFactorScalesEveryQuantity(t *testing.T) {
	f := setup(t)
	f.line(t, "Harina", "gr", 0, 0, 1000, 2000, 200, "gr")
	lines, err := recipeneeds.Load(context.Background(), f.pool, f.recipe, 2.5)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || math.Abs(lines[0].Quantity-500) > 1e-9 {
		t.Errorf("lines = %+v, want 500 gr (200 x 2.5)", lines)
	}
}

func TestN1_AMissingEquivalenceIsAConversionErrorThatNamesTheIngredient(t *testing.T) {
	f := setup(t)
	f.line(t, "Aceite", "lt", 0, 0, 1, 3000, 100, "gr")
	_, err := recipeneeds.Load(context.Background(), f.pool, f.recipe, 1)
	var conv *recipeneeds.ConversionError
	if !errors.As(err, &conv) || conv.Ingredient != "Aceite" {
		t.Fatalf("err = %v, want a ConversionError for Aceite", err)
	}
	var need *kernel.EquivalenceError
	if !errors.As(err, &need) || need.Need != "grams_per_liter" {
		t.Errorf("err = %v, want it to wrap the EquivalenceError that asks for the weight of 1 litre", err)
	}
}

func TestN1_ARecipeWithoutIngredientsGivesNoLines(t *testing.T) {
	f := setup(t)
	lines, err := recipeneeds.Load(context.Background(), f.pool, f.recipe, 1)
	if err != nil || len(lines) != 0 {
		t.Errorf("lines = %v, err = %v; want none", lines, err)
	}
}
