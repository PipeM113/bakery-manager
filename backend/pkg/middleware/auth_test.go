package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/PipeM113/bakery-manager/pkg/middleware"
	"github.com/golang-jwt/jwt/v5"
)

const secret = "0123456789abcdef0123456789abcdef"

func token(t *testing.T, signWith string, exp time.Time) string {
	t.Helper()
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  "user-1",
		"role": "owner",
		"exp":  exp.Unix(),
	}).SignedString([]byte(signWith))
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func call(t *testing.T, authHeader string) (status int, body string, reached bool, got middleware.UserClaims) {
	t.Helper()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		got, _ = r.Context().Value(middleware.UserKey).(middleware.UserClaims)
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rec := httptest.NewRecorder()

	middleware.Auth(secret)(next).ServeHTTP(rec, req)

	return rec.Code, rec.Body.String(), reached, got
}

// AC6: a token signed with the configured secret passes and carries the claims.
func TestAC6_ValidTokenReachesTheHandlerWithItsClaims(t *testing.T) {
	status, _, reached, claims := call(t, "Bearer "+token(t, secret, time.Now().Add(time.Hour)))

	if status != http.StatusOK || !reached {
		t.Fatalf("status = %d, reached = %v; want 200 and reached", status, reached)
	}
	if claims.ID != "user-1" || claims.Role != "owner" {
		t.Fatalf("claims = %+v, want user-1 / owner", claims)
	}
}

// AC6: anything else is a 401 and never reaches the handler.
func TestAC6_InvalidCredentialsAreRejected(t *testing.T) {
	cases := []struct{ name, header string }{
		{"no header", ""},
		{"not a bearer scheme", "Basic dXNlcjpwYXNz"},
		{"garbage token", "Bearer not.a.token"},
		{"signed with another secret", "Bearer " + token(t, "a-completely-different-secret-value!!", time.Now().Add(time.Hour))},
		{"signed with an empty secret", "Bearer " + token(t, "", time.Now().Add(time.Hour))},
		{"expired", "Bearer " + token(t, secret, time.Now().Add(-time.Minute))},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			status, body, reached, _ := call(t, c.header)

			if status != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", status)
			}
			if reached {
				t.Error("the handler must not run")
			}
			if !strings.Contains(body, "no autorizado") {
				t.Errorf("body = %q, want the JSON error", body)
			}
		})
	}
}

// AC6: the middleware cannot be built without a secret (a programming error, not a request error).
func TestAC6_AuthWithAnEmptySecretPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic")
		}
	}()
	middleware.Auth("")
}
