package config_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/PipeM113/bakery-manager/internal/config"
)

// A 32-character secret: the minimum accepted. Real ones come from `openssl rand`.
const goodSecret = "0123456789abcdef0123456789abcdef"

func env(m map[string]string) func(string) string {
	return func(key string) string { return m[key] }
}

func validEnv() map[string]string {
	return map[string]string{
		"DATABASE_URL": "postgres://u:p@127.0.0.1:5432/x",
		"JWT_SECRET":   goodSecret,
	}
}

// AC1: a minimal valid environment loads, with documented defaults.
func TestAC1_LoadAcceptsAMinimalValidEnvironment(t *testing.T) {
	cfg, err := config.Load(env(validEnv()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want the default 8080", cfg.Port)
	}
	if cfg.DBMaxConns != 4 {
		t.Errorf("DBMaxConns = %d, want the default 4", cfg.DBMaxConns)
	}
	if len(cfg.CORSOrigins) != 0 {
		t.Errorf("CORSOrigins = %v, want none (no cross-origin access by default)", cfg.CORSOrigins)
	}
	if cfg.JWTSecret != goodSecret || cfg.DatabaseURL != validEnv()["DATABASE_URL"] {
		t.Errorf("secret or database url were not copied into the config")
	}
}

// AC1: DATABASE_URL and JWT_SECRET are mandatory; the error names the variable.
func TestAC1_LoadRequiresDatabaseURLAndJWTSecret(t *testing.T) {
	for _, missing := range []string{"DATABASE_URL", "JWT_SECRET"} {
		t.Run(missing, func(t *testing.T) {
			e := validEnv()
			delete(e, missing)

			_, err := config.Load(env(e))

			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), missing) {
				t.Errorf("the error should name %s: %v", missing, err)
			}
		})
	}
}

// AC1: the JWT secret needs at least 32 characters.
func TestAC1_JWTSecretMustHaveAtLeast32Characters(t *testing.T) {
	cases := []struct {
		name, secret string
		wantErr      bool
	}{
		{"31 characters", strings.Repeat("a", 31), true},
		{"32 characters", strings.Repeat("a", 32), false},
		{"64 characters", strings.Repeat("a", 64), false},
		{"whitespace only", strings.Repeat(" ", 40), true},
		{"the placeholder of .env.example", "your_long_random_secret_here", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := validEnv()
			e["JWT_SECRET"] = c.secret

			_, err := config.Load(env(e))

			if (err != nil) != c.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, c.wantErr)
			}
			if err != nil {
				if !strings.Contains(err.Error(), "JWT_SECRET") || !strings.Contains(err.Error(), "32") {
					t.Errorf("the error should name JWT_SECRET and the minimum 32: %v", err)
				}
				if c.secret != "" && strings.Contains(err.Error(), strings.TrimSpace(c.secret)) {
					t.Errorf("the error repeats the secret: %v", err)
				}
			}
		})
	}
}

// AC1: no error may repeat a secret value, whichever setting is wrong.
func TestAC1_ErrorsNeverRepeatSecretValues(t *testing.T) {
	e := validEnv()
	e["DATABASE_URL"] = "postgres://user:s3cr3t-pass@db.example.com:5432/x"
	e["DB_MAX_CONNS"] = "not-a-number"

	_, err := config.Load(env(e))

	if err == nil {
		t.Fatal("expected an error")
	}
	for _, leaked := range []string{"s3cr3t-pass", goodSecret} {
		if strings.Contains(err.Error(), leaked) {
			t.Fatalf("the error leaks a secret value: %v", err)
		}
	}
}

// AC1: DB_MAX_CONNS must be a positive integer.
func TestAC1_DBMaxConns(t *testing.T) {
	cases := []struct {
		value   string
		want    int32
		wantErr bool
	}{
		{"2", 2, false},
		{"10", 10, false},
		{"0", 0, true},
		{"-1", 0, true},
		{"abc", 0, true},
		{"1.5", 0, true},
	}
	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			e := validEnv()
			e["DB_MAX_CONNS"] = c.value

			cfg, err := config.Load(env(e))

			if (err != nil) != c.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, c.wantErr)
			}
			if err == nil && cfg.DBMaxConns != c.want {
				t.Fatalf("DBMaxConns = %d, want %d", cfg.DBMaxConns, c.want)
			}
			if err != nil && !strings.Contains(err.Error(), "DB_MAX_CONNS") {
				t.Errorf("the error should name DB_MAX_CONNS: %v", err)
			}
		})
	}
}

// AC2: CORS origins are exact, normalized, and never a wildcard.
func TestAC2_CORSOriginsAreNormalized(t *testing.T) {
	e := validEnv()
	e["CORS_ORIGINS"] = " https://manager.example.com/ , http://localhost:5173,, "

	cfg, err := config.Load(env(e))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"https://manager.example.com", "http://localhost:5173"}
	if !slices.Equal(cfg.CORSOrigins, want) {
		t.Fatalf("CORSOrigins = %v, want %v", cfg.CORSOrigins, want)
	}
}

func TestAC2_CORSOriginsRejectWildcardsAndNonOrigins(t *testing.T) {
	cases := []struct{ name, value string }{
		{"star", "*"},
		{"star among valid origins", "https://manager.example.com,*"},
		{"wildcard subdomain", "https://*.vercel.app"},
		{"no scheme", "manager.example.com"},
		{"has a path", "https://manager.example.com/app"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := validEnv()
			e["CORS_ORIGINS"] = c.value

			_, err := config.Load(env(e))

			if err == nil {
				t.Fatalf("expected %q to be rejected", c.value)
			}
			if !strings.Contains(err.Error(), "CORS_ORIGINS") {
				t.Errorf("the error should name CORS_ORIGINS: %v", err)
			}
		})
	}
}
