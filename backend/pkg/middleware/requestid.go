package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"

	chimw "github.com/go-chi/chi/v5/middleware"
)

// RequestID gives every request its own identifier: it goes in the X-Request-Id header of the response (so a
// person can read it out when something fails) and in the context under the key the chi logger prints, so the
// line of the log carries it. An identifier sent by the client is ignored: it could be forged or carry a line
// break into the log.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b [8]byte
		if _, err := rand.Read(b[:]); err != nil { // the system random source failed: keep serving, with a weaker id
			copy(b[:], "fallback")
		}
		id := hex.EncodeToString(b[:])
		w.Header().Set("X-Request-Id", id)
		ctx := context.WithValue(r.Context(), chimw.RequestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
