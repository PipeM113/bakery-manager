package app_test

import (
	"net/http"
	"strings"
	"testing"
)

// Sprint 17: a quotation that is pending or cancelled can be deleted (X1-X6); with it gone, the recipe it
// used is free to delete. A confirmed one is not deleted: it is cancelled first.

func quoteOf(t *testing.T, api *client, recipe string) string {
	t.Helper()
	return createQuote(t, api, quote("Ana", 0, ql(recipe, 1, "")))["id"].(string)
}

func TestX1_APendingQuotationIsDeletedWithItsLines(t *testing.T) {
	api, admin := newReadyAPI(t)
	_, cake, _ := bakery(t, api)
	id := quoteOf(t, api, cake)
	if status, raw := api.do("DELETE", "/quotations/"+id, nil); status != http.StatusNoContent {
		t.Fatalf("DELETE: status = %d, want 204: %s", status, raw)
	}
	if status, _ := api.do("GET", "/quotations/"+id, nil); status != http.StatusNotFound {
		t.Errorf("the deleted quotation still answers (status %d)", status)
	}
	var lines int
	if err := admin.QueryRow(t.Context(), `select count(*) from quotation_lines`).Scan(&lines); err != nil || lines != 0 {
		t.Errorf("lines left after the delete = %d (err %v), want 0", lines, err)
	}
}

func TestX2_ACancelledQuotationIsDeleted(t *testing.T) {
	api, _ := newReadyAPI(t)
	_, cake, _ := bakery(t, api)
	id := quoteOf(t, api, cake)
	if status, raw := api.do("PUT", "/quotations/"+id+"/cancel", nil); status != http.StatusNoContent {
		t.Fatalf("cancel: %d %s", status, raw)
	}
	if status, raw := api.do("DELETE", "/quotations/"+id, nil); status != http.StatusNoContent {
		t.Fatalf("DELETE: status = %d, want 204: %s", status, raw)
	}
}

func TestX3_AConfirmedQuotationIsNotDeletedAndTheMessageSaysWhy(t *testing.T) {
	api, _ := newReadyAPI(t)
	_, cake, _ := bakery(t, api)
	id := quoteOf(t, api, cake)
	if status, raw := api.do("PUT", "/quotations/"+id+"/confirm", nil); status != http.StatusNoContent {
		t.Fatalf("confirm: %d %s", status, raw)
	}
	status, raw := api.do("DELETE", "/quotations/"+id, nil)
	if status != http.StatusConflict {
		t.Fatalf("DELETE: status = %d, want 409: %s", status, raw)
	}
	if !strings.Contains(lowerBody(raw), "confirmada") {
		t.Errorf("the message %q should say that the quotation is confirmed", raw)
	}
	if status, _ := api.do("GET", "/quotations/"+id, nil); status != http.StatusOK {
		t.Error("the refused delete removed the quotation")
	}
}

func TestX4_ANotFoundOrMalformedQuotationIsA404(t *testing.T) {
	api, _ := newReadyAPI(t)
	for _, id := range []string{"00000000-0000-0000-0000-000000000000", "not-an-id"} {
		if status, raw := api.do("DELETE", "/quotations/"+id, nil); status != http.StatusNotFound {
			t.Errorf("DELETE %s: status = %d, want 404: %s", id, status, raw)
		}
	}
}

func TestX5_ARecipeIsFreeToDeleteOnceItsQuotationIsDeleted(t *testing.T) {
	api, _ := newReadyAPI(t)
	_, cake, _ := bakery(t, api)
	id := quoteOf(t, api, cake)
	api.do("PUT", "/quotations/"+id+"/cancel", nil)
	if status, raw := api.do("DELETE", "/recipes/"+cake, nil); status != http.StatusConflict || !strings.Contains(lowerBody(raw), "cotizaci") {
		t.Fatalf("with a (cancelled) quotation the recipe must stay protected and say why: %d %s", status, raw)
	}
	if status, raw := api.do("DELETE", "/quotations/"+id, nil); status != http.StatusNoContent {
		t.Fatalf("delete quotation: %d %s", status, raw)
	}
	if status, raw := api.do("DELETE", "/recipes/"+cake, nil); status != http.StatusNoContent {
		t.Errorf("DELETE recipe after its quotation is gone: status = %d, want 204: %s", status, raw)
	}
}

func TestX6_DeletingOneQuotationLeavesTheOthers(t *testing.T) {
	api, _ := newReadyAPI(t)
	_, cake, pie := bakery(t, api)
	a := quoteOf(t, api, cake)
	b := quoteOf(t, api, pie)
	if status, raw := api.do("DELETE", "/quotations/"+a, nil); status != http.StatusNoContent {
		t.Fatalf("DELETE: %d %s", status, raw)
	}
	if status, _ := api.do("GET", "/quotations/"+b, nil); status != http.StatusOK {
		t.Error("another quotation was removed")
	}
}
