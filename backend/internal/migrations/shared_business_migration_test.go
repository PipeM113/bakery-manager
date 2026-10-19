package migrations_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/PipeM113/bakery-manager/internal/testsupport/pgtest"
	"github.com/jackc/pgx/v5"
)

// Sprint 11, rule N1: migration 23 turns user_id into created_by in the seven tables that have it.
// If the user is deleted the records stay.

const sharedBusinessMigration = "000023"

var userTables = []string{"recipes", "fixed_costs", "sales", "orders", "operational_expenses", "quotations", "stock_adjustments"}

func sharedBusinessMigrationExists(t *testing.T) {
	t.Helper()
	if m, _ := filepath.Glob(filepath.Join(migrationsDir, sharedBusinessMigration+"_*.up.sql")); len(m) != 1 {
		t.Fatalf("expected exactly one %s_*.up.sql, found %v", sharedBusinessMigration, m)
	}
}

type queryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func columnCount(t *testing.T, pool queryer, table, column string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), `select count(*) from information_schema.columns
		where table_schema = current_schema() and table_name = $1 and column_name = $2`, table, column).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestN1_UserIDBecomesCreatedBy(t *testing.T) {
	sharedBusinessMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	for _, table := range userTables {
		if columnCount(t, pool, table, "created_by") != 1 {
			t.Errorf("%s has no created_by", table)
		}
		if columnCount(t, pool, table, "user_id") != 0 {
			t.Errorf("%s still has user_id", table)
		}
	}
}

func TestN1_ExistingRecordsKeepTheirCreator(t *testing.T) {
	sharedBusinessMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	runMigrationFiles(t, pool, "up", func(v string) bool { return v < sharedBusinessMigration })
	ctx := context.Background()
	var user, recipe, sale, fixed string
	if err := pool.QueryRow(ctx, `insert into users (name, email, password, role) values ('Owner', 'o@example.test', 'x', 'owner') returning id`).Scan(&user); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `insert into recipes (user_id, name, description, yield, yield_unit) values ($1, 'Torta', 'x', 1, 'porciones') returning id`, user).Scan(&recipe); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `insert into sales (user_id, recipe_id, quantity_sold, unit_price, line_total, sale_date) values ($1, $2, 1, 1000, 1000, '2026-03-01') returning id`, user, recipe).Scan(&sale); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `insert into fixed_costs (user_id, name) values ($1, 'Gas') returning id`, user).Scan(&fixed); err != nil {
		t.Fatal(err)
	}

	runMigrationFiles(t, pool, "up", func(v string) bool { return v == sharedBusinessMigration })

	for table, id := range map[string]string{"recipes": recipe, "sales": sale, "fixed_costs": fixed} {
		var by string
		if err := pool.QueryRow(ctx, `select created_by::text from `+table+` where id = $1`, id).Scan(&by); err != nil || by != user {
			t.Errorf("%s.created_by = %q (%v), want %q", table, by, err, user)
		}
	}
}

func TestN1_DeletingAUserKeepsTheRecords(t *testing.T) {
	sharedBusinessMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	ctx := context.Background()
	var user, recipe string
	if err := pool.QueryRow(ctx, `insert into users (name, email, password, role) values ('Owner', 'o@example.test', 'x', 'owner') returning id`).Scan(&user); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `insert into recipes (created_by, name, description, yield, yield_unit) values ($1, 'Torta', 'x', 1, 'porciones') returning id`, user).Scan(&recipe); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `delete from users where id = $1`, user); err != nil {
		t.Fatalf("delete user: %v", err)
	}
	var by *string
	if err := pool.QueryRow(ctx, `select created_by::text from recipes where id = $1`, recipe).Scan(&by); err != nil {
		t.Fatalf("the recipe was deleted with its creator: %v", err)
	}
	if by != nil {
		t.Errorf("created_by = %v, want none", *by)
	}
}

func TestN1_DownRestoresUserID(t *testing.T) {
	sharedBusinessMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	ctx := context.Background()
	var user, recipe string
	if err := pool.QueryRow(ctx, `insert into users (name, email, password, role) values ('Owner', 'o@example.test', 'x', 'owner') returning id`).Scan(&user); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `insert into recipes (created_by, name, description, yield, yield_unit) values ($1, 'Torta', 'x', 1, 'porciones') returning id`, user).Scan(&recipe); err != nil {
		t.Fatal(err)
	}

	runMigrationFiles(t, pool, "down", func(v string) bool { return v == sharedBusinessMigration })

	for _, table := range userTables {
		if columnCount(t, pool, table, "user_id") != 1 || columnCount(t, pool, table, "created_by") != 0 {
			t.Errorf("%s did not go back to user_id", table)
		}
	}
	var by string
	if err := pool.QueryRow(ctx, `select user_id::text from recipes where id = $1`, recipe).Scan(&by); err != nil || by != user {
		t.Errorf("user_id = %q (%v), want %q", by, err, user)
	}
}
