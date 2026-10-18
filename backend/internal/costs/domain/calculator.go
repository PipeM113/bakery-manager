package domain

import (
	"math/big"
	"strconv"
)

// Money is calculated with exact fractions (math/big.Rat) and rounded once at the end.
// The inputs arrive as float64 because that is what the database and the JSON give us; each
// one is read as the shortest decimal that identifies it (0.2 is one fifth, not
// 0.200000000000000011...), which is what the person typed.

type IngredientCost struct {
	IngredientID string  `json:"ingredient_id"`
	Name         string  `json:"name"`
	Quantity     float64 `json:"quantity"`
	Unit         string  `json:"unit"`
	PricePerUnit float64 `json:"price_per_unit"`
	Subtotal     float64 `json:"subtotal"`

	// PackagePrice and PackageSize, when both are positive, give the exact unit price
	// (PackagePrice / PackageSize). PricePerUnit is the stored copy, rounded to four decimals.
	PackagePrice float64 `json:"-"`
	PackageSize  float64 `json:"-"`
}

type CostBreakdown struct {
	RecipeID         string           `json:"recipe_id"`
	RecipeName       string           `json:"recipe_name"`
	Yield            float64          `json:"yield"`
	YieldUnit        string           `json:"yield_unit"`
	Ingredients      []IngredientCost `json:"ingredients"`
	IngredientsTotal float64          `json:"ingredients_total"` // whole pesos
	IndirectCosts    float64          `json:"indirect_costs"`    // whole pesos
	LaborCosts       float64          `json:"labor_costs"`       // whole pesos
	TotalCost        float64          `json:"total_cost"`        // whole pesos, rounded from the exact sum
	CostPerPortion   float64          `json:"cost_per_portion"`  // with markup, not rounded
	MarginPct        float64          `json:"margin_pct"`        // the recipe markup over cost (a fraction)
	BasePrice        float64          `json:"base_price"`        // total cost * (1+markup), before extra/delivery/rounding
	ExtraCharge      float64          `json:"extra_charge"`      // hidden surcharge
	DeliveryCost     float64          `json:"delivery_cost"`     // shown in quotation
	SuggestedPrice   float64          `json:"suggested_price"`   // ceilTo500(base_price+extra+delivery)

	// costPerPortionNoMarkup is the exact cost of one portion before the markup; Simulate
	// starts from it.
	costPerPortionNoMarkup *big.Rat
}

type PriceSuggestion struct {
	CostBreakdown
	MarginPct      float64 `json:"margin_pct"`
	ProfitAmount   float64 `json:"profit_amount"`
	SuggestedPrice float64 `json:"suggested_price"`
}

// rat reads a float64 as the shortest decimal that round-trips, exactly.
func rat(f float64) *big.Rat {
	r, ok := new(big.Rat).SetString(strconv.FormatFloat(f, 'f', -1, 64))
	if !ok {
		return new(big.Rat)
	}
	return r
}

func toFloat(r *big.Rat) float64 {
	f, _ := r.Float64()
	return f
}

// roundHalfUp rounds a non-negative fraction to the nearest whole peso, halves going up.
func roundHalfUp(r *big.Rat) float64 {
	twice := new(big.Int).Mul(r.Num(), big.NewInt(2))
	twice.Add(twice, r.Denom())
	den := new(big.Int).Mul(r.Denom(), big.NewInt(2))
	whole := new(big.Int).Div(twice, den)
	return float64(whole.Int64())
}

// ceilTo500 rounds up to the nearest multiple of 500; zero stays zero.
func ceilTo500(price float64) float64 { return ceilRat500(rat(price)) }

func ceilRat500(r *big.Rat) float64 {
	if r.Sign() <= 0 {
		return 0
	}
	q := new(big.Rat).Quo(r, big.NewRat(500, 1))
	n := new(big.Int).Div(q.Num(), q.Denom())
	if new(big.Rat).SetInt(n).Cmp(q) < 0 {
		n.Add(n, big.NewInt(1))
	}
	return float64(n.Int64() * 500)
}

func unitPrice(ing IngredientCost) *big.Rat {
	if ing.PackagePrice > 0 && ing.PackageSize > 0 {
		return new(big.Rat).Quo(rat(ing.PackagePrice), rat(ing.PackageSize))
	}
	return rat(ing.PricePerUnit)
}

func Calculate(
	recipeID, recipeName string,
	yield float64, yieldUnit string,
	indirectPct, laborPct, marginPct float64,
	extraCharge, deliveryCost int,
	ingredients []IngredientCost,
) CostBreakdown {
	ingredientsTotal := new(big.Rat)
	for i, ing := range ingredients {
		subtotal := new(big.Rat).Mul(rat(ing.Quantity), unitPrice(ing))
		ingredients[i].Subtotal = toFloat(subtotal)
		ingredientsTotal.Add(ingredientsTotal, subtotal)
	}

	indirectCosts := new(big.Rat).Mul(ingredientsTotal, rat(indirectPct))
	laborCosts := new(big.Rat).Mul(ingredientsTotal, rat(laborPct))
	totalCost := new(big.Rat).Add(ingredientsTotal, indirectCosts)
	totalCost.Add(totalCost, laborCosts)

	markup := new(big.Rat).Add(big.NewRat(1, 1), rat(marginPct))
	yieldR := rat(yield)

	costPerPortionNoMarkup := new(big.Rat)
	costPerPortion := new(big.Rat)
	basePrice := new(big.Rat)
	if yieldR.Sign() > 0 {
		costPerPortionNoMarkup.Quo(totalCost, yieldR)
		costPerPortion.Mul(costPerPortionNoMarkup, markup)
		// basePrice = full recipe price with markup, before extra surcharges
		basePrice.Mul(costPerPortion, yieldR)
	}

	// suggestedPrice = final rounded price including all charges
	final := new(big.Rat).Add(basePrice, big.NewRat(int64(extraCharge), 1))
	final.Add(final, big.NewRat(int64(deliveryCost), 1))

	return CostBreakdown{
		RecipeID:               recipeID,
		RecipeName:             recipeName,
		Yield:                  yield,
		YieldUnit:              yieldUnit,
		Ingredients:            ingredients,
		IngredientsTotal:       roundHalfUp(ingredientsTotal),
		IndirectCosts:          roundHalfUp(indirectCosts),
		LaborCosts:             roundHalfUp(laborCosts),
		TotalCost:              roundHalfUp(totalCost),
		CostPerPortion:         toFloat(costPerPortion),
		MarginPct:              marginPct,
		BasePrice:              toFloat(basePrice),
		ExtraCharge:            float64(extraCharge),
		DeliveryCost:           float64(deliveryCost),
		SuggestedPrice:         ceilRat500(final),
		costPerPortionNoMarkup: costPerPortionNoMarkup,
	}
}

// Simulate prices one portion with a different markup. It starts from the cost without
// markup, so the markup is applied once whatever markup the breakdown was built with.
func Simulate(breakdown CostBreakdown, marginPct float64) PriceSuggestion {
	base := breakdown.costPerPortionNoMarkup
	if base == nil {
		// A breakdown that did not come from Calculate: undo the markup it carries.
		base = new(big.Rat).Quo(rat(breakdown.CostPerPortion), new(big.Rat).Add(big.NewRat(1, 1), rat(breakdown.MarginPct)))
	}
	profit := new(big.Rat).Mul(base, rat(marginPct))
	suggested := new(big.Rat).Add(base, profit)

	return PriceSuggestion{
		CostBreakdown:  breakdown,
		MarginPct:      marginPct,
		ProfitAmount:   toFloat(profit),
		SuggestedPrice: toFloat(suggested),
	}
}
