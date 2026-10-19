package migrations_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/PipeM113/bakery-manager/internal/testsupport/pgtest"
)

// Sprint 6, rule F1: migration 19 moves the amount of a fixed cost into versions by month and the
// on/off switch into a month in which the cost ends.

const fixedCostsMigration = "000019"

func fixedCostsMigrationExists(t *testing.T) {
	t.Helper()
	if m, _ := filepath.Glob(filepath.Join(migrationsDir, fixedCostsMigration+"_*.up.sql")); len(m) != 1 {
		t.Fatalf("expected exactly one %s_*.up.sql, found %v", fixedCostsMigration, m)
	}
}

func TestF1_ExistingCostsKeepTheirAmountFromTheirCreationMonth(t *testing.T) {
	fixedCostsMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	runMigrationFiles(t, pool, "up", func(v string) bool { return v < fixedCostsMigration })
	ctx := context.Background()
	var user string
	if err := pool.QueryRow(ctx, `insert into users (name, email, password, role) values ('Owner', 'o@example.test', 'x', 'owner') returning id`).Scan(&user); err != nil {
		t.Fatal(err)
	}
	// 02:00 UTC on 1 February is 23:00 of 31 January in Santiago: the cost belongs to January.
	var active, inactive string
	if err := pool.QueryRow(ctx, `insert into fixed_costs (user_id, name, monthly_amount, is_active, created_at)
		values ($1, 'Arriendo', 250000, true, '2026-02-01 02:00:00+00') returning id`, user).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `insert into fixed_costs (user_id, name, monthly_amount, is_active, created_at, updated_at)
		values ($1, 'Antiguo', 777, false, '2025-11-10 12:00:00+00', '2026-03-15 12:00:00+00') returning id`, user).Scan(&inactive); err != nil {
		t.Fatal(err)
	}

	runMigrationFiles(t, pool, "up", func(v string) bool { return v == fixedCostsMigration })

	date := func(q string, args ...any) string {
		var d *time.Time
		if err := pool.QueryRow(ctx, q, args...).Scan(&d); err != nil {
			t.Fatal(err)
		}
		if d == nil {
			return ""
		}
		return d.Format("2006-01-02")
	}
	if got := date(`select valid_from from fixed_cost_versions where fixed_cost_id = $1`, active); got != "2026-01-01" {
		t.Errorf("valid_from of the active cost = %q, want 2026-01-01 (its creation month in Santiago)", got)
	}
	if got := date(`select ended_from from fixed_costs where id = $1`, active); got != "" {
		t.Errorf("ended_from of the active cost = %q, want none", got)
	}
	if got := date(`select valid_from from fixed_cost_versions where fixed_cost_id = $1`, inactive); got != "2025-11-01" {
		t.Errorf("valid_from of the inactive cost = %q, want 2025-11-01", got)
	}
	if got := date(`select ended_from from fixed_costs where id = $1`, inactive); got != "2026-03-01" {
		t.Errorf("ended_from of the inactive cost = %q, want 2026-03-01 (the month it was last updated)", got)
	}
	var amount int
	if err := pool.QueryRow(ctx, `select monthly_amount from fixed_cost_versions where fixed_cost_id = $1`, active).Scan(&amount); err != nil || amount != 250000 {
		t.Errorf("amount of the version = %d (%v), want 250000", amount, err)
	}
}

func TestF1_VersionsTableIsClosedToTheDataAPI(t *testing.T) {
	fixedCostsMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	ctx := context.Background()
	var rls bool
	if err := pool.QueryRow(ctx, `select relrowsecurity from pg_class where relname = 'fixed_cost_versions' and relnamespace = current_schema()::regnamespace`).Scan(&rls); err != nil {
		t.Fatalf("fixed_cost_versions: %v", err)
	}
	if !rls {
		t.Error("row level security is off on fixed_cost_versions")
	}
	var policies int
	if err := pool.QueryRow(ctx, `select count(*) from pg_policies where tablename = 'fixed_cost_versions'`).Scan(&policies); err != nil || policies != 0 {
		t.Errorf("policies = %d (%v), want none", policies, err)
	}
}

func TestF1_DownRestoresAmountAndSwitch(t *testing.T) {
	fixedCostsMigrationExists(t)
	pool := pgtest.NewDatabase(t)
	pgtest.Migrate(t, pool, migrationsDir, "up")
	ctx := context.Background()
	var user, id, ended string
	if err := pool.QueryRow(ctx, `insert into users (name, email, password, role) values ('Owner', 'o@example.test', 'x', 'owner') returning id`).Scan(&user); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `insert into fixed_costs (created_by, name) values ($1, 'Gas') returning id`, user).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `insert into fixed_cost_versions (fixed_cost_id, valid_from, monthly_amount)
		values ($1, '2026-01-01', 100), ($1, '2026-03-01', 150)`, id); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `insert into fixed_costs (created_by, name, ended_from) values ($1, 'Viejo', '2026-02-01') returning id`, user).Scan(&ended); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `insert into fixed_cost_versions (fixed_cost_id, valid_from, monthly_amount) values ($1, '2025-12-01', 77)`, ended); err != nil {
		t.Fatal(err)
	}

	runMigrationFiles(t, pool, "down", func(v string) bool { return v == fixedCostsMigration })

	var amount int
	var isActive bool
	if err := pool.QueryRow(ctx, `select monthly_amount, is_active from fixed_costs where id = $1`, id).Scan(&amount, &isActive); err != nil {
		t.Fatal(err)
	}
	if amount != 150 || !isActive {
		t.Errorf("after down: amount = %d, active = %v, want the latest amount 150 and active", amount, isActive)
	}
	if err := pool.QueryRow(ctx, `select monthly_amount, is_active from fixed_costs where id = $1`, ended).Scan(&amount, &isActive); err != nil {
		t.Fatal(err)
	}
	if amount != 77 || isActive {
		t.Errorf("after down: amount = %d, active = %v, want 77 and inactive", amount, isActive)
	}
	var left bool
	if err := pool.QueryRow(ctx, `select to_regclass('fixed_cost_versions') is not null`).Scan(&left); err != nil || left {
		t.Errorf("fixed_cost_versions still exists after down (%v, %v)", left, err)
	}
}
