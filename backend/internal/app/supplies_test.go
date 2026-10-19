package app_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// Sprint 3: supplies (boxes, cardboard) are ingredients of another kind. They are priced,
// stocked, scaled and costed in the sales like any ingredient, but the indirect and labor
// percentages are not applied to them.

func createSupply(t *testing.T, api *client, name string, stock int) string {
	t.Helper()
	return createIngredient(t, api, map[string]any{
		"name": name, "kind": "supply", "default_unit": "und", "package_size": 1, "package_price": 300,
		"brand": "Cartonera", "stock_quantity": stock,
	})
}

// boxedCake is a recipe of 1 portion: 500 gr of flour (1000) and 1 box (300), with the
// percentages 15% / 30% / 30%.
func boxedCake(t *testing.T, api *client) (recipeID, flourID, boxID string) {
	t.Helper()
	flourID = createIngredient(t, api, map[string]any{
		"name": "Harina", "default_unit": "gr", "package_size": 1000, "package_price": 2000, "stock_quantity": 5000,
	})
	boxID = createSupply(t, api, "Caja torta", 100)
	recipeID = createRecipe(t, api, map[string]any{
		"name": "Torta", "description": "Torta con caja", "yield": 1, "yield_unit": "porciones",
		"ingredients": []map[string]any{
			{"ingredient_id": flourID, "quantity": 500, "unit": "gr"},
			{"ingredient_id": boxID, "quantity": 1, "unit": "und"},
		},
	})
	setCosts(t, api, recipeID, 0.15, 0.30, 0.30)
	return
}

// I1/I2: the kind is part of the ingredient and can be filtered.
func TestI1_KindIsStoredFilteredAndKeptOnUpdate(t *testing.T) {
	api, _ := newPricingAPI(t)
	flour := createIngredient(t, api, map[string]any{"name": "Harina", "default_unit": "gr", "package_size": 1000, "package_price": 2000})
	box := createSupply(t, api, "Caja torta", 0)

	t.Run("the default kind is ingredient, a supply says so", func(t *testing.T) {
		list := map[string]string{}
		_, raw := api.do("GET", "/ingredients", nil)
		var rows []map[string]any
		if err := json.Unmarshal(raw, &rows); err != nil {
			t.Fatal(err)
		}
		for _, r := range rows {
			list[r["name"].(string)], _ = r["kind"].(string)
		}
		if list["Harina"] != "ingredient" || list["Caja torta"] != "supply" {
			t.Errorf("kinds = %v, want Harina ingredient and Caja torta supply", list)
		}
	})

	t.Run("the list can be filtered by kind", func(t *testing.T) {
		for kind, want := range map[string]string{"supply": "Caja torta", "ingredient": "Harina"} {
			_, raw := api.do("GET", "/ingredients?kind="+kind, nil)
			var rows []map[string]any
			if err := json.Unmarshal(raw, &rows); err != nil || len(rows) != 1 || rows[0]["name"] != want {
				t.Errorf("kind=%s: %v (%v), want only %s", kind, rows, err, want)
			}
		}
		if status, _ := api.do("GET", "/ingredients?kind=caja", nil); status != http.StatusBadRequest {
			t.Errorf("kind=caja: status = %d, want 400", status)
		}
	})

	t.Run("an invalid kind is refused", func(t *testing.T) {
		status, _ := api.do("POST", "/ingredients", map[string]any{
			"name": "Raro", "kind": "caja", "default_unit": "und", "package_size": 1, "package_price": 100, "brand": "M",
		})
		if status != http.StatusBadRequest {
			t.Errorf("create with kind caja: status = %d, want 400", status)
		}
	})

	t.Run("an update that does not send the kind keeps it, one that sends it changes it", func(t *testing.T) {
		put := func(extra map[string]any) map[string]any {
			body := map[string]any{"name": "Caja torta", "default_unit": "und", "package_size": 1, "package_price": 350, "brand": "Cartonera"}
			for k, v := range extra {
				body[k] = v
			}
			status, raw := api.do("PUT", "/ingredients/"+box, body)
			if status != http.StatusOK {
				t.Fatalf("update: status = %d: %s", status, raw)
			}
			return decode(t, raw)
		}
		if got := put(nil)["kind"]; got != "supply" {
			t.Errorf("kind after an update without kind = %v, want supply", got)
		}
		if got := put(map[string]any{"kind": "ingredient"})["kind"]; got != "ingredient" {
			t.Errorf("kind after sending ingredient = %v, want ingredient", got)
		}
		if status, _ := api.do("PUT", "/ingredients/"+flour, map[string]any{
			"name": "Harina", "kind": "caja", "default_unit": "gr", "package_size": 1000, "package_price": 2000, "brand": "Marca",
		}); status != http.StatusBadRequest {
			t.Errorf("update with kind caja: status = %d, want 400", status)
		}
	})
}

// I3/I4: the breakdown separates ingredients and supplies; the percentages only touch
// the ingredients. 1000 + 15% + 30% = 1450, + 300 of box = 1750, x1.30 = 2275, up to 2500.
func TestI4_CostBreakdownSeparatesSupplies(t *testing.T) {
	api, _ := newPricingAPI(t)
	recipe, _, _ := boxedCake(t, api)

	body := getJSON(t, api, "/recipes/"+recipe+"/cost")
	want := map[string]float64{
		"ingredients_total": 1000, "supplies_total": 300, "indirect_costs": 150,
		"labor_costs": 300, "total_cost": 1750, "base_price": 2275, "suggested_price": 2500,
	}
	for k, v := range want {
		if got := number(t, body, k); got != v {
			t.Errorf("%s = %v, want %v", k, got, v)
		}
	}
	kinds := map[string]string{}
	for _, row := range body["ingredients"].([]any) {
		r := row.(map[string]any)
		kinds[r["name"].(string)], _ = r["kind"].(string)
	}
	if kinds["Harina"] != "ingredient" || kinds["Caja torta"] != "supply" {
		t.Errorf("kinds in the breakdown = %v", kinds)
	}
}

// I3: the recipe itself says which of its lines are supplies, so the screen can group them.
func TestI3_RecipeLinesCarryTheKind(t *testing.T) {
	api, _ := newPricingAPI(t)
	recipe, _, _ := boxedCake(t, api)

	body := getJSON(t, api, "/recipes/"+recipe)
	kinds := map[string]string{}
	for _, row := range body["ingredients"].([]any) {
		r := row.(map[string]any)
		kinds[r["name"].(string)], _ = r["kind"].(string)
	}
	if kinds["Harina"] != "ingredient" || kinds["Caja torta"] != "supply" {
		t.Errorf("kinds in the recipe = %v", kinds)
	}
}

// I5: boxes grow with the portions, in the preview and in the saved copy.
func TestI5_SuppliesScaleInThePreviewAndTheSavedCopy(t *testing.T) {
	api, _ := newPricingAPI(t)
	recipe, _, _ := boxedCake(t, api)

	preview := getJSON(t, api, "/recipes/"+recipe+"/cost?portions=10")
	// ingredients 10000 + 15% + 30% = 14500; boxes 10 x 300 = 3000; 17500 x 1.30 = 22750 -> 23000
	for k, v := range map[string]float64{"ingredients_total": 10000, "supplies_total": 3000, "total_cost": 17500, "suggested_price": 23000} {
		if got := number(t, preview, k); got != v {
			t.Errorf("preview %s = %v, want %v", k, got, v)
		}
	}

	status, raw := api.do("POST", "/recipes/"+recipe+"/save-as", map[string]any{"portions": 10, "new_name": "Torta x10"})
	if status != http.StatusCreated {
		t.Fatalf("save-as: status = %d: %s", status, raw)
	}
	copyID, _ := decode(t, raw)["id"].(string)
	saved := getJSON(t, api, "/recipes/"+copyID+"/cost")
	for _, k := range []string{"supplies_total", "total_cost", "suggested_price"} {
		if number(t, saved, k) != number(t, preview, k) {
			t.Errorf("%s: saved copy = %v, preview = %v; they must match", k, saved[k], preview[k])
		}
	}
}

// I6: a sale uses up the supplies and counts them in the cost of sales.
func TestI6_SalesUseSuppliesAndCountThemInTheCost(t *testing.T) {
	api, _ := newPricingAPI(t)
	recipe, _, box := boxedCake(t, api)

	if status, raw := api.do("POST", "/sales", map[string]any{"recipe_id": recipe, "quantity_sold": 2, "unit_price": 3500}); status != http.StatusCreated {
		t.Fatalf("sale: status = %d: %s", status, raw)
	}

	_, raw := api.do("GET", "/ingredients?kind=supply", nil)
	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil || len(rows) != 1 {
		t.Fatalf("supplies: %v %s", err, raw)
	}
	if rows[0]["id"] != box || number(t, rows[0], "stock_quantity") != 98 {
		t.Errorf("box stock = %v, want 98 after selling 2", rows[0]["stock_quantity"])
	}

	now := time.Now()
	body := getJSON(t, api, "/analytics/monthly?month="+itoa(int(now.Month()))+"&year="+itoa(now.Year()))
	// 2 x (1000 of ingredients + 300 of box)
	if got := number(t, body, "total_ingredients_cost"); got != 2600 {
		t.Errorf("total_ingredients_cost = %v, want 2600 (ingredients and supplies)", got)
	}
}
