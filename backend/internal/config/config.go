// Package config reads and validates the settings of the service from environment
// variables. Errors name the variable at fault but never repeat its value, because
// several of them are secrets.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	defaultPort       = "8080"
	defaultDBMaxConns = 4
	// minSecretLength is a floor, not a guarantee of randomness: HS256 wants 256 random
	// bits, which `openssl rand -base64 48` or `openssl rand -hex 32` produce.
	minSecretLength = 32
)

// Config holds the validated settings. It contains secrets: never log it whole.
type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
	CORSOrigins []string
	DBMaxConns  int32
	// Supabase Storage for recipe photos. Optional as a group: with none of the three set,
	// photo upload is disabled.
	SupabaseURL       string
	SupabaseSecretKey string
	SupabaseBucket    string
	// Business is what the quotation PDF prints about the business. Optional: without it the
	// PDF is refused (the quotations themselves still work).
	Business Business
}

// Business holds the two secrets of the quotation PDF that stay in the environment. The rest of
// the business data (name, phone, bank...) is edited in the app and lives in the database.
type Business struct {
	Account, RUT string
}

// Missing names the environment variables that are not set.
func (b Business) Missing() []string {
	var missing []string
	for _, f := range []struct{ env, value string }{{"BIZ_ACCOUNT", b.Account}, {"BIZ_RUT", b.RUT}} {
		if strings.TrimSpace(f.value) == "" {
			missing = append(missing, f.env)
		}
	}
	return missing
}

// Load reads the settings through getenv (os.Getenv in production) and reports every
// problem it finds at once.
func Load(getenv func(string) string) (Config, error) {
	var problems []error

	rawSecret := getenv("JWT_SECRET")
	cfg := Config{
		Port:        defaultPort,
		DatabaseURL: strings.TrimSpace(getenv("DATABASE_URL")),
		JWTSecret:   strings.TrimSpace(rawSecret),
		DBMaxConns:  defaultDBMaxConns,
		Business: Business{
			Account: strings.TrimSpace(getenv("BIZ_ACCOUNT")),
			RUT:     strings.TrimSpace(getenv("BIZ_RUT")),
		},
	}

	if port := strings.TrimSpace(getenv("PORT")); port != "" {
		cfg.Port = port
	}

	if cfg.DatabaseURL == "" {
		problems = append(problems, errors.New("DATABASE_URL es obligatoria"))
	}

	switch {
	case rawSecret == "":
		problems = append(problems, errors.New("JWT_SECRET es obligatoria"))
	case utf8.RuneCountInString(cfg.JWTSecret) < minSecretLength:
		problems = append(problems, fmt.Errorf("JWT_SECRET debe tener al menos %d caracteres", minSecretLength))
	}

	if raw := strings.TrimSpace(getenv("DB_MAX_CONNS")); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || n < 1 {
			problems = append(problems, errors.New("DB_MAX_CONNS debe ser un entero mayor o igual a 1"))
		} else {
			cfg.DBMaxConns = int32(n)
		}
	}

	origins, err := parseOrigins(getenv("CORS_ORIGINS"))
	if err != nil {
		problems = append(problems, err)
	}
	cfg.CORSOrigins = origins

	if err := loadStorage(getenv, &cfg); err != nil {
		problems = append(problems, err)
	}

	if len(problems) > 0 {
		return Config{}, errors.Join(problems...)
	}
	return cfg, nil
}

// parseOrigins turns "https://a.com/, http://b.com" into ["https://a.com", "http://b.com"].
// Only exact origins are accepted: no wildcards, no paths.
func parseOrigins(raw string) ([]string, error) {
	var origins []string
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		origin, ok := normalizeOrigin(entry)
		if !ok {
			return nil, fmt.Errorf("CORS_ORIGINS contiene un origen inválido (%q): use scheme://host[:puerto], sin comodines ni rutas", entry)
		}
		origins = append(origins, origin)
	}
	return origins, nil
}

func normalizeOrigin(entry string) (string, bool) {
	if strings.Contains(entry, "*") {
		return "", false
	}
	u, err := url.Parse(entry)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", false
	}
	if (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return "", false
	}
	return u.Scheme + "://" + u.Host, true
}

// bucketName follows what Supabase buckets use; it ends up inside URLs.
var bucketName = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,62}$`)

// loadStorage reads SUPABASE_URL, SUPABASE_SECRET_KEY and SUPABASE_STORAGE_BUCKET. They work
// as a group: none disables photos, a partial set is a mistake worth stopping the start for.
func loadStorage(getenv func(string) string, cfg *Config) error {
	vars := []struct{ name, value string }{
		{"SUPABASE_URL", strings.TrimSpace(getenv("SUPABASE_URL"))},
		{"SUPABASE_SECRET_KEY", strings.TrimSpace(getenv("SUPABASE_SECRET_KEY"))},
		{"SUPABASE_STORAGE_BUCKET", strings.TrimSpace(getenv("SUPABASE_STORAGE_BUCKET"))},
	}
	var set, missing []string
	for _, v := range vars {
		if v.value == "" {
			missing = append(missing, v.name)
		} else {
			set = append(set, v.name)
		}
	}
	if len(set) == 0 {
		return nil
	}

	var problems []error
	if len(missing) > 0 {
		problems = append(problems, fmt.Errorf("el almacenamiento de fotos necesita SUPABASE_URL, SUPABASE_SECRET_KEY y SUPABASE_STORAGE_BUCKET juntas; falta: %s", strings.Join(missing, ", ")))
	}
	if raw := vars[0].value; raw != "" {
		origin, ok := normalizeStorageURL(raw)
		if !ok {
			problems = append(problems, errors.New("SUPABASE_URL debe ser el origen https del proyecto, por ejemplo https://<ref>.supabase.co, sin ruta ni credenciales"))
		}
		cfg.SupabaseURL = origin
	}
	cfg.SupabaseSecretKey = vars[1].value
	if bucket := vars[2].value; bucket != "" {
		if !bucketName.MatchString(bucket) {
			problems = append(problems, errors.New("SUPABASE_STORAGE_BUCKET solo admite minúsculas, números, guion y guion bajo"))
		}
		cfg.SupabaseBucket = bucket
	}
	return errors.Join(problems...)
}

func normalizeStorageURL(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return "", false
	}
	if (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return "", false
	}
	return "https://" + u.Host, true
}
