package migrations_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/PipeM113/bakery-manager/internal/testsupport/pgtest"
)

// Sprint 12, rule Q11: migration 25 keeps the data of the business that the quotation PDF prints.
// The RUT and the bank account are secrets of the environment and are not stored here.

const businessProfileMigration = "000025"

func businessProfileMigrationExists(t *testing.T) {
	t.Helper()
	if m, _ := filepath.Glob(filepath.Join(migrationsDir, businessProfileMigration+"_*.up.sql")); len(m) != 1 {
		t.Fatalf("expected exactly one %s_*.up.sql, found %v", businessProfileMigration, m)
	}
}

func TestQ11_ProfileTableHasOnlyWhatIsEdited(t *testing.T) {
	businessProfileMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	for _, col := range []string{"name", "phone", "instagram", "bank", "holder", "updated_at", "updated_by"} {
		if columnCount(t, pool, "business_profile", col) != 1 {
			t.Errorf("business_profile has no %s", col)
		}
	}
	for _, col := range []string{"account", "rut", "bank_account"} {
		if columnCount(t, pool, "business_profile", col) != 0 {
			t.Errorf("business_profile must not store %s (it comes from the environment)", col)
		}
	}
	var rls bool
	if err := pool.QueryRow(context.Background(), `select relrowsecurity from pg_class
		where relname = 'business_profile' and relnamespace = current_schema()::regnamespace`).Scan(&rls); err != nil || !rls {
		t.Errorf("business_profile must have row level security on (%v, %v)", rls, err)
	}
}

func TestQ11_ThereIsOneEmptyRowAndNeverMore(t *testing.T) {
	businessProfileMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	ctx := context.Background()
	var n int
	var name, phone, instagram, bank, holder string
	if err := pool.QueryRow(ctx, `select count(*) from business_profile`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("rows = %d (%v), want 1", n, err)
	}
	if err := pool.QueryRow(ctx, `select name, phone, instagram, bank, holder from business_profile`).Scan(&name, &phone, &instagram, &bank, &holder); err != nil {
		t.Fatal(err)
	}
	if name+phone+instagram+bank+holder != "" {
		t.Errorf("the profile does not start empty: %q %q %q %q %q", name, phone, instagram, bank, holder)
	}
	if _, err := pool.Exec(ctx, `insert into business_profile (name) values ('otra')`); err == nil {
		t.Error("a second row was accepted")
	}
}

func TestQ11_DownDropsTheProfile(t *testing.T) {
	businessProfileMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	runMigrationFiles(t, pool, "down", func(v string) bool { return v == businessProfileMigration })
	var n int
	if err := pool.QueryRow(context.Background(), `select count(*) from information_schema.tables
		where table_schema = current_schema() and table_name = 'business_profile'`).Scan(&n); err != nil || n != 0 {
		t.Errorf("business_profile is still there (%v)", err)
	}
}
