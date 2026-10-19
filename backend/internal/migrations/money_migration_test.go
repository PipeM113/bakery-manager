package migrations_test

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/PipeM113/bakery-manager/internal/testsupport/pgtest"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Sprint 2b, rules D1, D3 and D4: migration 16 turns money into whole pesos and replaces the
// free-text expense category with a table. These tests stage the database at version 15, put
// data in it the way the old program did, and then apply migration 16 alone.

const moneyMigration = "000016"

func runMigrationFiles(t *testing.T, pool *pgxpool.Pool, direction string, keep func(version string) bool) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*."+direction+".sql"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	if direction == "down" {
		for i, j := 0, len(files)-1; i < j; i, j = i+1, j-1 {
			files[i], files[j] = files[j], files[i]
		}
	}
	ctx := context.Background()
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	for _, f := range files {
		version := strings.SplitN(filepath.Base(f), "_", 2)[0]
		if !keep(version) {
			continue
		}
		sql, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := conn.Conn().PgConn().Exec(ctx, string(sql)).ReadAll(); err != nil {
			t.Fatalf("migration %s (%s): %v", filepath.Base(f), direction, err)
		}
	}
}

func migrationFileExists(t *testing.T) {
	t.Helper()
	if m, _ := filepath.Glob(filepath.Join(migrationsDir, moneyMigration+"_*.up.sql")); len(m) != 1 {
		t.Fatalf("expected exactly one %s_*.up.sql, found %v", moneyMigration, m)
	}
}

// stageLegacy applies migrations 1 to 15 and inserts one of each money row, with decimals.
func stageLegacy(t *testing.T) *pgxpool.Pool {
	t.Helper()
	migrationFileExists(t)
	pool := pgtest.NewDatabase(t)
	runMigrationFiles(t, pool, "up", func(v string) bool { return v < moneyMigration })
	ctx := context.Background()
	stmts := []string{
		`insert into users (id, name, email, password, role) values ('00000000-0000-0000-0000-0000000000a1', 'Owner', 'o@example.test', 'x', 'owner')`,
		`insert into recipes (id, user_id, name, description) values ('00000000-0000-0000-0000-0000000000b1', '00000000-0000-0000-0000-0000000000a1', 'Torta', 'd')`,
		`insert into sales (user_id, recipe_id, quantity_sold, unit_price) values ('00000000-0000-0000-0000-0000000000a1', '00000000-0000-0000-0000-0000000000b1', 2, 3500.50)`,
		`insert into fixed_costs (user_id, name, monthly_amount) values ('00000000-0000-0000-0000-0000000000a1', 'Arriendo', 250000.40)`,
		`insert into quotations (user_id, recipe_id, client_name, margin_pct, final_price) values ('00000000-0000-0000-0000-0000000000a1', '00000000-0000-0000-0000-0000000000b1', 'Ana', 0.3, 2000.00)`,
		`insert into operational_expenses (user_id, description, amount, category, expense_date) values
			('00000000-0000-0000-0000-0000000000a1', 'Harina', 999.49, 'ingredientes', '2026-03-10'),
			('00000000-0000-0000-0000-0000000000a1', 'Gas', 15000.00, 'servicios', '2026-03-11'),
			('00000000-0000-0000-0000-0000000000a1', 'Cajas', 4000.50, 'otros', '2026-03-12')`,
	}
	for _, s := range stmts {
		if _, err := pool.Exec(ctx, s); err != nil {
			t.Fatalf("stage legacy data: %v\n%s", err, s)
		}
	}
	return pool
}

func applyMoneyMigration(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	runMigrationFiles(t, pool, "up", func(v string) bool { return v == moneyMigration })
}

// D1: the four money columns are whole pesos.
func TestD1_MoneyColumnsAreIntegers(t *testing.T) {
	pool := pgtest.NewDatabase(t)
	migrationFileExists(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	for _, c := range [][2]string{
		{"sales", "unit_price"}, {"fixed_cost_versions", "monthly_amount"},
		{"operational_expenses", "amount"}, {"quotations", "final_price"},
	} {
		var typ string
		err := pool.QueryRow(context.Background(), `
			select data_type from information_schema.columns
			where table_schema = current_schema() and table_name = $1 and column_name = $2`, c[0], c[1]).Scan(&typ)
		if err != nil {
			t.Fatal(err)
		}
		if typ != "integer" {
			t.Errorf("%s.%s is %s, want integer", c[0], c[1], typ)
		}
	}
}

// D1: existing amounts are rounded to the nearest peso (half up), not lost.
func TestD1_ExistingAmountsAreRounded(t *testing.T) {
	pool := stageLegacy(t)
	applyMoneyMigration(t, pool)
	ctx := context.Background()

	check := func(query string, want int) {
		t.Helper()
		var got int
		if err := pool.QueryRow(ctx, query).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("%s = %d, want %d", query, got, want)
		}
	}
	check(`select unit_price from sales`, 3501)
	check(`select monthly_amount from fixed_costs`, 250000)
	check(`select final_price from quotations`, 2000)
	check(`select amount from operational_expenses where description = 'Harina'`, 999)
	check(`select amount from operational_expenses where description = 'Cajas'`, 4001)
}

// D4: every existing expense keeps its category, now as a row of expense_categories; the
// categories already in use survive even when they are not in the starting list.
func TestD4_ExistingExpensesKeepTheirCategory(t *testing.T) {
	pool := stageLegacy(t)
	applyMoneyMigration(t, pool)
	ctx := context.Background()

	rows, err := pool.Query(ctx, `
		select e.description, c.name
		from operational_expenses e join expense_categories c on c.id = e.category_id
		order by e.description`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	got := map[string]string{}
	for rows.Next() {
		var d, c string
		if err := rows.Scan(&d, &c); err != nil {
			t.Fatal(err)
		}
		got[d] = c
	}
	want := map[string]string{"Harina": "ingredientes", "Gas": "servicios", "Cajas": "otros"}
	if len(got) != 3 {
		t.Fatalf("expenses with a category = %v, want 3", got)
	}
	for d, c := range want {
		if !strings.EqualFold(got[d], c) {
			t.Errorf("expense %q is in category %q, want %q", d, got[d], c)
		}
	}

	// The starting list is there, and "otros" was not duplicated by the legacy value.
	var n int
	if err := pool.QueryRow(ctx, `select count(*) from expense_categories where lower(name) = 'otros'`).Scan(&n); err != nil || n != 1 {
		t.Errorf("categories named otros = %d (%v), want 1", n, err)
	}
	for _, name := range []string{"gas", "luz", "agua", "arriendo", "envases", "transporte", "sueldos", "otros"} {
		if err := pool.QueryRow(ctx, `select count(*) from expense_categories where lower(name) = $1`, name).Scan(&n); err != nil || n != 1 {
			t.Errorf("starting category %q: count = %d (%v), want 1", name, n, err)
		}
	}
}

// D3: category names are unique ignoring case, and the table has row level security like
// every other table of the schema.
func TestD3_CategoryTableRules(t *testing.T) {
	pool := pgtest.NewDatabase(t)
	migrationFileExists(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `insert into expense_categories (name) values ('GAS')`); err == nil {
		t.Error("a second category whose name differs only in case should be refused")
	}
	var rls bool
	err := pool.QueryRow(ctx, `select relrowsecurity from pg_class where oid = 'expense_categories'::regclass`).Scan(&rls)
	if err != nil {
		t.Fatal(err)
	}
	if !rls {
		t.Error("expense_categories should have row level security enabled")
	}
}

// D1/D4: the down migration restores the old shape. A category the old check did not allow
// goes back as 'otros'; the three old ones keep their name.
func TestD1_DownRestoresTheOldShape(t *testing.T) {
	pool := stageLegacy(t)
	applyMoneyMigration(t, pool)
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		insert into operational_expenses (user_id, description, amount, category_id, expense_date)
		select '00000000-0000-0000-0000-0000000000a1', 'Caja de envío', 800, id, '2026-03-13'
		from expense_categories where lower(name) = 'envases'`)
	if err != nil {
		t.Fatal(err)
	}

	runMigrationFiles(t, pool, "down", func(v string) bool { return v == moneyMigration })

	var typ string
	if err := pool.QueryRow(ctx, `
		select data_type from information_schema.columns
		where table_schema = current_schema() and table_name = 'operational_expenses' and column_name = 'amount'`).Scan(&typ); err != nil {
		t.Fatal(err)
	}
	if typ != "numeric" {
		t.Errorf("amount is %s after down, want numeric", typ)
	}
	cats := map[string]string{}
	rows, err := pool.Query(ctx, `select description, category from operational_expenses`)
	if err != nil {
		t.Fatalf("category column after down: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var d, c string
		if err := rows.Scan(&d, &c); err != nil {
			t.Fatal(err)
		}
		cats[d] = c
	}
	want := map[string]string{"Harina": "ingredientes", "Gas": "servicios", "Cajas": "otros", "Caja de envío": "otros"}
	for d, c := range want {
		if cats[d] != c {
			t.Errorf("%q: category = %q after down, want %q", d, cats[d], c)
		}
	}
	var exists bool
	if err := pool.QueryRow(ctx, `select to_regclass('expense_categories') is not null`).Scan(&exists); err != nil || exists {
		t.Errorf("expense_categories should be gone after down (exists=%v, err=%v)", exists, err)
	}
}
