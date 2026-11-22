package middleware

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	chimw "github.com/go-chi/chi/v5/middleware"
)

// Review of 2026-11-22: every request gets an identifier that follows it into the log, so a failure can be
// traced from what the person saw to the line of the log.

var idShape = regexp.MustCompile(`^[0-9a-f]{16,32}$`)

func serve(t *testing.T, header string) (id string, inContext string) {
	t.Helper()
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inContext = chimw.GetReqID(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest("GET", "/health", nil)
	if header != "" {
		req.Header.Set("X-Request-Id", header)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Header().Get("X-Request-Id"), inContext
}

func TestR1_EveryResponseCarriesADifferentIdentifierThatTheLoggerCanSee(t *testing.T) {
	a, inA := serve(t, "")
	b, _ := serve(t, "")
	if !idShape.MatchString(a) || !idShape.MatchString(b) {
		t.Fatalf("identifiers %q and %q are not hexadecimal of 16 to 32 characters", a, b)
	}
	if a == b {
		t.Error("two requests got the same identifier")
	}
	if inA != a {
		t.Errorf("the context carries %q but the response says %q (the logger reads the context)", inA, a)
	}
}

func TestR2_AnIdentifierSentByTheClientIsNotTrusted(t *testing.T) {
	for _, sent := range []string{"abc", "x\nINJECTED line", "a-very-long-" + string(make([]byte, 0)) + "0123456789012345678901234567890123456789"} {
		id, _ := serve(t, sent)
		if id == sent || !idShape.MatchString(id) {
			t.Errorf("a client value %q ended up as the identifier %q", sent, id)
		}
	}
}
