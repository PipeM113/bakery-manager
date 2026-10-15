package app_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PipeM113/bakery-manager/internal/app"
	"github.com/PipeM113/bakery-manager/internal/config"
)

const goodSecret = "0123456789abcdef0123456789abcdef"

// router builds the API with no database: these tests only exercise middleware.
func router(origins []string) http.Handler {
	return app.New(config.Config{
		Port:        "8080",
		DatabaseURL: "postgres://u:p@127.0.0.1:1/x",
		JWTSecret:   goodSecret,
		DBMaxConns:  4,
		CORSOrigins: origins,
	}, nil, nil)
}

func preflight(h http.Handler, origin string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodOptions, "/ingredients", nil)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// AC3: a listed origin gets CORS headers, and credentials are never allowed.
func TestAC3_AllowedOriginGetsCORSHeaders(t *testing.T) {
	h := router([]string{"https://manager.example.com", "http://localhost:5173"})

	rec := preflight(h, "https://manager.example.com")

	if rec.Code >= 300 {
		t.Fatalf("preflight status = %d, want 2xx", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://manager.example.com" {
		t.Errorf("Allow-Origin = %q, want the request origin", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, "POST") {
		t.Errorf("Allow-Methods = %q, want it to include POST", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Errorf("Allow-Credentials = %q, must never be sent: the token travels in a header", got)
	}
}

// AC3: an origin that is not listed gets no CORS headers at all.
func TestAC3_UnlistedOriginGetsNoCORSHeaders(t *testing.T) {
	h := router([]string{"https://manager.example.com"})

	for _, origin := range []string{
		"https://evil.example.com",
		"https://manager.example.com.evil.example.com",
		"http://manager.example.com", // same host, other scheme
	} {
		rec := preflight(h, origin)
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("origin %q got Allow-Origin %q, want none", origin, got)
		}
	}
}

// AC3: with no configured origins there is no cross-origin access.
func TestAC3_NoConfiguredOriginsMeansNoCrossOriginAccess(t *testing.T) {
	rec := preflight(router(nil), "https://manager.example.com")

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Allow-Origin = %q, want none", got)
	}
}

// AC3: simple requests from a listed origin carry the header too, and /health needs no token.
func TestAC3_HealthAnswersWithoutAuthenticationAndWithCORS(t *testing.T) {
	h := router([]string{"https://manager.example.com"})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "https://manager.example.com")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://manager.example.com" {
		t.Errorf("Allow-Origin = %q", got)
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Errorf("body = %q", rec.Body.String())
	}
}
