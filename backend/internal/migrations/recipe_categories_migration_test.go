package migrations_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/PipeM113/bakery-manager/internal/testsupport/pgtest"
)

// Sprint 16, rule T6: migration 29 adds the types of recipe and the optional type of a recipe.

const recipeCategoriesMigration = "000029"

func recipeCategoriesMigrationExists(t *testing.T) {
	t.Helper()
	if m, _ := filepath.Glob(filepath.Join(migrationsDir, recipeCategoriesMigration+"_*.up.sql")); len(m) != 1 {
		t.Fatalf("expected exactly one %s_*.up.sql, found %v", recipeCategoriesMigration, m)
	}
}

func TestT6_TheTableAndTheColumnExistAndAreProtected(t *testing.T) {
	recipeCategoriesMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	ctx := context.Background()
	var rls bool
	if err := pool.QueryRow(ctx, `select relrowsecurity from pg_class where relname = 'recipe_categories' and relnamespace = current_schema()::regnamespace`).Scan(&rls); err != nil || !rls {
		t.Errorf("recipe_categories must exist with row level security on (%v, %v)", rls, err)
	}
	var nullable string
	if err := pool.QueryRow(ctx, `select is_nullable from information_schema.columns where table_schema = current_schema() and table_name = 'recipes' and column_name = 'category_id'`).Scan(&nullable); err != nil || nullable != "YES" {
		t.Errorf("recipes.category_id must exist and be optional (%q, %v)", nullable, err)
	}
	if _, err := pool.Exec(ctx, `insert into recipe_categories (name) values ('Hojarasca')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `insert into recipe_categories (name) values ('HOJARASCA')`); err == nil {
		t.Error("two types that only differ in case must collide")
	}
	if _, err := pool.Exec(ctx, `insert into recipe_categories (name) values ('   ')`); err == nil {
		t.Error("a name of spaces must be refused")
	}
}

func TestT6_DeletingATypeKeepsItsRecipes(t *testing.T) {
	recipeCategoriesMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	ctx := context.Background()
	var cat, recipe string
	if err := pool.QueryRow(ctx, `insert into recipe_categories (name) values ('Merengue') returning id`).Scan(&cat); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `insert into recipes (name, description, yield, yield_unit, category_id) values ('Pavlova', 'x', 1, 'porciones', $1) returning id`, cat).Scan(&recipe); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `delete from recipe_categories where id = $1`, cat); err != nil {
		t.Fatal(err)
	}
	var got *string
	if err := pool.QueryRow(ctx, `select category_id::text from recipes where id = $1`, recipe).Scan(&got); err != nil || got != nil {
		t.Errorf("category_id = %v (%v), want the recipe kept without type", got, err)
	}
}

func TestT6_ExistingRecipesStartWithoutType(t *testing.T) {
	recipeCategoriesMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	runMigrationFiles(t, pool, "up", func(v string) bool { return v < recipeCategoriesMigration })
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `insert into recipes (name, description, yield, yield_unit) values ('Pan', 'x', 1, 'porciones')`); err != nil {
		t.Fatal(err)
	}
	runMigrationFiles(t, pool, "up", func(v string) bool { return v == recipeCategoriesMigration })
	var got *string
	if err := pool.QueryRow(ctx, `select category_id::text from recipes`).Scan(&got); err != nil || got != nil {
		t.Errorf("category_id = %v (%v), want none", got, err)
	}
}

func TestT6_DownRemovesBothAndKeepsTheRecipes(t *testing.T) {
	recipeCategoriesMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `insert into recipes (name, description, yield, yield_unit) values ('Pan', 'x', 1, 'porciones')`); err != nil {
		t.Fatal(err)
	}
	runMigrationFiles(t, pool, "down", func(v string) bool { return v == recipeCategoriesMigration })
	if columnCount(t, pool, "recipes", "category_id") != 0 {
		t.Error("down left recipes.category_id")
	}
	var tables int
	if err := pool.QueryRow(ctx, `select count(*) from information_schema.tables where table_schema = current_schema() and table_name = 'recipe_categories'`).Scan(&tables); err != nil || tables != 0 {
		t.Errorf("down left recipe_categories (%d, %v)", tables, err)
	}
	var n int
	if err := pool.QueryRow(ctx, `select count(*) from recipes`).Scan(&n); err != nil || n != 1 {
		t.Errorf("down lost the recipes (%d, %v)", n, err)
	}
}
