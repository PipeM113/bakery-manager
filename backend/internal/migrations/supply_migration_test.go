package migrations_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/PipeM113/bakery-manager/internal/testsupport/pgtest"
)

// Sprint 3, rule I1: ingredients get a kind, 'ingredient' or 'supply'. Migration 17 alone is
// applied on top of a database that already holds data.

const supplyMigration = "000017"

func supplyMigrationExists(t *testing.T) {
	t.Helper()
	if m, _ := filepath.Glob(filepath.Join(migrationsDir, supplyMigration+"_*.up.sql")); len(m) != 1 {
		t.Fatalf("expected exactly one %s_*.up.sql, found %v", supplyMigration, m)
	}
}

func TestI1_ExistingIngredientsBecomeIngredients(t *testing.T) {
	supplyMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	runMigrationFiles(t, pool, "up", func(v string) bool { return v < supplyMigration })
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `insert into ingredients (name, default_unit, package_size, package_price, price_per_unit, brand)
		values ('Harina', 'gr', 1000, 2000, 2, 'Marca')`); err != nil {
		t.Fatal(err)
	}

	runMigrationFiles(t, pool, "up", func(v string) bool { return v == supplyMigration })

	var kind string
	if err := pool.QueryRow(ctx, `select kind from ingredients where name = 'Harina'`).Scan(&kind); err != nil {
		t.Fatal(err)
	}
	if kind != "ingredient" {
		t.Errorf("existing ingredient has kind %q, want ingredient", kind)
	}
}

func TestI1_OnlyTwoKindsAreAllowed(t *testing.T) {
	supplyMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	ctx := context.Background()

	insert := func(kind string) error {
		_, err := pool.Exec(ctx, `insert into ingredients (name, default_unit, package_size, package_price, price_per_unit, brand, kind)
			values ('x-'||$1, 'und', 1, 300, 300, 'M', $1)`, kind)
		return err
	}
	if err := insert("supply"); err != nil {
		t.Errorf("kind supply should be accepted: %v", err)
	}
	if err := insert("caja"); err == nil {
		t.Error("kind caja should be refused")
	}
}

func TestI1_DownRemovesTheKind(t *testing.T) {
	supplyMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")

	runMigrationFiles(t, pool, "down", func(v string) bool { return v == supplyMigration })

	var n int
	err := pool.QueryRow(context.Background(), `select count(*) from information_schema.columns
		where table_schema = current_schema() and table_name = 'ingredients' and column_name = 'kind'`).Scan(&n)
	if err != nil || n != 0 {
		t.Errorf("column kind after down: count = %d (%v), want 0", n, err)
	}
}
