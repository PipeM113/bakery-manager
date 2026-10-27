package app_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// Sprint 14: ingredients that differ by brand or supplier (E1-E5), weights of a litre and of a unit so that
// gr <-> ml and und <-> gr work (E6-E8), and the quick "compra" of stock (E9).

func ing(t *testing.T, api *client, name, brand, supplier string, extra map[string]any) string {
	t.Helper()
	body := map[string]any{"name": name, "brand": brand, "supplier": supplier, "default_unit": "gr", "package_size": 1000, "package_price": 2000}
	for k, v := range extra {
		body[k] = v
	}
	status, raw := api.do("POST", "/ingredients", body)
	if status != http.StatusCreated {
		t.Fatalf("create %v: status = %d: %s", body, status, raw)
	}
	return decode(t, raw)["id"].(string)
}

func ingredientBody(t *testing.T, api *client, id string) map[string]any {
	t.Helper()
	for _, i := range getList(t, api, "/ingredients") {
		if i["id"] == id {
			return i
		}
	}
	t.Fatalf("ingredient %s not found", id)
	return nil
}

// putIng sends the whole ingredient again with some fields changed.
func putIng(t *testing.T, api *client, id string, changes map[string]any) (int, []byte) {
	t.Helper()
	cur := ingredientBody(t, api, id)
	body := map[string]any{}
	for _, k := range []string{"name", "brand", "supplier", "kind", "default_unit", "package_size", "package_price", "stock_quantity", "alert_threshold", "grams_per_liter", "grams_per_unit"} {
		if v, ok := cur[k]; ok {
			body[k] = v
		}
	}
	for k, v := range changes {
		body[k] = v
	}
	return api.do("PUT", "/ingredients/"+id, body)
}

func recipeWith(t *testing.T, api *client, name, ingredient string, quantity float64, unit string) string {
	t.Helper()
	return createRecipe(t, api, map[string]any{
		"name": name, "description": "x", "yield": 1, "yield_unit": "porciones",
		"ingredients": []map[string]any{{"ingredient_id": ingredient, "quantity": quantity, "unit": unit}},
	})
}

func ingredientsTotal(t *testing.T, api *client, recipe string) float64 {
	t.Helper()
	return number(t, getJSON(t, api, "/recipes/"+recipe+"/cost"), "ingredients_total")
}

// --- E1, E2: the key is name + brand + supplier ---

func TestE1_SameNameWithAnotherBrandOrSupplierIsAnotherIngredient(t *testing.T) {
	api, _ := newPricingAPI(t)
	ing(t, api, "Mantequilla", "Soprole", "La Vega", map[string]any{"package_price": 2500})
	ing(t, api, "Mantequilla", "Soprole", "Líder", map[string]any{"package_price": 2900})
	ing(t, api, "Mantequilla", "Colún", "La Vega", nil)
	ing(t, api, "Mantequilla", "Soprole", "", nil)
	if got := len(getList(t, api, "/ingredients")); got != 4 {
		t.Errorf("ingredients = %d, want 4 (butter of 4 different origins)", got)
	}
}

func TestE1_TheSameThreeAreADuplicateAndTheMessageSaysSo(t *testing.T) {
	api, _ := newPricingAPI(t)
	ing(t, api, "Mantequilla", "Soprole", "La Vega", nil)
	for _, c := range []struct{ name, brand, supplier string }{
		{"Mantequilla", "Soprole", "La Vega"},
		{"  mantequilla ", "SOPROLE", "la vega"}, // case and spaces do not make it another one
	} {
		status, raw := api.do("POST", "/ingredients", map[string]any{"name": c.name, "brand": c.brand, "supplier": c.supplier,
			"default_unit": "gr", "package_size": 1000, "package_price": 1})
		msg := strings.ToLower(string(raw))
		if status != http.StatusConflict || !strings.Contains(msg, "marca") || !strings.Contains(msg, "proveedor") {
			t.Errorf("%+v: status = %d, body %s; want 409 naming the name, brand and supplier", c, status, raw)
		}
	}
}

func TestE1_RenamingOntoAnotherOneIsAConflictNotAnError(t *testing.T) {
	api, _ := newPricingAPI(t)
	ing(t, api, "Mantequilla", "Soprole", "La Vega", nil)
	other := ing(t, api, "Mantequilla", "Soprole", "Líder", nil)
	if status, raw := putIng(t, api, other, map[string]any{"supplier": "La Vega"}); status != http.StatusConflict {
		t.Errorf("status = %d, want 409: %s", status, raw)
	}
	if status, raw := putIng(t, api, other, map[string]any{"supplier": "Jumbo"}); status != http.StatusOK {
		t.Errorf("a free combination: status = %d, want 200: %s", status, raw)
	}
}

func TestE2_SupplierIsOptionalTrimmedAndLimited(t *testing.T) {
	api, _ := newPricingAPI(t)
	id := ing(t, api, "Harina", "Selecta", "  La Vega  ", nil)
	if got := ingredientBody(t, api, id)["supplier"]; got != "La Vega" {
		t.Errorf("supplier = %v, want %q", got, "La Vega")
	}
	plain := ing(t, api, "Sal", "Lobos", "", nil)
	if got, ok := ingredientBody(t, api, plain)["supplier"]; !ok || got != "" {
		t.Errorf("supplier = %v (%v), want an empty text", got, ok)
	}
	if status, _ := api.do("POST", "/ingredients", map[string]any{"name": "X", "brand": "Y", "supplier": strings.Repeat("a", 101),
		"default_unit": "gr", "package_size": 1, "package_price": 1}); status != http.StatusBadRequest {
		t.Errorf("a 101 character supplier: status = %d, want 400", status)
	}
}

// --- E3: import and export ---

func importFile(t *testing.T, api *client, rows [][]string) (int, map[string]any) {
	t.Helper()
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	for i, r := range rows {
		for j, v := range r {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+1)
			if err := f.SetCellStr(sheet, cell, v); err != nil {
				t.Fatal(err)
			}
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, _ := mw.CreateFormFile("file", "insumos.xlsx")
	part.Write(buf.Bytes()) //nolint:errcheck
	mw.Close()
	req := httptest.NewRequest("POST", "/ingredients/import", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+api.token)
	rec := httptest.NewRecorder()
	api.h.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestE3_ImportMatchesByNameBrandAndSupplier(t *testing.T) {
	api, _ := newPricingAPI(t)
	header := []string{"Nombre", "Marca", "Unidad", "Tamaño", "Precio", "Proveedor"}
	status, res := importFile(t, api, [][]string{header,
		{"Mantequilla", "Soprole", "gr", "250", "2500", "La Vega"},
		{"Mantequilla", "Soprole", "gr", "250", "2900", "Líder"},
		{"Mantequilla", "Soprole", "gr", "250", "3000", "la vega"}, // the same as row 2 of the file
	})
	if status != http.StatusOK {
		t.Fatalf("status = %d: %v", status, res)
	}
	if number(t, res, "imported") != 2 {
		t.Errorf("imported = %v, want 2 (two suppliers)", res["imported"])
	}
	if errs, _ := res["errors"].([]any); len(errs) != 1 {
		t.Errorf("errors = %v, want the repeated row only", res["errors"])
	}

	// Importing again updates the one with the same three, and only that one.
	_, res = importFile(t, api, [][]string{header, {"mantequilla", "soprole", "gr", "250", "2600", "LÍDER"}})
	if number(t, res, "updated") != 1 || number(t, res, "imported") != 0 {
		t.Errorf("second import = %v, want 1 updated and none new", res)
	}
	for _, i := range getList(t, api, "/ingredients") {
		want := 2500.0
		if i["supplier"] == "Líder" {
			want = 2600
		}
		if number(t, i, "package_price") != want {
			t.Errorf("%v / %v: package_price = %v, want %v", i["brand"], i["supplier"], i["package_price"], want)
		}
	}
}

func TestE3_AFileWithoutTheSupplierColumnStillImports(t *testing.T) {
	api, _ := newPricingAPI(t)
	status, res := importFile(t, api, [][]string{
		{"Nombre", "Marca", "Unidad", "Tamaño", "Precio"},
		{"Harina", "Selecta", "gr", "1000", "1500"},
	})
	if status != http.StatusOK || number(t, res, "imported") != 1 {
		t.Fatalf("status = %d, result %v", status, res)
	}
	if got := getList(t, api, "/ingredients")[0]["supplier"]; got != "" {
		t.Errorf("supplier = %v, want empty", got)
	}
}

func TestE3_ExportWritesTheSupplier(t *testing.T) {
	api, _ := newPricingAPI(t)
	ing(t, api, "Mantequilla", "Soprole", "La Vega", nil)
	status, raw := api.do("GET", "/ingredients/export", nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	f, err := excelize.OpenReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := f.GetRows(f.GetSheetName(0))
	if err != nil || len(rows) < 2 {
		t.Fatalf("rows = %v (%v)", rows, err)
	}
	col := -1
	for i, h := range rows[0] {
		if strings.EqualFold(strings.TrimSpace(h), "Proveedor") {
			col = i
		}
	}
	if col < 0 || len(rows[1]) <= col || rows[1][col] != "La Vega" {
		t.Errorf("the export has no Proveedor column with La Vega: %v", rows)
	}
}

// --- E4, E5: order and brand ---

func TestE4_ListsAreAlphabeticalWithoutCaseOrAccents(t *testing.T) {
	api, _ := newPricingAPI(t)
	for _, n := range []string{"Zanahoria", "azúcar", "Álamo", "Éter", "bicarbonato"} {
		ing(t, api, n, "M", "", nil)
	}
	var got []string
	for _, i := range getList(t, api, "/ingredients") {
		got = append(got, i["name"].(string))
	}
	if want := "Álamo,azúcar,bicarbonato,Éter,Zanahoria"; strings.Join(got, ",") != want {
		t.Errorf("ingredients = %v, want %s", got, want)
	}

	// Same name: by brand, then by supplier.
	ing(t, api, "Leche", "Soprole", "Z", nil)
	ing(t, api, "Leche", "Colún", "B", nil)
	ing(t, api, "Leche", "Colún", "A", nil)
	var milk []string
	for _, i := range getList(t, api, "/ingredients") {
		if i["name"] == "Leche" {
			milk = append(milk, fmt.Sprintf("%v/%v", i["brand"], i["supplier"]))
		}
	}
	if want := "Colún/A,Colún/B,Soprole/Z"; strings.Join(milk, ",") != want {
		t.Errorf("milk = %v, want %s", milk, want)
	}
}

func TestE4_RecipesAreAlphabeticalToo(t *testing.T) {
	api, _ := newPricingAPI(t)
	flour := ing(t, api, "Harina", "M", "", nil)
	for _, n := range []string{"torta", "Alfajor", "Éclair", "brownie"} {
		recipeWith(t, api, n, flour, 100, "gr")
	}
	var got []string
	for _, r := range getList(t, api, "/recipes") {
		got = append(got, r["name"].(string))
	}
	if want := "Alfajor,brownie,Éclair,torta"; strings.Join(got, ",") != want {
		t.Errorf("recipes = %v, want %s", got, want)
	}
}

func TestE5_RecipeLinesCarryTheBrandAndTheSupplier(t *testing.T) {
	api, _ := newPricingAPI(t)
	butter := ing(t, api, "Mantequilla", "Soprole", "La Vega", nil)
	recipe := recipeWith(t, api, "Galletas", butter, 100, "gr")
	check := func(label string, line map[string]any) {
		if line["brand"] != "Soprole" || line["supplier"] != "La Vega" {
			t.Errorf("%s: line = %v, want brand Soprole and supplier La Vega", label, line)
		}
	}
	lines := func(r map[string]any) map[string]any { return r["ingredients"].([]any)[0].(map[string]any) }
	check("GET /recipes/{id}", lines(getJSON(t, api, "/recipes/"+recipe)))
	check("GET /recipes", lines(getList(t, api, "/recipes")[0]))
}

// --- E6, E7: the equivalences and the units of a recipe ---

func TestE6_TheEquivalencesAreOptionalAndValidated(t *testing.T) {
	api, _ := newPricingAPI(t)
	plain := ing(t, api, "Harina", "M", "", nil)
	if b := ingredientBody(t, api, plain); number(t, b, "grams_per_liter") != 0 || number(t, b, "grams_per_unit") != 0 {
		t.Errorf("without them both are 0: %v", b)
	}
	for label, extra := range map[string]map[string]any{
		"negative litre":    {"grams_per_liter": -1},
		"huge litre":        {"grams_per_liter": 100000},
		"negative unit":     {"grams_per_unit": -5},
		"huge unit":         {"grams_per_unit": 10000000},
		"text as the litre": {"grams_per_liter": "mucho"},
	} {
		body := map[string]any{"name": "X" + label, "brand": "M", "default_unit": "lt", "package_size": 1, "package_price": 1}
		for k, v := range extra {
			body[k] = v
		}
		if status, _ := api.do("POST", "/ingredients", body); status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", label, status)
		}
	}
	milk := ing(t, api, "Leche", "Soprole", "", map[string]any{"default_unit": "lt", "package_size": 1, "package_price": 1000, "grams_per_liter": 1030.5})
	if got := number(t, ingredientBody(t, api, milk), "grams_per_liter"); got != 1030.5 {
		t.Errorf("grams_per_liter = %v, want 1030.5", got)
	}
}

func TestE6_CostsUseTheEquivalence(t *testing.T) {
	api, _ := newPricingAPI(t)
	// A litre costs 1000 and weighs 1250 gr: 625 gr are half a litre, 500 pesos.
	milk := ing(t, api, "Leche", "Soprole", "", map[string]any{"default_unit": "lt", "package_size": 1, "package_price": 1000, "grams_per_liter": 1250})
	if got := ingredientsTotal(t, api, recipeWith(t, api, "Flan", milk, 625, "gr")); got != 500 {
		t.Errorf("625 gr of milk cost %v, want 500", got)
	}
	if got := ingredientsTotal(t, api, recipeWith(t, api, "Flan2", milk, 500, "ml")); got != 500 {
		t.Errorf("500 ml of milk cost %v, want 500 (the same family needs nothing)", got)
	}
	// An egg: 12 for 3600, 60 gr each. 120 gr are 2 eggs, 600 pesos.
	egg := ing(t, api, "Huevo", "Doña Ana", "", map[string]any{"default_unit": "und", "package_size": 12, "package_price": 3600, "grams_per_unit": 60})
	if got := ingredientsTotal(t, api, recipeWith(t, api, "Tortilla", egg, 120, "gr")); got != 600 {
		t.Errorf("120 gr of egg cost %v, want 600", got)
	}
}

func TestE6_SellingTakesTheStockThroughTheEquivalence(t *testing.T) {
	api, _ := newPricingAPI(t)
	milk := ing(t, api, "Leche", "Soprole", "", map[string]any{"default_unit": "lt", "package_size": 1, "package_price": 1000, "grams_per_liter": 1250, "stock_quantity": 2})
	recipe := recipeWith(t, api, "Flan", milk, 625, "gr")
	sellOK(t, api, map[string]any{"recipe_id": recipe, "quantity_sold": 1})
	if got := stockOf(t, api, milk); !near(got, 1.5) {
		t.Errorf("stock = %v lt, want 1.5 (625 gr are half a litre)", got)
	}
}

func TestE6_AdjustmentsConvertThroughTheEquivalence(t *testing.T) {
	api, _ := newPricingAPI(t)
	milk := ing(t, api, "Leche", "Soprole", "", map[string]any{"default_unit": "lt", "package_size": 1, "package_price": 1000, "grams_per_liter": 1250, "stock_quantity": 2})
	adjustOK(t, api, milk, map[string]any{"reason": "merma", "quantity": 625, "unit": "gr"})
	if got := stockOf(t, api, milk); !near(got, 1.5) {
		t.Errorf("stock = %v, want 1.5", got)
	}
	plain := ing(t, api, "Aceite", "Chef", "", map[string]any{"default_unit": "lt", "package_size": 1, "package_price": 3000, "stock_quantity": 2})
	status, raw := adjust(t, api, plain, map[string]any{"reason": "merma", "quantity": 500, "unit": "gr"})
	if status != http.StatusBadRequest || !strings.Contains(strings.ToLower(string(raw)), "peso de 1 litro") {
		t.Errorf("without the equivalence: status = %d, body %s; want 400 asking for the weight of 1 litre", status, raw)
	}
}

func TestE6_ATestOfTheQuotationAndTheCostScreenAgree(t *testing.T) {
	api, _ := newPricingAPI(t)
	milk := ing(t, api, "Leche", "Soprole", "", map[string]any{"default_unit": "lt", "package_size": 1, "package_price": 1000, "grams_per_liter": 1250})
	recipe := recipeWith(t, api, "Flan", milk, 625, "gr")
	cost := getJSON(t, api, "/recipes/"+recipe+"/cost?indirect_cost_pct=0&labor_cost_pct=0.2&margin_pct=0.25")
	q := createQuote(t, api, quote("Ana", 0, ql(recipe, 1, "")))
	if got, want := number(t, quoteLines(t, q)[0], "unit_price"), number(t, cost, "suggested_price"); got != want {
		t.Errorf("quotation = %v, cost screen = %v: they must agree", got, want)
	}
}

func TestE7_ARecipeWithAnIncompatibleUnitIsRefusedAndSaysWhatToDo(t *testing.T) {
	api, _ := newPricingAPI(t)
	flour := ing(t, api, "Harina", "M", "", nil)                                                                           // gr
	oil := ing(t, api, "Aceite", "M", "", map[string]any{"default_unit": "lt", "package_size": 1, "package_price": 3000})  // lt
	egg := ing(t, api, "Huevo", "M", "", map[string]any{"default_unit": "und", "package_size": 12, "package_price": 3600}) // und
	try := func(label, ingredient, unit, want string) {
		status, raw := api.do("POST", "/recipes", map[string]any{
			"name": "R " + label, "description": "x", "yield": 1, "yield_unit": "porciones",
			"ingredients": []map[string]any{{"ingredient_id": ingredient, "quantity": 100, "unit": unit}},
		})
		if status != http.StatusBadRequest || !strings.Contains(strings.ToLower(string(raw)), want) {
			t.Errorf("%s: status = %d, body %s; want 400 mentioning %q", label, status, raw, want)
		}
	}
	try("ml of flour", flour, "ml", "peso de 1 litro")
	try("gr of oil", oil, "gr", "peso de 1 litro")
	try("gr of eggs", egg, "gr", "peso de 1 unidad")
	try("und of flour", flour, "und", "peso de 1 unidad")

	// Updating a recipe is checked the same way.
	good := recipeWith(t, api, "Pan", flour, 100, "gr")
	status, raw := api.do("PUT", "/recipes/"+good, map[string]any{
		"name": "Pan", "description": "x", "yield": 1, "yield_unit": "porciones",
		"ingredients": []map[string]any{{"ingredient_id": flour, "quantity": 100, "unit": "ml"}},
	})
	if status != http.StatusBadRequest {
		t.Errorf("update with ml of flour: status = %d, want 400: %s", status, raw)
	}
	// Units of the same family and unit spellings that already worked keep working.
	if status, raw := api.do("POST", "/recipes", map[string]any{
		"name": "Pan2", "description": "x", "yield": 1, "yield_unit": "porciones",
		"ingredients": []map[string]any{{"ingredient_id": flour, "quantity": 0.1, "unit": "kg"}},
	}); status != http.StatusCreated {
		t.Errorf("kg of flour: status = %d, want 201: %s", status, raw)
	}
}

// --- E8: changing the unit of an ingredient ---

func TestE8_ChangingTheUnitFreelyWhenNoRecipeUsesIt(t *testing.T) {
	api, _ := newPricingAPI(t)
	id := ing(t, api, "Leche", "Soprole", "", nil)
	if status, raw := putIng(t, api, id, map[string]any{"default_unit": "ml"}); status != http.StatusOK {
		t.Errorf("gr to ml with no recipes: status = %d, want 200: %s", status, raw)
	}
}

func TestE8_ChangingTheFamilyAsksForTheEquivalenceOnlyWhenARecipeNeedsIt(t *testing.T) {
	api, _ := newPricingAPI(t)
	id := ing(t, api, "Leche", "Soprole", "", nil) // 1000 gr for 2000
	recipe := recipeWith(t, api, "Flan", id, 500, "gr")

	status, raw := putIng(t, api, id, map[string]any{"default_unit": "ml"})
	msg := strings.ToLower(string(raw))
	if status != http.StatusConflict || !strings.Contains(msg, "flan") || !strings.Contains(msg, "peso de 1 litro") {
		t.Errorf("without the equivalence: status = %d, body %s; want 409 naming the recipe and asking for the weight of 1 litre", status, raw)
	}
	if got := ingredientBody(t, api, id)["default_unit"]; got != "gr" {
		t.Errorf("a refused change altered the unit: %v", got)
	}

	// With the equivalence in the same request the change goes through, and the recipe still has a cost:
	// 1000 ml for 2000 is 2 per ml; 500 gr at 1250 gr/lt are 400 ml, 800 pesos.
	if status, raw := putIng(t, api, id, map[string]any{"default_unit": "ml", "grams_per_liter": 1250}); status != http.StatusOK {
		t.Fatalf("with the equivalence: status = %d, want 200: %s", status, raw)
	}
	if got := ingredientsTotal(t, api, recipe); got != 800 {
		t.Errorf("the recipe costs %v, want 800", got)
	}
}

func TestE8_FromUnitsToWeightAsksForTheWeightOfOneUnit(t *testing.T) {
	api, _ := newPricingAPI(t)
	egg := ing(t, api, "Huevo", "Doña Ana", "", map[string]any{"default_unit": "und", "package_size": 12, "package_price": 3600})
	recipe := recipeWith(t, api, "Tortilla", egg, 2, "und")
	status, raw := putIng(t, api, egg, map[string]any{"default_unit": "gr"})
	if status != http.StatusConflict || !strings.Contains(strings.ToLower(string(raw)), "peso de 1 unidad") {
		t.Errorf("status = %d, body %s; want 409 asking for the weight of 1 unit", status, raw)
	}
	// 12 eggs of 60 gr are 720 gr for 3600: 5 per gram; 2 eggs are 120 gr, 600 pesos.
	if status, raw := putIng(t, api, egg, map[string]any{"default_unit": "gr", "package_size": 720, "grams_per_unit": 60}); status != http.StatusOK {
		t.Fatalf("with the equivalence: status = %d: %s", status, raw)
	}
	if got := ingredientsTotal(t, api, recipe); got != 600 {
		t.Errorf("the recipe costs %v, want 600", got)
	}
}

func TestE8_WithinTheSameFamilyNothingIsAsked(t *testing.T) {
	api, _ := newPricingAPI(t)
	id := ing(t, api, "Harina", "M", "", map[string]any{"default_unit": "kg", "package_size": 1, "package_price": 2000})
	recipeWith(t, api, "Pan", id, 500, "gr")
	if status, raw := putIng(t, api, id, map[string]any{"default_unit": "gr", "package_size": 1000}); status != http.StatusOK {
		t.Errorf("kg to gr with a recipe: status = %d, want 200: %s", status, raw)
	}
}

func TestE8_RemovingAnEquivalenceThatARecipeNeedsIsRefused(t *testing.T) {
	api, _ := newPricingAPI(t)
	milk := ing(t, api, "Leche", "Soprole", "", map[string]any{"default_unit": "lt", "package_size": 1, "package_price": 1000, "grams_per_liter": 1250})
	recipeWith(t, api, "Flan", milk, 625, "gr")
	status, raw := putIng(t, api, milk, map[string]any{"grams_per_liter": 0})
	msg := strings.ToLower(string(raw))
	if status != http.StatusConflict || !strings.Contains(msg, "flan") || !strings.Contains(msg, "peso de 1 litro") {
		t.Errorf("status = %d, body %s; want 409 naming the recipe", status, raw)
	}
	if got := number(t, ingredientBody(t, api, milk), "grams_per_liter"); got != 1250 {
		t.Errorf("a refused change altered the equivalence: %v", got)
	}
}

// --- E9: the quick "compra" ---

func TestE9_APurchaseAddsToTheStockAndIsNotALoss(t *testing.T) {
	api, _ := newPricingAPI(t)
	id := ing(t, api, "Harina", "M", "", map[string]any{"stock_quantity": 100})
	a := adjustOK(t, api, id, map[string]any{"reason": "compra", "quantity": 500})
	if a["reason"] != "compra" || number(t, a, "stock_before") != 100 || number(t, a, "stock_after") != 600 || number(t, a, "quantity_delta") != 500 || number(t, a, "loss_amount") != 0 {
		t.Errorf("adjustment = %v, want a purchase of +500 that is not a loss", a)
	}
	if got := stockOf(t, api, id); got != 600 {
		t.Errorf("stock = %v, want 600", got)
	}
	// In another unit of the same family, and it stays in the history.
	adjustOK(t, api, id, map[string]any{"reason": "compra", "quantity": 1, "unit": "kg"})
	if got := stockOf(t, api, id); got != 1600 {
		t.Errorf("stock = %v, want 1600 (1 kg more)", got)
	}
	history := getList(t, api, "/ingredients/"+id+"/stock-adjustments")
	if len(history) != 2 || history[0]["reason"] != "compra" {
		t.Errorf("history = %v, want the two purchases", history)
	}
}

func TestE9_PurchaseRules(t *testing.T) {
	api, _ := newPricingAPI(t)
	milk := ing(t, api, "Leche", "Soprole", "", map[string]any{"default_unit": "lt", "package_size": 1, "package_price": 1000, "grams_per_liter": 1250, "stock_quantity": 1})
	for label, body := range map[string]map[string]any{
		"zero":      {"reason": "compra", "quantity": 0},
		"negative":  {"reason": "compra", "quantity": -5},
		"no amount": {"reason": "compra"},
	} {
		if status, raw := adjust(t, api, milk, body); status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400: %s", label, status, raw)
		}
	}
	adjustOK(t, api, milk, map[string]any{"reason": "compra", "quantity": 1250, "unit": "gr"})
	if got := stockOf(t, api, milk); !near(got, 2) {
		t.Errorf("stock = %v lt, want 2 (1250 gr are a litre)", got)
	}
}
