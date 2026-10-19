package migrations_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/PipeM113/bakery-manager/internal/testsupport/pgtest"
)

// Sprint 7: migration 20 adds how a sale was made (whole recipe or portions) and how much stock
// each of its lines really took.

const salesStockMigration = "000020"

func salesStockMigrationExists(t *testing.T) {
	t.Helper()
	if m, _ := filepath.Glob(filepath.Join(migrationsDir, salesStockMigration+"_*.up.sql")); len(m) != 1 {
		t.Fatalf("expected exactly one %s_*.up.sql, found %v", salesStockMigration, m)
	}
}

func TestV3_ExistingSalesAreOfWholeRecipesAndTookWhatTheyUsed(t *testing.T) {
	salesStockMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	runMigrationFiles(t, pool, "up", func(v string) bool { return v < salesStockMigration })
	ctx := context.Background()
	var user, recipe, ingredient, sale string
	if err := pool.QueryRow(ctx, `insert into users (name, email, password, role) values ('Owner', 'o@example.test', 'x', 'owner') returning id`).Scan(&user); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `insert into recipes (user_id, name, description, yield, yield_unit) values ($1, 'Torta', 'x', 1, 'porciones') returning id`, user).Scan(&recipe); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `insert into ingredients (name, default_unit, package_size, package_price, price_per_unit, brand, kind)
		values ('Harina', 'gr', 1000, 2000, 2, 'M', 'ingredient') returning id`).Scan(&ingredient); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `insert into sales (user_id, recipe_id, quantity_sold, unit_price, sale_date) values ($1, $2, 1, 1000, '2026-03-01') returning id`, user, recipe).Scan(&sale); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `insert into sale_ingredients (sale_id, ingredient_id, quantity_used, unit, price_at_time) values ($1, $2, 500, 'gr', 2)`, sale, ingredient); err != nil {
		t.Fatal(err)
	}

	runMigrationFiles(t, pool, "up", func(v string) bool { return v == salesStockMigration })

	var soldAs string
	var deducted float64
	if err := pool.QueryRow(ctx, `select sold_as from sales where id = $1`, sale).Scan(&soldAs); err != nil || soldAs != "recipe" {
		t.Errorf("sold_as = %q (%v), want recipe", soldAs, err)
	}
	if err := pool.QueryRow(ctx, `select stock_deducted from sale_ingredients where sale_id = $1`, sale).Scan(&deducted); err != nil || deducted != 500 {
		t.Errorf("stock_deducted = %v (%v), want 500 (what the line used)", deducted, err)
	}
	if _, err := pool.Exec(ctx, `update sales set sold_as = 'kilo' where id = $1`, sale); err == nil {
		t.Error("sold_as accepted a value that is neither recipe nor portion")
	}
}

func TestV3_DownRemovesTheColumns(t *testing.T) {
	salesStockMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")

	runMigrationFiles(t, pool, "down", func(v string) bool { return v == salesStockMigration })

	var left int
	if err := pool.QueryRow(context.Background(), `select count(*) from information_schema.columns
		where table_schema = current_schema() and ((table_name = 'sales' and column_name = 'sold_as')
		   or (table_name = 'sale_ingredients' and column_name = 'stock_deducted'))`).Scan(&left); err != nil || left != 0 {
		t.Errorf("columns left after down = %d (%v), want 0", left, err)
	}
}
