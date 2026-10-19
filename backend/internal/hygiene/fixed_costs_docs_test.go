package hygiene

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Sprint 6, rule F12: fixed costs with history are written down.
func TestF12_FixedCostsAreDocumented(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "docs", "costos-fijos.md"))
	if err != nil {
		t.Fatalf("docs/costos-fijos.md is missing: %v", err)
	}
	for _, want := range []string{"fixed_cost_versions", "valid_from", "ended_from", "effective_from", "?month="} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("docs/costos-fijos.md should mention %q", want)
		}
	}
	deploy, err := os.ReadFile(filepath.Join(root, "docs", "despliegue.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(deploy), "19") || !strings.Contains(string(deploy), "fixed_cost_versions") {
		t.Error("docs/despliegue.md should explain migration 19 and its check")
	}
	glossary, err := os.ReadFile(filepath.Join(root, "docs", "glosario-dinero.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(glossary), "Costo fijo") {
		t.Error("the glossary should define \"Costo fijo\" (amount by month, history kept)")
	}
}
