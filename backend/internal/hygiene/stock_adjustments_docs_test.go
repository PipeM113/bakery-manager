package hygiene

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Sprint 8, rule A11: stock adjustments and waste are written down.
func TestA11_StockAdjustmentsAreDocumented(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "docs", "ajustes-de-stock.md"))
	if err != nil {
		t.Fatalf("docs/ajustes-de-stock.md is missing: %v", err)
	}
	for _, want := range []string{"stock_adjustments", "merma", "correccion", "otro", "loss_amount", "total_waste_cost", "counted_quantity"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("docs/ajustes-de-stock.md should mention %q", want)
		}
	}
	glossary, err := os.ReadFile(filepath.Join(root, "docs", "glosario-dinero.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(glossary), "Merma") {
		t.Error("the glossary should define \"Merma\" (a loss that counts against the month)")
	}
}
