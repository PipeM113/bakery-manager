package hygiene

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Sprint 11, rule N8: how the two accounts share the business is written down.
func TestN8_AccessIsDocumented(t *testing.T) {
	root := repoRoot(t)
	read := func(parts ...string) string {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(append([]string{root}, parts...)...))
		if err != nil {
			t.Fatalf("%s is missing: %v", filepath.Join(parts...), err)
		}
		return string(raw)
	}
	access := read("docs", "acceso.md")
	for _, want := range []string{"created_by", "negocio único", "borra", "hashpassword", "contraseña"} {
		if !strings.Contains(access, want) {
			t.Errorf("docs/acceso.md should mention %q", want)
		}
	}
	adrs, _ := filepath.Glob(filepath.Join(root, "docs", "adr", "002-*.md"))
	if len(adrs) != 1 {
		t.Fatalf("expected exactly one docs/adr/002-*.md, found %v", adrs)
	}
	adr, _ := os.ReadFile(adrs[0])
	if !strings.Contains(string(adr), "Aceptado") || !strings.Contains(string(adr), "created_by") {
		t.Error("ADR-002 should be accepted and explain created_by")
	}
	deploy := read("docs", "despliegue.md")
	for _, want := range []string{"cuenta de tu mamá", "hashpassword", "cambiar una contraseña"} {
		if !strings.Contains(deploy, want) {
			t.Errorf("docs/despliegue.md should explain %q", want)
		}
	}
}
