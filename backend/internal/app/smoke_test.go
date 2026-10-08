package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/PipeM113/bakery-manager/internal/app"
	"github.com/PipeM113/bakery-manager/internal/config"
	"github.com/PipeM113/bakery-manager/internal/shared/kernel"
	"github.com/PipeM113/bakery-manager/internal/testsupport/pgtest"
	"golang.org/x/crypto/bcrypt"
)

const (
	testEmail    = "owner@example.test"
	testPassword = "test-password-123" // throwaway credential of a throwaway database
)

type client struct {
	t     *testing.T
	h     http.Handler
	token string
}

func (c *client) do(method, path string, body any) (int, []byte) {
	c.t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			c.t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, req)
	return rec.Code, rec.Body.Bytes()
}

func decode(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("response is not a JSON object: %v\n%s", err, raw)
	}
	return out
}

func number(t *testing.T, m map[string]any, key string) float64 {
	t.Helper()
	v, ok := m[key].(float64)
	if !ok {
		t.Fatalf("field %q is not a number in %v", key, m)
	}
	return v
}

// newAPI migrates an empty database, seeds one owner and serves the real router through the
// same pool settings used in production (no prepared statements).
func newAPI(t *testing.T) *client {
	t.Helper()
	admin := pgtest.NewDatabase(t)
	pgtest.Migrate(t, admin, "../../db/migrations", "up")

	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	_, err = admin.Exec(context.Background(),
		`insert into users (name, email, password, role) values ('Test Owner', $1, $2, 'owner')`,
		testEmail, string(hash))
	if err != nil {
		t.Fatal(err)
	}

	pool, err := kernel.Open(context.Background(), admin.Config().ConnConfig.ConnString(), 4)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	h := app.New(config.Config{
		Port:        "8080",
		DatabaseURL: "unused-here",
		JWTSecret:   goodSecret,
		DBMaxConns:  4,
	}, pool, nil)
	return &client{t: t, h: h}
}

// AC5 + AC6: the main flows of the system work end to end through the production pool
// settings: login, ingredients, recipes with cost, sales that move stock, expenses, analytics.
// The expected values were worked out by hand; see the comments.
func TestAC5_CoreFlowsWorkWithoutPreparedStatements(t *testing.T) {
	api := newAPI(t)

	t.Run("login rejects a wrong password", func(t *testing.T) {
		status, _ := api.do("POST", "/auth/login", map[string]string{"email": testEmail, "password": "wrong"})
		if status != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", status)
		}
	})

	t.Run("protected routes need a token", func(t *testing.T) {
		if status, _ := api.do("GET", "/ingredients", nil); status != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", status)
		}
	})

	t.Run("login returns a token the API accepts", func(t *testing.T) {
		status, raw := api.do("POST", "/auth/login", map[string]string{"email": testEmail, "password": testPassword})
		if status != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", status, raw)
		}
		api.token, _ = decode(t, raw)["token"].(string)
		if api.token == "" {
			t.Fatal("no token in the response")
		}
		if status, _ := api.do("GET", "/ingredients", nil); status != http.StatusOK {
			t.Fatalf("with the token: status = %d, want 200", status)
		}
	})

	var ingredientID, recipeID string

	t.Run("create an ingredient: 2000 per 1000 gr is 2 per gr", func(t *testing.T) {
		status, raw := api.do("POST", "/ingredients", map[string]any{
			"name": "Harina", "brand": "Marca", "default_unit": "gr",
			"package_size": 1000, "package_price": 2000,
			"stock_quantity": 5000, "alert_threshold": 100,
		})
		if status != http.StatusCreated {
			t.Fatalf("status = %d, want 201: %s", status, raw)
		}
		body := decode(t, raw)
		ingredientID, _ = body["id"].(string)
		if ingredientID == "" {
			t.Fatal("no id in the response")
		}
		if got := number(t, body, "price_per_unit"); got != 2 {
			t.Fatalf("price_per_unit = %v, want 2", got)
		}
	})

	t.Run("a duplicate ingredient name is a conflict", func(t *testing.T) {
		status, _ := api.do("POST", "/ingredients", map[string]any{
			"name": "Harina", "brand": "Otra", "default_unit": "gr",
			"package_size": 500, "package_price": 900,
		})
		if status != http.StatusConflict {
			t.Fatalf("status = %d, want 409", status)
		}
	})

	t.Run("create a recipe with 500 gr of that ingredient", func(t *testing.T) {
		status, raw := api.do("POST", "/recipes", map[string]any{
			"name": "Bizcocho", "description": "Bizcocho simple", "yield": 10, "yield_unit": "porciones",
			"ingredients": []map[string]any{{"ingredient_id": ingredientID, "quantity": 500, "unit": "gr"}},
		})
		if status != http.StatusCreated {
			t.Fatalf("status = %d, want 201: %s", status, raw)
		}
		recipeID, _ = decode(t, raw)["id"].(string)
		if recipeID == "" {
			t.Fatal("no id in the response")
		}
	})

	t.Run("cost: 500 gr x 2 = 1000; with 15% indirect and 30% labor 1450; margin 30% gives 1885, rounded up to 2000", func(t *testing.T) {
		status, raw := api.do("GET", "/recipes/"+recipeID+"/cost?margin_pct=0.3", nil)
		if status != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", status, raw)
		}
		body := decode(t, raw)
		if got := number(t, body, "ingredients_total"); got < 999.999 || got > 1000.001 {
			t.Errorf("ingredients_total = %v, want 1000", got)
		}
		if got := number(t, body, "suggested_price"); got != 2000 {
			t.Errorf("suggested_price = %v, want 2000", got)
		}
	})

	t.Run("a sale of 2 units moves the stock from 5000 to 4000", func(t *testing.T) {
		status, raw := api.do("POST", "/sales", map[string]any{
			"recipe_id": recipeID, "quantity_sold": 2, "unit_price": 3500, "notes": "prueba",
		})
		if status != http.StatusCreated {
			t.Fatalf("status = %d, want 201: %s", status, raw)
		}
		if got := number(t, decode(t, raw), "total_price"); got != 7000 {
			t.Errorf("total_price = %v, want 7000", got)
		}

		status, raw = api.do("GET", "/ingredients", nil)
		if status != http.StatusOK {
			t.Fatalf("list: status = %d", status)
		}
		var list []map[string]any
		if err := json.Unmarshal(raw, &list); err != nil || len(list) != 1 {
			t.Fatalf("unexpected ingredient list: %v %s", err, raw)
		}
		if got := number(t, list[0], "stock_quantity"); got != 4000 {
			t.Errorf("stock_quantity = %v, want 4000", got)
		}
	})

	t.Run("a sale that needs more stock than available is refused (400 today)", func(t *testing.T) {
		status, _ := api.do("POST", "/sales", map[string]any{
			"recipe_id": recipeID, "quantity_sold": 100, "unit_price": 3500,
		})
		if status != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", status)
		}
	})

	t.Run("an expense can be created and listed", func(t *testing.T) {
		// Two days ago: it avoids the known same-day defect listed in docs/backlog.md.
		day := time.Now().AddDate(0, 0, -2)
		status, raw := api.do("POST", "/expenses", map[string]any{
			"description": "Gas del horno", "amount": 15000, "category": "servicios",
			"expense_date": day.Format("2006-01-02"),
		})
		if status != http.StatusCreated {
			t.Fatalf("status = %d, want 201: %s", status, raw)
		}

		path := "/expenses?month=" + itoa(int(day.Month())) + "&year=" + itoa(day.Year())
		status, raw = api.do("GET", path, nil)
		if status != http.StatusOK {
			t.Fatalf("list: status = %d, want 200: %s", status, raw)
		}
		if !strings.Contains(string(raw), "Gas del horno") {
			t.Errorf("the created expense is not in the list: %s", raw)
		}
	})

	t.Run("monthly analytics answers", func(t *testing.T) {
		if status, raw := api.do("GET", "/analytics/monthly", nil); status != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", status, raw)
		}
	})
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}
