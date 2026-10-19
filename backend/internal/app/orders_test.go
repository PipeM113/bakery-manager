package app_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// Sprint 10: orders with several products. A line of an order is a sale (so the stock and the
// reports work as before), the order groups them and may have its own total price. The clock of
// these tests stands on 2026-03-31 (Santiago).

type line = map[string]any

// ln is a line of an order: quantity of a recipe (or of its portions) at a normal unit price.
func ln(recipe string, quantity, price int, soldAs string) line {
	l := line{"recipe_id": recipe, "quantity": quantity, "unit_price": price}
	if soldAs != "" {
		l["sold_as"] = soldAs
	}
	return l
}

func postOrder(t *testing.T, api *client, body map[string]any) (int, []byte) {
	t.Helper()
	return api.do("POST", "/orders", body)
}

func orderOK(t *testing.T, api *client, body map[string]any) map[string]any {
	t.Helper()
	status, raw := postOrder(t, api, body)
	if status != http.StatusCreated {
		t.Fatalf("POST /orders: status = %d: %s", status, raw)
	}
	return decode(t, raw)
}

func orderLines(t *testing.T, order map[string]any) []map[string]any {
	t.Helper()
	raw, _ := order["lines"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, r := range raw {
		out = append(out, r.(map[string]any))
	}
	return out
}

func lineTotals(t *testing.T, order map[string]any) []float64 {
	t.Helper()
	var out []float64
	for _, l := range orderLines(t, order) {
		out = append(out, number(t, l, "line_total"))
	}
	return out
}

func sumOf(xs []float64) float64 {
	var s float64
	for _, x := range xs {
		s += x
	}
	return s
}

func salesCount(t *testing.T, api *timeAPI) int {
	t.Helper()
	var n int
	if err := api.db.QueryRow(context.Background(), `select count(*) from sales`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func ordersCount(t *testing.T, api *timeAPI) int {
	t.Helper()
	var n int
	if err := api.db.QueryRow(context.Background(), `select count(*) from orders`).Scan(&n); err != nil {
		t.Fatalf("orders: %v", err)
	}
	return n
}

// shop makes a flour that is plentiful and two recipes of it: a cake (500 gr, yields 10 portions)
// and a pie (200 gr).
func shop(t *testing.T, api *client) (flour, cake, pie string) {
	t.Helper()
	flour = createIngredient(t, api, map[string]any{
		"name": "Harina", "default_unit": "gr", "package_size": 1000, "package_price": 2000, "stock_quantity": 100000,
	})
	cake = recipeOf(t, api, flour, 500, "gr", 10, "porciones")
	pie = recipeOf(t, api, flour, 200, "gr", 1, "porciones")
	return
}

// O0: a single sale keeps working as before.
func TestO0_SingleSalesAreUnchanged(t *testing.T) {
	api := newAPIAt(t, marchClock)
	_, cake, _ := shop(t, api.client)
	sale := sellOK(t, api.client, map[string]any{"recipe_id": cake, "quantity_sold": 2, "unit_price": 1500})
	if got := number(t, sale, "total_price"); got != 3000 {
		t.Errorf("total_price = %v, want 3000", got)
	}
	if sale["order_id"] != nil {
		t.Errorf("order_id = %v, want none for a single sale", sale["order_id"])
	}
}

// O2: an order is created with its lines, all or nothing.
func TestO2_CreatingAnOrder(t *testing.T) {
	api := newAPIAt(t, marchClock)
	_, cake, pie := shop(t, api.client)

	order := orderOK(t, api.client, map[string]any{
		"customer_name": "María González", "notes": "Para el sábado",
		"lines": []line{ln(cake, 1, 12000, ""), ln(pie, 3, 4000, "")},
	})
	if order["customer_name"] != "María González" || order["notes"] != "Para el sábado" {
		t.Errorf("order = %v", order)
	}
	if order["order_date"] != "2026-03-31" {
		t.Errorf("order_date = %v, want 2026-03-31 (today in Santiago)", order["order_date"])
	}
	if id, _ := order["id"].(string); id == "" {
		t.Error("the order has no id")
	}
	if got := len(orderLines(t, order)); got != 2 {
		t.Errorf("lines = %d, want 2", got)
	}
	if got := salesCount(t, api); got != 2 {
		t.Errorf("sales = %d, want 2 (one per line)", got)
	}
	if got := ordersCount(t, api); got != 1 {
		t.Errorf("orders = %d, want 1", got)
	}

	t.Run("without a date it is the Santiago day, even late at night", func(t *testing.T) {
		api.setAt(lateMarch31)
		o := orderOK(t, api.client, map[string]any{"lines": []line{ln(pie, 1, 1000, "")}})
		if o["order_date"] != "2026-03-31" {
			t.Errorf("order_date = %v, want 2026-03-31", o["order_date"])
		}
	})

	t.Run("a given date is used", func(t *testing.T) {
		o := orderOK(t, api.client, map[string]any{"order_date": "2026-03-05", "lines": []line{ln(pie, 1, 1000, "")}})
		if o["order_date"] != "2026-03-05" {
			t.Errorf("order_date = %v", o["order_date"])
		}
	})

	t.Run("the customer and the notes are optional", func(t *testing.T) {
		orderOK(t, api.client, map[string]any{"lines": []line{ln(pie, 1, 1000, "")}})
	})

	t.Run("50 lines are fine, 51 or none are not", func(t *testing.T) {
		many := func(n int) []line {
			ls := make([]line, n)
			for i := range ls {
				ls[i] = ln(pie, 1, 100, "")
			}
			return ls
		}
		orderOK(t, api.client, map[string]any{"lines": many(50)})
		for _, n := range []int{0, 51} {
			if status, _ := postOrder(t, api.client, map[string]any{"lines": many(n)}); status != http.StatusBadRequest {
				t.Errorf("%d lines: status = %d, want 400", n, status)
			}
		}
	})
}

// O3: the price of an order.
func TestO3_PriceOfAnOrder(t *testing.T) {
	api := newAPIAt(t, marchClock)
	_, cake, pie := shop(t, api.client)

	t.Run("without a total it is the sum of the lines", func(t *testing.T) {
		o := orderOK(t, api.client, map[string]any{"lines": []line{ln(pie, 2, 1000, ""), ln(cake, 1, 3000, "portion")}})
		if got := number(t, o, "total_price"); got != 5000 {
			t.Errorf("total_price = %v, want 5000", got)
		}
		if got := lineTotals(t, o); got[0] != 2000 || got[1] != 3000 {
			t.Errorf("line totals = %v, want [2000 3000]", got)
		}
	})

	t.Run("a different total is shared in proportion to the normal prices", func(t *testing.T) {
		o := orderOK(t, api.client, map[string]any{
			"total_price": 4000, "lines": []line{ln(pie, 2, 1000, ""), ln(cake, 1, 3000, "")},
		})
		if got := number(t, o, "total_price"); got != 4000 {
			t.Errorf("total_price = %v, want 4000", got)
		}
		if got := lineTotals(t, o); got[0] != 1600 || got[1] != 2400 {
			t.Errorf("line totals = %v, want [1600 2400]", got)
		}
	})

	t.Run("the lines always add up to the total, in whole pesos", func(t *testing.T) {
		o := orderOK(t, api.client, map[string]any{
			"total_price": 1000, "lines": []line{ln(pie, 1, 100, ""), ln(pie, 1, 100, ""), ln(pie, 1, 100, "")},
		})
		got := lineTotals(t, o)
		if sumOf(got) != 1000 {
			t.Errorf("line totals %v add up to %v, want exactly 1000", got, sumOf(got))
		}
		for _, v := range got {
			if v != 333 && v != 334 {
				t.Errorf("line total %v: want 333 or 334", v)
			}
		}
	})

	t.Run("lines without a normal price share the total equally", func(t *testing.T) {
		o := orderOK(t, api.client, map[string]any{
			"total_price": 900, "lines": []line{ln(pie, 1, 0, ""), ln(pie, 2, 0, ""), ln(cake, 1, 0, "")},
		})
		if got := lineTotals(t, o); got[0] != 300 || got[1] != 300 || got[2] != 300 {
			t.Errorf("line totals = %v, want [300 300 300]", got)
		}
	})

	t.Run("a total of 0 is a gift", func(t *testing.T) {
		o := orderOK(t, api.client, map[string]any{"total_price": 0, "lines": []line{ln(pie, 1, 1000, ""), ln(cake, 1, 2000, "")}})
		if sumOf(lineTotals(t, o)) != 0 || number(t, o, "total_price") != 0 {
			t.Errorf("order = %v", o)
		}
	})

	t.Run("a line alone takes the whole total", func(t *testing.T) {
		o := orderOK(t, api.client, map[string]any{"total_price": 777, "lines": []line{ln(pie, 3, 1000, "")}})
		if got := lineTotals(t, o); got[0] != 777 {
			t.Errorf("line totals = %v, want [777]", got)
		}
	})

	t.Run("the sales list shows each line with its order and its share", func(t *testing.T) {
		o := orderOK(t, api.client, map[string]any{
			"order_date": "2026-02-10", "total_price": 4000, "lines": []line{ln(pie, 2, 1000, ""), ln(cake, 1, 3000, "")},
		})
		status, raw := api.do("GET", "/sales?from=2026-02-10&to=2026-02-10", nil)
		if status != http.StatusOK {
			t.Fatalf("status = %d: %s", status, raw)
		}
		var rows []map[string]any
		if err := json.Unmarshal(raw, &rows); err != nil {
			t.Fatal(err)
		}
		var totals []float64
		for _, r := range rows {
			if r["order_id"] == o["id"] {
				totals = append(totals, r["total_price"].(float64))
			}
		}
		if sumOf(totals) != 4000 || len(totals) != 2 {
			t.Errorf("sales of the order = %v, want 2 lines adding up to 4000", totals)
		}
	})
}

// O4: the stock is taken line by line, as in a sale.
func TestO4_StockPerLine(t *testing.T) {
	api := newAPIAt(t, marchClock)

	t.Run("units, portions and the same ingredient in several lines", func(t *testing.T) {
		manjar := createIngredient(t, api.client, map[string]any{
			"name": "Manjar", "default_unit": "kg", "package_size": 1, "package_price": 8000, "stock_quantity": 2,
		})
		flan := recipeOf(t, api.client, manjar, 200, "gr", 1, "porciones")   // 0.2 kg
		torta := recipeOf(t, api.client, manjar, 600, "gr", 12, "porciones") // 0.6 kg, 12 portions
		orderOK(t, api.client, map[string]any{"lines": []line{
			ln(flan, 2, 1000, ""),        // 0.4 kg
			ln(torta, 3, 500, "portion"), // 0.15 kg
		}})
		if got := stockOf(t, api.client, manjar); !near(got, 1.45) {
			t.Errorf("stock = %v kg, want 1.45 (2 - 0.4 - 0.15)", got)
		}
	})

	t.Run("without enough stock the order is registered and says what was missing", func(t *testing.T) {
		sugar := createIngredient(t, api.client, map[string]any{
			"name": "Azúcar", "default_unit": "gr", "package_size": 1000, "package_price": 1000, "stock_quantity": 500,
		})
		a := recipeOf(t, api.client, sugar, 400, "gr", 1, "porciones")
		b := recipeOf(t, api.client, sugar, 400, "gr", 1, "porciones")
		o := orderOK(t, api.client, map[string]any{"lines": []line{ln(a, 1, 1000, ""), ln(b, 1, 1000, "")}})
		if got := stockOf(t, api.client, sugar); got != 0 {
			t.Errorf("stock = %v, want 0 (never negative)", got)
		}
		short := shortfalls(t, o)
		if len(short) != 1 || short[0].Ingredient != "Azúcar" || !near(short[0].Missing, 300) || short[0].Unit != "gr" {
			t.Errorf("stock_shortfalls = %v, want Azúcar 300 gr once (500 for 800 needed)", short)
		}
	})
}

// O5: a bad order leaves nothing behind.
func TestO5_BadOrdersAreRefusedWhole(t *testing.T) {
	api := newAPIAt(t, marchClock)
	flour, cake, pie := shop(t, api.client)
	before := stockOf(t, api.client, flour)

	bad := map[string]map[string]any{
		"unknown recipe":     {"lines": []line{ln(pie, 1, 1000, ""), ln("00000000-0000-0000-0000-000000000000", 1, 1, "")}},
		"unknown sold_as":    {"lines": []line{ln(pie, 1, 1000, ""), ln(cake, 1, 1, "kilo")}},
		"quantity 0":         {"lines": []line{ln(pie, 1, 1000, ""), ln(cake, 0, 1, "")}},
		"decimal quantity":   {"lines": []line{ln(pie, 1, 1000, ""), {"recipe_id": cake, "quantity": 1.5, "unit_price": 1}}},
		"negative price":     {"lines": []line{ln(pie, 1, 1000, ""), ln(cake, 1, -5, "")}},
		"negative total":     {"total_price": -1, "lines": []line{ln(pie, 1, 1000, "")}},
		"total too big":      {"total_price": 3000000000, "lines": []line{ln(pie, 1, 1000, "")}},
		"badly written date": {"order_date": "31-03-2026", "lines": []line{ln(pie, 1, 1000, "")}},
		"future date":        {"order_date": "2026-04-01", "lines": []line{ln(pie, 1, 1000, "")}},
	}
	for label, body := range bad {
		if status, raw := postOrder(t, api.client, body); status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400: %s", label, status, raw)
		}
	}

	t.Run("incompatible units in the second line", func(t *testing.T) {
		odd := recipeOf(t, api.client, flour, 100, "gr", 1, "porciones")
		if _, err := api.db.Exec(context.Background(), `update recipe_ingredients set unit = 'ml' where recipe_id = $1`, odd); err != nil {
			t.Fatal(err)
		}
		status, raw := postOrder(t, api.client, map[string]any{"lines": []line{ln(pie, 1, 1000, ""), ln(odd, 1, 1000, "")}})
		if status != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400: %s", status, raw)
		}
	})

	if got := stockOf(t, api.client, flour); got != before {
		t.Errorf("stock = %v, want %v: no line of a refused order may take stock", got, before)
	}
	if salesCount(t, api) != 0 || ordersCount(t, api) != 0 {
		t.Errorf("sales = %d, orders = %d, want none", salesCount(t, api), ordersCount(t, api))
	}
}

// secondUser logs in as another user of the same business.
func secondUser(t *testing.T, api *timeAPI) *client {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := api.db.Exec(context.Background(),
		`insert into users (name, email, password, role) values ('Helper', 'helper@example.test', $1, 'helper')`, string(hash)); err != nil {
		t.Fatal(err)
	}
	other := &client{t: t, h: api.h}
	status, raw := other.do("POST", "/auth/login", map[string]string{"email": "helper@example.test", "password": testPassword})
	if status != http.StatusOK {
		t.Fatalf("login: status = %d: %s", status, raw)
	}
	other.token, _ = decode(t, raw)["token"].(string)
	return other
}

// O6: reading orders.
func TestO6_ListAndGet(t *testing.T) {
	api := newAPIAt(t, marchClock)
	_, cake, pie := shop(t, api.client)
	feb := orderOK(t, api.client, map[string]any{"order_date": "2026-02-10", "customer_name": "Febrero", "lines": []line{ln(pie, 1, 1000, "")}})
	older := orderOK(t, api.client, map[string]any{"order_date": "2026-03-10", "customer_name": "Antes", "lines": []line{ln(pie, 1, 1000, "")}})
	newer := orderOK(t, api.client, map[string]any{"order_date": "2026-03-20", "customer_name": "Después", "lines": []line{ln(pie, 2, 1000, ""), ln(cake, 1, 5000, "")}})

	list := func(path string) []map[string]any {
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
	names := func(rows []map[string]any) string {
		var n []string
		for _, r := range rows {
			n = append(n, r["customer_name"].(string))
		}
		return strings.Join(n, ",")
	}

	if got := names(list("/orders")); got != "Después,Antes" {
		t.Errorf("this month = %s, want newest first and only March", got)
	}
	if got := names(list("/orders?month=2026-02")); got != "Febrero" {
		t.Errorf("February = %s", got)
	}
	if got := list("/orders?month=2026-04"); len(got) != 0 {
		t.Errorf("April = %v, want none", got)
	}
	for _, bad := range []string{"?month=2026-13", "?month=abc", "?month="} {
		if status, _ := api.do("GET", "/orders"+bad, nil); status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", bad, status)
		}
	}
	if got := len(orderLines(t, list("/orders")[0])); got != 2 {
		t.Errorf("the listed order has %d lines, want 2", got)
	}

	status, raw := api.do("GET", "/orders/"+newer["id"].(string), nil)
	if status != http.StatusOK {
		t.Fatalf("GET one: status = %d: %s", status, raw)
	}
	one := decode(t, raw)
	if number(t, one, "total_price") != 7000 || len(orderLines(t, one)) != 2 {
		t.Errorf("order = %v", one)
	}
	if status, _ := api.do("GET", "/orders/00000000-0000-0000-0000-000000000000", nil); status != http.StatusNotFound {
		t.Errorf("unknown order: status = %d, want 404", status)
	}
	if status, _ := api.do("GET", "/orders/not-an-id", nil); status != http.StatusNotFound {
		t.Errorf("badly written id: status = %d, want 404", status)
	}

	other := secondUser(t, api)
	if status, _ := other.do("GET", "/orders/"+older["id"].(string), nil); status != http.StatusNotFound {
		t.Errorf("someone else's order: status = %d, want 404", status)
	}
	if got := list("/orders?month=2026-02"); got[0]["id"] != feb["id"] {
		t.Errorf("unexpected: %v", got)
	}
}

// O7: cancelling an order gives back what its lines took.
func TestO7_CancellingAnOrder(t *testing.T) {
	api := newAPIAt(t, marchClock)

	manjar := createIngredient(t, api.client, map[string]any{
		"name": "Manjar", "default_unit": "kg", "package_size": 1, "package_price": 8000, "stock_quantity": 1,
	})
	scarce := createIngredient(t, api.client, map[string]any{
		"name": "Nueces", "default_unit": "gr", "package_size": 100, "package_price": 1000, "stock_quantity": 100,
	})
	flan := recipeOf(t, api.client, manjar, 200, "gr", 1, "porciones")
	nuts := recipeOf(t, api.client, scarce, 500, "gr", 1, "porciones")

	o := orderOK(t, api.client, map[string]any{"lines": []line{ln(flan, 2, 1000, ""), ln(nuts, 1, 1000, "")}})
	if stockOf(t, api.client, manjar) != 0.6 || stockOf(t, api.client, scarce) != 0 {
		t.Fatalf("stock after the order: manjar %v, nuts %v", stockOf(t, api.client, manjar), stockOf(t, api.client, scarce))
	}

	other := secondUser(t, api)
	if status, _ := other.do("DELETE", "/orders/"+o["id"].(string), nil); status != http.StatusNotFound {
		t.Errorf("someone else's order: status = %d, want 404", status)
	}

	if status, raw := api.do("DELETE", "/orders/"+o["id"].(string), nil); status != http.StatusNoContent {
		t.Fatalf("DELETE: status = %d: %s", status, raw)
	}
	if got := stockOf(t, api.client, manjar); !near(got, 1) {
		t.Errorf("manjar = %v kg after cancelling, want 1", got)
	}
	if got := stockOf(t, api.client, scarce); !near(got, 100) {
		t.Errorf("nuts = %v gr after cancelling, want 100 (what there was, not the 500 asked for)", got)
	}
	if salesCount(t, api) != 0 || ordersCount(t, api) != 0 {
		t.Errorf("sales = %d, orders = %d, want none", salesCount(t, api), ordersCount(t, api))
	}
	if status, _ := api.do("GET", "/orders/"+o["id"].(string), nil); status != http.StatusNotFound {
		t.Errorf("GET after cancelling: status = %d, want 404", status)
	}
	if status, _ := api.do("DELETE", "/orders/"+o["id"].(string), nil); status != http.StatusNotFound {
		t.Errorf("cancelling twice: status = %d, want 404", status)
	}

	t.Run("a line of an order cannot be cancelled alone", func(t *testing.T) {
		_, _, pie := shop(t, api.client)
		o := orderOK(t, api.client, map[string]any{"lines": []line{ln(pie, 1, 1000, ""), ln(pie, 2, 1000, "")}})
		saleID := orderLines(t, o)[0]["sale_id"].(string)
		status, raw := api.do("DELETE", "/sales/"+saleID, nil)
		if status != http.StatusConflict {
			t.Fatalf("status = %d, want 409: %s", status, raw)
		}
		if !strings.Contains(string(raw), "pedido") {
			t.Errorf("the message should send the owner to the order: %s", raw)
		}
		if salesCount(t, api) != 2 {
			t.Errorf("sales = %d, want 2: nothing was cancelled", salesCount(t, api))
		}
	})
}

// O8: the reports count orders with their share of the total.
func TestO8_ReportsCountOrders(t *testing.T) {
	api := newAPIAt(t, marchClock)
	_, cake, pie := shop(t, api.client)

	// order of 4000 for what normally costs 2 x 1000 + 3000
	orderOK(t, api.client, map[string]any{"order_date": "2026-03-10", "total_price": 4000, "lines": []line{ln(pie, 2, 1000, ""), ln(cake, 1, 3000, "")}})
	sellOK(t, api.client, map[string]any{"recipe_id": pie, "quantity_sold": 1, "unit_price": 1000, "sale_date": "2026-03-12"})

	status, raw := api.do("GET", "/analytics/monthly?month=3&year=2026", nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d: %s", status, raw)
	}
	if got := number(t, decode(t, raw), "total_revenue"); got != 5000 {
		t.Errorf("total_revenue = %v, want 5000 (4000 of the order + 1000 of the sale)", got)
	}

	status, raw = api.do("GET", "/analytics/recipes?month=3&year=2026", nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d: %s", status, raw)
	}
	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(err)
	}
	revenue, units := map[string]float64{}, map[string]float64{}
	for _, r := range rows {
		revenue[r["recipe_id"].(string)] = r["total_revenue"].(float64)
		units[r["recipe_id"].(string)] = r["units_sold"].(float64)
	}
	if revenue[pie] != 2600 || units[pie] != 3 {
		t.Errorf("pie: revenue %v, units %v, want 2600 (1600 + 1000) and 3", revenue[pie], units[pie])
	}
	if revenue[cake] != 2400 || units[cake] != 1 {
		t.Errorf("cake: revenue %v, units %v, want 2400 and 1", revenue[cake], units[cake])
	}

	api.count.n.Store(0)
	status, raw = api.do("GET", "/analytics/trends?months=1", nil)
	queries := api.count.n.Load()
	if status != http.StatusOK {
		t.Fatalf("trends: status = %d: %s", status, raw)
	}
	var points []map[string]any
	if err := json.Unmarshal(raw, &points); err != nil || len(points) != 1 {
		t.Fatalf("trends = %s (%v)", raw, err)
	}
	if got := points[0]["revenue"].(float64); got != 5000 {
		t.Errorf("trend revenue = %v, want 5000", got)
	}
	if queries > 4 {
		t.Errorf("the trend ran %d queries, want at most 4", queries)
	}
}

// O9: orders at the same time never break the stock, and never block each other for good.
func TestO9_SimultaneousOrders(t *testing.T) {
	api := newAPIAt(t, marchClock)
	a := createIngredient(t, api.client, map[string]any{
		"name": "Harina", "default_unit": "gr", "package_size": 1000, "package_price": 2000, "stock_quantity": 1000,
	})
	b := createIngredient(t, api.client, map[string]any{
		"name": "Azúcar", "default_unit": "gr", "package_size": 1000, "package_price": 1000, "stock_quantity": 1000,
	})
	// Two recipes that use the same two ingredients in opposite order.
	ab := createRecipe(t, api.client, map[string]any{
		"name": "AB", "description": "x", "yield": 1, "yield_unit": "porciones",
		"ingredients": []map[string]any{{"ingredient_id": a, "quantity": 100, "unit": "gr"}, {"ingredient_id": b, "quantity": 100, "unit": "gr"}},
	})
	ba := createRecipe(t, api.client, map[string]any{
		"name": "BA", "description": "x", "yield": 1, "yield_unit": "porciones",
		"ingredients": []map[string]any{{"ingredient_id": b, "quantity": 100, "unit": "gr"}, {"ingredient_id": a, "quantity": 100, "unit": "gr"}},
	})

	const orders = 8
	statuses := make([]int, orders)
	bodies := make([]string, orders)
	var wg sync.WaitGroup
	for i := 0; i < orders; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			lines := []line{ln(ab, 1, 1000, ""), ln(ba, 1, 1000, "")}
			if i%2 == 1 {
				lines = []line{ln(ba, 1, 1000, ""), ln(ab, 1, 1000, "")}
			}
			raw, _ := json.Marshal(map[string]any{"lines": lines})
			req := httptest.NewRequest("POST", "/orders", strings.NewReader(string(raw)))
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
			t.Errorf("order %d: status = %d, want 201: %s", i, s, bodies[i])
		}
	}
	for name, id := range map[string]string{"flour": a, "sugar": b} {
		if got := stockOf(t, api.client, id); got != 0 {
			t.Errorf("%s stock = %v, want 0", name, got)
		}
		var deducted float64
		if err := api.db.QueryRow(context.Background(),
			`select coalesce(sum(stock_deducted), 0) from sale_ingredients where ingredient_id = $1`, id).Scan(&deducted); err != nil {
			t.Fatal(err)
		}
		if !near(deducted, 1000) {
			t.Errorf("%s: stock taken in all = %v, want exactly the 1000 there was", name, deducted)
		}
	}
}
