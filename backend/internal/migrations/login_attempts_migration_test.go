package migrations_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/PipeM113/bakery-manager/internal/testsupport/pgtest"
)

// Sprint 13, rule C1: migration 26 keeps the failed login attempts, in the database because every
// serverless instance is a different process.

const loginAttemptsMigration = "000026"

func loginAttemptsMigrationExists(t *testing.T) {
	t.Helper()
	if m, _ := filepath.Glob(filepath.Join(migrationsDir, loginAttemptsMigration+"_*.up.sql")); len(m) != 1 {
		t.Fatalf("expected exactly one %s_*.up.sql, found %v", loginAttemptsMigration, m)
	}
}

func TestC1_LoginAttemptsTable(t *testing.T) {
	loginAttemptsMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	for _, col := range []string{"email", "attempted_at"} {
		if columnCount(t, pool, "login_attempts", col) != 1 {
			t.Errorf("login_attempts has no %s", col)
		}
	}
	for _, col := range []string{"password", "ip"} {
		if columnCount(t, pool, "login_attempts", col) != 0 {
			t.Errorf("login_attempts must not keep %s", col)
		}
	}
	var rls bool
	if err := pool.QueryRow(context.Background(), `select relrowsecurity from pg_class
		where relname = 'login_attempts' and relnamespace = current_schema()::regnamespace`).Scan(&rls); err != nil || !rls {
		t.Errorf("login_attempts must have row level security on (%v, %v)", rls, err)
	}
	var indexes int
	if err := pool.QueryRow(context.Background(), `select count(*) from pg_indexes
		where schemaname = current_schema() and tablename = 'login_attempts' and indexname <> 'login_attempts_pkey'`).Scan(&indexes); err != nil || indexes < 1 {
		t.Errorf("login_attempts needs an index to count by e-mail and time (%d, %v)", indexes, err)
	}
}

func TestC1_DownDropsTheAttempts(t *testing.T) {
	loginAttemptsMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	runMigrationFiles(t, pool, "down", func(v string) bool { return v == loginAttemptsMigration })
	var n int
	if err := pool.QueryRow(context.Background(), `select count(*) from information_schema.tables
		where table_schema = current_schema() and table_name = 'login_attempts'`).Scan(&n); err != nil || n != 0 {
		t.Errorf("login_attempts is still there (%v)", err)
	}
}
