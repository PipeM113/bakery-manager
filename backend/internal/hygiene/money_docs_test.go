package hygiene

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Sprint 2a, rule P9: the words for money are defined once, in the repository.
func TestP9_GlossaryDefinesMarkupAndMargin(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "docs", "glosario-dinero.md"))
	if err != nil {
		t.Fatalf("docs/glosario-dinero.md is missing: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Recargo sobre costo", // the recipe markup, a fraction of the cost
		"Margen de ganancia",  // the dashboard figure, a percentage of sales
		"margin_pct",          // the unchanged name on the wire and in the database
		"mano de obra",        // an estimate used for pricing, not a paid wage
		"redondeo",            // when and how amounts are rounded
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the glossary should mention %q", want)
		}
	}
}

// Sprint 2b: expense categories and whole pesos are written down too.
func TestD6_GlossaryExplainsCategoriesAndWholePesos(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "docs", "glosario-dinero.md"))
	if err != nil {
		t.Fatalf("docs/glosario-dinero.md is missing: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Categorías de gasto",  // what they are and that anyone can add one
		"expenses_by_category", // the dashboard split
		"pesos enteros",        // money is stored and shown without decimals
		"como gasto",           // ingredients and recipe supplies are not logged as an expense
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the glossary should mention %q", want)
		}
	}
}

// Sprint 3: supplies of a recipe are written down too.
func TestI7_GlossaryExplainsSupplies(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "docs", "glosario-dinero.md"))
	if err != nil {
		t.Fatalf("docs/glosario-dinero.md is missing: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Insumos de receta", // what they are
		"supplies_total",    // the field of the breakdown
		"caja",              // the example
		"sin porcentajes",   // indirect and labor do not apply to them
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the glossary should mention %q", want)
		}
	}
}
