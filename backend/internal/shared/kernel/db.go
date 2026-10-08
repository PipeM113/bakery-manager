package kernel

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPoolConfig builds the pool settings for a serverless deployment behind Supabase's
// transaction pooler, which does not support prepared statements. pgx's Exec mode sends
// each query in a single round trip without preparing it. The mode is forced after parsing,
// so a default_query_exec_mode inside the connection string cannot bring it back.
//
// The error never repeats the connection string: it contains the password.
func NewPoolConfig(databaseURL string, maxConns int32) (*pgxpool.Config, error) {
	if maxConns < 1 {
		return nil, fmt.Errorf("el máximo de conexiones debe ser al menos 1")
	}
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("DATABASE_URL no es una cadena de conexión válida")
	}

	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	cfg.MaxConns = maxConns
	// Short-lived instances: do not hold idle connections the pooler may have dropped.
	cfg.MaxConnIdleTime = 2 * time.Minute
	cfg.MaxConnLifetime = 30 * time.Minute
	return cfg, nil
}

// Open creates the pool. It does not connect: connections are made on first use, so a
// cold start never fails because the database is slow or down.
func Open(ctx context.Context, databaseURL string, maxConns int32) (*pgxpool.Pool, error) {
	cfg, err := NewPoolConfig(databaseURL, maxConns)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("no se pudo crear el pool de conexiones")
	}
	return pool, nil
}
