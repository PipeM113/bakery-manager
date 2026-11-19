package app_test

import (
	"net/http"
	"strings"
	"testing"
)

// ADR-005: an ingredient is name + brand; a purchase carries the supplier, the size of the package and the price
// paid, and the price of the ingredient becomes the last one paid. Rules B1-B10 (purchases), A1-A4 (the key)
// and G1-G2 (the Excel import).

func buy(t *testing.T, api *client, id string, body map[string]any) map[string]any {
	t.Helper()
	body["reason"] = "compra"
	return adjustOK(t, api, id, body)
}

func priceHistory(t *testing.T, api *client, id string) []map[string]any {
	t.Helper()
	return getList(t, api, "/ingredients/"+id+"/history")
}

// flour is 1000 gr for $1.000 (1 peso per gram) with no stock.
func flour(t *testing.T, api *client) string {
	t.Helper()
	return ing(t, api, "Harina", "Selecta", "", map[string]any{"package_size": 1000, "package_price": 1000})
}

func TestB1_APurchaseWithItsPackageSetsTheLastPricePaidTheStockAndTheSupplier(t *testing.T) {
	api, _ := newPricingAPI(t)
	id := flour(t, api)
	buy(t, api, id, map[string]any{
		"quantity": 50, "unit": "kg", "package_size": 25, "package_unit": "kg", "package_price": 20000, "supplier": "Mayorista", "adjusted_on": "2026-03-10",
	})
	got := ingredientBody(t, api, id)
	if number(t, got, "stock_quantity") != 50000 {
		t.Errorf("stock = %v, want 50000 gr (50 kg bought)", got["stock_quantity"])
	}
	if number(t, got, "package_size") != 25000 || number(t, got, "package_price") != 20000 {
		t.Errorf("package = %v x $%v, want 25000 gr for $20000", got["package_size"], got["package_price"])
	}
	if p := number(t, got, "price_per_unit"); p < 0.7999 || p > 0.8001 {
		t.Errorf("price_per_unit = %v, want 0.8 (20000 / 25000 gr)", p)
	}
	if got["supplier"] != "Mayorista" {
		t.Errorf("supplier = %v, want Mayorista", got["supplier"])
	}
	if n := len(priceHistory(t, api, id)); n != 2 {
		t.Errorf("history rows = %d, want 2 (the creation and the purchase)", n)
	}
}

func TestB2_ThePackageWithoutAUnitIsInTheUnitOfTheIngredient(t *testing.T) {
	api, _ := newPricingAPI(t)
	id := flour(t, api)
	buy(t, api, id, map[string]any{"quantity": 500, "package_size": 500, "package_price": 900})
	if p := number(t, ingredientBody(t, api, id), "price_per_unit"); p < 1.7999 || p > 1.8001 {
		t.Errorf("price_per_unit = %v, want 1.8 (900 / 500 gr)", p)
	}
}

func TestB3_APurchaseOlderThanTheLastOneIsKeptButDoesNotMoveThePrice(t *testing.T) {
	api, _ := newPricingAPI(t)
	id := flour(t, api)
	buy(t, api, id, map[string]any{"quantity": 1000, "package_size": 1000, "package_price": 2000, "supplier": "Súper", "adjusted_on": "2026-03-10"})
	buy(t, api, id, map[string]any{"quantity": 1000, "package_size": 1000, "package_price": 500, "supplier": "Mayorista", "adjusted_on": "2026-03-01"})
	got := ingredientBody(t, api, id)
	if number(t, got, "package_price") != 2000 || got["supplier"] != "Súper" {
		t.Errorf("the older purchase moved the price: $%v at %v, want $2000 at Súper", got["package_price"], got["supplier"])
	}
	if number(t, got, "stock_quantity") != 2000 {
		t.Errorf("stock = %v, want 2000 (both purchases count)", got["stock_quantity"])
	}
	if n := len(priceHistory(t, api, id)); n != 3 {
		t.Errorf("history rows = %d, want 3 (the creation and both purchases)", n)
	}
}

func TestB4_OnTheSameDayTheLastOneRegisteredWins(t *testing.T) {
	api, _ := newPricingAPI(t)
	id := flour(t, api)
	buy(t, api, id, map[string]any{"quantity": 1000, "package_size": 1000, "package_price": 2000, "adjusted_on": "2026-03-10"})
	buy(t, api, id, map[string]any{"quantity": 1000, "package_size": 1000, "package_price": 1700, "adjusted_on": "2026-03-10"})
	if got := number(t, ingredientBody(t, api, id), "package_price"); got != 1700 {
		t.Errorf("package_price = %v, want 1700 (the second of the day)", got)
	}
}

func TestB5_APurchaseWithoutPackageDataOnlyAddsStock(t *testing.T) {
	api, _ := newPricingAPI(t)
	id := flour(t, api)
	before := ingredientBody(t, api, id)
	buy(t, api, id, map[string]any{"quantity": 500})
	got := ingredientBody(t, api, id)
	if number(t, got, "stock_quantity") != 500 {
		t.Errorf("stock = %v, want 500", got["stock_quantity"])
	}
	if got["package_price"] != before["package_price"] || got["supplier"] != before["supplier"] || got["price_per_unit"] != before["price_per_unit"] {
		t.Errorf("a purchase without price changed the price or the supplier: %v -> %v", before, got)
	}
	if n := len(priceHistory(t, api, id)); n != 1 {
		t.Errorf("history rows = %d, want 1 (only the creation)", n)
	}
}

func TestB6_ThePackageSizeAndThePriceComeTogetherOrNotAtAll(t *testing.T) {
	api, _ := newPricingAPI(t)
	id := flour(t, api)
	for name, body := range map[string]map[string]any{
		"size without price": {"quantity": 500, "package_size": 500},
		"price without size": {"quantity": 500, "package_price": 900},
	} {
		body["reason"] = "compra"
		if status, raw := adjust(t, api, id, body); status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400: %s", name, status, raw)
		}
	}
	if got := number(t, ingredientBody(t, api, id), "stock_quantity"); got != 0 {
		t.Errorf("a refused purchase changed the stock to %v", got)
	}
}

func TestB7_InvalidPurchaseDataIsRefusedAndNothingIsWritten(t *testing.T) {
	api, _ := newPricingAPI(t)
	id := flour(t, api)
	cases := map[string]map[string]any{
		"size 0":                  {"quantity": 1, "package_size": 0, "package_price": 100},
		"negative size":           {"quantity": 1, "package_size": -5, "package_price": 100},
		"price 0":                 {"quantity": 1, "package_size": 1, "package_price": 0},
		"negative price":          {"quantity": 1, "package_size": 1, "package_price": -1},
		"supplier too long":       {"quantity": 1, "package_size": 1, "package_price": 100, "supplier": strings.Repeat("a", 101)},
		"size beyond the column":  {"quantity": 1, "package_size": 1e12, "package_price": 100},
		"price beyond the column": {"quantity": 1, "package_size": 1, "package_price": 1e12},
		"a unit of another family without equivalence": {"quantity": 1, "package_size": 1, "package_unit": "lt", "package_price": 100},
	}
	for name, body := range cases {
		body["reason"] = "compra"
		if status, raw := adjust(t, api, id, body); status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400: %s", name, status, raw)
		}
	}
	if got := ingredientBody(t, api, id); number(t, got, "stock_quantity") != 0 || number(t, got, "package_price") != 1000 {
		t.Errorf("a refused purchase wrote something: %v", got)
	}
	if n := len(priceHistory(t, api, id)); n != 1 {
		t.Errorf("history rows = %d, want 1", n)
	}
}

func TestB8_PackageDataOnlyBelongsToAPurchase(t *testing.T) {
	api, _ := newPricingAPI(t)
	id := ing(t, api, "Harina", "Selecta", "", map[string]any{"stock_quantity": 1000})
	for _, body := range []map[string]any{
		{"reason": "merma", "quantity": 10, "package_size": 1000, "package_price": 1},
		{"reason": "correccion", "counted_quantity": 5, "supplier": "Líder"},
		{"reason": "otro", "quantity_delta": 5, "note": "x", "package_price": 1},
	} {
		if status, raw := adjust(t, api, id, body); status != http.StatusBadRequest {
			t.Errorf("%v: status = %d, want 400: %s", body, status, raw)
		}
	}
}

func TestB9_TheSameProductBoughtInTwoPlacesIsOneIngredientWithOneStock(t *testing.T) {
	api, _ := newPricingAPI(t)
	id := ing(t, api, "Mantequilla", "Soprole", "", map[string]any{"package_size": 250, "package_price": 1300})
	buy(t, api, id, map[string]any{"quantity": 25000, "package_size": 25000, "package_price": 100000, "supplier": "Mayorista", "adjusted_on": "2026-03-01"})
	buy(t, api, id, map[string]any{"quantity": 1000, "package_size": 250, "package_price": 1300, "supplier": "Súper", "adjusted_on": "2026-03-10"})
	list := getList(t, api, "/ingredients")
	if len(list) != 1 {
		t.Fatalf("ingredients = %d, want 1", len(list))
	}
	if number(t, list[0], "stock_quantity") != 26000 || list[0]["supplier"] != "Súper" {
		t.Errorf("stock %v at %v, want 26000 at Súper (the last purchase)", list[0]["stock_quantity"], list[0]["supplier"])
	}
}

func TestB10_TheCostOfARecipeFollowsTheLastPricePaidAndTheHistoryKeepsTheDetail(t *testing.T) {
	api, _ := newPricingAPI(t)
	id := flour(t, api)
	recipe := recipeWith(t, api, "Pan", id, 100, "gr")
	before := ingredientsTotal(t, api, recipe)
	buy(t, api, id, map[string]any{"quantity": 1000, "package_size": 1000, "package_price": 2000, "supplier": "Súper"})
	if after := ingredientsTotal(t, api, recipe); after < 1.99*before || after > 2.01*before {
		t.Errorf("recipe cost %v -> %v, want it doubled by the new price", before, after)
	}
	var purchase map[string]any
	for _, a := range getList(t, api, "/ingredients/"+id+"/stock-adjustments") {
		if a["reason"] == "compra" {
			purchase = a
		}
	}
	if purchase == nil || purchase["supplier"] != "Súper" || number(t, purchase, "package_size") != 1000 || number(t, purchase, "package_price") != 2000 {
		t.Errorf("the purchase in the history lost its detail: %v", purchase)
	}
}

func TestB11_AnOmittedSupplierLeavesTheSupplierAndAnEmptyOneClearsIt(t *testing.T) {
	api, _ := newPricingAPI(t)
	id := ing(t, api, "Harina", "Selecta", "Mayorista", map[string]any{"package_size": 1000, "package_price": 1000})
	buy(t, api, id, map[string]any{"quantity": 1, "package_size": 1000, "package_price": 1100, "adjusted_on": "2026-03-01"})
	if got := ingredientBody(t, api, id)["supplier"]; got != "Mayorista" {
		t.Errorf("supplier = %v, want Mayorista (not given: it stays)", got)
	}
	buy(t, api, id, map[string]any{"quantity": 1, "package_size": 1000, "package_price": 1200, "supplier": "", "adjusted_on": "2026-03-02"})
	if got := ingredientBody(t, api, id)["supplier"]; got != "" {
		t.Errorf("supplier = %v, want empty (given empty: it clears)", got)
	}
}

// --- A: the key is name + brand ---

func TestA1_TheSameNameAndBrandWithAnotherSupplierIsTheSameIngredient(t *testing.T) {
	api, _ := newPricingAPI(t)
	ing(t, api, "Mantequilla", "Soprole", "La Vega", nil)
	for _, c := range []struct{ name, brand, supplier string }{
		{"Mantequilla", "Soprole", "Líder"},
		{"  mantequilla ", "SOPROLE", "la vega"},
		{"Mantequilla", "Soprole", ""},
	} {
		status, raw := api.do("POST", "/ingredients", map[string]any{"name": c.name, "brand": c.brand, "supplier": c.supplier,
			"default_unit": "gr", "package_size": 1000, "package_price": 1})
		msg := strings.ToLower(string(raw))
		if status != http.StatusConflict || !strings.Contains(msg, "marca") || strings.Contains(msg, "proveedor") {
			t.Errorf("%+v: status = %d, body %s; want 409 naming the name and the brand, not the supplier", c, status, raw)
		}
	}
}

// (The API asks for a brand, so "no brand" is not a case of the key here; the database does allow it, see M1.)
func TestA2_AnotherBrandIsAnotherIngredient(t *testing.T) {
	api, _ := newPricingAPI(t)
	ing(t, api, "Mantequilla", "Soprole", "La Vega", nil)
	ing(t, api, "Mantequilla", "Colún", "La Vega", nil)
	ing(t, api, "Mantequilla", "Mantecol", "", nil)
	if got := len(getList(t, api, "/ingredients")); got != 3 {
		t.Errorf("ingredients = %d, want 3", got)
	}
}

func TestA3_ChangingTheSupplierIsFreeButTheBrandCannotLandOnAnotherIngredient(t *testing.T) {
	api, _ := newPricingAPI(t)
	ing(t, api, "Mantequilla", "Soprole", "La Vega", nil)
	other := ing(t, api, "Mantequilla", "Colún", "La Vega", nil)
	if status, raw := putIng(t, api, other, map[string]any{"supplier": "Jumbo"}); status != http.StatusOK {
		t.Errorf("changing the supplier: status = %d, want 200: %s", status, raw)
	}
	if status, raw := putIng(t, api, other, map[string]any{"brand": "Soprole"}); status != http.StatusConflict {
		t.Errorf("renaming onto another: status = %d, want 409: %s", status, raw)
	}
}

// --- G: the Excel import ---

func TestG1_ARowWithTheSameNameAndBrandButAnotherSupplierUpdatesTheIngredient(t *testing.T) {
	api, _ := newPricingAPI(t)
	header := []string{"Nombre", "Marca", "Unidad", "Tamaño", "Precio", "Proveedor"}
	status, res := importFile(t, api, [][]string{header,
		{"Mantequilla", "Soprole", "gr", "250", "2500", "La Vega"},
		{"Harina", "Selecta", "gr", "1000", "1500", ""},
	})
	if status != http.StatusOK || number(t, res, "imported") != 2 {
		t.Fatalf("first import: %d %v", status, res)
	}
	_, res = importFile(t, api, [][]string{header, {"mantequilla", "soprole", "gr", "250", "2600", "Líder"}})
	if number(t, res, "updated") != 1 || number(t, res, "imported") != 0 {
		t.Fatalf("second import = %v, want 1 updated and none new", res)
	}
	for _, i := range getList(t, api, "/ingredients") {
		if i["name"] == "Mantequilla" && (number(t, i, "package_price") != 2600 || i["supplier"] != "Líder") {
			t.Errorf("the ingredient was not updated: %v", i)
		}
	}
	if got := len(getList(t, api, "/ingredients")); got != 2 {
		t.Errorf("ingredients = %d, want 2", got)
	}
}

func TestG2_TwoRowsOfTheFileWithTheSameNameAndBrandAreReportedOnce(t *testing.T) {
	api, _ := newPricingAPI(t)
	status, res := importFile(t, api, [][]string{
		{"Nombre", "Marca", "Unidad", "Tamaño", "Precio", "Proveedor"},
		{"Mantequilla", "Soprole", "gr", "250", "2500", "La Vega"},
		{"Mantequilla", "Soprole", "gr", "250", "2900", "Líder"},
	})
	if status != http.StatusOK || number(t, res, "imported") != 1 {
		t.Fatalf("status = %d, result %v", status, res)
	}
	errs, _ := res["errors"].([]any)
	if len(errs) != 1 || !strings.Contains(strings.ToLower(errs[0].(map[string]any)["reason"].(string)), "nombre y marca") {
		t.Errorf("errors = %v, want one saying the name and brand are repeated", res["errors"])
	}
}
