// Package handler is the Vercel entry point: the whole API runs as one function.
package handler

import (
	"net/http"
	"os"
	"strings"

	"github.com/PipeM113/bakery-manager/internal/app"
)

// The application is built on the first request from the environment variables of the
// Vercel project, and reused while the instance lives.
var serve = app.NewLazyHandler(os.Getenv)

// Handler serves every route. Depending on how the rewrite hands over the path, it may
// arrive as /ingredients or as /api/ingredients; both reach the same route.
func Handler(w http.ResponseWriter, r *http.Request) {
	if p := r.URL.Path; p == "/api" || strings.HasPrefix(p, "/api/") {
		r.URL.Path = strings.TrimPrefix(p, "/api")
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}
		r.URL.RawPath = ""
	}
	serve(w, r)
}
