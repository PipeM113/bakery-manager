package migrations_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/PipeM113/bakery-manager/internal/testsupport/pgtest"
)

// Sprint 13+, rule Q11g: the name of the business and its Instagram belong to the brand and live in the
// code. Migration 27 removes their columns from business_profile.

const brandFieldsMigration = "000027"

func brandFieldsMigrationExists(t *testing.T) {
	t.Helper()
	if m, _ := filepath.Glob(filepath.Join(migrationsDir, brandFieldsMigration+"_*.up.sql")); len(m) != 1 {
		t.Fatalf("expected exactly one %s_*.up.sql, found %v", brandFieldsMigration, m)
	}
}

func TestQ11g_TheBrandColumnsAreGone(t *testing.T) {
	brandFieldsMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	for _, col := range []string{"name", "instagram"} {
		if columnCount(t, pool, "business_profile", col) != 0 {
			t.Errorf("business_profile still has %s", col)
		}
	}
	var n int
	if err := pool.QueryRow(context.Background(), `select count(*) from business_profile`).Scan(&n); err != nil || n != 1 {
		t.Errorf("the profile row must stay (%d, %v)", n, err)
	}
}

func TestQ11g_DownBringsThemBackEmpty(t *testing.T) {
	brandFieldsMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	runMigrationFiles(t, pool, "down", func(v string) bool { return v == brandFieldsMigration })
	for _, col := range []string{"name", "instagram"} {
		if columnCount(t, pool, "business_profile", col) != 1 {
			t.Errorf("down did not bring %s back", col)
		}
	}
	var name, instagram string
	if err := pool.QueryRow(context.Background(), `select name, instagram from business_profile`).Scan(&name, &instagram); err != nil || name != "" || instagram != "" {
		t.Errorf("name/instagram = %q/%q (%v), want empty texts", name, instagram, err)
	}
}
