package hygiene

import (
	"strings"
	"testing"
)

// Sprint 15, N1: the conversion of a recipe line to the unit of its ingredient is written once. Only
// these files may call kernel.Convert; costs and sales go through internal/recipeneeds.
func TestN1_OnlyTheKnownPlacesCallKernelConvert(t *testing.T) {
	root := repoRoot(t)
	allowed := map[string]bool{
		"backend/internal/recipeneeds/recipeneeds.go":                 true, // the one rule
		"backend/internal/stock_adjustments/repository/repository.go": true, // the unit a person types for a stock change
		"backend/internal/recipes/repository/repository.go":           true, // CheckUnits: a line must be convertible
		"backend/internal/ingredients/repository/repository.go":       true, // checkRecipes: a unit change must keep recipes working
	}
	for path, text := range goSources(t, root, "backend/internal") {
		if strings.HasSuffix(path, "_test.go") || strings.Contains(path, "/shared/kernel/") {
			continue
		}
		if !strings.Contains(text, "kernel.Convert(") {
			continue
		}
		rel := strings.TrimPrefix(path, root+"/")
		if !allowed[rel] {
			t.Errorf("%s calls kernel.Convert: use recipeneeds.Load for the lines of a recipe, or add the place to this list with a reason", rel)
		}
	}
}

func TestN1_CostsAndSalesUseTheSameRule(t *testing.T) {
	root := repoRoot(t)
	for _, dir := range []string{"backend/internal/costs/service", "backend/internal/sales/repository"} {
		var uses bool
		for path, text := range goSources(t, root, dir) {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			if strings.Contains(text, "recipeneeds.Load(") {
				uses = true
			}
			if strings.Contains(text, "FROM recipe_ingredients") {
				t.Errorf("%s reads recipe_ingredients itself: that query lives in internal/recipeneeds", path)
			}
		}
		if !uses {
			t.Errorf("%s does not call recipeneeds.Load", dir)
		}
	}
}
