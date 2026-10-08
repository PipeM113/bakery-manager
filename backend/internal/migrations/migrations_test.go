package migrations_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/PipeM113/bakery-manager/internal/testsupport/pgtest"
	"github.com/jackc/pgx/v5/pgxpool"
)

const migrationsDir = "../../db/migrations"

var expectedTables = []string{
	"fixed_costs",
	"ingredient_price_history",
	"ingredients",
	"operational_expenses",
	"quotations",
	"recipe_ingredients",
	"recipes",
	"sale_ingredients",
	"sales",
	"users",
}

func publicTables(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		select table_name from information_schema.tables
		where table_schema = 'public' and table_type = 'BASE TABLE'
		order by table_name`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		names = append(names, n)
	}
	return names
}

func publicEnums(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		select t.typname from pg_type t join pg_namespace n on n.oid = t.typnamespace
		where n.nspname = 'public' and t.typtype = 'e' order by 1`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		names = append(names, n)
	}
	return names
}

// AC4: the existing migrations (1 to 14) apply on an empty database.
func TestAC4_MigrationsApplyOnAnEmptyDatabase(t *testing.T) {
	pool := pgtest.NewDatabase(t)

	pgtest.Migrate(t, pool, migrationsDir, "up")

	got := publicTables(t, pool)
	if !slices.Equal(got, expectedTables) {
		t.Fatalf("tables after up = %v, want %v", got, expectedTables)
	}
	if enums := publicEnums(t, pool); !slices.Equal(enums, []string{"quotation_status"}) {
		t.Fatalf("enums after up = %v, want [quotation_status]", enums)
	}
}

// AC4: the down migrations undo everything the up migrations created.
func TestAC4_MigrationsRevertToAnEmptyDatabase(t *testing.T) {
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")

	pgtest.Migrate(t, pool, migrationsDir, "down")

	if got := publicTables(t, pool); len(got) != 0 {
		t.Fatalf("tables left after down = %v", got)
	}
	if enums := publicEnums(t, pool); len(enums) != 0 {
		t.Fatalf("enums left after down = %v", enums)
	}
}

// AC4: applying, reverting and applying again works (a down must leave no residue).
func TestAC4_MigrationsApplyAgainAfterReverting(t *testing.T) {
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	pgtest.Migrate(t, pool, migrationsDir, "down")

	pgtest.Migrate(t, pool, migrationsDir, "up")

	if got := publicTables(t, pool); !slices.Equal(got, expectedTables) {
		t.Fatalf("tables after the second up = %v", got)
	}
}

// AC4: every up migration has a matching down, and each version appears once.
func TestAC4_EveryUpHasADownAndVersionsAreUnique(t *testing.T) {
	ups, _ := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	if len(ups) == 0 {
		t.Fatal("no migrations found")
	}
	seen := map[string]bool{}
	for _, up := range ups {
		base := strings.TrimSuffix(filepath.Base(up), ".up.sql")
		version := strings.SplitN(base, "_", 2)[0]
		if seen[version] {
			t.Errorf("version %s appears twice", version)
		}
		seen[version] = true
		if _, err := os.Stat(strings.TrimSuffix(up, ".up.sql") + ".down.sql"); err != nil {
			t.Errorf("%s has no down migration", filepath.Base(up))
		}
	}
}
