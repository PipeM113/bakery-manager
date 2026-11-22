package app_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

// Review of 2026-11-22: the lists that grow with the years (quotations, sales) take limit and offset, give the
// newest first and say whether there is more. Without them the answer is the first 200, never all of the table.

func page(t *testing.T, api *client, path string) (items []map[string]any, hasMore string, status int) {
	t.Helper()
	rec := newRecorder(api, "GET", path)
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
			t.Fatalf("%s: %v: %s", path, err, rec.Body.String())
		}
	}
	return items, rec.Header().Get("X-Has-More"), rec.Code
}

func TestL1_QuotationsComeInPagesNewestFirstAndSayWhetherThereIsMore(t *testing.T) {
	api, _ := newReadyAPI(t)
	_, cake, _ := bakery(t, api)
	for _, client := range []string{"Ana", "Bea", "Carla"} {
		createQuote(t, api, quote(client, 0, ql(cake, 1, "")))
	}

	all, more, status := page(t, api, "/quotations")
	if status != http.StatusOK || len(all) != 3 || more != "false" {
		t.Fatalf("no parameters: status %d, %d items, has-more %q; want all 3 and false", status, len(all), more)
	}
	if all[0]["client_name"] != "Carla" || all[2]["client_name"] != "Ana" {
		t.Errorf("order = %v ... %v, want the newest first", all[0]["client_name"], all[2]["client_name"])
	}

	first, more, _ := page(t, api, "/quotations?limit=2")
	if len(first) != 2 || more != "true" || first[0]["client_name"] != "Carla" {
		t.Errorf("limit=2: %d items, has-more %q, first %v", len(first), more, first[0]["client_name"])
	}
	rest, more, _ := page(t, api, "/quotations?limit=2&offset=2")
	if len(rest) != 1 || more != "false" || rest[0]["client_name"] != "Ana" {
		t.Errorf("limit=2&offset=2: %d items, has-more %q", len(rest), more)
	}
	if beyond, more, _ := page(t, api, "/quotations?limit=2&offset=10"); len(beyond) != 0 || more != "false" {
		t.Errorf("past the end: %d items, has-more %q, want none and false", len(beyond), more)
	}
}

func TestL2_APageKeepsTheStatusFilter(t *testing.T) {
	api, _ := newReadyAPI(t)
	_, cake, _ := bakery(t, api)
	for _, client := range []string{"Ana", "Bea", "Carla"} {
		id := createQuote(t, api, quote(client, 0, ql(cake, 1, "")))["id"].(string)
		if client != "Bea" {
			api.do("PUT", "/quotations/"+id+"/cancel", nil)
		}
	}
	got, more, _ := page(t, api, "/quotations?status=cancelled&limit=1")
	if len(got) != 1 || more != "true" || got[0]["status"] != "cancelled" {
		t.Errorf("filtered page: %d items, has-more %q, %v", len(got), more, got)
	}
}

func TestL3_SalesComeInPagesToo(t *testing.T) {
	api, _ := newPricingAPI(t)
	flour := ing(t, api, "Harina", "Selecta", "", map[string]any{"stock_quantity": 100000})
	recipe := recipeWith(t, api, "Pan", flour, 100, "gr")
	for i := 0; i < 3; i++ {
		sellOK(t, api, map[string]any{"recipe_id": recipe, "quantity_sold": 1})
	}
	first, more, status := page(t, api, "/sales?limit=2")
	if status != http.StatusOK || len(first) != 2 || more != "true" {
		t.Fatalf("limit=2: status %d, %d items, has-more %q", status, len(first), more)
	}
	rest, more, _ := page(t, api, "/sales?limit=2&offset=2")
	if len(rest) != 1 || more != "false" {
		t.Errorf("second page: %d items, has-more %q", len(rest), more)
	}
	all, more, _ := page(t, api, "/sales")
	if len(all) != 3 || more != "false" {
		t.Errorf("no parameters: %d items, has-more %q; want all 3 and false", len(all), more)
	}
}

func TestL4_AnInvalidLimitOrOffsetIsRefused(t *testing.T) {
	api, _ := newReadyAPI(t)
	for _, path := range []string{
		"/quotations?limit=0", "/quotations?limit=1001", "/quotations?limit=abc", "/quotations?offset=-1", "/quotations?offset=x",
		"/sales?limit=0", "/sales?limit=1001", "/sales?offset=-1",
	} {
		if _, _, status := page(t, api, path); status != http.StatusBadRequest {
			t.Errorf("GET %s: status = %d, want 400", path, status)
		}
	}
}
