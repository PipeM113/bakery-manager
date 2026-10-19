package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

// Sprint 3, rules I4 and I5: supplies (boxes, cardboard) are a direct cost. The indirect and
// labor percentages apply to the ingredients only; supplies are added as they are.

func box(quantity, price float64) IngredientCost {
	return IngredientCost{Name: "caja", Kind: KindSupply, Quantity: quantity, PricePerUnit: price}
}

func flour(total float64) IngredientCost {
	return IngredientCost{Name: "harina", Quantity: 1, PricePerUnit: total}
}

// ingredients 1000 + 15% + 30% = 1450; supplies 300; total 1750; x1.30 = 2275; rounded 2500.
func TestI4_SuppliesAreAddedWithoutPercentages(t *testing.T) {
	got := Calculate("r", "n", 1, "u", 0.15, 0.30, 0.30, 0, 0, []IngredientCost{flour(1000), box(1, 300)})

	checks := map[string]struct{ got, want float64 }{
		"IngredientsTotal": {got.IngredientsTotal, 1000},
		"SuppliesTotal":    {got.SuppliesTotal, 300},
		"IndirectCosts":    {got.IndirectCosts, 150},
		"LaborCosts":       {got.LaborCosts, 300},
		"TotalCost":        {got.TotalCost, 1750},
		"BasePrice":        {got.BasePrice, 2275},
		"CostPerPortion":   {got.CostPerPortion, 2275},
	}
	for name, c := range checks {
		if !approx(c.got, c.want) {
			t.Errorf("%s = %v, want %v", name, c.got, c.want)
		}
	}
	if got.SuggestedPrice != 2500 {
		t.Errorf("SuggestedPrice = %v, want 2500", got.SuggestedPrice)
	}
}

func TestI4_SuppliesDoNotChangeWhenThePercentagesDo(t *testing.T) {
	a := Calculate("r", "n", 1, "u", 0, 0, 0, 0, 0, []IngredientCost{flour(1000), box(2, 300)})
	if a.TotalCost != 1600 { // 1000 + 2 x 300
		t.Errorf("TotalCost with 0%% = %v, want 1600", a.TotalCost)
	}
	b := Calculate("r", "n", 1, "u", 1, 1, 0, 0, 0, []IngredientCost{flour(1000), box(2, 300)})
	if b.TotalCost != 3600 { // 1000 x 3 + 600
		t.Errorf("TotalCost with 100%%+100%% = %v, want 3600", b.TotalCost)
	}
}

// Rounding follows the same rule as the rest: exact sums, whole pesos half up at the end.
func TestI4_SuppliesRoundHalfUpAndFeedTheExactTotal(t *testing.T) {
	// ingredients 0.4 (+50% +50% = 0.8) and supplies 0.4: exact total 1.2
	got := Calculate("r", "n", 1, "u", 0.5, 0.5, 0, 0, 0, []IngredientCost{flour(0.4), box(1, 0.4)})
	if got.SuppliesTotal != 0 || got.IngredientsTotal != 0 {
		t.Errorf("parts = %v and %v, want 0 and 0", got.IngredientsTotal, got.SuppliesTotal)
	}
	if got.TotalCost != 1 {
		t.Errorf("TotalCost = %v, want 1 (rounded from the exact 1.2)", got.TotalCost)
	}
	half := Calculate("r", "n", 1, "u", 0, 0, 0, 0, 0, []IngredientCost{box(1, 2.5)})
	if half.SuppliesTotal != 3 {
		t.Errorf("SuppliesTotal = %v, want 3", half.SuppliesTotal)
	}
}

// I5: for 10 portions of a recipe of 1: ingredients 10000 + 15% + 30% = 14500; supplies 3000.
func TestI5_SuppliesScaleWithThePortions(t *testing.T) {
	got := CalculateScaled("r", "n", 1, 10, "u", 0.15, 0.30, 0.30, 0, 0, []IngredientCost{flour(1000), box(1, 300)})
	if got.SuppliesTotal != 3000 {
		t.Errorf("SuppliesTotal = %v, want 3000", got.SuppliesTotal)
	}
	if got.TotalCost != 17500 {
		t.Errorf("TotalCost = %v, want 17500", got.TotalCost)
	}
	if got.SuggestedPrice != 23000 { // 17500 x 1.30 = 22750
		t.Errorf("SuggestedPrice = %v, want 23000", got.SuggestedPrice)
	}
}

func TestI4_TheJSONCarriesTheSuppliesAndTheKind(t *testing.T) {
	got := Calculate("r", "n", 1, "u", 0, 0, 0, 0, 0, []IngredientCost{flour(1000), box(1, 300)})
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"supplies_total":300`, `"kind":"supply"`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("the JSON should contain %s: %s", want, raw)
		}
	}
}

// Without supplies nothing changes.
func TestI4_NoSuppliesMeansNoSuppliesTotal(t *testing.T) {
	got := Calculate("r", "n", 1, "u", 0.15, 0.30, 0.30, 0, 0, []IngredientCost{flour(1000)})
	if got.SuppliesTotal != 0 || got.TotalCost != 1450 {
		t.Errorf("SuppliesTotal = %v, TotalCost = %v; want 0 and 1450", got.SuppliesTotal, got.TotalCost)
	}
}
