package kernel_test

import (
	"context"
	"testing"
	"time"

	"github.com/PipeM113/bakery-manager/internal/shared/kernel"
	"github.com/PipeM113/bakery-manager/internal/testsupport/pgtest"
	"github.com/jackc/pgx/v5"
)

const localURL = "postgres://u:p@127.0.0.1:5432/x"

// AC4: Supabase's transaction pooler does not support prepared statements, so the pool
// must use pgx's Exec mode (extended protocol, one round trip, no prepared statements).
func TestAC4_PoolUsesExecModeAndTheConfiguredLimits(t *testing.T) {
	cfg, err := kernel.NewPoolConfig(localURL, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ConnConfig.DefaultQueryExecMode != pgx.QueryExecModeExec {
		t.Errorf("DefaultQueryExecMode = %v, want Exec", cfg.ConnConfig.DefaultQueryExecMode)
	}
	if cfg.MaxConns != 3 {
		t.Errorf("MaxConns = %d, want 3", cfg.MaxConns)
	}
	timeout := cfg.ConnConfig.ConnectTimeout
	if timeout <= 0 || timeout > 10*time.Second {
		t.Errorf("ConnectTimeout = %v, want between 1ns and 10s", timeout)
	}
}

// AC4: a setting hidden in the connection string must not bring prepared statements back.
func TestAC4_ConnectionStringCannotSwitchBackToPreparedStatements(t *testing.T) {
	cfg, err := kernel.NewPoolConfig(localURL+"?default_query_exec_mode=cache_statement", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ConnConfig.DefaultQueryExecMode != pgx.QueryExecModeExec {
		t.Fatalf("DefaultQueryExecMode = %v, want Exec", cfg.ConnConfig.DefaultQueryExecMode)
	}
}

func TestAC4_PoolConfigRejectsBadInput(t *testing.T) {
	if _, err := kernel.NewPoolConfig(localURL, 0); err == nil {
		t.Error("a pool of 0 connections must be rejected")
	}
	if _, err := kernel.NewPoolConfig("::not a url::", 3); err == nil {
		t.Error("an unparseable url must be rejected")
	}
}

// AC4: the pool really works against Postgres in that mode, and opening it does not
// require the database to be reachable (a serverless cold start must not fail on that).
func TestAC4_PoolRunsParameterizedQueries(t *testing.T) {
	admin := pgtest.NewDatabase(t)
	pool, err := kernel.Open(context.Background(), admin.Config().ConnConfig.ConnString(), 2)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer pool.Close()

	var got int
	if err := pool.QueryRow(context.Background(), `select $1::int + 1`, 41).Scan(&got); err != nil {
		t.Fatalf("query: %v", err)
	}
	if got != 42 {
		t.Fatalf("got %d, want 42", got)
	}
}

func TestAC4_OpenDoesNotNeedTheDatabaseToBeUp(t *testing.T) {
	pool, err := kernel.Open(context.Background(), "postgres://u:p@127.0.0.1:1/x", 2)
	if err != nil {
		t.Fatalf("Open must not fail when the database is down: %v", err)
	}
	pool.Close()
}
