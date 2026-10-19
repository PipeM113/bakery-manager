package app_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/PipeM113/bakery-manager/internal/app"
	"github.com/PipeM113/bakery-manager/internal/config"
	"github.com/PipeM113/bakery-manager/internal/shared/kernel"
	"github.com/PipeM113/bakery-manager/internal/testsupport/pgtest"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// Sprint 2a: the price rules (P3, P4, P5, P7) seen through the real router and a real database.

// newPricingAPI is newAPI with a login done and the admin pool exposed for direct SQL.
func newPricingAPI(t *testing.T) (*client, *pgxpool.Pool) {
	t.Helper()
	admin := pgtest.NewDatabase(t)
	pgtest.Migrate(t, admin, "../../db/migrations", "up")
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(context.Background(),
		`insert into users (name, email, password, role) values ('Test Owner', $1, $2, 'owner')`, testEmail, string(hash)); err != nil {
		t.Fatal(err)
	}
	pool, err := kernel.Open(context.Background(), admin.Config().ConnConfig.ConnString(), 4)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	api := &client{t: t, h: app.New(config.Config{Port: "8080", DatabaseURL: "unused", JWTSecret: goodSecret, DBMaxConns: 4}, pool, nil)}
	status, raw := api.do("POST", "/auth/login", map[string]string{"email": testEmail, "password": testPassword})
	if status != http.StatusOK {
		t.Fatalf("login: status = %d: %s", status, raw)
	}
	api.token, _ = decode(t, raw)["token"].(string)
	return api, admin
}

func createIngredient(t *testing.T, api *client, body map[string]any) string {
	t.Helper()
	if _, ok := body["brand"]; !ok {
		body["brand"] = "Marca"
	}
	status, raw := api.do("POST", "/ingredients", body)
	if status != http.StatusCreated {
		t.Fatalf("create ingredient: status = %d: %s", status, raw)
	}
	id, _ := decode(t, raw)["id"].(string)
	return id
}

func createRecipe(t *testing.T, api *client, body map[string]any) string {
	t.Helper()
	status, raw := api.do("POST", "/recipes", body)
	if status != http.StatusCreated {
		t.Fatalf("create recipe: status = %d: %s", status, raw)
	}
	id, _ := decode(t, raw)["id"].(string)
	return id
}

// flourRecipe is a recipe whose ingredients cost exactly 1000 (500 gr at 2 per gr).
func flourRecipe(t *testing.T, api *client, extra map[string]any) string {
	t.Helper()
	flour := createIngredient(t, api, map[string]any{
		"name": "Harina", "default_unit": "gr", "package_size": 1000, "package_price": 2000,
	})
	body := map[string]any{
		"name": "Bizcocho", "description": "Bizcocho simple", "yield": 1, "yield_unit": "porciones",
		"ingredients": []map[string]any{{"ingredient_id": flour, "quantity": 500, "unit": "gr"}},
	}
	for k, v := range extra {
		body[k] = v
	}
	return createRecipe(t, api, body)
}

func getJSON(t *testing.T, api *client, path string) map[string]any {
	t.Helper()
	status, raw := api.do("GET", path, nil)
	if status != http.StatusOK {
		t.Fatalf("GET %s: status = %d: %s", path, status, raw)
	}
	return decode(t, raw)
}

// setCosts stores the percentages of a recipe the way the app does when the user edits them.
func setCosts(t *testing.T, api *client, id string, indirect, labor, markup float64) {
	t.Helper()
	status, raw := api.do("PUT", "/recipes/"+id+"/costs", map[string]any{
		"indirect_cost_pct": indirect, "labor_cost_pct": labor, "margin_pct": markup,
	})
	if status != http.StatusOK {
		t.Fatalf("set costs: status = %d: %s", status, raw)
	}
}

// P4: a percentage that is not sent falls back to the one stored in the recipe; one that is
// sent is used, 0 included. Stored here: 15% indirect, 30% labor and 30% markup.
func TestP4_PercentagesFallBackToTheRecipeOnlyWhenAbsent(t *testing.T) {
	api, _ := newPricingAPI(t)
	id := flourRecipe(t, api, nil)
	setCosts(t, api, id, 0.15, 0.30, 0.30)

	t.Run("nothing sent: the three stored percentages apply", func(t *testing.T) {
		// 1000 + 150 + 300 = 1450; markup 30% gives 1885, rounded up to 2000.
		body := getJSON(t, api, "/recipes/"+id+"/cost")
		if got := number(t, body, "indirect_costs"); got != 150 {
			t.Errorf("indirect_costs = %v, want 150", got)
		}
		if got := number(t, body, "labor_costs"); got != 300 {
			t.Errorf("labor_costs = %v, want 300", got)
		}
		if got := number(t, body, "margin_pct"); got != 0.30 {
			t.Errorf("margin_pct = %v, want 0.30", got)
		}
		if got := number(t, body, "suggested_price"); got != 2000 {
			t.Errorf("suggested_price = %v, want 2000", got)
		}
	})

	t.Run("an explicit 0 is respected", func(t *testing.T) {
		body := getJSON(t, api, "/recipes/"+id+"/cost?indirect_cost_pct=0&labor_cost_pct=0&margin_pct=0")
		if got := number(t, body, "indirect_costs"); got != 0 {
			t.Errorf("indirect_costs = %v, want 0", got)
		}
		if got := number(t, body, "labor_costs"); got != 0 {
			t.Errorf("labor_costs = %v, want 0", got)
		}
		if got := number(t, body, "suggested_price"); got != 1000 {
			t.Errorf("suggested_price = %v, want 1000", got)
		}
	})

	t.Run("only the markup sent as 0: the stored indirect and labor stay", func(t *testing.T) {
		body := getJSON(t, api, "/recipes/"+id+"/cost?margin_pct=0")
		if got := number(t, body, "suggested_price"); got != 1500 { // 1450 rounded up
			t.Errorf("suggested_price = %v, want 1500", got)
		}
	})

	t.Run("the limits are accepted", func(t *testing.T) {
		for _, q := range []string{"indirect_cost_pct=1", "labor_cost_pct=1", "margin_pct=10"} {
			if status, raw := api.do("GET", "/recipes/"+id+"/cost?"+q, nil); status != http.StatusOK {
				t.Errorf("%s: status = %d, want 200: %s", q, status, raw)
			}
		}
	})

	t.Run("out of range or invalid values are rejected", func(t *testing.T) {
		for _, q := range []string{
			"indirect_cost_pct=1.01", "labor_cost_pct=1.5", "margin_pct=10.01",
			"indirect_cost_pct=-0.1", "labor_cost_pct=-1", "margin_pct=-0.5",
			"indirect_cost_pct=abc", "margin_pct=abc", "labor_cost_pct=NaN",
		} {
			if status, _ := api.do("GET", "/recipes/"+id+"/cost?"+q, nil); status != http.StatusBadRequest {
				t.Errorf("%s: status = %d, want 400", q, status)
			}
		}
	})

	t.Run("saving costs outside the range is rejected", func(t *testing.T) {
		for _, body := range []map[string]any{
			{"indirect_cost_pct": 1.5, "labor_cost_pct": 0.3, "margin_pct": 0.3},
			{"indirect_cost_pct": 0.1, "labor_cost_pct": 1.01, "margin_pct": 0.3},
			{"indirect_cost_pct": 0.1, "labor_cost_pct": 0.3, "margin_pct": 11},
		} {
			if status, _ := api.do("PUT", "/recipes/"+id+"/costs", body); status != http.StatusBadRequest {
				t.Errorf("%v: status = %d, want 400", body, status)
			}
		}
	})
}

// P4: a recipe can be created with 0% indirect and 0% labor; leaving them out gives the defaults.
func TestP4_RecipeCreationKeepsAnExplicitZero(t *testing.T) {
	api, _ := newPricingAPI(t)

	zero := flourRecipe(t, api, map[string]any{"indirect_cost_pct": 0, "labor_cost_pct": 0})
	costs := getJSON(t, api, "/recipes/"+zero+"/costs")
	if number(t, costs, "indirect_cost_pct") != 0 || number(t, costs, "labor_cost_pct") != 0 {
		t.Errorf("explicit zeros were replaced: %v", costs)
	}
}

// P4: editing a recipe without sending its percentages must not wipe them (today a PUT
// without them stores 0% indirect and 0% labor); sending them, 0 included, changes them.
func TestP4_UpdatingARecipeKeepsThePercentagesItDoesNotSend(t *testing.T) {
	api, _ := newPricingAPI(t)
	id := flourRecipe(t, api, nil)
	setCosts(t, api, id, 0.20, 0.40, 0.30)
	sugar := createIngredient(t, api, map[string]any{"name": "Azúcar", "default_unit": "gr", "package_size": 1000, "package_price": 1500})
	edit := func(extra map[string]any) {
		body := map[string]any{
			"name": "Bizcocho", "description": "d", "yield": 2, "yield_unit": "porciones",
			"ingredients": []map[string]any{{"ingredient_id": sugar, "quantity": 100, "unit": "gr"}},
		}
		for k, v := range extra {
			body[k] = v
		}
		if status, raw := api.do("PUT", "/recipes/"+id, body); status != http.StatusOK {
			t.Fatalf("update: status = %d: %s", status, raw)
		}
	}

	edit(nil)
	costs := getJSON(t, api, "/recipes/"+id+"/costs")
	if number(t, costs, "indirect_cost_pct") != 0.20 || number(t, costs, "labor_cost_pct") != 0.40 {
		t.Errorf("the percentages were wiped by an update that did not send them: %v", costs)
	}

	edit(map[string]any{"indirect_cost_pct": 0, "labor_cost_pct": 0.10})
	costs = getJSON(t, api, "/recipes/"+id+"/costs")
	if number(t, costs, "indirect_cost_pct") != 0 || number(t, costs, "labor_cost_pct") != 0.10 {
		t.Errorf("the percentages that were sent were not stored: %v", costs)
	}
}

// A new recipe starts with 30% in the three percentages (owner decision, Sprint 2a); from
// then on each recipe keeps its own.
func TestP4_NewRecipeStartsWithThirtyPercentInTheThree(t *testing.T) {
	api, _ := newPricingAPI(t)
	id := flourRecipe(t, api, nil)
	costs := getJSON(t, api, "/recipes/"+id+"/costs")
	for _, k := range []string{"indirect_cost_pct", "labor_cost_pct", "margin_pct"} {
		if got := number(t, costs, k); got != 0.30 {
			t.Errorf("%s = %v, want 0.30 for a new recipe", k, got)
		}
	}
}

// P3 through the endpoint: with the recipe defaults the cost is 1450; a 30% markup is 1885
// per portion, charged once.
func TestP3_SimulateEndpointAppliesTheMarkupOnce(t *testing.T) {
	api, _ := newPricingAPI(t)
	id := flourRecipe(t, api, nil)
	setCosts(t, api, id, 0.15, 0.30, 0.30)

	status, raw := api.do("POST", "/recipes/"+id+"/cost/simulate", map[string]any{"margin_pct": 0.3})
	if status != http.StatusOK {
		t.Fatalf("status = %d: %s", status, raw)
	}
	body := decode(t, raw)
	if got := number(t, body, "suggested_price"); got < 1884.999 || got > 1885.001 {
		t.Errorf("suggested_price = %v, want 1885 (it was 2450.5 with the margin applied twice)", got)
	}
	if got := number(t, body, "profit_amount"); got < 434.999 || got > 435.001 {
		t.Errorf("profit_amount = %v, want 435", got)
	}
}

// P5: the yield must be positive everywhere it can come in.
func TestP5_YieldMustBePositive(t *testing.T) {
	api, db := newPricingAPI(t)
	id := flourRecipe(t, api, nil)

	t.Run("updating a recipe with a non-positive yield is rejected", func(t *testing.T) {
		flour := createIngredient(t, api, map[string]any{"name": "Azúcar", "default_unit": "gr", "package_size": 1000, "package_price": 1500})
		for _, y := range []float64{0, -3} {
			status, raw := api.do("PUT", "/recipes/"+id, map[string]any{
				"name": "Bizcocho", "description": "d", "yield": y, "yield_unit": "porciones",
				"ingredients": []map[string]any{{"ingredient_id": flour, "quantity": 100, "unit": "gr"}},
			})
			if status != http.StatusBadRequest {
				t.Errorf("yield %v: status = %d, want 400: %s", y, status, raw)
			}
		}
	})

	t.Run("a stored recipe with yield 0 gets a clear 422 from the cost endpoints", func(t *testing.T) {
		if _, err := db.Exec(context.Background(), `update recipes set yield = 0 where id = $1`, id); err != nil {
			t.Fatal(err)
		}
		status, raw := api.do("GET", "/recipes/"+id+"/cost", nil)
		if status != http.StatusUnprocessableEntity {
			t.Fatalf("cost: status = %d, want 422: %s", status, raw)
		}
		if msg, _ := decode(t, raw)["error"].(string); !strings.Contains(strings.ToLower(msg), "rendimiento") {
			t.Errorf("the error should mention the yield: %s", raw)
		}

		status, _ = api.do("POST", "/recipes/"+id+"/cost/simulate", map[string]any{"margin_pct": 0.3})
		if status != http.StatusUnprocessableEntity {
			t.Errorf("simulate: status = %d, want 422", status)
		}
		status, _ = api.do("POST", "/quotations/generate", map[string]any{"recipe_id": id, "client_name": "Ana", "margin_pct": 0.3})
		if status != http.StatusUnprocessableEntity {
			t.Errorf("quotation: status = %d, want 422", status)
		}
	})
}

// P7: the cost of an ingredient is package price / package size, exactly. A third of 2000 is
// stored with four decimals as 666.6667, and three of them would cost 2000.0001.
func TestP7_IngredientCostComesFromThePackageNotTheRoundedUnitPrice(t *testing.T) {
	api, db := newPricingAPI(t)
	third := createIngredient(t, api, map[string]any{"name": "Huevo", "default_unit": "und", "package_size": 3, "package_price": 2000})
	var stored float64
	if err := db.QueryRow(context.Background(), `select price_per_unit from ingredients where id = $1`, third).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != 666.6667 {
		t.Fatalf("precondition: stored price_per_unit = %v, want 666.6667", stored)
	}
	id := createRecipe(t, api, map[string]any{
		"name": "Tres huevos", "description": "d", "yield": 1, "yield_unit": "porciones",
		"ingredients": []map[string]any{{"ingredient_id": third, "quantity": 3, "unit": "und"}},
	})

	body := getJSON(t, api, "/recipes/"+id+"/cost?indirect_cost_pct=0&labor_cost_pct=0&margin_pct=0")
	if got := number(t, body, "ingredients_total"); got != 2000 {
		t.Errorf("ingredients_total = %v, want 2000", got)
	}
	if got := number(t, body, "suggested_price"); got != 2000 {
		t.Errorf("suggested_price = %v, want 2000 (2500 when the rounded unit price is used)", got)
	}
}

// P6 + P4: the quotation uses the same calculation as the cost endpoint, including an
// explicit 0%. Ingredients 1000, indirect 0%, labor 20%, markup 25%, yield 9 cost exactly 1500.
func TestP6_QuotationPriceIsTheCostEndpointPrice(t *testing.T) {
	api, _ := newPricingAPI(t)
	flour := createIngredient(t, api, map[string]any{"name": "Harina", "default_unit": "gr", "package_size": 1000, "package_price": 1000})
	id := createRecipe(t, api, map[string]any{
		"name": "Torta", "description": "Torta de prueba", "yield": 9, "yield_unit": "porciones",
		"ingredients": []map[string]any{{"ingredient_id": flour, "quantity": 1000, "unit": "gr"}},
	})

	cost := getJSON(t, api, "/recipes/"+id+"/cost?indirect_cost_pct=0&labor_cost_pct=0.2&margin_pct=0.25")
	if got := number(t, cost, "suggested_price"); got != 1500 {
		t.Fatalf("cost endpoint: suggested_price = %v, want 1500", got)
	}

	status, raw := api.do("POST", "/quotations/generate", map[string]any{
		"recipe_id": id, "client_name": "Ana", "margin_pct": 0.25,
		"indirect_cost_pct": 0, "labor_cost_pct": 0.2,
	})
	if status != http.StatusOK {
		t.Fatalf("generate: status = %d: %s", status, raw)
	}
	if !strings.HasPrefix(string(raw), "%PDF") {
		t.Fatalf("the response is not a PDF")
	}

	status, raw = api.do("GET", "/quotations", nil)
	if status != http.StatusOK {
		t.Fatalf("list: status = %d: %s", status, raw)
	}
	var list []map[string]any
	if err := json.Unmarshal(raw, &list); err != nil || len(list) != 1 {
		t.Fatalf("unexpected list: %v %s", err, raw)
	}
	if got := number(t, list[0], "final_price"); got != 1500 {
		t.Errorf("saved final_price = %v, want 1500", got)
	}
}

// P9: the cost breakdown no longer carries the duplicated subtotal.
func TestP9_CostResponseHasNoSubtotalSinMargen(t *testing.T) {
	api, _ := newPricingAPI(t)
	id := flourRecipe(t, api, nil)
	_, raw := api.do("GET", "/recipes/"+id+"/cost", nil)
	if strings.Contains(string(raw), "subtotal_sin_margen") {
		t.Errorf("the response still carries subtotal_sin_margen: %s", raw)
	}
}
