package app_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Sprint 11: the business is one. Both accounts see and administer everything; a record only
// remembers who registered it (created_by). The clock stands on 2026-03-31 (Santiago).

type business struct {
	api                          *timeAPI
	a, b                         *client
	idA, idB                     string
	flour, recipe, pie           string
	sale, order, expense, fixed  string
	quotation, adjustment, spare string // spare: an unsold recipe
}

func listOf(t *testing.T, api *client, path string) []map[string]any {
	t.Helper()
	status, raw := api.do("GET", path, nil)
	if status != http.StatusOK {
		t.Fatalf("GET %s: status = %d: %s", path, status, raw)
	}
	var out []map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("GET %s: not a list: %v\n%s", path, err, raw)
	}
	return out
}

func hasID(rows []map[string]any, id string) bool {
	for _, r := range rows {
		if r["id"] == id {
			return true
		}
	}
	return false
}

func idOf(t *testing.T, api *timeAPI, email string) string {
	t.Helper()
	var id string
	if err := api.db.QueryRow(context.Background(), `select id from users where email = $1`, email).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// newBusiness makes the two accounts and lets the first one register a record of every kind.
func newBusiness(t *testing.T) *business {
	t.Helper()
	api := newAPIAt(t, marchClock)
	b := &business{api: api, a: api.client, b: secondUser(t, api)}
	b.idA, b.idB = idOf(t, api, testEmail), idOf(t, api, "helper@example.test")

	flour, cake, pie := shop(t, b.a)
	b.flour, b.recipe, b.pie = flour, cake, pie
	b.spare = recipeOf(t, b.a, createIngredient(t, b.a, map[string]any{
		"name": "Sal", "default_unit": "gr", "package_size": 1000, "package_price": 500, "stock_quantity": 1000,
	}), 5, "gr", 1, "porciones")

	b.sale = sellOK(t, b.a, map[string]any{"recipe_id": pie, "quantity_sold": 2, "unit_price": 1500, "sale_date": "2026-03-10"})["id"].(string)
	b.order = orderOK(t, b.a, map[string]any{
		"order_date": "2026-03-12", "total_price": 4000, "lines": []line{ln(pie, 2, 1000, ""), ln(cake, 1, 3000, "")},
	})["id"].(string)

	status, raw := b.a.do("POST", "/expenses", map[string]any{
		"description": "Gas", "amount": 400, "category_id": categoryID(t, b.a, "gas"), "expense_date": "2026-03-11",
	})
	if status != http.StatusCreated {
		t.Fatalf("expense: status = %d: %s", status, raw)
	}
	b.expense = decode(t, raw)["id"].(string)
	b.fixed = newFixed(t, b.a, map[string]any{"name": "Arriendo", "monthly_amount": 100, "valid_from": "2026-01"})

	status, raw = b.a.do("POST", "/quotations/generate", map[string]any{
		"recipe_id": pie, "client_name": "María", "margin_pct": 0.3, "indirect_cost_pct": 0.3,
		"labor_cost_pct": 0.3, "extra_charge": 0, "delivery_cost": 0,
	})
	if status != http.StatusOK {
		t.Fatalf("quotation: status = %d: %.200s", status, raw)
	}
	if err := api.db.QueryRow(context.Background(), `select id from quotations limit 1`).Scan(&b.quotation); err != nil {
		t.Fatalf("the quotation was not saved: %v", err)
	}
	b.adjustment = adjustOK(t, b.a, createIngredient(t, b.a, map[string]any{
		"name": "Azúcar", "default_unit": "gr", "package_size": 1000, "package_price": 1000, "stock_quantity": 500,
	}), map[string]any{"reason": "merma", "quantity": 10, "adjusted_on": "2026-03-15"})["id"].(string)
	return b
}

// N2: both accounts see the whole business, and the same numbers.
func TestN2_EverybodySeesTheWholeBusiness(t *testing.T) {
	b := newBusiness(t)

	for _, c := range []struct {
		name, path, id string
	}{
		{"sales", "/sales", b.sale},
		{"orders", "/orders", b.order},
		{"expenses", "/expenses?month=3&year=2026", b.expense},
		{"fixed costs", "/fixed-costs?month=2026-03", b.fixed},
		{"quotations", "/quotations", b.quotation},
		{"stock adjustments", "/stock-adjustments?month=2026-03", b.adjustment},
		{"recipes", "/recipes", b.recipe},
	} {
		if !hasID(listOf(t, b.b, c.path), c.id) {
			t.Errorf("the second account does not see the %s of the first (%s)", c.name, c.path)
		}
	}

	same := func(path string) {
		t.Helper()
		statusA, rawA := b.a.do("GET", path, nil)
		statusB, rawB := b.b.do("GET", path, nil)
		if statusA != http.StatusOK || statusB != http.StatusOK {
			t.Fatalf("GET %s: statuses %d and %d", path, statusA, statusB)
		}
		if string(rawA) != string(rawB) {
			t.Errorf("GET %s differs between the accounts:\n A: %s\n B: %s", path, rawA, rawB)
		}
	}
	same("/analytics/monthly?month=3&year=2026")
	same("/analytics/recipes?month=3&year=2026")
	same("/analytics/trends?months=3")

	status, raw := b.b.do("GET", "/analytics/monthly?month=3&year=2026", nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	m := decode(t, raw)
	// 3000 of the sale + 4000 of the order, 100 of the fixed cost, 400 of the expense, 10 of the merma (10 gr at 1 per gr)
	if got := number(t, m, "total_revenue"); got != 7000 {
		t.Errorf("total_revenue = %v, want 7000 (the sale and the order of the other account)", got)
	}
	if got := number(t, m, "total_fixed_costs"); got != 100 {
		t.Errorf("total_fixed_costs = %v, want 100", got)
	}
	if got := number(t, m, "total_operational_expenses"); got != 400 {
		t.Errorf("total_operational_expenses = %v, want 400", got)
	}
	if got := number(t, m, "total_waste_cost"); got != 10 {
		t.Errorf("total_waste_cost = %v, want 10", got)
	}

	t.Run("what the second account registers is seen by the first", func(t *testing.T) {
		bSale := sellOK(t, b.b, map[string]any{"recipe_id": b.pie, "quantity_sold": 1, "unit_price": 1000, "sale_date": "2026-03-20"})
		if !hasID(listOf(t, b.a, "/sales"), bSale["id"].(string)) {
			t.Error("the first account does not see the sale of the second")
		}
		_, rawA := b.a.do("GET", "/analytics/monthly?month=3&year=2026", nil)
		if got := number(t, decode(t, rawA), "total_revenue"); got != 8000 {
			t.Errorf("total_revenue = %v, want 8000", got)
		}
	})
}

// N3: either account may change or cancel what the other registered.
func TestN3_EverybodyCanChangeTheOthersRecords(t *testing.T) {
	b := newBusiness(t)
	ok := func(label string, status int, want ...int) {
		t.Helper()
		for _, w := range want {
			if status == w {
				return
			}
		}
		t.Errorf("%s: status = %d, want one of %v", label, status, want)
	}

	status, raw := b.b.do("PUT", "/recipes/"+b.recipe, map[string]any{
		"name": "Torta renombrada", "description": "Cambiada por la otra cuenta", "yield": 10, "yield_unit": "porciones",
		"ingredients": []map[string]any{{"ingredient_id": b.flour, "quantity": 500, "unit": "gr"}},
	})
	ok("edit a recipe of the other account", status, http.StatusOK)
	if status == http.StatusOK && !strings.Contains(string(raw), "Torta renombrada") {
		t.Errorf("the recipe was not renamed: %s", raw)
	}
	status, _ = b.b.do("DELETE", "/recipes/"+b.spare, nil)
	ok("delete a recipe of the other account", status, http.StatusNoContent)

	status, _ = b.b.do("PUT", "/expenses/"+b.expense, map[string]any{
		"description": "Gas corregido", "amount": 500, "category_id": categoryID(t, b.b, "gas"), "expense_date": "2026-03-11",
	})
	ok("edit an expense of the other account", status, http.StatusOK)
	status, _ = b.b.do("DELETE", "/expenses/"+b.expense, nil)
	ok("delete an expense of the other account", status, http.StatusNoContent)

	status, _ = putFixed(t, b.b, b.fixed, map[string]any{"name": "Arriendo", "monthly_amount": 150, "is_active": true})
	ok("change a fixed cost of the other account", status, http.StatusOK)
	status, _ = b.b.do("DELETE", "/fixed-costs/"+b.fixed, nil)
	ok("end a fixed cost of the other account", status, http.StatusNoContent)

	status, _ = b.b.do("PUT", "/quotations/"+b.quotation+"/cancel", nil)
	ok("cancel a quotation of the other account", status, http.StatusOK, http.StatusNoContent)

	status, _ = b.b.do("DELETE", "/orders/"+b.order, nil)
	ok("cancel an order of the other account", status, http.StatusNoContent)
	status, _ = b.b.do("DELETE", "/sales/"+b.sale, nil)
	ok("cancel a sale of the other account", status, http.StatusNoContent)

	if got := ordersCount(t, b.api); got != 0 {
		t.Errorf("orders = %d, want 0", got)
	}
}

// N4: a record remembers who registered it, and says so as created_by (no more user_id).
func TestN4_RecordsKeepWhoRegisteredThem(t *testing.T) {
	b := newBusiness(t)
	ctx := context.Background()

	bExpense := func() string {
		status, raw := b.b.do("POST", "/expenses", map[string]any{
			"description": "Envases", "amount": 100, "category_id": categoryID(t, b.b, "envases"), "expense_date": "2026-03-12",
		})
		if status != http.StatusCreated {
			t.Fatalf("expense of B: status = %d: %s", status, raw)
		}
		return decode(t, raw)["id"].(string)
	}()
	bOrder := orderOK(t, b.b, map[string]any{"lines": []line{ln(b.pie, 1, 1000, "")}})

	for table, id := range map[string]string{
		"sales": b.sale, "orders": b.order, "operational_expenses": b.expense, "fixed_costs": b.fixed,
		"quotations": b.quotation, "stock_adjustments": b.adjustment, "recipes": b.recipe,
	} {
		var by *string
		if err := b.api.db.QueryRow(ctx, `select created_by::text from `+table+` where id = $1`, id).Scan(&by); err != nil {
			t.Fatalf("%s.created_by: %v", table, err)
		}
		if by == nil || *by != b.idA {
			t.Errorf("%s.created_by = %v, want the first account", table, by)
		}
	}
	for table, id := range map[string]string{"operational_expenses": bExpense, "orders": bOrder["id"].(string)} {
		var by *string
		if err := b.api.db.QueryRow(ctx, `select created_by::text from `+table+` where id = $1`, id).Scan(&by); err != nil || by == nil || *by != b.idB {
			t.Errorf("%s.created_by = %v (%v), want the second account", table, by, err)
		}
	}

	t.Run("the API says created_by and no longer user_id", func(t *testing.T) {
		for name, path := range map[string]string{
			"sales": "/sales", "orders": "/orders", "expenses": "/expenses?month=3&year=2026",
			"fixed costs": "/fixed-costs?month=2026-03", "stock adjustments": "/stock-adjustments?month=2026-03", "recipes": "/recipes",
		} {
			rows := listOf(t, b.a, path)
			if len(rows) == 0 {
				t.Fatalf("%s: nothing listed", name)
			}
			if rows[0]["created_by"] != b.idA && rows[0]["created_by"] != b.idB {
				t.Errorf("%s: created_by = %v, want the id of who registered it", name, rows[0]["created_by"])
			}
			if _, has := rows[0]["user_id"]; has {
				t.Errorf("%s: still says user_id", name)
			}
		}
	})
}

// N1 (behavior): deleting a user keeps what the user registered.
func TestN1_DeletingAUserKeepsTheirRecords(t *testing.T) {
	b := newBusiness(t)
	ctx := context.Background()
	_, before := b.b.do("GET", "/analytics/monthly?month=3&year=2026", nil)

	if _, err := b.api.db.Exec(ctx, `delete from users where id = $1`, b.idA); err != nil {
		t.Fatalf("delete user: %v", err)
	}

	for _, table := range []string{"sales", "orders", "operational_expenses", "fixed_costs", "quotations", "stock_adjustments", "recipes"} {
		var n int
		var orphans int
		if err := b.api.db.QueryRow(ctx, `select count(*), count(*) filter (where created_by is null) from `+table).Scan(&n, &orphans); err != nil {
			t.Fatalf("%s: %v", table, err)
		}
		if n == 0 || orphans != n {
			t.Errorf("%s: %d rows, %d without creator; want all kept and with no creator", table, n, orphans)
		}
	}
	_, after := b.b.do("GET", "/analytics/monthly?month=3&year=2026", nil)
	if string(before) != string(after) {
		t.Errorf("the dashboard changed when a user was deleted:\n before: %s\n after: %s", before, after)
	}
}

// N5: a token that is not what the server issues is a 401, never a 500.
func TestN5_MalformedTokensAreUnauthorized(t *testing.T) {
	api := newAPIAt(t, marchClock)
	sign := func(claims jwt.MapClaims) string {
		s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(goodSecret))
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	exp := time.Now().Add(time.Hour).Unix()
	for label, claims := range map[string]jwt.MapClaims{
		"no sub":           {"role": "owner", "exp": exp},
		"no role":          {"sub": "x", "exp": exp},
		"numeric sub":      {"sub": 42, "role": "owner", "exp": exp},
		"numeric role":     {"sub": "x", "role": 7, "exp": exp},
		"null sub":         {"sub": nil, "role": "owner", "exp": exp},
		"no claims at all": {"exp": exp},
	} {
		c := &client{t: t, h: api.h, token: sign(claims)}
		if status, raw := c.do("GET", "/ingredients", nil); status != http.StatusUnauthorized {
			t.Errorf("%s: status = %d, want 401: %s", label, status, raw)
		}
	}
	if status, _ := (&client{t: t, h: api.h, token: "not.a.token"}).do("GET", "/ingredients", nil); status != http.StatusUnauthorized {
		t.Errorf("garbage: status = %d, want 401", status)
	}
}
