// Package pgtest gives tests a throwaway Postgres database.
//
// It only ever talks to a server on this machine: EnsureLocal refuses anything else
// (for example a Supabase URL), so a test can never touch real data.
package pgtest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EnvName is the variable that holds the test server URL (user needs CREATEDB).
const EnvName = "TEST_DATABASE_URL"

var localHosts = map[string]bool{"localhost": true, "127.0.0.1": true, "::1": true}

// EnsureLocal returns an error unless rawURL points at a server on this machine.
// The error never repeats the URL, because it can contain a password.
func EnsureLocal(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("the database url cannot be parsed")
	}
	if !localHosts[u.Hostname()] {
		return fmt.Errorf("refusing to run tests against non-local host %q", u.Hostname())
	}
	return nil
}

// NewDatabase creates an empty, uniquely named database on the test server and
// drops it when the test ends. Without TEST_DATABASE_URL the test is skipped,
// except when CI is set, where a missing database is a failure.
func NewDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	base := os.Getenv(EnvName)
	if base == "" {
		if os.Getenv("CI") != "" {
			t.Fatalf("%s must be set in CI", EnvName)
		}
		t.Skipf("%s is not set: skipping the database test", EnvName)
	}
	if err := EnsureLocal(base); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	name := "bm_test_" + randomHex(t)

	admin, err := pgx.Connect(ctx, base)
	if err != nil {
		t.Fatalf("connect to the test server: %v", err)
	}
	defer admin.Close(ctx)
	// The name is generated here (hex only), so building the statement is safe.
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("create test database: %v", err)
	}

	u, _ := url.Parse(base)
	u.Path = "/" + name
	pool, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		cleaner, err := pgx.Connect(context.Background(), base)
		if err != nil {
			t.Logf("cleanup: connect: %v", err)
			return
		}
		defer cleaner.Close(context.Background())
		if _, err := cleaner.Exec(context.Background(), "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)"); err != nil {
			t.Logf("cleanup: drop %s: %v", name, err)
		}
	})
	return pool
}

// Migrate runs every *.up.sql in dir in version order (direction "up"), or every
// *.down.sql newest first (direction "down"). Files may hold several statements.
func Migrate(t *testing.T, pool *pgxpool.Pool, dir, direction string) {
	t.Helper()
	if direction != "up" && direction != "down" {
		t.Fatalf("direction must be up or down, got %q", direction)
	}
	files, err := filepath.Glob(filepath.Join(dir, "*."+direction+".sql"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files) // versions are zero-padded, so name order is version order
	if direction == "down" {
		for i, j := 0, len(files)-1; i < j; i, j = i+1, j-1 {
			files[i], files[j] = files[j], files[i]
		}
	}

	ctx := context.Background()
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire connection: %v", err)
	}
	defer conn.Release()

	for _, file := range files {
		sql, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		// The simple protocol accepts several statements in one call.
		if _, err := conn.Conn().PgConn().Exec(ctx, string(sql)).ReadAll(); err != nil {
			t.Fatalf("migration %s (%s): %v", filepath.Base(file), direction, err)
		}
	}
}

func randomHex(t *testing.T) string {
	t.Helper()
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return strings.ToLower(hex.EncodeToString(b))
}
