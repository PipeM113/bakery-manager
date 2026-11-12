package app_test

import (
	"net/http"
	"strings"
	"testing"
)

// Sprint 15: deleting something that has history or is in use is a conflict that says why (D1-D4), and a
// recipe only gets its photo through the upload (P1).

func exists(t *testing.T, api *client, path string) bool {
	t.Helper()
	status, _ := api.do("GET", path, nil)
	return status == http.StatusOK
}

func lowerBody(raw []byte) string { return strings.ToLower(string(raw)) }

// --- D1, D2: ingredients ---

func TestD1_AnIngredientUsedByRecipesIsNotDeletedAndTheMessageNamesThem(t *testing.T) {
	api, _ := newPricingAPI(t)
	flour := ing(t, api, "Harina", "M", "", nil)
	for _, name := range []string{"Dulce", "Cheesecake", "Alfajor", "Brownie"} {
		recipeWith(t, api, name, flour, 100, "gr")
	}
	status, raw := api.do("DELETE", "/ingredients/"+flour, nil)
	msg := string(raw)
	if status != http.StatusConflict {
		t.Fatalf("status = %d, want 409: %s", status, raw)
	}
	for _, want := range []string{"Alfajor", "Brownie", "Cheesecake"} {
		if !strings.Contains(msg, want) {
			t.Errorf("the message %q should name %s (the first three in alphabetical order)", msg, want)
		}
	}
	if strings.Contains(msg, "Dulce") || !strings.Contains(lowerBody(raw), "más") {
		t.Errorf("the message %q should name only three recipes and say that there are more", msg)
	}
	if status, _ := api.do("GET", "/ingredients", nil); status != http.StatusOK || len(getList(t, api, "/ingredients")) != 1 {
		t.Error("the refused delete removed the ingredient")
	}
}

func TestD2_AnIngredientOnlyInTheSalesHistoryIsNotDeleted(t *testing.T) {
	api, _ := newPricingAPI(t)
	old := ing(t, api, "Harina vieja", "M", "", map[string]any{"stock_quantity": 5000})
	fresh := ing(t, api, "Harina nueva", "M", "", map[string]any{"stock_quantity": 5000})
	recipe := recipeWith(t, api, "Pan", old, 100, "gr")
	sellOK(t, api, map[string]any{"recipe_id": recipe, "quantity_sold": 1})
	// The recipe moves to the new flour: the old one is now only in the history of that sale.
	if status, raw := api.do("PUT", "/recipes/"+recipe, map[string]any{
		"name": "Pan", "description": "x", "yield": 1, "yield_unit": "porciones",
		"ingredients": []map[string]any{{"ingredient_id": fresh, "quantity": 100, "unit": "gr"}},
	}); status != http.StatusOK {
		t.Fatalf("update recipe: %d %s", status, raw)
	}
	status, raw := api.do("DELETE", "/ingredients/"+old, nil)
	if status != http.StatusConflict || !strings.Contains(lowerBody(raw), "venta") {
		t.Errorf("status = %d, body %s; want 409 saying that it is in the sales history", status, raw)
	}
}

// --- D3: recipes ---

func TestD3_ARecipeWithSalesIsNotDeletedAndTheMessageCountsThem(t *testing.T) {
	api, _ := newPricingAPI(t)
	flour := ing(t, api, "Harina", "M", "", map[string]any{"stock_quantity": 5000})
	recipe := recipeWith(t, api, "Pan", flour, 100, "gr")
	sellOK(t, api, map[string]any{"recipe_id": recipe, "quantity_sold": 1})
	sellOK(t, api, map[string]any{"recipe_id": recipe, "quantity_sold": 1})
	status, raw := api.do("DELETE", "/recipes/"+recipe, nil)
	if status != http.StatusConflict || !strings.Contains(lowerBody(raw), "2 ventas") {
		t.Errorf("status = %d, body %s; want 409 saying that it has 2 ventas", status, raw)
	}
	if !exists(t, api, "/recipes/"+recipe) {
		t.Error("the refused delete removed the recipe")
	}
}

func TestD3_ARecipeInAQuotationIsNotDeleted(t *testing.T) {
	api, _ := newPricingAPI(t)
	flour := ing(t, api, "Harina", "M", "", nil)
	recipe := recipeWith(t, api, "Torta", flour, 100, "gr")
	createQuote(t, api, quote("Ana", 0, ql(recipe, 1, "")))
	status, raw := api.do("DELETE", "/recipes/"+recipe, nil)
	if status != http.StatusConflict || !strings.Contains(lowerBody(raw), "1 cotización") {
		t.Errorf("status = %d, body %s; want 409 saying that it has 1 cotización", status, raw)
	}
	if !exists(t, api, "/recipes/"+recipe) {
		t.Error("the refused delete removed the recipe")
	}
}

func TestD3_ARefusedDeleteKeepsThePhoto(t *testing.T) {
	store := &removingStore{}
	api, _, cake := newRemovingAPI(t, store)
	upload(t, api, cake)
	sellOK(t, api, map[string]any{"recipe_id": cake, "quantity_sold": 1})
	if status, raw := api.do("DELETE", "/recipes/"+cake, nil); status != http.StatusConflict {
		t.Fatalf("status = %d, want 409: %s", status, raw)
	}
	if got := store.removedList(); len(got) != 0 {
		t.Errorf("the photo was removed (%v) although the recipe stayed", got)
	}
}

// --- D4: what is not in use is still deleted ---

func TestD4_AnIngredientAndARecipeWithoutUseAreStillDeleted(t *testing.T) {
	api, _ := newPricingAPI(t)
	salt := ing(t, api, "Sal", "M", "", nil)
	if status, raw := api.do("DELETE", "/ingredients/"+salt, nil); status != http.StatusNoContent {
		t.Errorf("unused ingredient: status = %d, want 204: %s", status, raw)
	}
	flour := ing(t, api, "Harina", "M", "", nil)
	recipe := recipeWith(t, api, "Pan", flour, 100, "gr")
	if status, raw := api.do("DELETE", "/recipes/"+recipe, nil); status != http.StatusNoContent {
		t.Errorf("unused recipe: status = %d, want 204: %s", status, raw)
	}
	// With its recipe gone the ingredient is free too.
	if status, raw := api.do("DELETE", "/ingredients/"+flour, nil); status != http.StatusNoContent {
		t.Errorf("ingredient after its recipe: status = %d, want 204: %s", status, raw)
	}
}

// --- P1: the photo only comes through the upload ---

func TestP1_ACreatedRecipeCannotCarryAPhotoURL(t *testing.T) {
	api, _ := newPricingAPI(t)
	flour := ing(t, api, "Harina", "M", "", nil)
	status, raw := api.do("POST", "/recipes", map[string]any{
		"name": "Pan", "description": "x", "yield": 1, "yield_unit": "porciones",
		"photo_url":   "http://127.0.0.1:9/internal",
		"ingredients": []map[string]any{{"ingredient_id": flour, "quantity": 100, "unit": "gr"}},
	})
	if status != http.StatusBadRequest || !strings.Contains(lowerBody(raw), "foto") {
		t.Errorf("status = %d, body %s; want 400 saying that the photo is uploaded afterwards", status, raw)
	}
	if got := len(getList(t, api, "/recipes")); got != 0 {
		t.Errorf("recipes = %d, want none (the request was refused)", got)
	}
	// A request without it (or with an empty one) is as before.
	if status, raw := api.do("POST", "/recipes", map[string]any{
		"name": "Pan", "description": "x", "yield": 1, "yield_unit": "porciones", "photo_url": nil,
		"ingredients": []map[string]any{{"ingredient_id": flour, "quantity": 100, "unit": "gr"}},
	}); status != http.StatusCreated {
		t.Errorf("a null photo_url: status = %d, want 201: %s", status, raw)
	}
}

func TestP1_ASavedCopyStillKeepsThePhotoOfTheOriginal(t *testing.T) {
	store := &removingStore{}
	api, _, cake := newRemovingAPI(t, store)
	upload(t, api, cake)
	original := photoOf(api, cake)
	status, raw := api.do("POST", "/recipes/"+cake+"/save-as", map[string]any{"portions": 4, "new_name": "Torta chica"})
	if status != http.StatusCreated {
		t.Fatalf("save-as: %d %s", status, raw)
	}
	if got := decode(t, raw)["photo_url"]; got != original {
		t.Errorf("the copy has photo_url %v, want %q", got, original)
	}
}

// Support tests from the adversarial review of S15.
func TestD1_TwoRecipesWithTheSameNameCountAsTwo(t *testing.T) {
	api, _ := newPricingAPI(t)
	flour := ing(t, api, "Harina", "M", "", nil)
	for _, name := range []string{"Pan", "Pan", "Pan", "Pan", "Pan"} {
		recipeWith(t, api, name, flour, 100, "gr")
	}
	_, raw := api.do("DELETE", "/ingredients/"+flour, nil)
	if !strings.Contains(string(raw), "y 2 más") {
		t.Errorf("message %s: five recipes, three named, should say \"y 2 más\"", raw)
	}
}
