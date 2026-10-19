package app_test

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// Sprint 7: what a sale takes from the stock. The quantities of the recipe are converted to the
// unit of the ingredient, a sale can be of a whole recipe or of portions, and a sale without
// enough stock is registered anyway (the stock stops at 0 and the sale says what was missing).

func stockOf(t *testing.T, api *client, ingredientID string) float64 {
	t.Helper()
	status, raw := api.do("GET", "/ingredients", nil)
	if status != http.StatusOK {
		t.Fatalf("list ingredients: status = %d: %s", status, raw)
	}
	var list []map[string]any
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Fatal(err)
	}
	for _, i := range list {
		if i["id"] == ingredientID {
			return i["stock_quantity"].(float64)
		}
	}
	t.Fatalf("ingredient %s is not listed", ingredientID)
	return 0
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

// recipeOf makes a recipe with one ingredient: `quantity` `unit` of it, yielding `yield` `yieldUnit`.
func recipeOf(t *testing.T, api *client, ingredientID string, quantity float64, unit string, yield float64, yieldUnit string) string {
	t.Helper()
	return createRecipe(t, api, map[string]any{
		"name": "Receta", "description": "Receta de prueba", "yield": yield, "yield_unit": yieldUnit,
		"ingredients": []map[string]any{{"ingredient_id": ingredientID, "quantity": quantity, "unit": unit}},
	})
}

func sellOK(t *testing.T, api *client, body map[string]any) map[string]any {
	t.Helper()
	if _, ok := body["unit_price"]; !ok {
		body["unit_price"] = 1000
	}
	status, raw := api.do("POST", "/sales", body)
	if status != http.StatusCreated {
		t.Fatalf("POST /sales %v: status = %d: %s", body, status, raw)
	}
	return decode(t, raw)
}

func usedLines(t *testing.T, sale map[string]any) []map[string]any {
	t.Helper()
	raw, _ := sale["ingredients"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, r := range raw {
		out = append(out, r.(map[string]any))
	}
	return out
}

// V1 and V7: the recipe says 200 gr, the ingredient is kept in kg: 0.2 kg leave the stock and the
// cost is 20% of the price of a kg, not 200 kg.
func TestV1_RecipeQuantitiesAreConvertedToTheIngredientUnit(t *testing.T) {
	api := newAPIAt(t, marchClock)

	t.Run("200 gr of a product kept in kg (the manjar of V7)", func(t *testing.T) {
		manjar := createIngredient(t, api.client, map[string]any{
			"name": "Manjar", "default_unit": "kg", "package_size": 1, "package_price": 8000, "stock_quantity": 1,
		})
		recipe := recipeOf(t, api.client, manjar, 200, "gr", 1, "porciones")
		sale := sellOK(t, api.client, map[string]any{"recipe_id": recipe, "quantity_sold": 1})

		if got := stockOf(t, api.client, manjar); !near(got, 0.8) {
			t.Errorf("stock = %v kg, want 0.8 (1 kg bought, 200 gr used)", got)
		}
		lines := usedLines(t, sale)
		if len(lines) != 1 {
			t.Fatalf("lines = %v", lines)
		}
		if q := lines[0]["quantity_used"].(float64); !near(q, 0.2) {
			t.Errorf("quantity_used = %v, want 0.2", q)
		}
		if u := lines[0]["unit"]; u != "kg" {
			t.Errorf("unit = %v, want kg (the unit of the ingredient)", u)
		}
		cost := lines[0]["quantity_used"].(float64) * lines[0]["price_at_time"].(float64)
		if !near(cost, 1600) {
			t.Errorf("cost of the line = %v, want 1600 (20%% of the 8000 of a kg)", cost)
		}
	})

	t.Run("milliliters of a product kept in liters", func(t *testing.T) {
		milk := createIngredient(t, api.client, map[string]any{
			"name": "Leche", "default_unit": "lt", "package_size": 1, "package_price": 1000, "stock_quantity": 2,
		})
		recipe := recipeOf(t, api.client, milk, 250, "ml", 1, "porciones")
		sellOK(t, api.client, map[string]any{"recipe_id": recipe, "quantity_sold": 1})
		if got := stockOf(t, api.client, milk); !near(got, 1.75) {
			t.Errorf("stock = %v lt, want 1.75", got)
		}
	})

	t.Run("a product kept in gr and a recipe in kg", func(t *testing.T) {
		flour := createIngredient(t, api.client, map[string]any{
			"name": "Harina", "default_unit": "gr", "package_size": 1000, "package_price": 2000, "stock_quantity": 5000,
		})
		recipe := recipeOf(t, api.client, flour, 0.5, "kg", 1, "porciones")
		sellOK(t, api.client, map[string]any{"recipe_id": recipe, "quantity_sold": 2})
		if got := stockOf(t, api.client, flour); !near(got, 4000) {
			t.Errorf("stock = %v gr, want 4000 (2 x 0.5 kg)", got)
		}
	})

	t.Run("the same unit changes nothing", func(t *testing.T) {
		sugar := createIngredient(t, api.client, map[string]any{
			"name": "Azúcar", "default_unit": "gr", "package_size": 1000, "package_price": 1000, "stock_quantity": 1000,
		})
		recipe := recipeOf(t, api.client, sugar, 100, "gr", 1, "porciones")
		sellOK(t, api.client, map[string]any{"recipe_id": recipe, "quantity_sold": 3})
		if got := stockOf(t, api.client, sugar); !near(got, 700) {
			t.Errorf("stock = %v gr, want 700", got)
		}
	})
}

// V2: units that cannot be converted refuse the sale and leave everything as it was.
func TestV2_IncompatibleUnitsRefuseTheSale(t *testing.T) {
	api := newAPIAt(t, marchClock)
	flour := createIngredient(t, api.client, map[string]any{
		"name": "Harina", "default_unit": "gr", "package_size": 1000, "package_price": 2000, "stock_quantity": 5000,
	})
	recipe := recipeOf(t, api.client, flour, 100, "gr", 1, "porciones")
	// The recipe asks for it in ml, which cannot be turned into grams.
	if _, err := api.db.Exec(context.Background(), `update recipe_ingredients set unit = 'ml' where recipe_id = $1`, recipe); err != nil {
		t.Fatal(err)
	}

	status, raw := api.do("POST", "/sales", map[string]any{"recipe_id": recipe, "quantity_sold": 1, "unit_price": 1000})
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", status, raw)
	}
	if !strings.Contains(string(raw), "ml") || !strings.Contains(string(raw), "gr") {
		t.Errorf("the message should name both units: %s", raw)
	}
	if got := stockOf(t, api.client, flour); got != 5000 {
		t.Errorf("stock = %v, want 5000 (nothing was taken)", got)
	}
	var sales int
	if err := api.db.QueryRow(context.Background(), `select count(*) from sales`).Scan(&sales); err != nil || sales != 0 {
		t.Errorf("sales = %d (%v), want none", sales, err)
	}
}

// V3: a sale is of whole recipes (by default) or of portions.
func TestV3_SoldAsRecipeOrPortion(t *testing.T) {
	api := newAPIAt(t, marchClock)
	flour := createIngredient(t, api.client, map[string]any{
		"name": "Harina", "default_unit": "gr", "package_size": 1000, "package_price": 2000, "stock_quantity": 10000,
	})
	cake := recipeOf(t, api.client, flour, 600, "gr", 12, "porciones")

	expect := func(label string, want float64) {
		t.Helper()
		if got := stockOf(t, api.client, flour); !near(got, want) {
			t.Errorf("%s: stock = %v, want %v", label, got, want)
		}
	}

	sale := sellOK(t, api.client, map[string]any{"recipe_id": cake, "quantity_sold": 1})
	expect("by default a sale is of a whole recipe", 9400)
	if sale["sold_as"] != "recipe" {
		t.Errorf("sold_as = %v, want recipe", sale["sold_as"])
	}

	sellOK(t, api.client, map[string]any{"recipe_id": cake, "quantity_sold": 1, "sold_as": "recipe"})
	expect("explicit recipe", 8800)

	sale = sellOK(t, api.client, map[string]any{"recipe_id": cake, "quantity_sold": 3, "sold_as": "portion"})
	expect("3 portions of a recipe of 12 take 3/12 of it", 8650)
	if sale["sold_as"] != "portion" {
		t.Errorf("sold_as = %v, want portion", sale["sold_as"])
	}
	if got := sale["total_price"].(float64); got != 3000 {
		t.Errorf("total_price = %v, want 3000 (3 x 1000)", got)
	}

	t.Run("when the recipe does not yield portions, a portion is one unit of what it yields", func(t *testing.T) {
		bread := recipeOf(t, api.client, flour, 1000, "gr", 2, "kg")
		sellOK(t, api.client, map[string]any{"recipe_id": bread, "quantity_sold": 1, "sold_as": "portion"})
		expect("1 of the 2 kg", 8150)
	})

	t.Run("anything else is refused", func(t *testing.T) {
		for _, body := range []map[string]any{
			{"recipe_id": cake, "quantity_sold": 1, "unit_price": 1, "sold_as": "kilo"},
			{"recipe_id": cake, "quantity_sold": 1, "unit_price": 1, "sold_as": ""},
			{"recipe_id": cake, "quantity_sold": 0, "unit_price": 1, "sold_as": "portion"},
			{"recipe_id": cake, "quantity_sold": 1.5, "unit_price": 1, "sold_as": "portion"},
		} {
			if status, _ := api.do("POST", "/sales", body); status != http.StatusBadRequest {
				t.Errorf("%v: status = %d, want 400", body, status)
			}
		}
		if _, err := api.db.Exec(context.Background(), `update recipes set yield = 0 where id = $1`, cake); err != nil {
			t.Fatal(err)
		}
		if status, _ := api.do("POST", "/sales", map[string]any{"recipe_id": cake, "quantity_sold": 1, "unit_price": 1, "sold_as": "portion"}); status != http.StatusBadRequest {
			t.Errorf("portion of a recipe that yields 0: status = %d, want 400", status)
		}
		expect("nothing was taken by the refused sales", 8150)
	})
}

type shortfall struct {
	Ingredient string
	Missing    float64
	Unit       string
}

func shortfalls(t *testing.T, sale map[string]any) []shortfall {
	t.Helper()
	raw, ok := sale["stock_shortfalls"]
	if !ok || raw == nil {
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		t.Fatalf("stock_shortfalls is not a list: %v", raw)
	}
	var out []shortfall
	for _, r := range list {
		m := r.(map[string]any)
		out = append(out, shortfall{m["ingredient"].(string), m["missing"].(float64), m["unit"].(string)})
	}
	return out
}

// V4: a sale without enough stock goes through; the stock stops at 0 and the sale says what was missing.
func TestV4_SellingWithoutEnoughStockIsAllowed(t *testing.T) {
	api := newAPIAt(t, marchClock)
	flour := createIngredient(t, api.client, map[string]any{
		"name": "Harina", "default_unit": "gr", "package_size": 1000, "package_price": 2000, "stock_quantity": 100,
	})
	recipe := recipeOf(t, api.client, flour, 500, "gr", 1, "porciones")

	sale := sellOK(t, api.client, map[string]any{"recipe_id": recipe, "quantity_sold": 1})
	if got := stockOf(t, api.client, flour); got != 0 {
		t.Errorf("stock = %v, want 0 (never negative)", got)
	}
	short := shortfalls(t, sale)
	if len(short) != 1 || short[0].Ingredient != "Harina" || !near(short[0].Missing, 400) || short[0].Unit != "gr" {
		t.Errorf("stock_shortfalls = %v, want Harina 400 gr", short)
	}
	line := usedLines(t, sale)[0]
	if q := line["quantity_used"].(float64); !near(q, 500) {
		t.Errorf("quantity_used = %v, want 500: the cost counts the whole quantity", q)
	}

	t.Run("with enough stock there is nothing to warn about", func(t *testing.T) {
		rich := createIngredient(t, api.client, map[string]any{
			"name": "Azúcar", "default_unit": "gr", "package_size": 1000, "package_price": 1000, "stock_quantity": 5000,
		})
		sale := sellOK(t, api.client, map[string]any{"recipe_id": recipeOf(t, api.client, rich, 100, "gr", 1, "porciones"), "quantity_sold": 1})
		if short := shortfalls(t, sale); len(short) != 0 {
			t.Errorf("stock_shortfalls = %v, want none", short)
		}
	})

	t.Run("an ingredient with no stock loaded does not block the sale", func(t *testing.T) {
		none := createIngredient(t, api.client, map[string]any{
			"name": "Sal", "default_unit": "gr", "package_size": 1000, "package_price": 500,
		})
		sale := sellOK(t, api.client, map[string]any{"recipe_id": recipeOf(t, api.client, none, 10, "gr", 1, "porciones"), "quantity_sold": 2})
		if short := shortfalls(t, sale); len(short) != 1 || !near(short[0].Missing, 20) {
			t.Errorf("stock_shortfalls = %v, want 20 gr missing", short)
		}
	})
}

// V5: cancelling a sale gives back what was really taken.
func TestV5_CancellingReturnsWhatWasTaken(t *testing.T) {
	api := newAPIAt(t, marchClock)
	flour := createIngredient(t, api.client, map[string]any{
		"name": "Harina", "default_unit": "gr", "package_size": 1000, "package_price": 2000, "stock_quantity": 100,
	})
	recipe := recipeOf(t, api.client, flour, 500, "gr", 1, "porciones")

	short := sellOK(t, api.client, map[string]any{"recipe_id": recipe, "quantity_sold": 1})
	var deducted float64
	if err := api.db.QueryRow(context.Background(),
		`select stock_deducted from sale_ingredients where sale_id = $1`, short["id"]).Scan(&deducted); err != nil {
		t.Fatalf("sale_ingredients.stock_deducted: %v", err)
	}
	if !near(deducted, 100) {
		t.Errorf("stock_deducted = %v, want 100 (what there was)", deducted)
	}
	if status, raw := api.do("DELETE", "/sales/"+short["id"].(string), nil); status != http.StatusNoContent && status != http.StatusOK {
		t.Fatalf("DELETE: status = %d: %s", status, raw)
	}
	if got := stockOf(t, api.client, flour); !near(got, 100) {
		t.Errorf("stock after cancelling = %v, want 100 (not 500: 400 were never taken)", got)
	}

	t.Run("a normal sale gives back all of it, converted", func(t *testing.T) {
		manjar := createIngredient(t, api.client, map[string]any{
			"name": "Manjar", "default_unit": "kg", "package_size": 1, "package_price": 8000, "stock_quantity": 1,
		})
		sale := sellOK(t, api.client, map[string]any{"recipe_id": recipeOf(t, api.client, manjar, 200, "gr", 1, "porciones"), "quantity_sold": 2})
		if got := stockOf(t, api.client, manjar); !near(got, 0.6) {
			t.Fatalf("stock = %v, want 0.6", got)
		}
		if status, _ := api.do("DELETE", "/sales/"+sale["id"].(string), nil); status != http.StatusNoContent && status != http.StatusOK {
			t.Fatalf("DELETE: status = %d", status)
		}
		if got := stockOf(t, api.client, manjar); !near(got, 1) {
			t.Errorf("stock after cancelling = %v, want 1", got)
		}
	})
}

// V6: sales at the same time never leave a negative stock or an error.
func TestV6_SimultaneousSalesNeverBreakTheStock(t *testing.T) {
	api := newAPIAt(t, marchClock)
	flour := createIngredient(t, api.client, map[string]any{
		"name": "Harina", "default_unit": "gr", "package_size": 1000, "package_price": 2000, "stock_quantity": 1000,
	})
	recipe := recipeOf(t, api.client, flour, 500, "gr", 1, "porciones")

	const sales = 6
	statuses := make([]int, sales)
	bodies := make([]string, sales)
	var wg sync.WaitGroup
	for i := 0; i < sales; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			raw, _ := json.Marshal(map[string]any{"recipe_id": recipe, "quantity_sold": 1, "unit_price": 1000})
			req := httptest.NewRequest("POST", "/sales", strings.NewReader(string(raw)))
			req.Header.Set("Authorization", "Bearer "+api.token)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			api.h.ServeHTTP(rec, req)
			statuses[i], bodies[i] = rec.Code, rec.Body.String()
		}(i)
	}
	wg.Wait()

	for i, s := range statuses {
		if s != http.StatusCreated {
			t.Errorf("sale %d: status = %d, want 201: %s", i, s, bodies[i])
		}
	}
	if got := stockOf(t, api.client, flour); got != 0 {
		t.Errorf("stock = %v, want 0", got)
	}
	var deducted, used float64
	if err := api.db.QueryRow(context.Background(),
		`select coalesce(sum(stock_deducted), 0), coalesce(sum(quantity_used), 0) from sale_ingredients`).Scan(&deducted, &used); err != nil {
		t.Fatal(err)
	}
	if !near(deducted, 1000) {
		t.Errorf("stock taken in all = %v, want exactly the 1000 there was", deducted)
	}
	if !near(used, sales*500) {
		t.Errorf("quantity used in all = %v, want %d (the cost counts all of it)", used, sales*500)
	}
}
