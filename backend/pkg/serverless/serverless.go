// Package serverless is the public door of the application for the Vercel entry point.
//
// Vercel compiles api/ outside this module's import tree, where Go forbids importing
// internal packages. This package lives inside the tree, so it may import internal/app,
// and api/ only needs to import this one.
package serverless

import (
	"net/http"

	"github.com/PipeM113/bakery-manager/internal/app"
)

// NewLazyHandler builds the application on the first request from the environment read
// through getenv, and reuses it afterwards. See app.NewLazyHandler.
func NewLazyHandler(getenv func(string) string) http.HandlerFunc {
	return app.NewLazyHandler(getenv)
}
