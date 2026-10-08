package pgtest

import (
	"strings"
	"testing"
)

// AC1: tests must never run against anything but a local server.
func TestAC1_EnsureLocalAcceptsOnlyThisMachine(t *testing.T) {
	cases := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"loopback ip", "postgres://u:p@127.0.0.1:5432/x", false},
		{"localhost", "postgres://u:p@localhost:54329/x", false},
		{"ipv6 loopback", "postgres://u:p@[::1]:5432/x", false},
		{"supabase direct", "postgres://postgres:p@db.abcdefgh.supabase.co:5432/postgres", true},
		{"supabase pooler", "postgres://postgres.abcdefgh:p@aws-0-us-east-1.pooler.supabase.com:6543/postgres", true},
		{"lookalike host", "postgres://u:p@localhost.evil.example:5432/x", true},
		{"no host", "postgres:///x", true},
		{"not a url", "::::", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := EnsureLocal(c.url)
			if (err != nil) != c.wantErr {
				t.Fatalf("EnsureLocal(%q) error = %v, wantErr %v", c.url, err, c.wantErr)
			}
		})
	}
}

// AC1: a refusal must not leak the password it was given.
func TestAC1_EnsureLocalErrorDoesNotLeakPassword(t *testing.T) {
	err := EnsureLocal("postgres://postgres:s3cr3t-pass@db.abcdefgh.supabase.co:5432/postgres")
	if err == nil {
		t.Fatal("expected a refusal")
	}
	if strings.Contains(err.Error(), "s3cr3t-pass") {
		t.Fatalf("the error leaks the password: %v", err)
	}
}

// AC1: the helper really builds and drops an empty database.
func TestAC1_NewDatabaseGivesAnEmptyIsolatedDatabase(t *testing.T) {
	pool := NewDatabase(t)

	var tables int
	err := pool.QueryRow(t.Context(),
		`select count(*) from information_schema.tables where table_schema = 'public'`).Scan(&tables)
	if err != nil {
		t.Fatal(err)
	}
	if tables != 0 {
		t.Fatalf("expected an empty database, found %d tables", tables)
	}
}
