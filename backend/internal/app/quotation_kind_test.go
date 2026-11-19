package app_test

import (
	"net/http"
	"strings"
	"testing"
)

// Sprint 18: a quotation is for a person (default) or for a company. A company one carries the conditions of
// the order (delivery date, way of payment, days it is valid); the PDF it makes is told in
// quotations/service/pdf_kind_test.go.

func kindQuote(client, kind string, extra map[string]any, lines ...map[string]any) map[string]any {
	body := quote(client, 0, lines...)
	if kind != "" {
		body["kind"] = kind
	}
	for k, v := range extra {
		body[k] = v
	}
	return body
}

func TestK1_ANewQuotationIsForAPersonWhenNothingIsSaid(t *testing.T) {
	api, _ := newReadyAPI(t)
	_, cake, _ := bakery(t, api)
	q := createQuote(t, api, quote("Ana", 0, ql(cake, 1, "")))
	if q["kind"] != "person" {
		t.Errorf("kind = %v, want person", q["kind"])
	}
	for _, f := range []string{"delivery_date", "payment_terms", "valid_days"} {
		if v := q[f]; v != nil && v != "" {
			t.Errorf("a quotation for a person carries %s = %v", f, v)
		}
	}
}

func TestK2_ACompanyQuotationSavesAndReturnsItsConditions(t *testing.T) {
	api, _ := newReadyAPI(t)
	_, cake, _ := bakery(t, api)
	q := createQuote(t, api, kindQuote("Banco del Sur Ltda.", "company", map[string]any{
		"delivery_date": "2099-12-24", "payment_terms": "Transferencia a 30 días", "valid_days": 10,
	}, ql(cake, 3, "")))
	got := getJSON(t, api, "/quotations/"+q["id"].(string))
	if got["kind"] != "company" || got["delivery_date"] != "2099-12-24" || got["payment_terms"] != "Transferencia a 30 días" || got["valid_days"] != float64(10) {
		t.Errorf("the conditions were not saved as given: %v", got)
	}
}

func TestK3_ACompanyQuotationWithoutConditionsIsValidForSevenDays(t *testing.T) {
	api, _ := newReadyAPI(t)
	_, cake, _ := bakery(t, api)
	q := createQuote(t, api, kindQuote("Banco del Sur Ltda.", "company", nil, ql(cake, 1, "")))
	if q["valid_days"] != float64(7) {
		t.Errorf("valid_days = %v, want 7 by default", q["valid_days"])
	}
	if v := q["delivery_date"]; v != nil && v != "" {
		t.Errorf("delivery_date = %v, want none", v)
	}
}

func TestK4_AnUnknownKindOrConditionsOnAPersonQuotationAreRefused(t *testing.T) {
	api, admin := newReadyAPI(t)
	_, cake, _ := bakery(t, api)
	cases := map[string]map[string]any{
		"unknown kind":      kindQuote("Ana", "empresa", nil, ql(cake, 1, "")),
		"date on a person":  kindQuote("Ana", "person", map[string]any{"delivery_date": "2099-12-24"}, ql(cake, 1, "")),
		"terms on a person": kindQuote("Ana", "", map[string]any{"payment_terms": "Contado"}, ql(cake, 1, "")),
		"days on a person":  kindQuote("Ana", "person", map[string]any{"valid_days": 5}, ql(cake, 1, "")),
	}
	for name, body := range cases {
		if status, raw := api.do("POST", "/quotations", body); status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400: %s", name, status, raw)
		}
	}
	if n := quotationsCount(t, admin); n != 0 {
		t.Errorf("%d quotation(s) saved from refused requests, want 0", n)
	}
}

func TestK5_TheConditionsOfACompanyAreChecked(t *testing.T) {
	api, admin := newReadyAPI(t)
	_, cake, _ := bakery(t, api)
	cases := map[string]map[string]any{
		"0 days":            {"valid_days": 0},
		"91 days":           {"valid_days": 91},
		"days not a number": {"valid_days": "diez"},
		"date malformed":    {"delivery_date": "24-12-2099"},
		"date impossible":   {"delivery_date": "2099-02-30"},
		"date in the past":  {"delivery_date": "2000-01-01"},
		"terms too long":    {"payment_terms": strings.Repeat("x", 121)},
	}
	for name, extra := range cases {
		if status, raw := api.do("POST", "/quotations", kindQuote("Empresa", "company", extra, ql(cake, 1, ""))); status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400: %s", name, status, raw)
		}
	}
	if n := quotationsCount(t, admin); n != 0 {
		t.Errorf("%d quotation(s) saved from refused requests, want 0", n)
	}
}

func TestK6_TheListCarriesTheKind(t *testing.T) {
	api, _ := newReadyAPI(t)
	_, cake, pie := bakery(t, api)
	createQuote(t, api, quote("Ana", 0, ql(cake, 1, "")))
	createQuote(t, api, kindQuote("Empresa SpA", "company", nil, ql(pie, 1, "")))
	_, raw := api.do("GET", "/quotations", nil)
	kinds := map[string]any{}
	for _, q := range decodeList(t, raw) {
		kinds[q["client_name"].(string)] = q["kind"]
	}
	if kinds["Ana"] != "person" || kinds["Empresa SpA"] != "company" {
		t.Errorf("kinds in the list = %v", kinds)
	}
}

func TestK7_ThePDFFollowsTheKindThatWasSaved(t *testing.T) {
	api, _ := newReadyAPI(t)
	_, cake, pie := bakery(t, api)
	person := createQuote(t, api, quote("Ana", 500, ql(cake, 2, ""), ql(pie, 1, "")))["id"].(string)
	company := createQuote(t, api, kindQuote("Empresa SpA", "company", map[string]any{"payment_terms": "Transferencia a 30 días"}, ql(cake, 2, ""), ql(pie, 1, "")))["id"].(string)
	text := func(id string) string {
		status, pdf := api.do("GET", "/quotations/"+id+"/pdf", nil)
		if status != http.StatusOK {
			t.Fatalf("pdf status = %d", status)
		}
		return pdfTextOf(t, pdf)
	}
	p, c := text(person), text(company)
	if strings.Contains(p, "PRECIO UNIT") || strings.Contains(p, "SUBTOTAL") {
		t.Errorf("the PDF of a person shows the breakdown:\n%s", p)
	}
	if !strings.Contains(c, "PRECIO UNIT") || !strings.Contains(c, "Transferencia a 30 días") {
		t.Errorf("the PDF of a company misses the detail or its conditions:\n%s", c)
	}
}
