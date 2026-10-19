package middleware

import (
	"net/http"

	"github.com/PipeM113/bakery-manager/pkg/httputil"
)

// MaxBody refuses requests whose body is bigger than limit, with 413 when the size is declared
// (Content-Length) and never reading more than limit bytes otherwise (the handler then fails to
// decode and answers 400). skip lets a route keep its own limit, like the photo upload.
func MaxBody(limit int64, skip func(*http.Request) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body == nil || (skip != nil && skip(r)) {
				next.ServeHTTP(w, r)
				return
			}
			if r.ContentLength > limit {
				httputil.JSONError(w, "el cuerpo de la solicitud es demasiado grande", http.StatusRequestEntityTooLarge)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}
