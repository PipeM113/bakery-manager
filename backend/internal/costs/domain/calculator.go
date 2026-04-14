package domain

import "math"

type IngredientCost struct {
	IngredientID string  `json:"ingredient_id"`
	Name         string  `json:"name"`
	Quantity     float64 `json:"quantity"`
	Unit         string  `json:"unit"`
	PricePerUnit float64 `json:"price_per_unit"`
	Subtotal     float64 `json:"subtotal"`
}

type CostBreakdown struct {
	RecipeID          string           `json:"recipe_id"`
	RecipeName        string           `json:"recipe_name"`
	Yield             float64          `json:"yield"`
	YieldUnit         string           `json:"yield_unit"`
	Ingredients       []IngredientCost `json:"ingredients"`
	IngredientsTotal  float64          `json:"ingredients_total"`
	IndirectCosts     float64          `json:"indirect_costs"`
	LaborCosts        float64          `json:"labor_costs"`
	SubtotalSinMargen float64          `json:"subtotal_sin_margen"`
	TotalCost         float64          `json:"total_cost"`
	CostPerPortion    float64          `json:"cost_per_portion"`
	MarginPct         float64          `json:"margin_pct"`
	BasePrice         float64          `json:"base_price"`    // totalCost*(1+margin), before extra/delivery/rounding
	ExtraCharge       float64          `json:"extra_charge"`  // hidden surcharge
	DeliveryCost      float64          `json:"delivery_cost"` // shown in quotation
	SuggestedPrice    float64          `json:"suggested_price"` // ceilTo500(base_price+extra+delivery)
}

// ceilTo500 rounds up to the nearest multiple of 500.
func ceilTo500(price float64) float64 {
	if price == 0 {
		return 0
	}
	return math.Ceil(price/500) * 500
}

type PriceSuggestion struct {
	CostBreakdown
	MarginPct      float64 `json:"margin_pct"`
	ProfitAmount   float64 `json:"profit_amount"`
	SuggestedPrice float64 `json:"suggested_price"`
}

func Calculate(
	recipeID, recipeName string,
	yield float64, yieldUnit string,
	indirectPct, laborPct, marginPct float64,
	extraCharge, deliveryCost int,
	ingredients []IngredientCost,
) CostBreakdown {
	var ingredientsTotal float64
	for i, ing := range ingredients {
		ingredients[i].Subtotal = ing.Quantity * ing.PricePerUnit
		ingredientsTotal += ingredients[i].Subtotal
	}

	indirectCosts := ingredientsTotal * indirectPct
	laborCosts := ingredientsTotal * laborPct
	totalCost := ingredientsTotal + indirectCosts + laborCosts

	costPerPortion := 0.0
	if yield > 0 {
		costPerPortion = (totalCost / yield) * (1 + marginPct)
	}

	// basePrice = full recipe price with margin, before extra surcharges
	basePrice := costPerPortion * yield

	// suggestedPrice = final rounded price including all charges
	suggestedPrice := ceilTo500(basePrice + float64(extraCharge) + float64(deliveryCost))

	return CostBreakdown{
		RecipeID:          recipeID,
		RecipeName:        recipeName,
		Yield:             yield,
		YieldUnit:         yieldUnit,
		Ingredients:       ingredients,
		IngredientsTotal:  ingredientsTotal,
		IndirectCosts:     indirectCosts,
		LaborCosts:        laborCosts,
		SubtotalSinMargen: totalCost,
		TotalCost:         totalCost,
		CostPerPortion:    costPerPortion,
		MarginPct:         marginPct,
		BasePrice:         basePrice,
		ExtraCharge:       float64(extraCharge),
		DeliveryCost:      float64(deliveryCost),
		SuggestedPrice:    suggestedPrice,
	}
}

func Simulate(breakdown CostBreakdown, marginPct float64) PriceSuggestion {
	profitAmount := breakdown.CostPerPortion * marginPct
	suggestedPrice := breakdown.CostPerPortion + profitAmount

	return PriceSuggestion{
		CostBreakdown:  breakdown,
		MarginPct:      marginPct,
		ProfitAmount:   profitAmount,
		SuggestedPrice: suggestedPrice,
	}
}
