package deploy_test

import (
	"context"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/PipeM113/bakery-manager/internal/testsupport/pgtest"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Supabase's admin role (postgres) is not a superuser: it can create roles and owns the
// database, nothing more. The first real run of the setup script failed there with
// "must be able to SET ROLE" (2026-10-15) while the rehearsal, which ran as a superuser,
// passed. This test runs the script the way Supabase does, twice, because a failed first
// attempt leaves the role behind and the second run takes the ALTER ROLE branch.
func TestB1_SetupScriptWorksForAnAdminThatIsNotASuperuser(t *testing.T) {
	base := os.Getenv(pgtest.EnvName)
	if base == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	adminRole := "adm_" + randomPassword(t)[:10]
	adminPassword := randomPassword(t)

	// Registered before NewDatabase so it runs after the database is dropped: a role that
	// owns a database cannot be dropped.
	t.Cleanup(func() {
		conn, err := pgx.Connect(ctx, base)
		if err != nil {
			return
		}
		defer conn.Close(ctx)
		_, _ = conn.Exec(ctx, "DROP ROLE IF EXISTS "+adminRole)
	})

	su := pgtest.NewDatabase(t)
	ensureDataAPIRoles(t, su)
	runSQL(t, su, "DROP ROLE IF EXISTS "+appRole) // earlier tests create it as a superuser
	runSQL(t, su, "CREATE ROLE "+adminRole+" LOGIN PASSWORD '"+adminPassword+"' CREATEROLE")
	var dbName string
	if err := su.QueryRow(ctx, "select current_database()").Scan(&dbName); err != nil {
		t.Fatal(err)
	}
	runSQL(t, su, "ALTER DATABASE "+dbName+" OWNER TO "+adminRole)
	runSQL(t, su, "ALTER SCHEMA public OWNER TO "+adminRole)
	runSQL(t, su, "CREATE EXTENSION IF NOT EXISTS pgcrypto") // Supabase has it installed

	cc := su.Config().ConnConfig
	asAdminURL := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(adminRole, adminPassword),
		Host:     net.JoinHostPort(cc.Host, strconv.Itoa(int(cc.Port))),
		Path:     "/" + dbName,
		RawQuery: "sslmode=disable",
	}
	asAdmin, err := pgxpool.New(ctx, asAdminURL.String())
	if err != nil {
		t.Fatal(err)
	}
	defer asAdmin.Close()

	raw, err := os.ReadFile(backendPath("db", "setup", "01_manager_schema.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for run := 1; run <= 2; run++ {
		runSQL(t, asAdmin, strings.ReplaceAll(string(raw), passwordToken, randomPassword(t)))
	}

	var owner string
	if err := su.QueryRow(ctx, "select pg_get_userbyid(nspowner) from pg_namespace where nspname = 'manager'").Scan(&owner); err != nil {
		t.Fatal(err)
	}
	if owner != appRole {
		t.Errorf("schema manager is owned by %q, want %s", owner, appRole)
	}
}
