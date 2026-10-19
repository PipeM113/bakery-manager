package migrations_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/PipeM113/bakery-manager/internal/testsupport/pgtest"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Sprint 5, rule T7: migration 18 turns the TIMESTAMP columns into TIMESTAMPTZ. The server stored
// them in UTC, so the values already there are read as UTC.

const timestamptzMigration = "000018"

func timestamptzMigrationExists(t *testing.T) {
	t.Helper()
	if m, _ := filepath.Glob(filepath.Join(migrationsDir, timestamptzMigration+"_*.up.sql")); len(m) != 1 {
		t.Fatalf("expected exactly one %s_*.up.sql, found %v", timestamptzMigration, m)
	}
}

func naiveTimestampColumns(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		select table_name || '.' || column_name from information_schema.columns
		where table_schema = current_schema() and data_type = 'timestamp without time zone' order by 1`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	return out
}

func TestT7_NoColumnWithoutTimeZoneIsLeft(t *testing.T) {
	timestamptzMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	if left := naiveTimestampColumns(t, pool); len(left) != 0 {
		t.Errorf("columns still TIMESTAMP without time zone: %v", left)
	}
}

func TestT7_ExistingValuesAreReadAsUTC(t *testing.T) {
	timestamptzMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	runMigrationFiles(t, pool, "up", func(v string) bool { return v < timestamptzMigration })
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `insert into users (name, email, password, role, created_at)
		values ('Owner', 'o@example.test', 'x', 'owner', '2026-03-10 12:00:00')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `insert into ingredients (name, default_unit, package_size, package_price, price_per_unit, brand, created_at, updated_at, kind)
		values ('Harina', 'gr', 1000, 2000, 2, 'M', '2026-03-10 23:30:00', '2026-03-11 01:15:00', 'ingredient')`); err != nil {
		t.Fatal(err)
	}

	runMigrationFiles(t, pool, "up", func(v string) bool { return v == timestamptzMigration })

	var created, updated time.Time
	if err := pool.QueryRow(ctx, `select created_at, updated_at from ingredients where name = 'Harina'`).Scan(&created, &updated); err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 3, 10, 23, 30, 0, 0, time.UTC); !created.Equal(want) {
		t.Errorf("created_at = %s, want %s", created.UTC(), want)
	}
	if want := time.Date(2026, 3, 11, 1, 15, 0, 0, time.UTC); !updated.Equal(want) {
		t.Errorf("updated_at = %s, want %s", updated.UTC(), want)
	}
	var userCreated time.Time
	if err := pool.QueryRow(ctx, `select created_at from users`).Scan(&userCreated); err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC); !userCreated.Equal(want) {
		t.Errorf("users.created_at = %s, want %s", userCreated.UTC(), want)
	}
}

func TestT7_DownRestoresTheSameWallClockInUTC(t *testing.T) {
	timestamptzMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `insert into ingredients (name, default_unit, package_size, package_price, price_per_unit, brand, created_at, kind)
		values ('Harina', 'gr', 1000, 2000, 2, 'M', '2026-03-10 23:30:00+00', 'ingredient')`); err != nil {
		t.Fatal(err)
	}

	runMigrationFiles(t, pool, "down", func(v string) bool { return v == timestamptzMigration })

	var wall string
	if err := pool.QueryRow(ctx, `select to_char(created_at, 'YYYY-MM-DD HH24:MI:SS') from ingredients`).Scan(&wall); err != nil {
		t.Fatal(err)
	}
	if wall != "2026-03-10 23:30:00" {
		t.Errorf("created_at after down = %s, want 2026-03-10 23:30:00 (the UTC wall clock)", wall)
	}
	var typ string
	if err := pool.QueryRow(ctx, `select data_type from information_schema.columns
		where table_schema = current_schema() and table_name = 'ingredients' and column_name = 'created_at'`).Scan(&typ); err != nil {
		t.Fatal(err)
	}
	if typ != "timestamp without time zone" {
		t.Errorf("created_at is %s after down, want timestamp without time zone", typ)
	}
}
