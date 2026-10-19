package app_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// Sprint 6: fixed costs with history. A cost has an amount per month (versions that start on the
// first of a month), can be ended from a month, and the analytics read the month they are asked for.
//
// The clock of these tests stands on 2026-03-31 (Santiago), so "this month" is 2026-03.

const marchClock = "2026-03-31T15:00:00Z"

type fixedRow struct {
	ID     string
	Name   string
	Amount float64
	Active bool
}

func listFixed(t *testing.T, api *client, query string) map[string]fixedRow {
	t.Helper()
	status, raw := api.do("GET", "/fixed-costs"+query, nil)
	if status != http.StatusOK {
		t.Fatalf("GET /fixed-costs%s: status = %d: %s", query, status, raw)
	}
	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatalf("not a JSON list: %v\n%s", err, raw)
	}
	out := map[string]fixedRow{}
	for _, r := range rows {
		id, _ := r["id"].(string)
		name, _ := r["name"].(string)
		amount, _ := r["monthly_amount"].(float64)
		active, _ := r["is_active"].(bool)
		out[id] = fixedRow{ID: id, Name: name, Amount: amount, Active: active}
	}
	return out
}

func newFixed(t *testing.T, api *client, body map[string]any) string {
	t.Helper()
	status, raw := api.do("POST", "/fixed-costs", body)
	if status != http.StatusCreated {
		t.Fatalf("POST /fixed-costs %v: status = %d: %s", body, status, raw)
	}
	return decode(t, raw)["id"].(string)
}

func putFixed(t *testing.T, api *client, id string, body map[string]any) (int, []byte) {
	t.Helper()
	return api.do("PUT", "/fixed-costs/"+id, body)
}

// amountIn is the amount in force in a month; ok is false when the cost is not listed that month.
func amountIn(t *testing.T, api *client, id, month string) (amount float64, active, ok bool) {
	t.Helper()
	r, ok := listFixed(t, api, "?month="+month)[id]
	return r.Amount, r.Active, ok
}

func expectAmount(t *testing.T, api *client, id, month string, want float64) {
	t.Helper()
	got, active, ok := amountIn(t, api, id, month)
	if !ok || !active || got != want {
		t.Errorf("%s: amount = %v (listed %v, active %v), want %v", month, got, ok, active, want)
	}
}

func versionCount(t *testing.T, api *timeAPI, id string) int {
	t.Helper()
	var n int
	if err := api.db.QueryRow(context.Background(),
		`select count(*) from fixed_cost_versions where fixed_cost_id = $1`, id).Scan(&n); err != nil {
		t.Fatalf("fixed_cost_versions: %v", err)
	}
	return n
}

// F2: a cost is created from this month by default, or from the month the owner says.
func TestF2_CreateStartsInTheChosenMonth(t *testing.T) {
	api := newAPIAt(t, marchClock)
	byDefault := newFixed(t, api.client, map[string]any{"name": "Arriendo", "monthly_amount": 100})
	expectAmount(t, api.client, byDefault, "2026-03", 100)
	if _, _, ok := amountIn(t, api.client, byDefault, "2026-02"); ok {
		t.Error("a cost created this month must not be listed in February")
	}

	forgotten := newFixed(t, api.client, map[string]any{"name": "Luz", "monthly_amount": 40, "valid_from": "2026-01"})
	expectAmount(t, api.client, forgotten, "2026-01", 40)
	expectAmount(t, api.client, forgotten, "2026-03", 40)
	if _, _, ok := amountIn(t, api.client, forgotten, "2025-12"); ok {
		t.Error("a cost that starts in January must not be listed in December")
	}

	later := newFixed(t, api.client, map[string]any{"name": "Contador", "monthly_amount": 70, "valid_from": "2026-05"})
	if _, _, ok := amountIn(t, api.client, later, "2026-03"); ok {
		t.Error("a cost that starts in May must not be listed in March")
	}
	expectAmount(t, api.client, later, "2026-05", 70)

	for _, bad := range []string{"2026-13", "26-01", "2026-1", "2026-01-15", "enero", "2026/01"} {
		if status, _ := api.do("POST", "/fixed-costs", map[string]any{"name": "X", "monthly_amount": 1, "valid_from": bad}); status != http.StatusBadRequest {
			t.Errorf("valid_from %q: status = %d, want 400", bad, status)
		}
	}
}

// F3: changing the amount starts a version in the chosen month and leaves the other months alone,
// including months in the past (a forgotten change can be put right).
func TestF3_ChangingTheAmountOnlyTouchesItsMonths(t *testing.T) {
	api := newAPIAt(t, marchClock)
	id := newFixed(t, api.client, map[string]any{"name": "Gas", "monthly_amount": 100, "valid_from": "2026-01"})

	if status, raw := putFixed(t, api.client, id, map[string]any{"name": "Gas", "monthly_amount": 150, "is_active": true}); status != http.StatusOK {
		t.Fatalf("raise from this month: status = %d: %s", status, raw)
	}
	for month, want := range map[string]float64{"2026-01": 100, "2026-02": 100, "2026-03": 150, "2026-04": 150} {
		expectAmount(t, api.client, id, month, want)
	}

	t.Run("a month in the past can be corrected and keeps its own value", func(t *testing.T) {
		if status, raw := putFixed(t, api.client, id, map[string]any{"name": "Gas", "monthly_amount": 120, "is_active": true, "effective_from": "2026-02"}); status != http.StatusOK {
			t.Fatalf("status = %d: %s", status, raw)
		}
		for month, want := range map[string]float64{"2026-01": 100, "2026-02": 120, "2026-03": 150} {
			expectAmount(t, api.client, id, month, want)
		}
	})

	t.Run("a change in a month that already has a version replaces it", func(t *testing.T) {
		if status, raw := putFixed(t, api.client, id, map[string]any{"name": "Gas", "monthly_amount": 130, "is_active": true, "effective_from": "2026-03"}); status != http.StatusOK {
			t.Fatalf("status = %d: %s", status, raw)
		}
		expectAmount(t, api.client, id, "2026-03", 130)
		if n := versionCount(t, api, id); n != 3 {
			t.Errorf("versions = %d, want 3 (January, February, March)", n)
		}
	})

	t.Run("a bad month is refused", func(t *testing.T) {
		for _, bad := range []string{"2026-13", "marzo", "2026-03-01"} {
			if status, _ := putFixed(t, api.client, id, map[string]any{"name": "Gas", "monthly_amount": 1, "is_active": true, "effective_from": bad}); status != http.StatusBadRequest {
				t.Errorf("effective_from %q: status = %d, want 400", bad, status)
			}
		}
	})
}

// F4: renaming does not create a version, and the new name shows in every month.
func TestF4_RenamingKeepsTheVersions(t *testing.T) {
	api := newAPIAt(t, marchClock)
	id := newFixed(t, api.client, map[string]any{"name": "Gas", "monthly_amount": 100, "valid_from": "2026-01"})
	if status, raw := putFixed(t, api.client, id, map[string]any{"name": "Gas licuado", "monthly_amount": 100, "is_active": true}); status != http.StatusOK {
		t.Fatalf("status = %d: %s", status, raw)
	}
	if n := versionCount(t, api, id); n != 1 {
		t.Errorf("versions = %d, want 1: the amount did not change", n)
	}
	for _, month := range []string{"2026-01", "2026-03"} {
		if r := listFixed(t, api.client, "?month="+month)[id]; r.Name != "Gas licuado" {
			t.Errorf("%s: name = %q, want the new name", month, r.Name)
		}
	}
}

// F5: the list is of a month: this one by default.
func TestF5_ListIsOfAMonth(t *testing.T) {
	api := newAPIAt(t, marchClock)
	id := newFixed(t, api.client, map[string]any{"name": "Arriendo", "monthly_amount": 100, "valid_from": "2026-01"})
	if status, raw := putFixed(t, api.client, id, map[string]any{"name": "Arriendo", "monthly_amount": 200, "is_active": true, "effective_from": "2026-03"}); status != http.StatusOK {
		t.Fatalf("status = %d: %s", status, raw)
	}
	if got := listFixed(t, api.client, "")[id]; got.Amount != 200 {
		t.Errorf("without month: amount = %v, want 200 (March, the current month)", got.Amount)
	}
	expectAmount(t, api.client, id, "2026-02", 100)
	for _, bad := range []string{"?month=2026-13", "?month=abc", "?month=2026-03-01", "?month="} {
		if status, _ := api.do("GET", "/fixed-costs"+bad, nil); status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", bad, status)
		}
	}
}

// F6: deleting or switching a cost off ends it from a month; the months before keep counting.
func TestF6_EndingACostKeepsItsHistory(t *testing.T) {
	api := newAPIAt(t, marchClock)
	id := newFixed(t, api.client, map[string]any{"name": "Antiguo", "monthly_amount": 100, "valid_from": "2026-01"})

	if status, _ := api.do("DELETE", "/fixed-costs/"+id, nil); status != http.StatusNoContent {
		t.Fatalf("DELETE: status = %d, want 204", status)
	}
	expectAmount(t, api.client, id, "2026-02", 100)
	if _, active, _ := amountIn(t, api.client, id, "2026-03"); active {
		t.Error("March: the cost must not count after it was deleted this month")
	}
	var rows int
	if err := api.db.QueryRow(context.Background(), `select count(*) from fixed_costs where id = $1`, id).Scan(&rows); err != nil || rows != 1 {
		t.Errorf("the row must be kept (count = %d, err = %v)", rows, err)
	}

	t.Run("switching it back on clears the end", func(t *testing.T) {
		if status, raw := putFixed(t, api.client, id, map[string]any{"name": "Antiguo", "monthly_amount": 100, "is_active": true}); status != http.StatusOK {
			t.Fatalf("status = %d: %s", status, raw)
		}
		expectAmount(t, api.client, id, "2026-03", 100)
	})

	t.Run("switching it off from a past month ends it from there", func(t *testing.T) {
		if status, raw := putFixed(t, api.client, id, map[string]any{"name": "Antiguo", "monthly_amount": 100, "is_active": false, "effective_from": "2026-02"}); status != http.StatusOK {
			t.Fatalf("status = %d: %s", status, raw)
		}
		expectAmount(t, api.client, id, "2026-01", 100)
		if _, active, _ := amountIn(t, api.client, id, "2026-02"); active {
			t.Error("February: the cost must not count from the month it was ended")
		}
	})
}

func fixedCostWithHistory(t *testing.T, api *timeAPI) string {
	t.Helper()
	id := newFixed(t, api.client, map[string]any{"name": "Arriendo", "monthly_amount": 100, "valid_from": "2026-01"})
	if status, raw := putFixed(t, api.client, id, map[string]any{"name": "Arriendo", "monthly_amount": 150, "is_active": true, "effective_from": "2026-03"}); status != http.StatusOK {
		t.Fatalf("status = %d: %s", status, raw)
	}
	return id
}

// F7: the monthly dashboard uses the cost in force in the month it is asked for.
func TestF7_MonthlyMetricsUseThatMonthsCost(t *testing.T) {
	api := newAPIAt(t, marchClock)
	fixedCostWithHistory(t, api)
	for _, c := range []struct {
		month, year string
		want        float64
	}{{"1", "2026", 100}, {"2", "2026", 100}, {"3", "2026", 150}, {"12", "2025", 0}} {
		status, raw := api.do("GET", "/analytics/monthly?month="+c.month+"&year="+c.year, nil)
		if status != http.StatusOK {
			t.Fatalf("status = %d: %s", status, raw)
		}
		if got := number(t, decode(t, raw), "total_fixed_costs"); got != c.want {
			t.Errorf("%s/%s: total_fixed_costs = %v, want %v", c.month, c.year, got, c.want)
		}
	}
}

// F8: the trend uses each month's cost, still in a handful of queries.
func TestF8_TrendUsesEachMonthsCost(t *testing.T) {
	api := newAPIAt(t, marchClock)
	fixedCostWithHistory(t, api)
	api.count.n.Store(0)
	status, raw := api.do("GET", "/analytics/trends?months=4", nil)
	queries := api.count.n.Load()
	if status != http.StatusOK {
		t.Fatalf("status = %d: %s", status, raw)
	}
	var points []map[string]any
	if err := json.Unmarshal(raw, &points); err != nil {
		t.Fatal(err)
	}
	want := map[string]float64{"2025-12": 0, "2026-01": -100, "2026-02": -100, "2026-03": -150}
	if len(points) != len(want) {
		t.Fatalf("points = %v", points)
	}
	for _, p := range points {
		if got := p["net_profit"].(float64); got != want[p["date"].(string)] {
			t.Errorf("%v: net_profit = %v, want %v", p["date"], got, want[p["date"].(string)])
		}
	}
	if queries > 4 {
		t.Errorf("the trend ran %d queries, want at most 4", queries)
	}
}

// F9: the costs are the business's: both accounts see them, change them and end them, and get the same numbers.
func TestF9_CostsAreSharedByBothAccounts(t *testing.T) {
	api := newAPIAt(t, marchClock)
	id := fixedCostWithHistory(t, api)

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

	if got := listFixed(t, other, "?month=2026-03"); len(got) != 1 || got[id].Amount != 150 {
		t.Errorf("the other account sees %v, want the cost of 150", got)
	}
	if status, raw := putFixed(t, other, id, map[string]any{"name": "Arriendo", "monthly_amount": 200, "is_active": true}); status != http.StatusOK {
		t.Fatalf("PUT of the other account's cost: status = %d: %s", status, raw)
	}
	expectAmount(t, api.client, id, "2026-03", 200) // the first account sees the change
	if status, _ := other.do("DELETE", "/fixed-costs/"+id, nil); status != http.StatusNoContent {
		t.Fatalf("DELETE of the other account's cost: status = %d, want 204", status)
	}
	for _, month := range []string{"2", "3"} {
		statusA, rawA := api.do("GET", "/analytics/monthly?month="+month+"&year=2026", nil)
		statusB, rawB := other.do("GET", "/analytics/monthly?month="+month+"&year=2026", nil)
		if statusA != http.StatusOK || statusB != http.StatusOK {
			t.Fatalf("statuses = %d and %d", statusA, statusB)
		}
		a, b := number(t, decode(t, rawA), "total_fixed_costs"), number(t, decode(t, rawB), "total_fixed_costs")
		want := map[string]float64{"2": 100, "3": 0}[month]
		if a != want || b != want {
			t.Errorf("month %s: total_fixed_costs = %v and %v, want %v for both", month, a, b, want)
		}
	}
}
