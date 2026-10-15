package handler

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// Vercel compiles this directory under its own import path (handler/api), outside this
// module's tree, so Go refuses any import of an internal package ("use of internal package
// ... not allowed"). The entry point may only import public packages of the module.
// Observed in the first real build (2026-10-15); reproduced with a throwaway module that
// replaces this one.
func TestEntrypointImportsNoInternalPackages(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "index.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if strings.Contains(path, "/internal/") || strings.HasSuffix(path, "/internal") {
			t.Errorf("api/index.go imports %s: Vercel cannot build it; go through a public package such as pkg/serverless", path)
		}
	}
}
