package app_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Sprint 2b: whole pesos (D2), expense categories the user can create (D3, D4, D5) and the
// dashboard (D6), seen through the real router and a real database.

type category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func listCategories(t *testing.T, api *client) []category {
	t.Helper()
	status, raw := api.do("GET", "/expense-categories", nil)
	if status != http.StatusOK {
		t.Fatalf("list categories: status = %d: %s", status, raw)
	}
	var out []category
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("categories are not a JSON list: %v\n%s", err, raw)
	}
	return out
}

func categoryID(t *testing.T, api *client, name string) string {
	t.Helper()
	for _, c := range listCategories(t, api) {
		if strings.EqualFold(c.Name, name) {
			return c.ID
		}
	}
	t.Fatalf("no category named %q in %v", name, listCategories(t, api))
	return ""
}

func twoDaysAgo() string { return time.Now().AddDate(0, 0, -2).Format("2006-01-02") }

// D2: money enters as whole pesos, positive, and below two thousand million.
func TestD2_AmountsMustBeWholePositivePesos(t *testing.T) {
	api, _ := newPricingAPI(t)
	otros := categoryID(t, api, "otros")
	flourRecipeID := flourRecipe(t, api, nil)

	bad := []any{1500.5, 0, -5, 2000000001, "1500"}
	for _, amount := range bad {
		if status, _ := api.do("POST", "/fixed-costs", map[string]any{"name": "Arriendo", "monthly_amount": amount}); status != http.StatusBadRequest {
			t.Errorf("fixed cost %v: status = %d, want 400", amount, status)
		}
		if status, _ := api.do("POST", "/expenses", map[string]any{
			"description": "Gas", "amount": amount, "category_id": otros, "expense_date": twoDaysAgo(),
		}); status != http.StatusBadRequest {
			t.Errorf("expense %v: status = %d, want 400", amount, status)
		}
	}
	for _, price := range []any{1500.5, -1, 2000000001, "1500"} {
		if status, _ := api.do("POST", "/sales", map[string]any{"recipe_id": flourRecipeID, "quantity_sold": 1, "unit_price": price}); status != http.StatusBadRequest {
			t.Errorf("sale price %v: status = %d, want 400", price, status)
		}
	}

	t.Run("whole amounts at the edges are accepted", func(t *testing.T) {
		if status, raw := api.do("POST", "/fixed-costs", map[string]any{"name": "Arriendo", "monthly_amount": 250000}); status != http.StatusCreated {
			t.Errorf("fixed cost: status = %d: %s", status, raw)
		}
		if status, raw := api.do("POST", "/fixed-costs", map[string]any{"name": "Tope", "monthly_amount": 2000000000}); status != http.StatusCreated {
			t.Errorf("fixed cost at the limit: status = %d: %s", status, raw)
		}
		if status, raw := api.do("POST", "/expenses", map[string]any{
			"description": "Gas", "amount": 15000, "category_id": otros, "expense_date": twoDaysAgo(),
		}); status != http.StatusCreated {
			t.Errorf("expense: status = %d: %s", status, raw)
		}
		if status, raw := api.do("POST", "/sales", map[string]any{"recipe_id": flourRecipeID, "quantity_sold": 1, "unit_price": 0}); status != http.StatusCreated {
			t.Errorf("sale at 0: status = %d: %s", status, raw)
		}
	})
}

// D3: the starting list, and creating, renaming and deleting categories.
func TestD3_ExpenseCategoriesCanBeManaged(t *testing.T) {
	api, _ := newPricingAPI(t)

	t.Run("the starting list is there", func(t *testing.T) {
		have := map[string]bool{}
		for _, c := range listCategories(t, api) {
			have[strings.ToLower(c.Name)] = true
		}
		for _, name := range []string{"gas", "luz", "agua", "arriendo", "envases", "transporte", "sueldos", "otros"} {
			if !have[name] {
				t.Errorf("missing starting category %q", name)
			}
		}
		if have["ingredientes"] {
			t.Error("'ingredientes' should not be a starting category")
		}
	})

	var created category
	t.Run("a new category can be created", func(t *testing.T) {
		status, raw := api.do("POST", "/expense-categories", map[string]any{"name": "  Mantención del horno "})
		if status != http.StatusCreated {
			t.Fatalf("status = %d: %s", status, raw)
		}
		if err := json.Unmarshal(raw, &created); err != nil || created.ID == "" {
			t.Fatalf("unexpected response: %v %s", err, raw)
		}
		if created.Name != "Mantención del horno" {
			t.Errorf("name = %q, want it trimmed", created.Name)
		}
	})

	t.Run("a name that exists, ignoring case, is a conflict", func(t *testing.T) {
		for _, n := range []string{"mantención del HORNO", "GAS"} {
			if status, _ := api.do("POST", "/expense-categories", map[string]any{"name": n}); status != http.StatusConflict {
				t.Errorf("%q: status = %d, want 409", n, status)
			}
		}
	})

	t.Run("empty or too long names are rejected", func(t *testing.T) {
		for _, n := range []string{"", "   ", strings.Repeat("x", 51)} {
			if status, _ := api.do("POST", "/expense-categories", map[string]any{"name": n}); status != http.StatusBadRequest {
				t.Errorf("%q: status = %d, want 400", n, status)
			}
		}
	})

	t.Run("a category can be renamed", func(t *testing.T) {
		status, raw := api.do("PUT", "/expense-categories/"+created.ID, map[string]any{"name": "Mantención"})
		if status != http.StatusOK {
			t.Fatalf("status = %d: %s", status, raw)
		}
		if status, _ := api.do("PUT", "/expense-categories/"+created.ID, map[string]any{"name": "gas"}); status != http.StatusConflict {
			t.Errorf("renaming onto another name: status = %d, want 409", status)
		}
		if status, _ := api.do("PUT", "/expense-categories/00000000-0000-0000-0000-000000000000", map[string]any{"name": "Algo"}); status != http.StatusNotFound {
			t.Errorf("unknown id: status = %d, want 404", status)
		}
	})

	t.Run("a category without expenses can be deleted, one with expenses cannot", func(t *testing.T) {
		used := categoryID(t, api, "luz")
		if status, raw := api.do("POST", "/expenses", map[string]any{
			"description": "Cuenta de luz", "amount": 30000, "category_id": used, "expense_date": twoDaysAgo(),
		}); status != http.StatusCreated {
			t.Fatalf("expense: status = %d: %s", status, raw)
		}
		if status, _ := api.do("DELETE", "/expense-categories/"+used, nil); status != http.StatusConflict {
			t.Errorf("deleting a used category: status = %d, want 409", status)
		}
		if status, _ := api.do("DELETE", "/expense-categories/"+created.ID, nil); status != http.StatusNoContent {
			t.Errorf("deleting an unused category: status = %d, want 204", status)
		}
		if status, _ := api.do("DELETE", "/expense-categories/"+created.ID, nil); status != http.StatusNotFound {
			t.Errorf("deleting it again: status = %d, want 404", status)
		}
	})
}

// D5: ingredients are costed from the sales, so they are not an expense category.
func TestD5_IngredientsAreNotAnExpenseCategory(t *testing.T) {
	api, _ := newPricingAPI(t)
	for _, n := range []string{"ingredientes", "Ingredientes", "  INGREDIENTES  "} {
		status, raw := api.do("POST", "/expense-categories", map[string]any{"name": n})
		if status != http.StatusBadRequest {
			t.Errorf("create %q: status = %d, want 400", n, status)
		}
		if msg, _ := decode(t, raw)["error"].(string); !strings.Contains(strings.ToLower(msg), "ventas") {
			t.Errorf("create %q: the message should explain that ingredients are costed from sales: %s", n, raw)
		}
	}
	gas := categoryID(t, api, "gas")
	if status, _ := api.do("PUT", "/expense-categories/"+gas, map[string]any{"name": "Ingredientes"}); status != http.StatusBadRequest {
		t.Errorf("rename to ingredientes: status = %d, want 400", status)
	}
}

// D4: an expense points at a category.
func TestD4_ExpensesUseCategories(t *testing.T) {
	api, _ := newPricingAPI(t)
	gas := categoryID(t, api, "gas")
	day := time.Now().AddDate(0, 0, -2)

	t.Run("an expense needs an existing category", func(t *testing.T) {
		for _, body := range []map[string]any{
			{"description": "Gas", "amount": 1000, "expense_date": twoDaysAgo()},
			{"description": "Gas", "amount": 1000, "category_id": "", "expense_date": twoDaysAgo()},
			{"description": "Gas", "amount": 1000, "category_id": "00000000-0000-0000-0000-000000000000", "expense_date": twoDaysAgo()},
			{"description": "Gas", "amount": 1000, "category_id": "not-a-uuid", "expense_date": twoDaysAgo()},
		} {
			if status, _ := api.do("POST", "/expenses", body); status != http.StatusBadRequest {
				t.Errorf("%v: status = %d, want 400", body, status)
			}
		}
	})

	var id string
	t.Run("the list and the response carry the category name", func(t *testing.T) {
		status, raw := api.do("POST", "/expenses", map[string]any{
			"description": "Balón de gas", "amount": 21000, "category_id": gas, "expense_date": day.Format("2006-01-02"),
		})
		if status != http.StatusCreated {
			t.Fatalf("status = %d: %s", status, raw)
		}
		body := decode(t, raw)
		id, _ = body["id"].(string)
		if body["category_id"] != gas || !strings.EqualFold(body["category"].(string), "gas") {
			t.Errorf("response = %v, want category_id %s and category Gas", body, gas)
		}

		status, raw = api.do("GET", "/expenses?month="+itoa(int(day.Month()))+"&year="+itoa(day.Year()), nil)
		if status != http.StatusOK {
			t.Fatalf("list: status = %d", status)
		}
		var list []map[string]any
		if err := json.Unmarshal(raw, &list); err != nil || len(list) != 1 {
			t.Fatalf("unexpected list: %v %s", err, raw)
		}
		if !strings.EqualFold(list[0]["category"].(string), "gas") {
			t.Errorf("listed category = %v, want Gas", list[0]["category"])
		}
	})

	t.Run("renaming the category shows in the expenses", func(t *testing.T) {
		if status, raw := api.do("PUT", "/expense-categories/"+gas, map[string]any{"name": "Gas licuado"}); status != http.StatusOK {
			t.Fatalf("rename: status = %d: %s", status, raw)
		}
		_, raw := api.do("GET", "/expenses?month="+itoa(int(day.Month()))+"&year="+itoa(day.Year()), nil)
		if !strings.Contains(string(raw), "Gas licuado") {
			t.Errorf("the expense should show the new category name: %s", raw)
		}
	})

	t.Run("an expense can be moved to another category", func(t *testing.T) {
		luz := categoryID(t, api, "luz")
		status, raw := api.do("PUT", "/expenses/"+id, map[string]any{
			"description": "Balón de gas", "amount": 21000, "category_id": luz, "expense_date": day.Format("2006-01-02"),
		})
		if status != http.StatusOK {
			t.Fatalf("status = %d: %s", status, raw)
		}
		if got := decode(t, raw)["category_id"]; got != luz {
			t.Errorf("category_id = %v, want %v", got, luz)
		}
	})
}

// D6: the dashboard in whole pesos, with the expenses split by category.
//
//	Sales:    recipe A 3 x 3500 = 10500 (ingredients 3 x 1000 = 3000)
//	          recipe B 1 x 1000 = 1000  (ingredients 500 gr at 0.001 = 0.5)
//	Revenue 11500; ingredients 3000.5, shown as 3001 (half up).
//	Fixed costs 1000 (an inactive 777 does not count); expenses gas 1500 + envases 500.
//	Gross 11500 - 3001 = 8499; net 8499 - 1000 - 2000 = 5499; margin 5499 / 11500 = 47.8 %.
func TestD6_MonthlyDashboardInWholePesos(t *testing.T) {
	api, db := newPricingAPI(t)
	flour := createIngredient(t, api, map[string]any{"name": "Harina", "default_unit": "gr", "package_size": 1000, "package_price": 2000, "stock_quantity": 5000})
	salt := createIngredient(t, api, map[string]any{"name": "Sal", "default_unit": "gr", "package_size": 1000, "package_price": 1, "stock_quantity": 5000})
	recipeA := createRecipe(t, api, map[string]any{
		"name": "Bizcocho", "description": "d", "yield": 1,
		"ingredients": []map[string]any{{"ingredient_id": flour, "quantity": 500, "unit": "gr"}},
	})
	recipeB := createRecipe(t, api, map[string]any{
		"name": "Sal fina", "description": "d", "yield": 1,
		"ingredients": []map[string]any{{"ingredient_id": salt, "quantity": 500, "unit": "gr"}},
	})
	for _, s := range []map[string]any{
		{"recipe_id": recipeA, "quantity_sold": 3, "unit_price": 3500},
		{"recipe_id": recipeB, "quantity_sold": 1, "unit_price": 1000},
	} {
		if status, raw := api.do("POST", "/sales", s); status != http.StatusCreated {
			t.Fatalf("sale: status = %d: %s", status, raw)
		}
	}
	if status, raw := api.do("POST", "/fixed-costs", map[string]any{"name": "Arriendo", "monthly_amount": 1000}); status != http.StatusCreated {
		t.Fatalf("fixed cost: status = %d: %s", status, raw)
	}
	status, raw := api.do("POST", "/fixed-costs", map[string]any{"name": "Antiguo", "monthly_amount": 777})
	if status != http.StatusCreated {
		t.Fatalf("fixed cost: status = %d: %s", status, raw)
	}
	old := decode(t, raw)
	if status, raw := api.do("PUT", "/fixed-costs/"+old["id"].(string), map[string]any{"name": "Antiguo", "monthly_amount": 777, "is_active": false}); status != http.StatusOK {
		t.Fatalf("deactivate: status = %d: %s", status, raw)
	}
	for _, e := range []struct {
		cat    string
		amount int
	}{{"gas", 1500}, {"envases", 500}} {
		_, err := db.Exec(context.Background(), `
			insert into operational_expenses (user_id, description, amount, category_id, expense_date)
			select (select id from users limit 1), 'gasto', $2, id, current_date from expense_categories where lower(name) = $1`, e.cat, e.amount)
		if err != nil {
			t.Fatal(err)
		}
	}

	now := time.Now()
	path := "/analytics/monthly?month=" + itoa(int(now.Month())) + "&year=" + itoa(now.Year())
	status, raw = api.do("GET", path, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d: %s", status, raw)
	}
	body := decode(t, raw)
	want := map[string]float64{
		"total_revenue": 11500, "total_ingredients_cost": 3001, "total_fixed_costs": 1000,
		"total_operational_expenses": 2000, "gross_profit": 8499, "net_profit": 5499, "profit_margin": 47.8,
	}
	for k, v := range want {
		if got := number(t, body, k); got != v {
			t.Errorf("%s = %v, want %v", k, got, v)
		}
	}

	rows, _ := body["expenses_by_category"].([]any)
	if len(rows) != 2 {
		t.Fatalf("expenses_by_category = %v, want 2 rows", body["expenses_by_category"])
	}
	first, second := rows[0].(map[string]any), rows[1].(map[string]any)
	if !strings.EqualFold(first["category"].(string), "gas") || first["total"] != float64(1500) {
		t.Errorf("first row = %v, want Gas 1500 (largest first)", first)
	}
	if !strings.EqualFold(second["category"].(string), "envases") || second["total"] != float64(500) {
		t.Errorf("second row = %v, want Envases 500", second)
	}

	t.Run("a month without sales has a margin of 0, not a division by zero", func(t *testing.T) {
		status, raw := api.do("GET", "/analytics/monthly?month=1&year=2020", nil)
		if status != http.StatusOK {
			t.Fatalf("status = %d: %s", status, raw)
		}
		body := decode(t, raw)
		if number(t, body, "profit_margin") != 0 || number(t, body, "total_revenue") != 0 {
			t.Errorf("empty month = %v", body)
		}
		if rows, ok := body["expenses_by_category"].([]any); !ok || len(rows) != 0 {
			t.Errorf("expenses_by_category should be an empty list, got %v", body["expenses_by_category"])
		}
	})

	t.Run("the recipe margin has one decimal", func(t *testing.T) {
		status, raw := api.do("GET", "/analytics/recipes?month="+itoa(int(now.Month()))+"&year="+itoa(now.Year()), nil)
		if status != http.StatusOK {
			t.Fatalf("status = %d: %s", status, raw)
		}
		var list []map[string]any
		if err := json.Unmarshal(raw, &list); err != nil {
			t.Fatal(err)
		}
		for _, r := range list {
			if r["recipe_name"] != "Bizcocho" {
				continue
			}
			// (10500 - 3000) / 10500 = 71.428...: shown as 71.4
			if got := r["gross_margin"].(float64); got != 71.4 {
				t.Errorf("gross_margin = %v, want 71.4", got)
			}
			if r["cost_per_unit"] != float64(1000) || r["unit_profit"] != float64(2500) {
				t.Errorf("cost_per_unit/unit_profit = %v/%v, want 1000/2500", r["cost_per_unit"], r["unit_profit"])
			}
		}
	})
}
