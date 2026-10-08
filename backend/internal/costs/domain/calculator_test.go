package domain

import (
	"math"
	"testing"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

// AC2: characterization of the price calculation. Expected values were worked out by
// hand from the formulas, not copied from the program output.
//
//	ingredientsTotal = sum(quantity * pricePerUnit)
//	totalCost        = ingredientsTotal * (1 + indirectPct + laborPct)
//	basePrice        = totalCost / yield * (1 + margin) * yield
//	suggestedPrice   = ceilTo500(basePrice + extraCharge + deliveryCost)

func TestAC2_CeilTo500(t *testing.T) {
	cases := []struct{ in, want float64 }{
		{0, 0},
		{1, 500},
		{499.99, 500},
		{500, 500},
		{500.01, 1000},
		{4712.5, 5000},
		{5000, 5000},
		{5001, 5500},
	}
	for _, c := range cases {
		if got := ceilTo500(c.in); got != c.want {
			t.Errorf("ceilTo500(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestAC2_CalculateBreakdown(t *testing.T) {
	// 1000 gr at 2.0 = 2000 and 500 ml at 1.0 = 500, so ingredients cost 2500.
	ingredients := []IngredientCost{
		{IngredientID: "a", Name: "harina", Quantity: 1000, Unit: "gr", PricePerUnit: 2.0},
		{IngredientID: "b", Name: "leche", Quantity: 500, Unit: "ml", PricePerUnit: 1.0},
	}

	got := Calculate("r1", "torta", 10, "porciones", 0.15, 0.30, 0.30, 0, 0, ingredients)

	checks := []struct {
		name      string
		got, want float64
	}{
		{"IngredientsTotal", got.IngredientsTotal, 2500},
		{"IndirectCosts", got.IndirectCosts, 375},
		{"LaborCosts", got.LaborCosts, 750},
		{"TotalCost", got.TotalCost, 3625},
		{"CostPerPortion", got.CostPerPortion, 471.25},
		{"BasePrice", got.BasePrice, 4712.5},
		{"Ingredients[0].Subtotal", got.Ingredients[0].Subtotal, 2000},
		{"Ingredients[1].Subtotal", got.Ingredients[1].Subtotal, 500},
	}
	for _, c := range checks {
		if !approx(c.got, c.want) {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
	if got.SuggestedPrice != 5000 {
		t.Errorf("SuggestedPrice = %v, want 5000", got.SuggestedPrice)
	}
	if got.RecipeID != "r1" || got.RecipeName != "torta" || got.Yield != 10 || got.YieldUnit != "porciones" {
		t.Errorf("recipe fields were not copied: %+v", got)
	}
	if got.MarginPct != 0.30 {
		t.Errorf("MarginPct = %v, want 0.30", got.MarginPct)
	}
}

func TestAC2_CalculateSuggestedPrice(t *testing.T) {
	ing := func(total float64) []IngredientCost {
		return []IngredientCost{{Quantity: 1, PricePerUnit: total}}
	}
	cases := []struct {
		name                  string
		total                 float64
		ind, lab, margin      float64
		yield                 float64
		extra, delivery       int
		wantSuggested         float64
		wantBase, wantPerPort float64
	}{
		{"base already a multiple of 500 is not bumped", 2000, 0, 0, 1.5, 1, 0, 0, 5000, 5000, 5000},
		{"one peso over a multiple goes to the next step", 2000, 0, 0, 1.5, 1, 1, 0, 5500, 5000, 5000},
		{"extra and delivery are added before rounding", 2000, 0, 0, 1.5, 1, 100, 400, 5500, 5000, 5000},
		{"margin 0 keeps the cost", 3000, 0, 0, 0, 3, 0, 0, 3000, 3000, 1000},
		{"surcharge applies on top of margin", 4712.5, 0, 0, 0, 10, 300, 500, 6000, 4712.5, 471.25},
		{"no ingredients and no surcharge costs nothing", 0, 0.15, 0.30, 0.30, 4, 0, 0, 0, 0, 0},
		{"no ingredients but a delivery fee still rounds up", 0, 0.15, 0.30, 0.30, 4, 0, 250, 500, 0, 0},
		// CURRENT: a recipe with yield 0 gets no per-portion cost and a base price of 0;
		// only the surcharges remain.
		{"CURRENT yield 0 leaves only the surcharges", 2000, 0.15, 0.30, 0.30, 0, 0, 700, 1000, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Calculate("r", "n", c.yield, "u", c.ind, c.lab, c.margin, c.extra, c.delivery, ing(c.total))
			if got.SuggestedPrice != c.wantSuggested {
				t.Errorf("SuggestedPrice = %v, want %v", got.SuggestedPrice, c.wantSuggested)
			}
			if !approx(got.BasePrice, c.wantBase) {
				t.Errorf("BasePrice = %v, want %v", got.BasePrice, c.wantBase)
			}
			if !approx(got.CostPerPortion, c.wantPerPort) {
				t.Errorf("CostPerPortion = %v, want %v", got.CostPerPortion, c.wantPerPort)
			}
			if got.ExtraCharge != float64(c.extra) || got.DeliveryCost != float64(c.delivery) {
				t.Errorf("surcharges were not echoed: %+v", got)
			}
		})
	}
}

// AC2: Simulate adds the margin on top of CostPerPortion. When the breakdown was built
// with that same margin (as the /cost/simulate handler does), the margin is applied twice.
// CURRENT behaviour, frozen on purpose: it is a suspected defect listed in the backlog
// and any change needs the owner's approval.
func TestAC2_SimulateAppliesMarginOnTopOfTheBreakdown(t *testing.T) {
	ingredients := []IngredientCost{{Quantity: 1, PricePerUnit: 1000}}
	breakdown := Calculate("r", "n", 1, "u", 0, 0, 0.30, 0, 0, ingredients)
	if !approx(breakdown.CostPerPortion, 1300) {
		t.Fatalf("precondition: CostPerPortion = %v, want 1300", breakdown.CostPerPortion)
	}

	got := Simulate(breakdown, 0.30)

	if !approx(got.ProfitAmount, 390) {
		t.Errorf("ProfitAmount = %v, want 390", got.ProfitAmount)
	}
	if !approx(got.SuggestedPrice, 1690) {
		t.Errorf("SuggestedPrice = %v, want 1690 (1300 * 1.30)", got.SuggestedPrice)
	}
}

// AC2: KNOWN DEFECT, frozen as it behaves today. With float64 the base price comes out
// as 1500.000000000000227 instead of exactly 1500, so ceilTo500 charges 2000 for a
// recipe whose exact price is 1500. Sprint 2 (money) must change this expectation to
// 1500; until then the test keeps the problem visible and reproducible.
func TestAC2_KnownDefectFloatDriftRaisesThePriceStep(t *testing.T) {
	// ingredients 1000, indirect 0%, labor 20%, margin 25%, yield 9:
	// exact: 1000 * 1.20 = 1200; 1200 / 9 * 1.25 * 9 = 1500.
	ingredients := []IngredientCost{{Quantity: 1, PricePerUnit: 1000}}

	got := Calculate("r", "n", 9, "u", 0, 0.20, 0.25, 0, 0, ingredients)

	if got.SuggestedPrice != 2000 {
		t.Fatalf("SuggestedPrice = %v; the documented current value is 2000 (exact value: 1500)", got.SuggestedPrice)
	}
}
