package migrations_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PipeM113/bakery-manager/internal/testsupport/pgtest"
)

// Sprint 14, rules E1, E2, E6, E9: migration 28 adds the supplier and the two equivalences to the
// ingredients, makes name + brand + supplier the key and lets a stock adjustment be a purchase.

const ingredientKeysMigration = "000028"

func ingredientKeysMigrationExists(t *testing.T) {
	t.Helper()
	if m, _ := filepath.Glob(filepath.Join(migrationsDir, ingredientKeysMigration+"_*.up.sql")); len(m) != 1 {
		t.Fatalf("expected exactly one %s_*.up.sql, found %v", ingredientKeysMigration, m)
	}
}

const insertIngredient = `insert into ingredients (name, brand, unit, package_size, package_price) values ($1, $2, 'gr', 1000, 1000)`

func TestE1_TheNewColumnsExistAndDefaultToNothing(t *testing.T) {
	ingredientKeysMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	runMigrationFiles(t, pool, "up", func(v string) bool { return v < ingredientKeysMigration })
	ctx := context.Background()
	if _, err := pool.Exec(ctx, insertIngredient, "Harina", "Selecta"); err != nil {
		t.Fatal(err)
	}
	runMigrationFiles(t, pool, "up", func(v string) bool { return v == ingredientKeysMigration })
	var supplier string
	var perLiter, perUnit float64
	if err := pool.QueryRow(ctx, `select supplier, grams_per_liter, grams_per_unit from ingredients`).Scan(&supplier, &perLiter, &perUnit); err != nil {
		t.Fatalf("the new columns are missing: %v", err)
	}
	if supplier != "" || perLiter != 0 || perUnit != 0 {
		t.Errorf("existing rows got %q/%v/%v, want empty/0/0", supplier, perLiter, perUnit)
	}
}

func TestE1_TheKeyIsNameBrandAndSupplierWithoutCase(t *testing.T) {
	ingredientKeysMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	ctx := context.Background()
	add := func(name, brand, supplier string) error {
		_, err := pool.Exec(ctx, `insert into ingredients (name, brand, supplier, unit, package_size, package_price) values ($1, $2, $3, 'gr', 250, 1000)`, name, brand, supplier)
		return err
	}
	for _, c := range [][3]string{{"Mantequilla", "Soprole", "La Vega"}, {"Mantequilla", "Soprole", "Líder"}, {"Mantequilla", "Colún", "La Vega"}} {
		if err := add(c[0], c[1], c[2]); err != nil {
			t.Fatalf("%v should be allowed: %v", c, err)
		}
	}
	err := add("MANTEQUILLA", "soprole", "la vega")
	if err == nil || !strings.Contains(err.Error(), "23505") {
		t.Errorf("the same three with other case must be a unique violation, got %v", err)
	}
	if err := add("Mantequilla", "", ""); err != nil {
		t.Errorf("without brand and supplier is its own ingredient: %v", err)
	}
	if err := add("Mantequilla", "", ""); err == nil {
		t.Error("two with empty brand and supplier must collide")
	}
}

func TestE9_AStockAdjustmentCanBeAPurchase(t *testing.T) {
	ingredientKeysMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	ctx := context.Background()
	var ing string
	if err := pool.QueryRow(ctx, insertIngredient+` returning id`, "Harina", "").Scan(&ing); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `insert into stock_adjustments (ingredient_id, reason, quantity_delta, unit, adjusted_on) values ($1, 'compra', 500, 'gr', '2026-03-01')`, ing); err != nil {
		t.Fatalf("reason compra refused: %v", err)
	}
	if _, err := pool.Exec(ctx, `insert into stock_adjustments (ingredient_id, reason, quantity_delta, unit, adjusted_on) values ($1, 'robo', 1, 'gr', '2026-03-01')`, ing); err == nil {
		t.Error("an unknown reason must still be refused")
	}
}

func TestE1_DownGoesBackWhenNothingCollides(t *testing.T) {
	ingredientKeysMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	runMigrationFiles(t, pool, "down", func(v string) bool { return v == ingredientKeysMigration })
	for _, col := range []string{"supplier", "grams_per_liter", "grams_per_unit"} {
		if columnCount(t, pool, "ingredients", col) != 0 {
			t.Errorf("down left %s", col)
		}
	}
	ctx := context.Background()
	if _, err := pool.Exec(ctx, insertIngredient, "Harina", "A"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, insertIngredient, "Harina", "B"); err == nil {
		t.Error("after down the name is unique again")
	}
}
