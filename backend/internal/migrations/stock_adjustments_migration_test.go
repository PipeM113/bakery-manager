package migrations_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/PipeM113/bakery-manager/internal/testsupport/pgtest"
)

// Sprint 8, rule A1: migration 21 creates the table of stock adjustments.

const stockAdjustmentsMigration = "000021"

func stockAdjustmentsMigrationExists(t *testing.T) {
	t.Helper()
	if m, _ := filepath.Glob(filepath.Join(migrationsDir, stockAdjustmentsMigration+"_*.up.sql")); len(m) != 1 {
		t.Fatalf("expected exactly one %s_*.up.sql, found %v", stockAdjustmentsMigration, m)
	}
}

func TestA1_TableIsClosedToTheDataAPI(t *testing.T) {
	stockAdjustmentsMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	ctx := context.Background()
	var rls bool
	if err := pool.QueryRow(ctx, `select relrowsecurity from pg_class where relname = 'stock_adjustments' and relnamespace = current_schema()::regnamespace`).Scan(&rls); err != nil {
		t.Fatalf("stock_adjustments: %v", err)
	}
	if !rls {
		t.Error("row level security is off on stock_adjustments")
	}
	var policies int
	if err := pool.QueryRow(ctx, `select count(*) from pg_policies where tablename = 'stock_adjustments'`).Scan(&policies); err != nil || policies != 0 {
		t.Errorf("policies = %d (%v), want none", policies, err)
	}
}

func TestA1_TheTableRefusesWhatTheRulesRefuse(t *testing.T) {
	stockAdjustmentsMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	ctx := context.Background()
	var user, ingredient string
	if err := pool.QueryRow(ctx, `insert into users (name, email, password, role) values ('Owner', 'o@example.test', 'x', 'owner') returning id`).Scan(&user); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `insert into ingredients (name, default_unit, package_size, package_price, price_per_unit, brand, kind)
		values ('Harina', 'gr', 1000, 2000, 2, 'M', 'ingredient') returning id`).Scan(&ingredient); err != nil {
		t.Fatal(err)
	}
	insert := func(reason string, loss int, before, after float64) error {
		_, err := pool.Exec(ctx, `insert into stock_adjustments
			(user_id, ingredient_id, reason, quantity_delta, unit, stock_before, stock_after, loss_amount, note, adjusted_on)
			values ($1, $2, $3, $6::numeric - $5::numeric, 'gr', $5, $6, $4, '', '2026-03-10')`, user, ingredient, reason, loss, before, after)
		return err
	}
	if err := insert("merma", 400, 5000, 4800); err != nil {
		t.Fatalf("a valid merma was refused: %v", err)
	}
	if err := insert("robo", 0, 5000, 4800); err == nil {
		t.Error("an unknown reason was accepted")
	}
	if err := insert("merma", -1, 5000, 4800); err == nil {
		t.Error("a negative loss was accepted")
	}
	if err := insert("correccion", 0, 5000, -1); err == nil {
		t.Error("a negative stock after the adjustment was accepted")
	}
	if err := insert("correccion", 400, 5000, 4800); err == nil {
		t.Error("a loss on something that is not a merma was accepted")
	}
}

func TestA1_DownDropsTheTable(t *testing.T) {
	stockAdjustmentsMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")

	runMigrationFiles(t, pool, "down", func(v string) bool { return v == stockAdjustmentsMigration })

	var left bool
	if err := pool.QueryRow(context.Background(), `select to_regclass('stock_adjustments') is not null`).Scan(&left); err != nil || left {
		t.Errorf("stock_adjustments still exists after down (%v, %v)", left, err)
	}
}
