package hygiene

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Sprint 9, rule R9: how the screens adapt to a phone is written down.
func TestR9_ResponsiveDesignIsDocumented(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "docs", "diseno.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, "Diseño responsivo") {
		t.Fatal("docs/diseno.md should have a section \"Diseño responsivo\"")
	}
	for _, want := range []string{"360", "390", "768", "1280", "tarjetas", "44 px", "test:responsive"} {
		if !strings.Contains(text, want) {
			t.Errorf("the responsive section should mention %q", want)
		}
	}
}
