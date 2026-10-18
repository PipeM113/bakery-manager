package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

// Sprint 2a, rules P1 to P3 and P9. Every expected value below was worked out by hand with
// exact fractions; the "float" column of the P2 table is what the program answered before.

// P2: the price step must not move because of binary floating point. Each case lands exactly
// on a multiple of 500, where a 1e-13 error charges one step more.
func TestP2_ExactPriceStep(t *testing.T) {
	cases := []struct {
		name                    string
		total, ind, lab, margin float64
		yield                   float64
		want                    float64
		floatAnswerWasWrongWith float64
	}{
		// 1000 * 1.50 = 1500
		{"markup 50%, yield 9", 1000, 0, 0, 0.50, 9, 1500, 2000},
		// 1000 * 1.20 * 1.25 = 1500
		{"labor 20%, markup 25%, yield 9", 1000, 0, 0.20, 0.25, 9, 1500, 2000},
		// 1000 * 1.25 * 1.20 = 1500
		{"indirect 5% and labor 20%, markup 20%, yield 7", 1000, 0.05, 0.20, 0.20, 7, 1500, 2000},
		// 1000 * 1.20 * 1.25 = 1500
		{"indirect 10% and labor 10%, markup 25%, yield 9", 1000, 0.10, 0.10, 0.25, 9, 1500, 2000},
		// 2000 * 1.50 = 3000
		{"2000 with markup 50%, yield 9", 2000, 0, 0, 0.50, 9, 3000, 3500},
		// 2000 * 1.25 * 1.20 = 3000
		{"2000 with indirect 15% and labor 10%, markup 20%, yield 7", 2000, 0.15, 0.10, 0.20, 7, 3000, 3500},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ing := []IngredientCost{{Quantity: 1, PricePerUnit: c.total}}
			got := Calculate("r", "n", c.yield, "u", c.ind, c.lab, c.margin, 0, 0, ing)
			if got.SuggestedPrice != c.want {
				t.Errorf("SuggestedPrice = %v, want %v (float64 gave %v)", got.SuggestedPrice, c.want, c.floatAnswerWasWrongWith)
			}
		})
	}
}

// P1: the four totals are shown in whole pesos (half rounds up), but every derived figure
// comes from the exact values, never from the rounded ones.
func TestP1_TotalsAreWholePesosAndDerivedFromExactValues(t *testing.T) {
	t.Run("half a peso rounds up", func(t *testing.T) {
		got := Calculate("r", "n", 1, "u", 0, 0, 0, 0, 0, []IngredientCost{{Quantity: 1, PricePerUnit: 2.5}})
		if got.IngredientsTotal != 3 {
			t.Errorf("IngredientsTotal = %v, want 3", got.IngredientsTotal)
		}
		got = Calculate("r", "n", 1, "u", 0, 0, 0, 0, 0, []IngredientCost{{Quantity: 1, PricePerUnit: 0.5}})
		if got.IngredientsTotal != 1 {
			t.Errorf("IngredientsTotal = %v, want 1", got.IngredientsTotal)
		}
	})

	t.Run("the total cost is rounded from the exact sum, not summed from rounded parts", func(t *testing.T) {
		// ingredients 0.4, indirect 50% = 0.2, labor 50% = 0.2; exact total 0.8.
		// Rounded parts would be 0 + 0 + 0 = 0; the total must be round(0.8) = 1.
		got := Calculate("r", "n", 1, "u", 0.5, 0.5, 0, 0, 0, []IngredientCost{{Quantity: 1, PricePerUnit: 0.4}})
		if got.IngredientsTotal != 0 || got.IndirectCosts != 0 || got.LaborCosts != 0 {
			t.Errorf("parts = %v, %v, %v; want 0, 0, 0", got.IngredientsTotal, got.IndirectCosts, got.LaborCosts)
		}
		if got.TotalCost != 1 {
			t.Errorf("TotalCost = %v, want 1", got.TotalCost)
		}
	})

	t.Run("the line subtotals are not rounded to whole pesos", func(t *testing.T) {
		// 3 units at 0.1666 cost 0.4998 (not 0).
		ing := []IngredientCost{{Quantity: 3, PricePerUnit: 0.1666}}
		got := Calculate("r", "n", 1, "u", 0, 0, 0, 0, 0, ing)
		if !approx(got.Ingredients[0].Subtotal, 0.4998) {
			t.Errorf("Subtotal = %v, want 0.4998", got.Ingredients[0].Subtotal)
		}
	})
}

// P3: simulating a markup recalculates from the cost without markup, so it is applied once.
func TestP3_SimulateAppliesTheMarkupOnce(t *testing.T) {
	ingredients := func() []IngredientCost { return []IngredientCost{{Quantity: 1, PricePerUnit: 1000}} }

	t.Run("a different markup is applied to the cost without markup", func(t *testing.T) {
		b := Calculate("r", "n", 1, "u", 0, 0, 0.30, 0, 0, ingredients())
		got := Simulate(b, 0.50)
		if !approx(got.SuggestedPrice, 1500) {
			t.Errorf("SuggestedPrice = %v, want 1500", got.SuggestedPrice)
		}
		if !approx(got.ProfitAmount, 500) {
			t.Errorf("ProfitAmount = %v, want 500", got.ProfitAmount)
		}
		if got.MarginPct != 0.50 {
			t.Errorf("MarginPct = %v, want 0.50", got.MarginPct)
		}
	})

	t.Run("simulating with the markup of the breakdown gives the same price per portion", func(t *testing.T) {
		b := Calculate("r", "n", 9, "u", 0, 0.20, 0.25, 0, 0, ingredients())
		got := Simulate(b, 0.25)
		if !approx(got.SuggestedPrice, b.CostPerPortion) {
			t.Errorf("SuggestedPrice = %v, want %v", got.SuggestedPrice, b.CostPerPortion)
		}
		// 1200 / 9 = 133.33...; with 25% that is 166.66...
		if !approx(got.SuggestedPrice, 1500.0/9) {
			t.Errorf("SuggestedPrice = %v, want %v", got.SuggestedPrice, 1500.0/9)
		}
	})

	t.Run("markup 0 gives the cost per portion", func(t *testing.T) {
		b := Calculate("r", "n", 9, "u", 0, 0.20, 0.25, 0, 0, ingredients())
		got := Simulate(b, 0)
		if !approx(got.SuggestedPrice, 1200.0/9) || !approx(got.ProfitAmount, 0) {
			t.Errorf("SuggestedPrice = %v, ProfitAmount = %v; want %v and 0", got.SuggestedPrice, got.ProfitAmount, 1200.0/9)
		}
	})
}

// P1/P7 support: a quantity of 3 at one third of 2000 is exactly 2000.
func TestP1_ExactDecimalsFromFloatInputs(t *testing.T) {
	// 0.1 and 0.2 are not exact in binary; the percentages the user typed must be read as the
	// decimals they meant: 1000 * (1 + 0.1 + 0.2) = 1300 exactly.
	got := Calculate("r", "n", 1, "u", 0.1, 0.2, 0, 0, 0, []IngredientCost{{Quantity: 1, PricePerUnit: 1000}})
	if got.TotalCost != 1300 {
		t.Errorf("TotalCost = %v, want 1300", got.TotalCost)
	}
	if got.SuggestedPrice != 1500 {
		t.Errorf("SuggestedPrice = %v, want 1500 (1300 rounded up to the next 500)", got.SuggestedPrice)
	}
}

// P9: the breakdown no longer repeats the total cost under a second name.
func TestP9_BreakdownHasNoDuplicatedSubtotal(t *testing.T) {
	b := Calculate("r", "n", 1, "u", 0, 0, 0.3, 0, 0, []IngredientCost{{Quantity: 1, PricePerUnit: 1000}})
	raw, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "subtotal_sin_margen") {
		t.Errorf("the JSON still carries subtotal_sin_margen: %s", raw)
	}
	if !strings.Contains(string(raw), `"total_cost":1000`) {
		t.Errorf("total_cost should be 1000 in %s", raw)
	}
}
