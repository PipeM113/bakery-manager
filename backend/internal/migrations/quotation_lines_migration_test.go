package migrations_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/PipeM113/bakery-manager/internal/testsupport/pgtest"
)

// Sprint 12, rule Q7: migration 24 gives a quotation several lines. The quotations that already
// exist become quotations of one line and nothing is lost.

const quotationLinesMigration = "000024"

func quotationLinesMigrationExists(t *testing.T) {
	t.Helper()
	if m, _ := filepath.Glob(filepath.Join(migrationsDir, quotationLinesMigration+"_*.up.sql")); len(m) != 1 {
		t.Fatalf("expected exactly one %s_*.up.sql, found %v", quotationLinesMigration, m)
	}
}

func TestQ7_LinesTableAndColumns(t *testing.T) {
	quotationLinesMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	for _, col := range []string{"quotation_id", "position", "recipe_id", "recipe_name", "photo_url", "sold_as", "quantity",
		"unit_price", "line_total", "margin_pct", "indirect_cost_pct", "labor_cost_pct", "extra_charge"} {
		if columnCount(t, pool, "quotation_lines", col) != 1 {
			t.Errorf("quotation_lines has no %s", col)
		}
	}
	if columnCount(t, pool, "quotations", "delivery_cost") != 1 {
		t.Error("quotations has no delivery_cost")
	}
	var nullable string
	if err := pool.QueryRow(context.Background(), `select is_nullable from information_schema.columns
		where table_schema = current_schema() and table_name = 'quotations' and column_name = 'recipe_id'`).Scan(&nullable); err != nil {
		t.Fatal(err)
	}
	if nullable != "YES" {
		t.Error("quotations.recipe_id must be optional: the recipes now live in the lines")
	}
	var rls bool
	if err := pool.QueryRow(context.Background(), `select relrowsecurity from pg_class
		where relname = 'quotation_lines' and relnamespace = current_schema()::regnamespace`).Scan(&rls); err != nil || !rls {
		t.Errorf("quotation_lines must have row level security on (%v, %v)", rls, err)
	}
}

func TestQ7_ExistingQuotationsBecomeOneLine(t *testing.T) {
	quotationLinesMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	runMigrationFiles(t, pool, "up", func(v string) bool { return v < quotationLinesMigration })
	ctx := context.Background()
	var user, recipe, quotation string
	if err := pool.QueryRow(ctx, `insert into users (name, email, password, role) values ('Owner', 'o@example.test', 'x', 'owner') returning id`).Scan(&user); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `insert into recipes (created_by, name, description, yield, yield_unit, photo_url) values ($1, 'Torta', 'x', 1, 'porciones', 'https://example.test/t.jpg') returning id`, user).Scan(&recipe); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `insert into quotations (created_by, recipe_id, client_name, margin_pct, final_price, status)
		values ($1, $2, 'Ana', 0.3, 4000, 'confirmed') returning id`, user, recipe).Scan(&quotation); err != nil {
		t.Fatal(err)
	}

	runMigrationFiles(t, pool, "up", func(v string) bool { return v == quotationLinesMigration })

	var n int
	var name, photo, soldAs string
	var qty, unit, total int64
	var margin float64
	var line string
	if err := pool.QueryRow(ctx, `select count(*) from quotation_lines where quotation_id = $1`, quotation).Scan(&n); err != nil || n != 1 {
		t.Fatalf("lines = %d (%v), want 1", n, err)
	}
	if err := pool.QueryRow(ctx, `select recipe_id::text, recipe_name, coalesce(photo_url, ''), sold_as, quantity, unit_price, line_total, margin_pct
		from quotation_lines where quotation_id = $1`, quotation).Scan(&line, &name, &photo, &soldAs, &qty, &unit, &total, &margin); err != nil {
		t.Fatal(err)
	}
	if line != recipe || name != "Torta" || photo != "https://example.test/t.jpg" || soldAs != "recipe" || qty != 1 || unit != 4000 || total != 4000 || margin != 0.3 {
		t.Errorf("line = %s %q %q %s %d %d %d %v, want the old quotation as one line of 4000", line, name, photo, soldAs, qty, unit, total, margin)
	}
	var delivery, final int64
	var status string
	if err := pool.QueryRow(ctx, `select delivery_cost, final_price, status from quotations where id = $1`, quotation).Scan(&delivery, &final, &status); err != nil {
		t.Fatal(err)
	}
	if delivery != 0 || final != 4000 || status != "confirmed" {
		t.Errorf("quotation = delivery %d, final %d, %s; want 0, 4000, confirmed (the old total keeps its meaning)", delivery, final, status)
	}
}

func TestQ7_DeletingAQuotationDeletesItsLines(t *testing.T) {
	quotationLinesMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	ctx := context.Background()
	var recipe, quotation string
	if err := pool.QueryRow(ctx, `insert into recipes (name, description, yield, yield_unit) values ('Torta', 'x', 1, 'porciones') returning id`).Scan(&recipe); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `insert into quotations (client_name, final_price) values ('Ana', 1000) returning id`).Scan(&quotation); err != nil {
		t.Fatalf("a quotation can be saved without recipe_id: %v", err)
	}
	if _, err := pool.Exec(ctx, `insert into quotation_lines (quotation_id, position, recipe_id, recipe_name, sold_as, quantity, unit_price, line_total)
		values ($1, 1, $2, 'Torta', 'recipe', 1, 1000, 1000)`, quotation, recipe); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `delete from quotations where id = $1`, quotation); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx, `select count(*) from quotation_lines`).Scan(&n); err != nil || n != 0 {
		t.Errorf("lines left = %d (%v), want 0", n, err)
	}
}

func TestQ7_DownGoesBackToOneRecipePerQuotation(t *testing.T) {
	quotationLinesMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	ctx := context.Background()
	var cake, pie, quotation string
	if err := pool.QueryRow(ctx, `insert into recipes (name, description, yield, yield_unit) values ('Torta', 'x', 1, 'porciones') returning id`).Scan(&cake); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `insert into recipes (name, description, yield, yield_unit) values ('Pie', 'x', 1, 'porciones') returning id`).Scan(&pie); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `insert into quotations (client_name, final_price, delivery_cost) values ('Ana', 5500, 500) returning id`).Scan(&quotation); err != nil {
		t.Fatal(err)
	}
	for i, r := range []string{cake, pie} {
		if _, err := pool.Exec(ctx, `insert into quotation_lines (quotation_id, position, recipe_id, recipe_name, sold_as, quantity, unit_price, line_total)
			values ($1, $2, $3, 'x', 'recipe', 1, 2500, 2500)`, quotation, i+1, r); err != nil {
			t.Fatal(err)
		}
	}

	runMigrationFiles(t, pool, "down", func(v string) bool { return v == quotationLinesMigration })

	if columnCount(t, pool, "quotations", "delivery_cost") != 0 {
		t.Error("delivery_cost is still there")
	}
	var tableCount int
	if err := pool.QueryRow(ctx, `select count(*) from information_schema.tables where table_schema = current_schema() and table_name = 'quotation_lines'`).Scan(&tableCount); err != nil || tableCount != 0 {
		t.Errorf("quotation_lines is still there (%v)", err)
	}
	var recipe string
	var final int64
	if err := pool.QueryRow(ctx, `select recipe_id::text, final_price from quotations where id = $1`, quotation).Scan(&recipe, &final); err != nil {
		t.Fatal(err)
	}
	if recipe != cake || final != 5500 {
		t.Errorf("quotation = recipe %s, final %d; want the first line's recipe and the same total", recipe, final)
	}
	var nullable string
	if err := pool.QueryRow(ctx, `select is_nullable from information_schema.columns
		where table_schema = current_schema() and table_name = 'quotations' and column_name = 'recipe_id'`).Scan(&nullable); err != nil || nullable != "NO" {
		t.Errorf("recipe_id nullable = %q (%v), want NO again", nullable, err)
	}
}
