package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// AC7: the Vercel entry point. The environment is read on the first request, so the
// test sets it before calling. /health must not need the database (nothing listens on port 1).
func TestAC7_VercelEntrypointServesHealth(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@127.0.0.1:1/x")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("CORS_ORIGINS", "")

	// Vercel may hand the function the original path or one prefixed with /api; both work.
	for _, path := range []string{"/health", "/api/health"} {
		rec := httptest.NewRecorder()

		Handler(rec, httptest.NewRequest(http.MethodGet, path, nil))

		if rec.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200", path, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
			t.Errorf("%s: body = %q", path, rec.Body.String())
		}
	}
}
