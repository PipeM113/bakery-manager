package hygiene

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Sprint 10, rule O11: orders are written down.
func TestO11_OrdersAreDocumented(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "docs", "pedidos.md"))
	if err != nil {
		t.Fatalf("docs/pedidos.md is missing: %v", err)
	}
	for _, want := range []string{"orders", "order_id", "line_total", "total_price", "reparto", "anular"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("docs/pedidos.md should mention %q", want)
		}
	}
	glossary, err := os.ReadFile(filepath.Join(root, "docs", "glosario-dinero.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(glossary), "Pedido") {
		t.Error("the glossary should define \"Pedido\" (several products, one price)")
	}
}
