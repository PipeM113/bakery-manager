package migrations_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PipeM113/bakery-manager/internal/testsupport/pgtest"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ADR-005, rules M1-M6: migration 31 makes the key name + brand (merging what the old key kept apart), adds the
// detail of a purchase to stock_adjustments, and stops with a message when a merge needs a person to decide.

const mergeMigration = "000031"

func mergeMigrationExists(t *testing.T) {
	t.Helper()
	if m, _ := filepath.Glob(filepath.Join(migrationsDir, mergeMigration+"_*.up.sql")); len(m) != 1 {
		t.Fatalf("expected exactly one %s_*.up.sql, found %v", mergeMigration, m)
	}
}

// stage applies everything before migration 31 and hands the pool over.
func stage(t *testing.T) *pgxpool.Pool {
	t.Helper()
	mergeMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	runMigrationFiles(t, pool, "up", func(v string) bool { return v < mergeMigration })
	return pool
}

func up31(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	runMigrationFiles(t, pool, "up", func(v string) bool { return v == mergeMigration })
}

// runMigration applies migration 31 and returns its error instead of failing the test.
func runMigration(t *testing.T, pool *pgxpool.Pool) error {
	t.Helper()
	files, _ := filepath.Glob(filepath.Join(migrationsDir, mergeMigration+"_*.up.sql"))
	sql, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	_, err = conn.Conn().PgConn().Exec(context.Background(), string(sql)).ReadAll()
	return err
}

func exec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

func count(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return n
}

// butter inserts one ingredient of the usual shape and returns its id.
func butter(t *testing.T, pool *pgxpool.Pool, supplier string, stock, threshold float64, extra string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		insert into ingredients (name, brand, supplier, default_unit, price_per_unit, package_size, package_price, stock_quantity, alert_threshold`+extra+`)
		values ('Mantequilla', 'Soprole', $1, 'gr', 4, 250, 1000, $2, $3) returning id`, supplier, stock, threshold).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestM1_WithoutDuplicatesNothingChangesAndTheKeyIsNameAndBrand(t *testing.T) {
	pool := stage(t)
	butter(t, pool, "La Vega", 100, 10, "")
	exec(t, pool, `insert into ingredients (name, brand, supplier, default_unit, price_per_unit, package_size, package_price) values ('Mantequilla', 'Colún', 'La Vega', 'gr', 4, 250, 1000)`)
	up31(t, pool)
	if n := count(t, pool, `select count(*) from ingredients`); n != 2 {
		t.Errorf("ingredients = %d, want 2 (nothing to merge)", n)
	}
	_, err := pool.Exec(context.Background(), `insert into ingredients (name, brand, supplier, default_unit, price_per_unit, package_size, package_price) values ('MANTEQUILLA', 'soprole', 'Líder', 'gr', 4, 250, 1000)`)
	if err == nil || !strings.Contains(err.Error(), "23505") {
		t.Errorf("the same name and brand with another supplier must be a unique violation, got %v", err)
	}
}

func TestM2_TheDetailOfAPurchaseHasItsColumnsAndOnlyForPurchases(t *testing.T) {
	pool := stage(t)
	id := butter(t, pool, "", 0, 0, "")
	up31(t, pool)
	for _, col := range []string{"supplier", "package_size", "package_price"} {
		if columnCount(t, pool, "stock_adjustments", col) != 1 {
			t.Errorf("stock_adjustments.%s is missing", col)
		}
	}
	ok := `insert into stock_adjustments (ingredient_id, reason, quantity_delta, unit, stock_before, stock_after, adjusted_on, supplier, package_size, package_price)
	       values ($1, $2, 10, 'gr', 0, 10, '2026-03-01', 'Súper', 250, 1300)`
	exec(t, pool, ok, id, "compra")
	if _, err := pool.Exec(context.Background(), ok, id, "merma"); err == nil {
		t.Error("a loss with the data of a purchase must be refused by the schema")
	}
}

func TestM3_TwoRowsOfTheSameNameAndBrandBecomeOneWithTheirStockAndHistory(t *testing.T) {
	pool := stage(t)
	older := butter(t, pool, "La Vega", 100, 10, "")
	newer := butter(t, pool, "Líder", 50, 30, "")
	exec(t, pool, `insert into ingredient_price_history (ingredient_id, price_per_unit, unit, effective_date) values ($1, 4, 'gr', '2026-01-01'), ($2, 5, 'gr', '2026-02-01')`, older, newer)
	exec(t, pool, `update ingredients set package_price = 1300, price_per_unit = 5.2, package_size = 250 where id = $1`, newer)
	exec(t, pool, `insert into recipes (name, description, yield, yield_unit) values ('Pie', 'x', 8, 'porciones')`)
	exec(t, pool, `insert into recipe_ingredients (recipe_id, ingredient_id, quantity, unit) select id, $1, 200, 'gr' from recipes`, older)
	exec(t, pool, `insert into recipe_ingredients (recipe_id, ingredient_id, quantity, unit) select id, $1, 100, 'gr' from recipes`, newer)
	exec(t, pool, `insert into stock_adjustments (ingredient_id, reason, quantity_delta, unit, stock_before, stock_after, adjusted_on) values ($1, 'compra', 100, 'gr', 0, 100, '2026-01-02')`, older)
	exec(t, pool, `insert into sales (recipe_id, quantity_sold, unit_price, line_total) select id, 1, 1000, 1000 from recipes`)
	exec(t, pool, `insert into sale_ingredients (sale_id, ingredient_id, quantity_used, unit, price_at_time, stock_deducted) select id, $1, 200, 'gr', 4, 200 from sales`, older)

	up31(t, pool)

	if n := count(t, pool, `select count(*) from ingredients`); n != 1 {
		t.Fatalf("ingredients = %d, want 1", n)
	}
	var supplier string
	var stock, threshold, price float64
	var id string
	if err := pool.QueryRow(context.Background(), `select id, supplier, stock_quantity, alert_threshold, package_price from ingredients`).Scan(&id, &supplier, &stock, &threshold, &price); err != nil {
		t.Fatal(err)
	}
	if supplier != "Líder" || price != 1300 {
		t.Errorf("kept %q at $%v, want the row with the most recent price (Líder at $1300)", supplier, price)
	}
	if stock != 150 || threshold != 30 {
		t.Errorf("stock %v and threshold %v, want 150 (the sum) and 30 (the larger)", stock, threshold)
	}
	for name, sql := range map[string]string{
		"recipe lines":  `select count(*) from recipe_ingredients where ingredient_id = $1`,
		"price history": `select count(*) from ingredient_price_history where ingredient_id = $1`,
		"adjustments":   `select count(*) from stock_adjustments where ingredient_id = $1`,
		"sale lines":    `select count(*) from sale_ingredients where ingredient_id = $1`,
	} {
		want := map[string]int{"recipe lines": 2, "price history": 2, "adjustments": 1, "sale lines": 1}[name]
		if got := count(t, pool, sql, id); got != want {
			t.Errorf("%s repointed to the kept row = %d, want %d", name, got, want)
		}
	}
}

func TestM4_TheEquivalencesOfTheKeptRowWinAndTheOtherFillsWhatIsMissing(t *testing.T) {
	pool := stage(t)
	a := butter(t, pool, "La Vega", 0, 0, "")
	b := butter(t, pool, "Líder", 0, 0, "")
	exec(t, pool, `update ingredients set grams_per_liter = 920 where id = $1`, a)
	exec(t, pool, `update ingredients set grams_per_unit = 250 where id = $1`, b)
	exec(t, pool, `insert into ingredient_price_history (ingredient_id, price_per_unit, unit, effective_date) values ($1, 4, 'gr', '2026-01-01'), ($2, 5, 'gr', '2026-02-01')`, a, b)
	up31(t, pool)
	var perLiter, perUnit float64
	if err := pool.QueryRow(context.Background(), `select grams_per_liter, grams_per_unit from ingredients`).Scan(&perLiter, &perUnit); err != nil {
		t.Fatal(err)
	}
	if perLiter != 920 || perUnit != 250 {
		t.Errorf("equivalences %v / %v, want 920 (from one) and 250 (from the other)", perLiter, perUnit)
	}
}

func TestM5_ADifferenceThatNeedsAPersonStopsTheMigrationAndChangesNothing(t *testing.T) {
	cases := map[string]string{
		"different unit":             `update ingredients set default_unit = 'kg' where supplier = 'Líder'`,
		"different kind":             `update ingredients set kind = 'supply' where supplier = 'Líder'`,
		"two different equivalences": `update ingredients set grams_per_liter = 920 where supplier = 'La Vega'; update ingredients set grams_per_liter = 1030 where supplier = 'Líder'`,
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			pool := stage(t)
			butter(t, pool, "La Vega", 100, 0, "")
			butter(t, pool, "Líder", 50, 0, "")
			exec(t, pool, change)
			err := runMigration(t, pool)
			if err == nil || !strings.Contains(err.Error(), "Mantequilla") {
				t.Fatalf("the migration must stop and name the ingredient, got %v", err)
			}
			if n := count(t, pool, `select count(*) from ingredients`); n != 2 {
				t.Errorf("ingredients = %d after the refusal, want 2 (nothing changed)", n)
			}
			if columnCount(t, pool, "stock_adjustments", "package_price") != 0 {
				t.Error("the refused migration left its columns behind")
			}
		})
	}
}

func TestM6_DownRestoresTheOldKeyAndDropsTheColumnsButCannotSplitAMerge(t *testing.T) {
	pool := stage(t)
	butter(t, pool, "La Vega", 100, 0, "")
	butter(t, pool, "Líder", 50, 0, "")
	exec(t, pool, `insert into ingredient_price_history (ingredient_id, price_per_unit, unit, effective_date) select id, 4, 'gr', '2026-01-01' from ingredients`)
	up31(t, pool)
	runMigrationFiles(t, pool, "down", func(v string) bool { return v == mergeMigration })
	for _, col := range []string{"supplier", "package_size", "package_price"} {
		if columnCount(t, pool, "stock_adjustments", col) != 0 {
			t.Errorf("down left stock_adjustments.%s", col)
		}
	}
	if n := count(t, pool, `select count(*) from ingredients`); n != 1 {
		t.Errorf("ingredients = %d, want 1: the merge is not undone", n)
	}
	exec(t, pool, `insert into ingredients (name, brand, supplier, default_unit, price_per_unit, package_size, package_price) values ('Mantequilla', 'Soprole', 'Jumbo', 'gr', 4, 250, 1000)`)
}
