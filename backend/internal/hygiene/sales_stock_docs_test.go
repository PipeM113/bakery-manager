package hygiene

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Sprint 7, rule V9: what a sale takes from the stock is written down.
func TestV9_SalesAndStockAreDocumented(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "docs", "ventas-y-stock.md"))
	if err != nil {
		t.Fatalf("docs/ventas-y-stock.md is missing: %v", err)
	}
	for _, want := range []string{"sold_as", "stock_deducted", "stock_shortfalls", "portion", "kg"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("docs/ventas-y-stock.md should mention %q", want)
		}
	}
	glossary, err := os.ReadFile(filepath.Join(root, "docs", "glosario-dinero.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(glossary), "Porción") {
		t.Error("the glossary should define \"Porción\" (a sale of portions takes 1/rendimiento of the recipe)")
	}
}
