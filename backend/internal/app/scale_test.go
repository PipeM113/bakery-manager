package app_test

import (
	"net/http"
	"testing"
)

// Sprint 2a, scaling (found by the owner after the first delivery): the price for another
// number of portions comes from the backend, and "save scaled" keeps the recipe's markup,
// extra charge and delivery, so the saved copy costs what the preview said.

func setAllCosts(t *testing.T, api *client, id string, indirect, labor, markup float64, extra, delivery int) {
	t.Helper()
	status, raw := api.do("PUT", "/recipes/"+id+"/costs", map[string]any{
		"indirect_cost_pct": indirect, "labor_cost_pct": labor, "margin_pct": markup,
		"extra_charge": extra, "delivery_cost": delivery,
	})
	if status != http.StatusOK {
		t.Fatalf("set costs: status = %d: %s", status, raw)
	}
}

// E1: ?portions=N prices the recipe made for N portions: the ingredients scale by N / yield,
// the surcharges (extra charge, delivery) are per order and do not scale.
func TestE1_CostForAnotherNumberOfPortions(t *testing.T) {
	api, _ := newPricingAPI(t)
	id := flourRecipe(t, api, nil) // 1000 of ingredients for 1 portion
	setAllCosts(t, api, id, 0.15, 0.30, 0.30, 100, 400)

	t.Run("10 portions: 10000 + 15% + 30% = 14500, x1.30 = 18850, +500 = 19350, rounded up to 19500", func(t *testing.T) {
		body := getJSON(t, api, "/recipes/"+id+"/cost?portions=10&extra_charge=100&delivery_cost=400")
		want := map[string]float64{
			"yield": 10, "ingredients_total": 10000, "total_cost": 14500,
			"base_price": 18850, "cost_per_portion": 1885, "suggested_price": 19500,
		}
		for k, v := range want {
			if got := number(t, body, k); got != v {
				t.Errorf("%s = %v, want %v", k, got, v)
			}
		}
	})

	t.Run("without portions the recipe's own yield is used", func(t *testing.T) {
		body := getJSON(t, api, "/recipes/"+id+"/cost?extra_charge=100&delivery_cost=400")
		if got := number(t, body, "suggested_price"); got != 2500 { // 1885 + 500 = 2385
			t.Errorf("suggested_price = %v, want 2500", got)
		}
	})

	t.Run("explicit percentages still apply", func(t *testing.T) {
		body := getJSON(t, api, "/recipes/"+id+"/cost?portions=10&indirect_cost_pct=0&labor_cost_pct=0&margin_pct=0&extra_charge=100&delivery_cost=400")
		if got := number(t, body, "suggested_price"); got != 10500 {
			t.Errorf("suggested_price = %v, want 10500", got)
		}
	})

	t.Run("portions below 1 or not a number are rejected", func(t *testing.T) {
		for _, p := range []string{"0", "-2", "0.5", "abc", "NaN"} {
			if status, _ := api.do("GET", "/recipes/"+id+"/cost?portions="+p, nil); status != http.StatusBadRequest {
				t.Errorf("portions=%s: status = %d, want 400", p, status)
			}
		}
	})
}

// E1: a factor that is not a whole number: 4 portions of 1000 become 6 portions of 1500.
func TestE1_FractionalScaleFactor(t *testing.T) {
	api, _ := newPricingAPI(t)
	id := flourRecipe(t, api, map[string]any{"yield": 4})
	setAllCosts(t, api, id, 0.30, 0.30, 0.30, 0, 0)

	body := getJSON(t, api, "/recipes/"+id+"/cost?portions=6")
	// 1500 * 1.60 = 2400; * 1.30 = 3120; rounded up to 3500
	if got := number(t, body, "ingredients_total"); got != 1500 {
		t.Errorf("ingredients_total = %v, want 1500", got)
	}
	if got := number(t, body, "suggested_price"); got != 3500 {
		t.Errorf("suggested_price = %v, want 3500", got)
	}
}

// E3: the copy saved by "Guardar escalado" keeps the markup, extra charge and delivery of
// the original (it used to start from 0), so it is priced like the preview.
func TestE3_SavedScaledRecipeKeepsTheCostsAndMatchesThePreview(t *testing.T) {
	api, _ := newPricingAPI(t)
	id := flourRecipe(t, api, nil)
	setAllCosts(t, api, id, 0.15, 0.30, 0.30, 100, 400)

	preview := getJSON(t, api, "/recipes/"+id+"/cost?portions=10&extra_charge=100&delivery_cost=400")

	status, raw := api.do("POST", "/recipes/"+id+"/save-as", map[string]any{"portions": 10, "new_name": "Bizcocho x10"})
	if status != http.StatusCreated {
		t.Fatalf("save-as: status = %d: %s", status, raw)
	}
	copyID, _ := decode(t, raw)["id"].(string)

	costs := getJSON(t, api, "/recipes/"+copyID+"/costs")
	want := map[string]float64{
		"indirect_cost_pct": 0.15, "labor_cost_pct": 0.30, "margin_pct": 0.30,
		"extra_charge": 100, "delivery_cost": 400,
	}
	for k, v := range want {
		if got := number(t, costs, k); got != v {
			t.Errorf("copy %s = %v, want %v", k, got, v)
		}
	}

	saved := getJSON(t, api, "/recipes/"+copyID+"/cost?extra_charge=100&delivery_cost=400")
	for _, k := range []string{"yield", "ingredients_total", "total_cost", "suggested_price"} {
		if number(t, saved, k) != number(t, preview, k) {
			t.Errorf("%s: saved copy = %v, preview = %v; they must match", k, saved[k], preview[k])
		}
	}
	if got := number(t, saved, "suggested_price"); got != 19500 {
		t.Errorf("saved suggested_price = %v, want 19500", got)
	}
}
