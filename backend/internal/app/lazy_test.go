package app_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/PipeM113/bakery-manager/internal/app"
)

// countingEnv reads from a map and counts how often each variable is read.
type countingEnv struct {
	mu     sync.Mutex
	values map[string]string
	reads  map[string]int
}

func newCountingEnv(values map[string]string) *countingEnv {
	return &countingEnv{values: values, reads: map[string]int{}}
}

func (c *countingEnv) get(key string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reads[key]++
	return c.values[key]
}

func get(h http.Handler, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

// AC7: /health answers 200 even when the database is unreachable, and the application
// is built once (on the first request), not on every request.
func TestAC7_LazyHandlerServesHealthWithoutADatabaseAndBuildsOnce(t *testing.T) {
	env := newCountingEnv(map[string]string{
		"DATABASE_URL": "postgres://u:p@127.0.0.1:1/x", // nothing listens on port 1
		"JWT_SECRET":   goodSecret,
	})
	h := app.NewLazyHandler(env.get)

	for i := 0; i < 3; i++ {
		rec := get(h, "/health")
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want 200", i, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
			t.Fatalf("request %d: body = %q", i, rec.Body.String())
		}
	}
	if env.reads["DATABASE_URL"] != 1 {
		t.Errorf("DATABASE_URL was read %d times, want 1 (built once)", env.reads["DATABASE_URL"])
	}
}

// AC7: with an invalid environment every request gets a generic 500: no variable names,
// no values, nothing about the internals.
func TestAC7_LazyHandlerWithInvalidEnvironmentAnswersAGeneric500(t *testing.T) {
	env := newCountingEnv(map[string]string{
		"DATABASE_URL": "postgres://user:s3cr3t-pass@db.example.com:5432/x",
		// JWT_SECRET is missing
	})
	h := app.NewLazyHandler(env.get)

	for _, path := range []string{"/health", "/ingredients"} {
		rec := get(h, path)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("%s: status = %d, want 500", path, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Errorf("%s: Content-Type = %q, want JSON", path, ct)
		}
		body := rec.Body.String()
		if !strings.Contains(body, `"error"`) {
			t.Errorf("%s: body = %q, want a JSON error", path, body)
		}
		for _, leaked := range []string{"JWT_SECRET", "DATABASE_URL", "s3cr3t-pass", "postgres://"} {
			if strings.Contains(body, leaked) {
				t.Errorf("%s: the body leaks %q: %s", path, leaked, body)
			}
		}
	}
	if env.reads["DATABASE_URL"] != 1 {
		t.Errorf("DATABASE_URL was read %d times, want 1 (the failure is remembered)", env.reads["DATABASE_URL"])
	}
}
