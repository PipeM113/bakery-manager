package app_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/PipeM113/bakery-manager/internal/app"
	"github.com/PipeM113/bakery-manager/internal/config"
	"github.com/PipeM113/bakery-manager/internal/shared/kernel"
	"github.com/PipeM113/bakery-manager/internal/testsupport/pgtest"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// Sprint 5: the business clock (America/Santiago) in the handlers, the months of the
// trend, and how much work the trend asks of the database.

// 23:30 of the 31st of March in Santiago (UTC-3 until the first Saturday of April) is already
// 02:30 of the 1st of April in UTC, where the server runs.
const lateMarch31 = "2026-04-01T02:30:00Z"

type queryCounter struct{ n atomic.Int64 }

func (c *queryCounter) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	c.n.Add(1)
	return ctx
}
func (c *queryCounter) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

type timeAPI struct {
	*client
	db    *pgxpool.Pool
	setAt func(string)
	count *queryCounter
}

// newAPIAt serves the real router with a clock the test can move, and counts the statements
// that reach the database.
func newAPIAt(t *testing.T, instant string) *timeAPI {
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
	cfg, err := kernel.NewPoolConfig(admin.Config().ConnConfig.ConnString(), 4)
	if err != nil {
		t.Fatal(err)
	}
	counter := &queryCounter{}
	cfg.ConnConfig.Tracer = counter
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	now := mustParse(t, instant)
	clk := func() time.Time { return now }
	api := &client{t: t, h: app.NewWithClock(config.Config{Port: "8080", DatabaseURL: "unused", JWTSecret: goodSecret, DBMaxConns: 4}, pool, nil, clk)}
	status, raw := api.do("POST", "/auth/login", map[string]string{"email": testEmail, "password": testPassword})
	if status != http.StatusOK {
		t.Fatalf("login: status = %d: %s", status, raw)
	}
	api.token, _ = decode(t, raw)["token"].(string)
	return &timeAPI{client: api, db: admin, count: counter, setAt: func(s string) { now = mustParse(t, s) }}
}

func mustParse(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// rawGet is a GET that also gives the headers.
func (a *timeAPI) rawGet(path string) (int, http.Header, []byte) {
	req := httptest.NewRequest("GET", path, nil)
	req.Header.Set("Authorization", "Bearer "+a.token)
	rec := httptest.NewRecorder()
	a.h.ServeHTTP(rec, req)
	return rec.Code, rec.Header(), rec.Body.Bytes()
}

func stockedFlourRecipe(t *testing.T, api *client) (recipeID, flourID string) {
	t.Helper()
	flourID = createIngredient(t, api, map[string]any{
		"name": "Harina", "default_unit": "gr", "package_size": 1000, "package_price": 1000, "stock_quantity": 5000,
	})
	recipeID = createRecipe(t, api, map[string]any{
		"name": "Bizcocho", "description": "Bizcocho simple", "yield": 1, "yield_unit": "porciones",
		"ingredients": []map[string]any{{"ingredient_id": flourID, "quantity": 500, "unit": "gr"}},
	})
	return
}

// T2: a sale without a date is dated with the Santiago day.
func TestT2_SaleDefaultsToTheSantiagoDate(t *testing.T) {
	for _, c := range []struct{ name, instant, want string }{
		{"summer, UTC is already the next day", lateMarch31, "2026-03-31"},
		{"winter, UTC is already the next day", "2026-07-01T03:30:00Z", "2026-06-30"},
		{"midday", "2026-03-31T15:00:00Z", "2026-03-31"},
	} {
		t.Run(c.name, func(t *testing.T) {
			api := newAPIAt(t, c.instant)
			recipe, _ := stockedFlourRecipe(t, api.client)
			status, raw := api.do("POST", "/sales", map[string]any{"recipe_id": recipe, "quantity_sold": 1, "unit_price": 1000})
			if status != http.StatusCreated {
				t.Fatalf("status = %d: %s", status, raw)
			}
			if got := decode(t, raw)["sale_date"]; got != c.want {
				t.Errorf("sale_date = %v, want %s", got, c.want)
			}
		})
	}
}

// T2: the price history is dated with the Santiago day too.
func TestT2_PriceHistoryUsesTheSantiagoDate(t *testing.T) {
	api := newAPIAt(t, lateMarch31)
	_, flour := stockedFlourRecipe(t, api.client)
	status, raw := api.do("GET", "/ingredients/"+flour+"/history", nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d: %s", status, raw)
	}
	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil || len(rows) == 0 {
		t.Fatalf("history: %v %s", err, raw)
	}
	if got, _ := rows[0]["effective_date"].(string); !strings.HasPrefix(got, "2026-03-31") {
		t.Errorf("effective_date = %q, want 2026-03-31", got)
	}
}

// T2: with no month asked for, the dashboard and the expenses list are those of the Santiago month.
func TestT2_DefaultMonthIsTheSantiagoMonth(t *testing.T) {
	api := newAPIAt(t, lateMarch31)
	recipe, _ := stockedFlourRecipe(t, api.client)
	if status, raw := api.do("POST", "/sales", map[string]any{
		"recipe_id": recipe, "quantity_sold": 1, "unit_price": 1000, "sale_date": "2026-03-31",
	}); status != http.StatusCreated {
		t.Fatalf("sale: status = %d: %s", status, raw)
	}
	gas := categoryID(t, api.client, "gas")
	if status, raw := api.do("POST", "/expenses", map[string]any{
		"description": "Gas de marzo", "amount": 5000, "category_id": gas, "expense_date": "2026-03-31",
	}); status != http.StatusCreated {
		t.Fatalf("expense: status = %d: %s", status, raw)
	}

	monthly := getJSON(t, api.client, "/analytics/monthly")
	if number(t, monthly, "month") != 3 || number(t, monthly, "year") != 2026 {
		t.Errorf("monthly defaults to %v/%v, want 3/2026", monthly["month"], monthly["year"])
	}
	if got := number(t, monthly, "total_revenue"); got != 1000 {
		t.Errorf("total_revenue = %v, want 1000", got)
	}

	_, raw := api.do("GET", "/expenses", nil)
	if !strings.Contains(string(raw), "Gas de marzo") {
		t.Errorf("the expenses of the Santiago month (March) should be listed: %s", raw)
	}
	_, raw = api.do("GET", "/analytics/recipes", nil)
	if !strings.Contains(string(raw), "Bizcocho") {
		t.Errorf("the recipe metrics should default to March: %s", raw)
	}
}

// T2: the file names carry the Santiago date.
func TestT2_FileNamesCarryTheSantiagoDate(t *testing.T) {
	api := newAPIAt(t, lateMarch31)
	recipe, _ := stockedFlourRecipe(t, api.client)

	_, headers, _ := api.rawGet("/ingredients/export")
	if got := headers.Get("Content-Disposition"); !strings.Contains(got, "20260331") {
		t.Errorf("export Content-Disposition = %q, want the date 20260331", got)
	}

	body, _ := json.Marshal(map[string]any{"recipe_id": recipe, "client_name": "Ana", "margin_pct": 0.3})
	req := httptest.NewRequest("POST", "/quotations/generate", strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer "+api.token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	api.h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("quotation: status = %d: %s", rec.Code, rec.Body)
	}
	if got := rec.Header().Get("Content-Disposition"); !strings.Contains(got, "2026-03-31") {
		t.Errorf("quotation Content-Disposition = %q, want the date 2026-03-31", got)
	}
}

// T3: an expense of today is accepted at any hour; tomorrow (in Santiago) is not, even when it is
// already today in UTC.
func TestT3_ExpenseDateIsComparedWithTheSantiagoDay(t *testing.T) {
	for _, hour := range []struct{ name, instant string }{
		{"23:30 in Santiago", lateMarch31},
		{"early morning in Santiago", "2026-03-31T05:30:00Z"},
		{"midday", "2026-03-31T15:00:00Z"},
	} {
		t.Run(hour.name, func(t *testing.T) {
			api := newAPIAt(t, hour.instant)
			gas := categoryID(t, api.client, "gas")
			try := func(date string) int {
				status, _ := api.do("POST", "/expenses", map[string]any{
					"description": "Gas", "amount": 1000, "category_id": gas, "expense_date": date,
				})
				return status
			}
			if got := try("2026-03-31"); got != http.StatusCreated {
				t.Errorf("today (2026-03-31): status = %d, want 201", got)
			}
			if got := try("2026-03-30"); got != http.StatusCreated {
				t.Errorf("yesterday: status = %d, want 201", got)
			}
			if got := try("2026-04-01"); got != http.StatusBadRequest {
				t.Errorf("tomorrow (2026-04-01): status = %d, want 400", got)
			}
		})
	}
}

// T3: the same rule when an expense is edited.
func TestT3_EditingAnExpenseUsesTheSameRule(t *testing.T) {
	api := newAPIAt(t, lateMarch31)
	gas := categoryID(t, api.client, "gas")
	status, raw := api.do("POST", "/expenses", map[string]any{"description": "Gas", "amount": 1000, "category_id": gas, "expense_date": "2026-03-30"})
	if status != http.StatusCreated {
		t.Fatalf("create: status = %d: %s", status, raw)
	}
	id := decode(t, raw)["id"].(string)
	put := func(date string) int {
		s, _ := api.do("PUT", "/expenses/"+id, map[string]any{"description": "Gas", "amount": 1000, "category_id": gas, "expense_date": date})
		return s
	}
	if got := put("2026-03-31"); got != http.StatusOK {
		t.Errorf("moving it to today: status = %d, want 200", got)
	}
	if got := put("2026-04-01"); got != http.StatusBadRequest {
		t.Errorf("moving it to tomorrow: status = %d, want 400", got)
	}
}

func trendDates(t *testing.T, api *timeAPI, query string) []string {
	t.Helper()
	status, raw := api.do("GET", "/analytics/trends"+query, nil)
	if status != http.StatusOK {
		t.Fatalf("trends%s: status = %d: %s", query, status, raw)
	}
	var points []map[string]any
	if err := json.Unmarshal(raw, &points); err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(points))
	for _, p := range points {
		out = append(out, p["date"].(string))
	}
	return out
}

// T4: the trend lists consecutive months, none repeated and none missing, whatever the day.
func TestT4_TrendMonthsAreConsecutive(t *testing.T) {
	api := newAPIAt(t, "2026-03-31T15:00:00Z")
	cases := []struct {
		instant, query string
		want           []string
	}{
		{"2026-03-31T15:00:00Z", "?months=4", []string{"2025-12", "2026-01", "2026-02", "2026-03"}},
		{"2026-01-31T15:00:00Z", "?months=3", []string{"2025-11", "2025-12", "2026-01"}},
		{"2024-03-31T15:00:00Z", "?months=3", []string{"2024-01", "2024-02", "2024-03"}},
		{lateMarch31, "?months=2", []string{"2026-02", "2026-03"}},
	}
	for _, c := range cases {
		api.setAt(c.instant)
		got := trendDates(t, api, c.query)
		if strings.Join(got, ",") != strings.Join(c.want, ",") {
			t.Errorf("at %s %s: months = %v, want %v", c.instant, c.query, got, c.want)
		}
	}
}

// T5: the number of months is 1 to 24 (6 when it is not given).
func TestT5_MonthsParameter(t *testing.T) {
	api := newAPIAt(t, "2026-03-31T15:00:00Z")
	if got := len(trendDates(t, api, "")); got != 6 {
		t.Errorf("default: %d months, want 6", got)
	}
	for q, want := range map[string]int{"?months=1": 1, "?months=12": 12, "?months=24": 24} {
		if got := len(trendDates(t, api, q)); got != want {
			t.Errorf("%s: %d months, want %d", q, got, want)
		}
	}
	for _, q := range []string{"?months=0", "?months=-1", "?months=25", "?months=500", "?months=abc", "?months=1.5"} {
		if status, _ := api.do("GET", "/analytics/trends"+q, nil); status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", q, status)
		}
	}
}

// T6: each point of the trend equals the monthly dashboard of that month, and the trend asks
// the database a handful of questions, not four per month.
//
//	Jan: revenue 1000, ingredients 500, fixed 300                    -> 200
//	Feb: revenue 2 x 1500 = 3000, ingredients 1000, fixed 300, gas 400 -> 1300
//	Mar: revenue 2000, ingredients 500, fixed 300                    -> 1200
func TestT6_TrendEqualsTheMonthlyDashboardInFewQueries(t *testing.T) {
	api := newAPIAt(t, "2026-03-31T15:00:00Z")
	recipe, _ := stockedFlourRecipe(t, api.client) // 500 gr at 1 per gr = 500 per unit
	for _, s := range []map[string]any{
		{"recipe_id": recipe, "quantity_sold": 1, "unit_price": 1000, "sale_date": "2026-01-15"},
		{"recipe_id": recipe, "quantity_sold": 2, "unit_price": 1500, "sale_date": "2026-02-10"},
		{"recipe_id": recipe, "quantity_sold": 1, "unit_price": 2000, "sale_date": "2026-03-05"},
	} {
		if status, raw := api.do("POST", "/sales", s); status != http.StatusCreated {
			t.Fatalf("sale: status = %d: %s", status, raw)
		}
	}
	if status, raw := api.do("POST", "/fixed-costs", map[string]any{"name": "Arriendo", "monthly_amount": 300}); status != http.StatusCreated {
		t.Fatalf("fixed cost: status = %d: %s", status, raw)
	}
	if status, raw := api.do("POST", "/expenses", map[string]any{
		"description": "Gas", "amount": 400, "category_id": categoryID(t, api.client, "gas"), "expense_date": "2026-02-20",
	}); status != http.StatusCreated {
		t.Fatalf("expense: status = %d: %s", status, raw)
	}

	api.count.n.Store(0)
	status, raw := api.do("GET", "/analytics/trends?months=3", nil)
	queries := api.count.n.Load()
	if status != http.StatusOK {
		t.Fatalf("trends: status = %d: %s", status, raw)
	}
	var points []struct {
		Date      string  `json:"date"`
		Revenue   float64 `json:"revenue"`
		NetProfit float64 `json:"net_profit"`
	}
	if err := json.Unmarshal(raw, &points); err != nil || len(points) != 3 {
		t.Fatalf("trend: %v %s", err, raw)
	}
	want := []struct {
		date         string
		revenue, net float64
	}{{"2026-01", 1000, 200}, {"2026-02", 3000, 1300}, {"2026-03", 2000, 1200}}
	for i, w := range want {
		if points[i].Date != w.date || points[i].Revenue != w.revenue || points[i].NetProfit != w.net {
			t.Errorf("point %d = %+v, want %s revenue %v net %v", i, points[i], w.date, w.revenue, w.net)
		}
		month := getJSON(t, api.client, "/analytics/monthly?month="+itoa(i+1)+"&year=2026")
		if number(t, month, "total_revenue") != points[i].Revenue || number(t, month, "net_profit") != points[i].NetProfit {
			t.Errorf("%s: the trend (%v, %v) differs from the monthly dashboard (%v, %v)",
				w.date, points[i].Revenue, points[i].NetProfit, month["total_revenue"], month["net_profit"])
		}
	}

	if queries > 6 {
		t.Errorf("the trend of 3 months used %d statements; it should need a handful, not 4 per month", queries)
	}
	api.count.n.Store(0)
	if status, _ := api.do("GET", "/analytics/trends?months=24", nil); status != http.StatusOK {
		t.Fatal("trends 24 failed")
	}
	if got := api.count.n.Load(); got > 6 {
		t.Errorf("the trend of 24 months used %d statements; the number must not grow with the months", got)
	}
}
