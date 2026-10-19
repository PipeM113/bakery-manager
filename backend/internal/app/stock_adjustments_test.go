package app_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// Sprint 8: stock adjustments with a reason (merma, correccion, otro) and their history. The clock
// of these tests stands on 2026-03-31 (Santiago).

func adjust(t *testing.T, api *client, ingredientID string, body map[string]any) (int, []byte) {
	t.Helper()
	return api.do("POST", "/ingredients/"+ingredientID+"/stock-adjustments", body)
}

func adjustOK(t *testing.T, api *client, ingredientID string, body map[string]any) map[string]any {
	t.Helper()
	status, raw := adjust(t, api, ingredientID, body)
	if status != http.StatusCreated {
		t.Fatalf("adjust %v: status = %d: %s", body, status, raw)
	}
	return decode(t, raw)
}

// flourWithStock: 2 pesos per gram.
func flourWithStock(t *testing.T, api *client, stock float64) string {
	t.Helper()
	return createIngredient(t, api, map[string]any{
		"name": "Harina", "default_unit": "gr", "package_size": 1000, "package_price": 2000, "stock_quantity": stock,
	})
}

func adjustmentCount(t *testing.T, api *timeAPI) int {
	t.Helper()
	var n int
	if err := api.db.QueryRow(context.Background(), `select count(*) from stock_adjustments`).Scan(&n); err != nil {
		t.Fatalf("stock_adjustments: %v", err)
	}
	return n
}

// A2: a merma takes the stock down and counts as a loss of what it cost.
func TestA2_MermaLowersTheStockAndCountsAsALoss(t *testing.T) {
	api := newAPIAt(t, marchClock)

	t.Run("200 gr of a flour that costs 2 per gr", func(t *testing.T) {
		flour := flourWithStock(t, api.client, 5000)
		a := adjustOK(t, api.client, flour, map[string]any{"reason": "merma", "quantity": 200, "unit": "gr", "note": "se echó a perder"})
		if got := stockOf(t, api.client, flour); !near(got, 4800) {
			t.Errorf("stock = %v, want 4800", got)
		}
		for k, want := range map[string]float64{"quantity_delta": -200, "stock_before": 5000, "stock_after": 4800, "loss_amount": 400} {
			if got := number(t, a, k); !near(got, want) {
				t.Errorf("%s = %v, want %v", k, got, want)
			}
		}
		if a["reason"] != "merma" || a["unit"] != "gr" || a["note"] != "se echó a perder" || a["adjusted_on"] != "2026-03-31" {
			t.Errorf("adjustment = %v", a)
		}
	})

	t.Run("in another unit of the same kind: 200 gr of a product kept in kg", func(t *testing.T) {
		manjar := createIngredient(t, api.client, map[string]any{
			"name": "Manjar", "default_unit": "kg", "package_size": 1, "package_price": 8000, "stock_quantity": 1,
		})
		a := adjustOK(t, api.client, manjar, map[string]any{"reason": "merma", "quantity": 200, "unit": "gr"})
		if got := stockOf(t, api.client, manjar); !near(got, 0.8) {
			t.Errorf("stock = %v kg, want 0.8", got)
		}
		if got := number(t, a, "loss_amount"); got != 1600 {
			t.Errorf("loss_amount = %v, want 1600 (20%% of 8000)", got)
		}
		if a["unit"] != "kg" {
			t.Errorf("unit = %v, want kg (the unit of the ingredient)", a["unit"])
		}
	})

	t.Run("without a unit it is the unit of the ingredient", func(t *testing.T) {
		flour := createIngredient(t, api.client, map[string]any{
			"name": "Azúcar", "default_unit": "gr", "package_size": 1000, "package_price": 1000, "stock_quantity": 1000,
		})
		a := adjustOK(t, api.client, flour, map[string]any{"reason": "merma", "quantity": 100})
		if got := number(t, a, "loss_amount"); got != 100 {
			t.Errorf("loss_amount = %v, want 100", got)
		}
	})

	t.Run("the loss is rounded once, half up, to whole pesos", func(t *testing.T) {
		odd := createIngredient(t, api.client, map[string]any{
			"name": "Esencia", "default_unit": "gr", "package_size": 2, "package_price": 5, "stock_quantity": 10,
		})
		a := adjustOK(t, api.client, odd, map[string]any{"reason": "merma", "quantity": 1}) // 2.5 pesos
		if got := number(t, a, "loss_amount"); got != 3 {
			t.Errorf("loss_amount = %v, want 3", got)
		}
	})

	t.Run("more than the stock is refused and nothing changes", func(t *testing.T) {
		flour := flourWithStock(t, api.client, 100)
		before := adjustmentCount(t, api)
		status, raw := adjust(t, api.client, flour, map[string]any{"reason": "merma", "quantity": 150})
		if status != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400: %s", status, raw)
		}
		if got := stockOf(t, api.client, flour); got != 100 {
			t.Errorf("stock = %v, want 100", got)
		}
		if adjustmentCount(t, api) != before {
			t.Error("a refused merma left a record")
		}
	})

	t.Run("a merma of 0 or less is refused", func(t *testing.T) {
		flour := flourWithStock(t, api.client, 100)
		for _, q := range []float64{0, -5} {
			if status, _ := adjust(t, api.client, flour, map[string]any{"reason": "merma", "quantity": q}); status != http.StatusBadRequest {
				t.Errorf("quantity %v: status = %d, want 400", q, status)
			}
		}
	})
}

// A3: a correction sets the stock to what was counted; the difference is recorded, no loss.
func TestA3_CorrectionSetsTheCountedStock(t *testing.T) {
	api := newAPIAt(t, marchClock)
	flour := flourWithStock(t, api.client, 5000)

	a := adjustOK(t, api.client, flour, map[string]any{"reason": "correccion", "counted_quantity": 4500})
	if got := stockOf(t, api.client, flour); !near(got, 4500) {
		t.Errorf("stock = %v, want 4500", got)
	}
	for k, want := range map[string]float64{"quantity_delta": -500, "stock_before": 5000, "stock_after": 4500, "loss_amount": 0} {
		if got := number(t, a, k); !near(got, want) {
			t.Errorf("%s = %v, want %v", k, got, want)
		}
	}

	a = adjustOK(t, api.client, flour, map[string]any{"reason": "correccion", "counted_quantity": 5200})
	if got := number(t, a, "quantity_delta"); !near(got, 700) {
		t.Errorf("quantity_delta = %v, want +700", got)
	}

	t.Run("in another unit", func(t *testing.T) {
		adjustOK(t, api.client, flour, map[string]any{"reason": "correccion", "counted_quantity": 4.2, "unit": "kg"})
		if got := stockOf(t, api.client, flour); !near(got, 4200) {
			t.Errorf("stock = %v, want 4200", got)
		}
	})

	t.Run("counting 0 is allowed, a negative count is not", func(t *testing.T) {
		adjustOK(t, api.client, flour, map[string]any{"reason": "correccion", "counted_quantity": 0})
		if got := stockOf(t, api.client, flour); got != 0 {
			t.Errorf("stock = %v, want 0", got)
		}
		if status, _ := adjust(t, api.client, flour, map[string]any{"reason": "correccion", "counted_quantity": -1}); status != http.StatusBadRequest {
			t.Errorf("negative count: status = %d, want 400", status)
		}
	})
}

// A4: "otro" moves the stock up or down with a note, and is no loss.
func TestA4_OtroNeedsANoteAndKeepsTheStockNotNegative(t *testing.T) {
	api := newAPIAt(t, marchClock)
	flour := flourWithStock(t, api.client, 100)

	a := adjustOK(t, api.client, flour, map[string]any{"reason": "otro", "quantity_delta": -30, "note": "regalo"})
	if got := stockOf(t, api.client, flour); !near(got, 70) {
		t.Errorf("stock = %v, want 70", got)
	}
	if got := number(t, a, "loss_amount"); got != 0 {
		t.Errorf("loss_amount = %v, want 0", got)
	}
	adjustOK(t, api.client, flour, map[string]any{"reason": "otro", "quantity_delta": 30, "note": "devolución"})
	if got := stockOf(t, api.client, flour); !near(got, 100) {
		t.Errorf("stock = %v, want 100", got)
	}

	for label, body := range map[string]map[string]any{
		"no note":        {"reason": "otro", "quantity_delta": -1},
		"blank note":     {"reason": "otro", "quantity_delta": -1, "note": "   "},
		"zero":           {"reason": "otro", "quantity_delta": 0, "note": "x"},
		"negative stock": {"reason": "otro", "quantity_delta": -101, "note": "x"},
	} {
		if status, _ := adjust(t, api.client, flour, body); status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", label, status)
		}
	}
	if got := stockOf(t, api.client, flour); !near(got, 100) {
		t.Errorf("stock = %v after the refused ones, want 100", got)
	}
}

// A5: bad requests are refused and leave nothing behind.
func TestA5_BadAdjustmentsAreRefused(t *testing.T) {
	api := newAPIAt(t, marchClock)
	flour := flourWithStock(t, api.client, 1000)

	for label, body := range map[string]map[string]any{
		"no reason":          {"quantity": 10},
		"unknown reason":     {"reason": "robo", "quantity": 10},
		"incompatible unit":  {"reason": "merma", "quantity": 10, "unit": "ml"},
		"future date":        {"reason": "merma", "quantity": 10, "adjusted_on": "2026-04-01"},
		"badly written date": {"reason": "merma", "quantity": 10, "adjusted_on": "31-03-2026"},
	} {
		if status, raw := adjust(t, api.client, flour, body); status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400: %s", label, status, raw)
		}
	}
	if status, _ := adjust(t, api.client, "00000000-0000-0000-0000-000000000000", map[string]any{"reason": "merma", "quantity": 1}); status != http.StatusNotFound {
		t.Errorf("unknown ingredient: status = %d, want 404", status)
	}
	if got := stockOf(t, api.client, flour); got != 1000 || adjustmentCount(t, api) != 0 {
		t.Errorf("stock = %v, records = %d: the refused ones must leave nothing", got, adjustmentCount(t, api))
	}

	t.Run("a past date is fine and today is the default", func(t *testing.T) {
		a := adjustOK(t, api.client, flour, map[string]any{"reason": "merma", "quantity": 10, "adjusted_on": "2026-03-10"})
		if a["adjusted_on"] != "2026-03-10" {
			t.Errorf("adjusted_on = %v, want 2026-03-10", a["adjusted_on"])
		}
		// 23:30 of the 31st in Santiago is already the 1st in UTC: the default is the Santiago day.
		api.setAt(lateMarch31)
		a = adjustOK(t, api.client, flour, map[string]any{"reason": "merma", "quantity": 10})
		if a["adjusted_on"] != "2026-03-31" {
			t.Errorf("adjusted_on = %v, want 2026-03-31", a["adjusted_on"])
		}
	})
}

// A6: editing the stock of an ingredient leaves a correction behind.
func TestA6_EditingTheStockLeavesACorrection(t *testing.T) {
	api := newAPIAt(t, marchClock)
	flour := flourWithStock(t, api.client, 5000)
	body := func(stock float64, name string) map[string]any {
		return map[string]any{
			"name": name, "default_unit": "gr", "package_size": 1000, "package_price": 2000,
			"stock_quantity": stock, "brand": "Marca",
		}
	}

	if status, raw := api.do("PUT", "/ingredients/"+flour, body(4000, "Harina")); status != http.StatusOK {
		t.Fatalf("PUT: status = %d: %s", status, raw)
	}
	var reason string
	var delta, before, after float64
	var loss int
	if err := api.db.QueryRow(context.Background(), `
		select reason, quantity_delta, stock_before, stock_after, loss_amount from stock_adjustments where ingredient_id = $1`, flour,
	).Scan(&reason, &delta, &before, &after, &loss); err != nil {
		t.Fatalf("no adjustment was left: %v", err)
	}
	if reason != "correccion" || !near(delta, -1000) || !near(before, 5000) || !near(after, 4000) || loss != 0 {
		t.Errorf("adjustment = %s %v %v %v %d, want correccion -1000 5000 4000 0", reason, delta, before, after, loss)
	}

	if status, raw := api.do("PUT", "/ingredients/"+flour, body(4000, "Harina fina")); status != http.StatusOK {
		t.Fatalf("PUT (rename): status = %d: %s", status, raw)
	}
	if n := adjustmentCount(t, api); n != 1 {
		t.Errorf("records = %d, want 1: renaming does not change the stock", n)
	}
}

// A7: the history, newest first, for an ingredient and for a month.
func TestA7_History(t *testing.T) {
	api := newAPIAt(t, marchClock)
	flour := flourWithStock(t, api.client, 5000)
	sugar := createIngredient(t, api.client, map[string]any{
		"name": "Azúcar", "default_unit": "gr", "package_size": 1000, "package_price": 1000, "stock_quantity": 1000,
	})
	adjustOK(t, api.client, flour, map[string]any{"reason": "merma", "quantity": 10, "adjusted_on": "2026-02-27"})
	adjustOK(t, api.client, flour, map[string]any{"reason": "merma", "quantity": 20, "adjusted_on": "2026-03-10"})
	adjustOK(t, api.client, flour, map[string]any{"reason": "correccion", "counted_quantity": 4000, "adjusted_on": "2026-03-20"})
	adjustOK(t, api.client, sugar, map[string]any{"reason": "merma", "quantity": 5, "adjusted_on": "2026-03-15"})

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
	dates := func(rows []map[string]any) string {
		var d []string
		for _, r := range rows {
			d = append(d, r["adjusted_on"].(string))
		}
		return strings.Join(d, ",")
	}

	if got := dates(list("/ingredients/" + flour + "/stock-adjustments")); got != "2026-03-20,2026-03-10,2026-02-27" {
		t.Errorf("history of the flour = %s, want newest first", got)
	}
	if got := dates(list("/stock-adjustments?month=2026-03")); got != "2026-03-20,2026-03-15,2026-03-10" {
		t.Errorf("March = %s", got)
	}
	if got := dates(list("/stock-adjustments")); got != "2026-03-20,2026-03-15,2026-03-10" {
		t.Errorf("without month = %s, want the current month (March)", got)
	}
	if got := list("/stock-adjustments?month=2026-04"); len(got) != 0 {
		t.Errorf("April = %v, want none", got)
	}
	if rows := list("/stock-adjustments?month=2026-03"); rows[0]["ingredient_name"] == nil {
		t.Errorf("the monthly list should say which ingredient each one is: %v", rows[0])
	}
	for _, bad := range []string{"?month=2026-13", "?month=abc", "?month="} {
		if status, _ := api.do("GET", "/stock-adjustments"+bad, nil); status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", bad, status)
		}
	}
}

// A8: the losses of the month count in the dashboard and in the trend.
func TestA8_WasteReachesTheDashboardAndTheTrend(t *testing.T) {
	api := newAPIAt(t, marchClock)
	flour := flourWithStock(t, api.client, 5000)
	adjustOK(t, api.client, flour, map[string]any{"reason": "merma", "quantity": 200, "adjusted_on": "2026-03-10"})               // 400
	adjustOK(t, api.client, flour, map[string]any{"reason": "merma", "quantity": 50, "adjusted_on": "2026-02-10"})                // 100
	adjustOK(t, api.client, flour, map[string]any{"reason": "correccion", "counted_quantity": 1000, "adjusted_on": "2026-03-11"}) // no loss
	adjustOK(t, api.client, flour, map[string]any{"reason": "otro", "quantity_delta": -10, "note": "regalo", "adjusted_on": "2026-03-12"})

	for _, c := range []struct {
		query string
		waste float64
	}{{"month=3&year=2026", 400}, {"month=2&year=2026", 100}, {"month=1&year=2026", 0}} {
		status, raw := api.do("GET", "/analytics/monthly?"+c.query, nil)
		if status != http.StatusOK {
			t.Fatalf("status = %d: %s", status, raw)
		}
		body := decode(t, raw)
		if got := number(t, body, "total_waste_cost"); got != c.waste {
			t.Errorf("%s: total_waste_cost = %v, want %v", c.query, got, c.waste)
		}
		if got := number(t, body, "net_profit"); got != -c.waste {
			t.Errorf("%s: net_profit = %v, want %v (nothing sold, the waste is a loss)", c.query, got, -c.waste)
		}
	}

	api.count.n.Store(0)
	status, raw := api.do("GET", "/analytics/trends?months=3", nil)
	queries := api.count.n.Load()
	if status != http.StatusOK {
		t.Fatalf("trends: status = %d: %s", status, raw)
	}
	var points []map[string]any
	if err := json.Unmarshal(raw, &points); err != nil {
		t.Fatal(err)
	}
	want := map[string]float64{"2026-01": 0, "2026-02": -100, "2026-03": -400}
	for _, p := range points {
		if got := p["net_profit"].(float64); got != want[p["date"].(string)] {
			t.Errorf("%v: net_profit = %v, want %v", p["date"], got, want[p["date"].(string)])
		}
	}
	if queries > 4 {
		t.Errorf("the trend ran %d queries, want at most 4 (the waste goes inside the existing ones)", queries)
	}
}

// A9: adjustments at the same time keep the stock right.
func TestA9_SimultaneousMermasNeverOverdrawTheStock(t *testing.T) {
	api := newAPIAt(t, marchClock)
	flour := flourWithStock(t, api.client, 1000)

	const tries = 8
	statuses := make([]int, tries)
	var wg sync.WaitGroup
	for i := 0; i < tries; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			raw, _ := json.Marshal(map[string]any{"reason": "merma", "quantity": 200})
			req := httptest.NewRequest("POST", "/ingredients/"+flour+"/stock-adjustments", strings.NewReader(string(raw)))
			req.Header.Set("Authorization", "Bearer "+api.token)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			api.h.ServeHTTP(rec, req)
			statuses[i] = rec.Code
		}(i)
	}
	wg.Wait()

	created, refused := 0, 0
	for _, s := range statuses {
		switch s {
		case http.StatusCreated:
			created++
		case http.StatusBadRequest:
			refused++
		default:
			t.Errorf("status = %d: only 201 or 400 are possible", s)
		}
	}
	if created != 5 || refused != 3 {
		t.Errorf("created = %d, refused = %d, want 5 and 3 (1000 / 200)", created, refused)
	}
	if got := stockOf(t, api.client, flour); got != 0 {
		t.Errorf("stock = %v, want 0", got)
	}
	var losses int
	if err := api.db.QueryRow(context.Background(), `select coalesce(sum(loss_amount), 0) from stock_adjustments`).Scan(&losses); err != nil || losses != 5*400 {
		t.Errorf("losses = %d (%v), want 2000", losses, err)
	}
}
