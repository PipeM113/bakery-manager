package migrations_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/PipeM113/bakery-manager/internal/testsupport/pgtest"
)

// Sprint 10, rule O1: migration 22 adds the orders and the price each sale really had.

const ordersMigration = "000022"

func ordersMigrationExists(t *testing.T) {
	t.Helper()
	if m, _ := filepath.Glob(filepath.Join(migrationsDir, ordersMigration+"_*.up.sql")); len(m) != 1 {
		t.Fatalf("expected exactly one %s_*.up.sql, found %v", ordersMigration, m)
	}
}

func TestO1_OrdersTableIsClosedToTheDataAPI(t *testing.T) {
	ordersMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	ctx := context.Background()
	var rls bool
	if err := pool.QueryRow(ctx, `select relrowsecurity from pg_class where relname = 'orders' and relnamespace = current_schema()::regnamespace`).Scan(&rls); err != nil {
		t.Fatalf("orders: %v", err)
	}
	if !rls {
		t.Error("row level security is off on orders")
	}
	var policies int
	if err := pool.QueryRow(ctx, `select count(*) from pg_policies where tablename = 'orders'`).Scan(&policies); err != nil || policies != 0 {
		t.Errorf("policies = %d (%v), want none", policies, err)
	}
}

func TestO1_ExistingSalesHaveNoOrderAndKeepTheirTotal(t *testing.T) {
	ordersMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	runMigrationFiles(t, pool, "up", func(v string) bool { return v < ordersMigration })
	ctx := context.Background()
	var user, recipe, sale string
	if err := pool.QueryRow(ctx, `insert into users (name, email, password, role) values ('Owner', 'o@example.test', 'x', 'owner') returning id`).Scan(&user); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `insert into recipes (user_id, name, description, yield, yield_unit) values ($1, 'Torta', 'x', 1, 'porciones') returning id`, user).Scan(&recipe); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `insert into sales (user_id, recipe_id, quantity_sold, unit_price, sale_date) values ($1, $2, 3, 1000, '2026-03-01') returning id`, user, recipe).Scan(&sale); err != nil {
		t.Fatal(err)
	}

	runMigrationFiles(t, pool, "up", func(v string) bool { return v == ordersMigration })

	var orderID *string
	var lineTotal int64
	if err := pool.QueryRow(ctx, `select order_id, line_total from sales where id = $1`, sale).Scan(&orderID, &lineTotal); err != nil {
		t.Fatal(err)
	}
	if orderID != nil {
		t.Errorf("order_id = %v, want none", *orderID)
	}
	if lineTotal != 3000 {
		t.Errorf("line_total = %d, want 3000 (3 x 1000)", lineTotal)
	}
	if _, err := pool.Exec(ctx, `insert into sales (user_id, recipe_id, quantity_sold, unit_price, sale_date) values ($1, $2, 1, 1, '2026-03-02')`, user, recipe); err == nil {
		t.Error("a sale without its line_total was accepted")
	}
}

func TestO1_DownRestoresTheSales(t *testing.T) {
	ordersMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")

	runMigrationFiles(t, pool, "down", func(v string) bool { return v == ordersMigration })

	ctx := context.Background()
	var gone bool
	if err := pool.QueryRow(ctx, `select to_regclass('orders') is null`).Scan(&gone); err != nil || !gone {
		t.Errorf("orders still exists after down (%v, %v)", gone, err)
	}
	var left int
	if err := pool.QueryRow(ctx, `select count(*) from information_schema.columns
		where table_schema = current_schema() and table_name = 'sales' and column_name in ('order_id', 'line_total')`).Scan(&left); err != nil || left != 0 {
		t.Errorf("columns left after down = %d (%v), want 0", left, err)
	}
}
