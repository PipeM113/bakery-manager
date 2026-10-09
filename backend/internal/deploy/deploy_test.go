// Package deploy_test rehearses, on a local Postgres, the procedure of docs/despliegue.md:
// create the role and the schema, run the real migrate tool as that role, create the first
// user and use the API. Nothing here touches Supabase.
//
// These tests need the Go toolchain and network access to the Go module proxy: the
// migrate tool is built on demand with `go run <module>@<version>` and adds nothing to go.mod.
package deploy_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/PipeM113/bakery-manager/internal/app"
	"github.com/PipeM113/bakery-manager/internal/config"
	"github.com/PipeM113/bakery-manager/internal/shared/kernel"
	"github.com/PipeM113/bakery-manager/internal/testsupport/pgtest"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	migrateModule = "github.com/golang-migrate/migrate/v4/cmd/migrate@v4.20.1"
	passwordToken = "__MANAGER_APP_PASSWORD__"
	appRole       = "manager_app"
	jwtSecret     = "0123456789abcdef0123456789abcdef"
)

// backendPath resolves a path inside backend/ from this package's directory.
func backendPath(parts ...string) string {
	return filepath.Join(append([]string{"..", ".."}, parts...)...)
}

func randomPassword(t *testing.T) string {
	t.Helper()
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(b)
}

// runSQL executes a script that may hold several statements (simple protocol).
func runSQL(t *testing.T, pool *pgxpool.Pool, sql string) {
	t.Helper()
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	if _, err := conn.Conn().PgConn().Exec(context.Background(), sql).ReadAll(); err != nil {
		t.Fatalf("sql failed: %v", err)
	}
}

// ensureDataAPIRoles creates stand-ins for the roles Supabase's Data API uses, so the
// setup script has something to revoke from.
func ensureDataAPIRoles(t *testing.T, admin *pgxpool.Pool) {
	t.Helper()
	runSQL(t, admin, `
		DO $$
		DECLARE r text;
		BEGIN
		  FOREACH r IN ARRAY ARRAY['anon', 'authenticated', 'service_role'] LOOP
		    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = r) THEN
		      EXECUTE format('CREATE ROLE %I NOLOGIN', r);
		    END IF;
		  END LOOP;
		END $$;`)
}

func applySetup(t *testing.T, admin *pgxpool.Pool, password string) {
	t.Helper()
	raw, err := os.ReadFile(backendPath("db", "setup", "01_manager_schema.sql"))
	if err != nil {
		t.Fatalf("the setup script is missing: %v", err)
	}
	if !strings.Contains(string(raw), passwordToken) {
		t.Fatalf("the setup script must contain the %s placeholder", passwordToken)
	}
	runSQL(t, admin, strings.ReplaceAll(string(raw), passwordToken, password))
}

// appURL is the connection string of the application role. It carries no search_path:
// the role's own default must be enough.
func appURL(admin *pgxpool.Pool, password string) string {
	cc := admin.Config().ConnConfig
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(appRole, password),
		Host:     net.JoinHostPort(cc.Host, strconv.Itoa(int(cc.Port))),
		Path:     "/" + cc.Database,
		RawQuery: "sslmode=disable",
	}
	return u.String()
}

// migrate runs the real golang-migrate tool, exactly as the runbook does.
func migrate(t *testing.T, databaseURL, password string, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	full := append([]string{"run", "-tags", "postgres", migrateModule,
		"-path", backendPath("db", "migrations"), "-database", databaseURL}, args...)
	out, err := exec.CommandContext(ctx, "go", full...).CombinedOutput()
	text := strings.ReplaceAll(string(out), password, "***")
	if err != nil {
		t.Fatalf("migrate %v failed: %v\n%s", args, err, text)
	}
	return text
}

type rehearsal struct {
	admin    *pgxpool.Pool
	password string
	url      string
}

// rehearse prepares a fresh database the way the runbook prepares Supabase.
func rehearse(t *testing.T) rehearsal {
	t.Helper()
	admin := pgtest.NewDatabase(t)
	ensureDataAPIRoles(t, admin)
	password := randomPassword(t)
	applySetup(t, admin, password)
	r := rehearsal{admin: admin, password: password, url: appURL(admin, password)}
	migrate(t, r.url, password, "up")
	return r
}

func names(t *testing.T, pool *pgxpool.Pool, query string, args ...any) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(), query, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		out = append(out, n)
	}
	return out
}

// B1: the setup script holds the password only as a placeholder, never a real value.
func TestB1_SetupScriptHoldsOnlyThePasswordPlaceholder(t *testing.T) {
	raw, err := os.ReadFile(backendPath("db", "setup", "01_manager_schema.sql"))
	if err != nil {
		t.Fatalf("the setup script is missing: %v", err)
	}
	if !strings.Contains(string(raw), passwordToken) {
		t.Errorf("the script must use the %s placeholder", passwordToken)
	}
	literal := regexp.MustCompile(`(?i)password\s+'([^']*)'`)
	for _, m := range literal.FindAllStringSubmatch(string(raw), -1) {
		if !strings.Contains(m[1], "__") {
			t.Errorf("a literal password appears in the script")
		}
	}
}

// B1: the script can be run again (a failed attempt must be recoverable); the second run
// changes the password.
func TestB1_SetupScriptCanRunTwiceAndTheLastPasswordWins(t *testing.T) {
	admin := pgtest.NewDatabase(t)
	ensureDataAPIRoles(t, admin)
	first, second := randomPassword(t), randomPassword(t)

	applySetup(t, admin, first)
	applySetup(t, admin, second)

	if conn, err := pgx.Connect(context.Background(), appURL(admin, second)); err != nil {
		t.Fatalf("the new password must work: %v", err)
	} else {
		conn.Close(context.Background())
	}
	if conn, err := pgx.Connect(context.Background(), appURL(admin, first)); err == nil {
		conn.Close(context.Background())
		t.Fatal("the old password must stop working")
	}
}

// B1: the real migrate tool, run as the application role, puts everything in schema
// manager and nothing in public.
func TestB1_MigrationsLandInTheManagerSchemaOnly(t *testing.T) {
	r := rehearse(t)

	got := names(t, r.admin, `select table_name from information_schema.tables
		where table_schema = 'manager' and table_type = 'BASE TABLE' order by 1`)
	want := []string{"fixed_costs", "ingredient_price_history", "ingredients", "operational_expenses",
		"quotations", "recipe_ingredients", "recipes", "sale_ingredients", "sales", "schema_migrations", "users"}
	if !slices.Equal(got, want) {
		t.Errorf("tables in manager = %v, want %v", got, want)
	}

	if inPublic := names(t, r.admin, `select table_name from information_schema.tables where table_schema = 'public'`); len(inPublic) != 0 {
		t.Errorf("public must stay empty, found tables %v", inPublic)
	}
	enums := names(t, r.admin, `select n.nspname || '.' || t.typname from pg_type t
		join pg_namespace n on n.oid = t.typnamespace
		where t.typtype = 'e' and n.nspname in ('public', 'manager') order by 1`)
	if !slices.Equal(enums, []string{"manager.quotation_status"}) {
		t.Errorf("enums = %v, want only manager.quotation_status", enums)
	}

	var version int
	var dirty bool
	if err := r.admin.QueryRow(context.Background(), `select version, dirty from manager.schema_migrations`).Scan(&version, &dirty); err != nil {
		t.Fatal(err)
	}
	if version != 15 || dirty {
		t.Errorf("schema_migrations = version %d dirty %v, want 15 and clean", version, dirty)
	}
}

// B2: row level security is on for every table of the schema, with no policies.
func TestB2_EveryManagerTableHasRLSAndNoPolicies(t *testing.T) {
	r := rehearse(t)

	off := names(t, r.admin, `select c.relname from pg_class c join pg_namespace n on n.oid = c.relnamespace
		where n.nspname = 'manager' and c.relkind = 'r' and not c.relrowsecurity order by 1`)
	if len(off) != 0 {
		t.Errorf("tables without RLS: %v", off)
	}
	var policies int
	if err := r.admin.QueryRow(context.Background(), `select count(*) from pg_policies where schemaname = 'manager'`).Scan(&policies); err != nil {
		t.Fatal(err)
	}
	if policies != 0 {
		t.Errorf("found %d policies, want none", policies)
	}
}

// B2: the RLS migration can be reverted with the real tool.
func TestB2_RLSMigrationIsReversible(t *testing.T) {
	r := rehearse(t)

	migrate(t, r.url, r.password, "down", "1")

	var version int
	if err := r.admin.QueryRow(context.Background(), `select version from manager.schema_migrations`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 14 {
		t.Errorf("version after down 1 = %d, want 14", version)
	}
	on := names(t, r.admin, `select c.relname from pg_class c join pg_namespace n on n.oid = c.relnamespace
		where n.nspname = 'manager' and c.relkind = 'r' and c.relname <> 'schema_migrations' and c.relrowsecurity order by 1`)
	if len(on) != 0 {
		t.Errorf("tables still with RLS after the down: %v", on)
	}
}

// B3: the application role has the least privilege it needs.
func TestB3_ManagerAppHasLeastPrivilege(t *testing.T) {
	r := rehearse(t)
	ctx := context.Background()

	var super, createRole, createDB, bypassRLS bool
	err := r.admin.QueryRow(ctx, `select rolsuper, rolcreaterole, rolcreatedb, rolbypassrls from pg_roles where rolname = $1`, appRole).
		Scan(&super, &createRole, &createDB, &bypassRLS)
	if err != nil {
		t.Fatal(err)
	}
	if super || createRole || createDB || bypassRLS {
		t.Errorf("manager_app has extra attributes: super=%v createrole=%v createdb=%v bypassrls=%v", super, createRole, createDB, bypassRLS)
	}

	conn, err := pgx.Connect(ctx, r.url)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)

	var searchPath string
	if err := conn.QueryRow(ctx, `show search_path`).Scan(&searchPath); err != nil {
		t.Fatal(err)
	}
	if searchPath != "manager" {
		t.Errorf("search_path = %q, want manager", searchPath)
	}

	_, err = conn.Exec(ctx, `create table public.should_not_exist (a int)`)
	var pgErr *pgconn.PgError
	if err == nil || !asPgError(err, &pgErr) || pgErr.Code != "42501" {
		t.Errorf("creating a table in public must fail with permission denied, got %v", err)
	}

	// The Data API roles of Supabase must not reach the schema. Locally they are stand-ins,
	// so this check is weak; docs/despliegue.md has the query to run on the real project.
	for _, role := range []string{"anon", "authenticated", "service_role"} {
		var schemaUsage, tableSelect bool
		err := r.admin.QueryRow(ctx, `select has_schema_privilege($1, 'manager', 'USAGE'), has_table_privilege($1, 'manager.users', 'SELECT')`, role).
			Scan(&schemaUsage, &tableSelect)
		if err != nil {
			t.Fatal(err)
		}
		if schemaUsage || tableSelect {
			t.Errorf("role %s reaches schema manager (usage=%v select=%v)", role, schemaUsage, tableSelect)
		}
	}
}

func asPgError(err error, target **pgconn.PgError) bool {
	for e := err; e != nil; {
		if pe, ok := e.(*pgconn.PgError); ok {
			*target = pe
			return true
		}
		u, ok := e.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		e = u.Unwrap()
	}
	return false
}

// B5: the first-user procedure of the runbook works end to end: hash a password with the
// tool, insert the user as the application role, log in through the real API.
func TestB5_FirstUserCanLogInThroughTheRealAPI(t *testing.T) {
	r := rehearse(t)
	ctx := context.Background()
	const email, userPassword = "first.user@example.test", "a-throwaway-password-1"

	hasher := exec.Command("go", "run", "./cmd/hashpassword")
	hasher.Dir = backendPath()
	hasher.Stdin = strings.NewReader(userPassword + "\n")
	var hash, hashErr bytes.Buffer
	hasher.Stdout, hasher.Stderr = &hash, &hashErr
	if err := hasher.Run(); err != nil {
		t.Fatalf("hashpassword failed: %v\n%s", err, hashErr.String())
	}

	appPool, err := pgxpool.New(ctx, r.url)
	if err != nil {
		t.Fatal(err)
	}
	defer appPool.Close()
	if _, err := appPool.Exec(ctx, `insert into users (name, email, password, role) values ('Primera Usuaria', $1, $2, 'owner')`,
		email, strings.TrimSpace(hash.String())); err != nil {
		t.Fatalf("inserting the first user as manager_app: %v", err)
	}

	pool, err := kernel.Open(ctx, r.url, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	h := app.New(config.Config{Port: "8080", DatabaseURL: "unused", JWTSecret: jwtSecret, DBMaxConns: 2}, pool, nil)

	do := func(method, path, token string, body any) (int, []byte) {
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(method, path, bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code, rec.Body.Bytes()
	}

	if status, _ := do("POST", "/auth/login", "", map[string]string{"email": email, "password": "not-the-password"}); status != http.StatusUnauthorized {
		t.Errorf("wrong password: status = %d, want 401", status)
	}
	status, raw := do("POST", "/auth/login", "", map[string]string{"email": email, "password": userPassword})
	if status != http.StatusOK {
		t.Fatalf("login: status = %d, want 200: %s", status, raw)
	}
	var login struct{ Token string }
	if err := json.Unmarshal(raw, &login); err != nil || login.Token == "" {
		t.Fatalf("no token in the login response: %v %s", err, raw)
	}

	status, raw = do("POST", "/ingredients", login.Token, map[string]any{
		"name": "Azúcar", "brand": "Marca", "default_unit": "gr",
		"package_size": 1000, "package_price": 1500, "stock_quantity": 1000, "alert_threshold": 100,
	})
	if status != http.StatusCreated {
		t.Fatalf("creating an ingredient with the new token: status = %d, want 201: %s", status, raw)
	}
}
