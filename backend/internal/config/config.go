// Package config reads and validates the settings of the service from environment
// variables. Errors name the variable at fault but never repeat its value, because
// several of them are secrets.
package config

import (
	"errors"
	"fmt"
	"net/url"
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
	Port          string
	DatabaseURL   string
	JWTSecret     string
	CORSOrigins   []string
	DBMaxConns    int32
	CloudinaryURL string // optional: without it photo upload is disabled
}

// Load reads the settings through getenv (os.Getenv in production) and reports every
// problem it finds at once.
func Load(getenv func(string) string) (Config, error) {
	var problems []error

	rawSecret := getenv("JWT_SECRET")
	cfg := Config{
		Port:          defaultPort,
		DatabaseURL:   strings.TrimSpace(getenv("DATABASE_URL")),
		JWTSecret:     strings.TrimSpace(rawSecret),
		CloudinaryURL: strings.TrimSpace(getenv("CLOUDINARY_URL")),
		DBMaxConns:    defaultDBMaxConns,
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
