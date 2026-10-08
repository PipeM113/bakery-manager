package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserKey contextKey = "user"

type UserClaims struct {
	ID   string
	Role string
}

// Auth returns a middleware that accepts only requests with a Bearer token signed with
// secret. The secret is validated once at startup (see internal/config), so an empty one
// here is a programming error.
func Auth(secret string) func(http.Handler) http.Handler {
	if secret == "" {
		panic("middleware.Auth: the JWT secret must not be empty")
	}
	key := []byte(secret)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				unauthorized(w)
				return
			}

			tokenStr := strings.TrimPrefix(header, "Bearer ")
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				return key, nil
			})
			if err != nil || !token.Valid {
				unauthorized(w)
				return
			}

			claims := token.Claims.(jwt.MapClaims)
			ctx := context.WithValue(r.Context(), UserKey, UserClaims{
				ID:   claims["sub"].(string),
				Role: claims["role"].(string),
			})

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"no autorizado"}`))
}
