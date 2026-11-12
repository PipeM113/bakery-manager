package app_test

import (
	"net/http"
	"strings"
	"testing"
)

// Sprint 16: types of recipe (T1-T5) and the cost in the list of recipes (L1, L2).

func newType(t *testing.T, api *client, name string) string {
	t.Helper()
	status, raw := api.do("POST", "/recipe-categories", map[string]any{"name": name})
	if status != http.StatusCreated {
		t.Fatalf("create type %q: status = %d: %s", name, status, raw)
	}
	return decode(t, raw)["id"].(string)
}

func typeNames(t *testing.T, api *client) []string {
	t.Helper()
	var out []string
	for _, c := range getList(t, api, "/recipe-categories") {
		out = append(out, c["name"].(string))
	}
	return out
}

func recipeOfType(t *testing.T, api *client, name, ingredient, category string) string {
	t.Helper()
	body := map[string]any{
		"name": name, "description": "x", "yield": 1, "yield_unit": "porciones",
		"ingredients": []map[string]any{{"ingredient_id": ingredient, "quantity": 100, "unit": "gr"}},
	}
	if category != "" {
		body["category_id"] = category
	}
	return createRecipe(t, api, body)
}

func recipeJSON(t *testing.T, api *client, id string) map[string]any {
	t.Helper()
	return getJSON(t, api, "/recipes/"+id)
}

// --- T1: create and list ---

func TestT1_ATypeIsCreatedTrimmedAndLimited(t *testing.T) {
	api, _ := newPricingAPI(t)
	status, raw := api.do("POST", "/recipe-categories", map[string]any{"name": "  Torta   bizcocho  "})
	if status != http.StatusCreated || decode(t, raw)["name"] != "Torta bizcocho" {
		t.Errorf("status = %d, body %s; want 201 with the name trimmed and its spaces collapsed", status, raw)
	}
	for label, name := range map[string]string{"empty": "", "only spaces": "   ", "51 characters": strings.Repeat("a", 51)} {
		if status, _ := api.do("POST", "/recipe-categories", map[string]any{"name": name}); status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", label, status)
		}
	}
	if status, _ := api.do("POST", "/recipe-categories", map[string]any{"name": strings.Repeat("a", 50)}); status != http.StatusCreated {
		t.Errorf("50 characters: status = %d, want 201", status)
	}
}

func TestT1_ARepeatedNameIsAConflictWithoutCaseOrAccents(t *testing.T) {
	api, _ := newPricingAPI(t)
	newType(t, api, "Hojarasca")
	for _, name := range []string{"hojarasca", "HOJARASCA", "Hójarasca", "  hojarasca "} {
		status, raw := api.do("POST", "/recipe-categories", map[string]any{"name": name})
		if status != http.StatusConflict || !strings.Contains(lowerBody(raw), "ya existe") {
			t.Errorf("%q: status = %d, body %s; want 409 saying that it already exists", name, status, raw)
		}
	}
}

func TestT1_TheListIsAlphabeticalWithTheNumberOfRecipes(t *testing.T) {
	api, _ := newPricingAPI(t)
	flour := ing(t, api, "Harina", "M", "", nil)
	merengue := newType(t, api, "merengue")
	sponge := newType(t, api, "Bizcocho")
	newType(t, api, "Álamo")
	recipeOfType(t, api, "Pavlova", flour, merengue)
	recipeOfType(t, api, "Beso", flour, merengue)
	recipeOfType(t, api, "Torta", flour, sponge)
	list := getList(t, api, "/recipe-categories")
	var names []string
	counts := map[string]float64{}
	for _, c := range list {
		names = append(names, c["name"].(string))
		counts[c["name"].(string)] = number(t, c, "recipe_count")
	}
	if want := "Álamo,Bizcocho,merengue"; strings.Join(names, ",") != want {
		t.Errorf("types = %v, want %s", names, want)
	}
	if counts["merengue"] != 2 || counts["Bizcocho"] != 1 || counts["Álamo"] != 0 {
		t.Errorf("recipe_count = %v, want merengue 2, Bizcocho 1, Álamo 0", counts)
	}
}

func TestT1_TheTypesNeedALogin(t *testing.T) {
	api, _ := newPricingAPI(t)
	anon := &client{t: t, h: api.h}
	for _, c := range []struct{ method, path string }{
		{"GET", "/recipe-categories"}, {"POST", "/recipe-categories"},
		{"PUT", "/recipe-categories/x"}, {"DELETE", "/recipe-categories/x"},
	} {
		if status, _ := anon.do(c.method, c.path, map[string]any{"name": "x"}); status != http.StatusUnauthorized {
			t.Errorf("%s %s without a token: status = %d, want 401", c.method, c.path, status)
		}
	}
}

// --- T2: rename ---

func TestT2_RenamingChangesTheNameEverywhere(t *testing.T) {
	api, _ := newPricingAPI(t)
	flour := ing(t, api, "Harina", "M", "", nil)
	id := newType(t, api, "Hojarasa")
	recipe := recipeOfType(t, api, "Alfajor", flour, id)
	status, raw := api.do("PUT", "/recipe-categories/"+id, map[string]any{"name": " Hojarasca "})
	if status != http.StatusOK || decode(t, raw)["name"] != "Hojarasca" {
		t.Fatalf("status = %d, body %s; want 200 with the new name", status, raw)
	}
	if got := recipeJSON(t, api, recipe)["category"]; got != "Hojarasca" {
		t.Errorf("the recipe shows %v, want Hojarasca", got)
	}
}

func TestT2_RenameRules(t *testing.T) {
	api, _ := newPricingAPI(t)
	a := newType(t, api, "Hojarasca")
	newType(t, api, "Merengue")
	if status, raw := api.do("PUT", "/recipe-categories/"+a, map[string]any{"name": "merengue"}); status != http.StatusConflict {
		t.Errorf("onto another one: status = %d, want 409: %s", status, raw)
	}
	if status, raw := api.do("PUT", "/recipe-categories/"+a, map[string]any{"name": "HOJARASCA"}); status != http.StatusOK {
		t.Errorf("its own name in another case: status = %d, want 200: %s", status, raw)
	}
	if status, _ := api.do("PUT", "/recipe-categories/"+a, map[string]any{"name": ""}); status != http.StatusBadRequest {
		t.Errorf("empty name: status = %d, want 400", status)
	}
	for _, id := range []string{"00000000-0000-4000-8000-000000000000", "not-a-uuid"} {
		if status, _ := api.do("PUT", "/recipe-categories/"+id, map[string]any{"name": "Otro"}); status != http.StatusNotFound {
			t.Errorf("%s: status = %d, want 404", id, status)
		}
	}
}

// --- T3: delete ---

func TestT3_DeletingATypeLeavesItsRecipesWithoutType(t *testing.T) {
	api, _ := newPricingAPI(t)
	flour := ing(t, api, "Harina", "M", "", nil)
	id := newType(t, api, "Merengue")
	recipe := recipeOfType(t, api, "Pavlova", flour, id)
	if status, raw := api.do("DELETE", "/recipe-categories/"+id, nil); status != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: %s", status, raw)
	}
	got := recipeJSON(t, api, recipe)
	if got["category"] != "" || got["category_id"] != nil {
		t.Errorf("the recipe has category %v / %v, want none", got["category"], got["category_id"])
	}
	if names := typeNames(t, api); len(names) != 0 {
		t.Errorf("types = %v, want none", names)
	}
	for _, again := range []string{id, "not-a-uuid"} {
		if status, _ := api.do("DELETE", "/recipe-categories/"+again, nil); status != http.StatusNotFound {
			t.Errorf("%s: status = %d, want 404", again, status)
		}
	}
}

// --- T4: the type of a recipe ---

func TestT4_ARecipeCarriesItsTypeAndItIsOptional(t *testing.T) {
	api, _ := newPricingAPI(t)
	flour := ing(t, api, "Harina", "M", "", nil)
	hoja := newType(t, api, "Hojarasca")
	with := recipeOfType(t, api, "Alfajor", flour, hoja)
	without := recipeOfType(t, api, "Pan", flour, "")
	if got := recipeJSON(t, api, with); got["category_id"] != hoja || got["category"] != "Hojarasca" {
		t.Errorf("with type: %v / %v, want %s / Hojarasca", got["category_id"], got["category"], hoja)
	}
	if got := recipeJSON(t, api, without); got["category_id"] != nil || got["category"] != "" {
		t.Errorf("without type: %v / %v, want null / empty", got["category_id"], got["category"])
	}
	for _, r := range getList(t, api, "/recipes") {
		if _, ok := r["category"]; !ok {
			t.Errorf("GET /recipes: %v has no category field", r["name"])
		}
		if _, ok := r["category_id"]; !ok {
			t.Errorf("GET /recipes: %v has no category_id field", r["name"])
		}
	}
}

func TestT4_UpdatingKeepsClearsOrChangesTheType(t *testing.T) {
	api, _ := newPricingAPI(t)
	flour := ing(t, api, "Harina", "M", "", nil)
	hoja := newType(t, api, "Hojarasca")
	merengue := newType(t, api, "Merengue")
	recipe := recipeOfType(t, api, "Alfajor", flour, hoja)
	put := func(extra map[string]any) (int, []byte) {
		body := map[string]any{
			"name": "Alfajor", "description": "x", "yield": 1, "yield_unit": "porciones",
			"ingredients": []map[string]any{{"ingredient_id": flour, "quantity": 100, "unit": "gr"}},
		}
		for k, v := range extra {
			body[k] = v
		}
		return api.do("PUT", "/recipes/"+recipe, body)
	}
	if status, raw := put(nil); status != http.StatusOK || recipeJSON(t, api, recipe)["category_id"] != hoja {
		t.Errorf("without the field: status = %d, type = %v; want 200 and the type kept: %s", status, recipeJSON(t, api, recipe)["category_id"], raw)
	}
	if status, raw := put(map[string]any{"category_id": merengue}); status != http.StatusOK || recipeJSON(t, api, recipe)["category_id"] != merengue {
		t.Errorf("another type: status = %d: %s", status, raw)
	}
	for _, bad := range []any{"00000000-0000-4000-8000-000000000000", "not-a-uuid"} {
		if status, raw := put(map[string]any{"category_id": bad}); status != http.StatusBadRequest {
			t.Errorf("category_id %v: status = %d, want 400: %s", bad, status, raw)
		}
	}
	if got := recipeJSON(t, api, recipe)["category_id"]; got != merengue {
		t.Errorf("a refused update changed the type: %v", got)
	}
	if status, raw := put(map[string]any{"category_id": nil}); status != http.StatusOK || recipeJSON(t, api, recipe)["category_id"] != nil {
		t.Errorf("null: status = %d, type = %v; want 200 and no type: %s", status, recipeJSON(t, api, recipe)["category_id"], raw)
	}
}

func TestT4_CreatingWithAnUnknownTypeIsRefused(t *testing.T) {
	api, _ := newPricingAPI(t)
	flour := ing(t, api, "Harina", "M", "", nil)
	status, raw := api.do("POST", "/recipes", map[string]any{
		"name": "Pan", "description": "x", "yield": 1, "yield_unit": "porciones", "category_id": "00000000-0000-4000-8000-000000000000",
		"ingredients": []map[string]any{{"ingredient_id": flour, "quantity": 100, "unit": "gr"}},
	})
	if status != http.StatusBadRequest || !strings.Contains(lowerBody(raw), "tipo") {
		t.Errorf("status = %d, body %s; want 400 saying that the type does not exist", status, raw)
	}
	if got := len(getList(t, api, "/recipes")); got != 0 {
		t.Errorf("recipes = %d, want none", got)
	}
}

func TestT4_ACopyKeepsTheTypeOfTheOriginal(t *testing.T) {
	api, _ := newPricingAPI(t)
	flour := ing(t, api, "Harina", "M", "", map[string]any{"stock_quantity": 5000})
	hoja := newType(t, api, "Hojarasca")
	recipe := recipeOfType(t, api, "Alfajor", flour, hoja)
	for _, c := range []struct {
		path string
		body map[string]any
	}{
		{"/recipes/" + recipe + "/save-as", map[string]any{"portions": 4, "new_name": "Alfajor chico"}},
		{"/recipes/" + recipe + "/save-scaled", map[string]any{"scale_factor": 2, "new_name": "Alfajor doble"}},
	} {
		status, raw := api.do("POST", c.path, c.body)
		if status != http.StatusCreated {
			t.Fatalf("%s: %d %s", c.path, status, raw)
		}
		if got := decode(t, raw)["category_id"]; got != hoja {
			t.Errorf("%s: the copy has type %v, want %s", c.path, got, hoja)
		}
	}
}

// --- T5: without a type nothing changes ---

func TestT5_ARecipeWithoutTypeStillCostsSellsAndQuotes(t *testing.T) {
	api, _ := newPricingAPI(t)
	flour := ing(t, api, "Harina", "M", "", map[string]any{"stock_quantity": 5000})
	recipe := recipeOfType(t, api, "Pan", flour, "")
	if got := ingredientsTotal(t, api, recipe); got != 200 {
		t.Errorf("cost = %v, want 200 (100 gr at 2 per gram)", got)
	}
	sellOK(t, api, map[string]any{"recipe_id": recipe, "quantity_sold": 1})
	createQuote(t, api, quote("Ana", 0, ql(recipe, 1, "")))
}

// --- L1, L2: the cost in the list ---

func TestL1_EveryRecipeOfTheListCarriesItsCostAndItsSuggestedPrice(t *testing.T) {
	api, _ := newPricingAPI(t)
	flour := ing(t, api, "Harina", "M", "", nil)
	sugar := ing(t, api, "Azúcar", "M", "", map[string]any{"package_price": 3000})
	a := recipeOfType(t, api, "Pan", flour, "")
	b := recipeWith(t, api, "Dulce", sugar, 250, "gr")
	for _, r := range getList(t, api, "/recipes") {
		id := r["id"].(string)
		if id != a && id != b {
			continue
		}
		cost := getJSON(t, api, "/recipes/"+id+"/cost")
		if got, want := number(t, r, "total_cost"), number(t, cost, "total_cost"); got != want {
			t.Errorf("%v: total_cost in the list = %v, cost screen = %v", r["name"], got, want)
		}
		if got, want := number(t, r, "suggested_price"), number(t, cost, "suggested_price"); got != want {
			t.Errorf("%v: suggested_price in the list = %v, cost screen = %v", r["name"], got, want)
		}
	}
}

func TestL1_ARecipeThatCannotBePricedGivesNullAndDoesNotBreakTheList(t *testing.T) {
	api := newAPIAt(t, "2026-03-10T15:00:00Z")
	flour := ing(t, api.client, "Harina", "M", "", nil)
	oil := ing(t, api.client, "Aceite", "M", "", map[string]any{"default_unit": "lt", "package_size": 1, "package_price": 3000})
	good := recipeOfType(t, api.client, "Pan", flour, "")
	broken := recipeWith(t, api.client, "Frito", oil, 1, "lt")
	// A line in grams of an ingredient in litres without its weight: it cannot be converted.
	if _, err := api.db.Exec(t.Context(), `update recipe_ingredients set unit = 'gr' where recipe_id = $1`, broken); err != nil {
		t.Fatal(err)
	}
	status, raw := api.do("GET", "/recipes", nil)
	if status != http.StatusOK {
		t.Fatalf("GET /recipes: status = %d, want 200: %s", status, raw)
	}
	for _, r := range getList(t, api.client, "/recipes") {
		switch r["id"] {
		case good:
			if number(t, r, "total_cost") <= 0 {
				t.Errorf("the good recipe has total_cost %v", r["total_cost"])
			}
		case broken:
			if r["total_cost"] != nil || r["suggested_price"] != nil {
				t.Errorf("the recipe that cannot be priced has %v / %v, want null / null", r["total_cost"], r["suggested_price"])
			}
		}
	}
}

func TestL2_TheListUsesTheSameNumberOfQueriesWhateverTheNumberOfRecipes(t *testing.T) {
	api := newAPIAt(t, "2026-03-10T15:00:00Z")
	flour := ing(t, api.client, "Harina", "M", "", nil)
	hoja := newType(t, api.client, "Hojarasca")
	for i := 0; i < 2; i++ {
		recipeOfType(t, api.client, "R"+string(rune('A'+i)), flour, hoja)
	}
	measure := func() int64 {
		before := api.count.n.Load()
		if status, raw := api.do("GET", "/recipes", nil); status != http.StatusOK {
			t.Fatalf("GET /recipes: %d %s", status, raw)
		}
		return api.count.n.Load() - before
	}
	few := measure()
	for i := 0; i < 10; i++ {
		recipeOfType(t, api.client, "S"+string(rune('A'+i)), flour, hoja)
	}
	if many := measure(); many != few {
		t.Errorf("GET /recipes used %d queries with 2 recipes and %d with 12: it must not grow with the number of recipes", few, many)
	}
}
