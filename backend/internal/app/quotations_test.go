package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/PipeM113/bakery-manager/internal/app"
	"github.com/PipeM113/bakery-manager/internal/config"
	"github.com/PipeM113/bakery-manager/internal/shared/kernel"
	"github.com/PipeM113/bakery-manager/internal/testsupport/pgtest"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// Sprint 12: a quotation has several lines, one delivery and a PDF that is made from what was saved.
// Prices of the fixtures: flour 1000 gr at 1000 (1 per gr); "Torta" yields 9 portions from 1000 gr
// (cost 1000) and "Pie" yields 1 from 500 gr (cost 500). With indirect 0, labor 20% and markup 25%
// a Torta costs 1500 and a Pie 1000 (rounded up to 500).

// The RUT and the account come from the environment; the rest is edited in the app (see business_test.go).
var fullBusiness = config.Business{Account: "000-111222-3", RUT: "11.111.111-1"}

var profile = map[string]any{
	"name": "Panadería de Prueba", "phone": "+56 9 1111 2222", "instagram": "@prueba",
	"bank": "Banco Prueba", "account_type": "Cuenta Corriente", "holder": "Titular Prueba", "email": "pagos@prueba.test",
}

func fillProfile(t *testing.T, api *client) {
	t.Helper()
	if status, raw := api.do("PUT", "/business", profile); status != http.StatusOK {
		t.Fatalf("PUT /business: status = %d: %s", status, raw)
	}
}

// newReadyAPI has the environment and the profile filled: its PDFs can be made.
func newReadyAPI(t *testing.T) (*client, *pgxpool.Pool) {
	t.Helper()
	api, admin := newQuotesAPI(t, fullBusiness)
	fillProfile(t, api)
	return api, admin
}

func newQuotesAPI(t *testing.T, biz config.Business) (*client, *pgxpool.Pool) {
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
	pool, err := kernel.Open(context.Background(), admin.Config().ConnConfig.ConnString(), 4)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	api := &client{t: t, h: app.New(config.Config{Port: "8080", DatabaseURL: "unused", JWTSecret: goodSecret, DBMaxConns: 4, Business: biz}, pool, nil)}
	status, raw := api.do("POST", "/auth/login", map[string]string{"email": testEmail, "password": testPassword})
	if status != http.StatusOK {
		t.Fatalf("login: status = %d: %s", status, raw)
	}
	api.token, _ = decode(t, raw)["token"].(string)
	return api, admin
}

// bakery makes the flour, the Torta and the Pie.
func bakery(t *testing.T, api *client) (flour, cake, pie string) {
	t.Helper()
	flour = createIngredient(t, api, map[string]any{"name": "Harina", "default_unit": "gr", "package_size": 1000, "package_price": 1000})
	cake = createRecipe(t, api, map[string]any{
		"name": "Torta", "description": "Torta de prueba", "yield": 9, "yield_unit": "porciones",
		"ingredients": []map[string]any{{"ingredient_id": flour, "quantity": 1000, "unit": "gr"}},
	})
	pie = createRecipe(t, api, map[string]any{
		"name": "Pie", "description": "Pie de prueba", "yield": 1, "yield_unit": "porciones",
		"ingredients": []map[string]any{{"ingredient_id": flour, "quantity": 500, "unit": "gr"}},
	})
	return
}

// ql is a line of a quotation with the usual percentages (indirect 0, labor 20%, markup 25%).
func ql(recipe string, quantity int, soldAs string) map[string]any {
	l := map[string]any{"recipe_id": recipe, "quantity": quantity,
		"indirect_cost_pct": 0, "labor_cost_pct": 0.2, "margin_pct": 0.25}
	if soldAs != "" {
		l["sold_as"] = soldAs
	}
	return l
}

func quote(client string, delivery int, lines ...map[string]any) map[string]any {
	return map[string]any{"client_name": client, "delivery_cost": delivery, "lines": lines}
}

func createQuote(t *testing.T, api *client, body map[string]any) map[string]any {
	t.Helper()
	status, raw := api.do("POST", "/quotations", body)
	if status != http.StatusCreated {
		t.Fatalf("POST /quotations: status = %d: %s", status, raw)
	}
	return decode(t, raw)
}

func quoteLines(t *testing.T, q map[string]any) []map[string]any {
	t.Helper()
	raw, _ := q["lines"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, r := range raw {
		out = append(out, r.(map[string]any))
	}
	return out
}

func quotationsCount(t *testing.T, admin *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := admin.QueryRow(context.Background(), `select count(*) from quotations`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func decodeList(t *testing.T, raw []byte) []map[string]any {
	t.Helper()
	var out []map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("not a JSON list: %v\n%.300s", err, raw)
	}
	return out
}

func newRecorder(api *client, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+api.token)
	rec := httptest.NewRecorder()
	api.h.ServeHTTP(rec, req)
	return rec
}

func whole(v float64) bool { return v == math.Trunc(v) }

func pdfTextOf(t *testing.T, pdf []byte) string {
	t.Helper()
	path, err := exec.LookPath("pdftotext")
	if err != nil {
		t.Skip("pdftotext (poppler-utils) is not installed")
	}
	cmd := exec.Command(path, "-layout", "-", "-")
	cmd.Stdin = bytes.NewReader(pdf)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// Q1: several lines, from 1 to 30.
func TestQ1_AQuotationWithSeveralLines(t *testing.T) {
	api, _ := newReadyAPI(t)
	_, cake, pie := bakery(t, api)

	q := createQuote(t, api, quote("Ana Pérez", 500, ql(cake, 2, ""), ql(pie, 3, "")))
	if id, _ := q["id"].(string); id == "" {
		t.Errorf("no id: %v", q)
	}
	if q["client_name"] != "Ana Pérez" || q["status"] != "pending" {
		t.Errorf("client_name/status = %v/%v", q["client_name"], q["status"])
	}
	lines := quoteLines(t, q)
	if len(lines) != 2 {
		t.Fatalf("lines = %d, want 2", len(lines))
	}
	if lines[0]["recipe_id"] != cake || lines[1]["recipe_id"] != pie {
		t.Errorf("the lines do not keep their order: %v", lines)
	}
	if lines[0]["recipe_name"] != "Torta" || lines[1]["recipe_name"] != "Pie" {
		t.Errorf("recipe names = %v, %v", lines[0]["recipe_name"], lines[1]["recipe_name"])
	}
	if lines[0]["sold_as"] != "recipe" {
		t.Errorf("sold_as defaults to %v, want recipe", lines[0]["sold_as"])
	}
}

func TestQ1_BetweenOneAndThirtyLines(t *testing.T) {
	api, admin := newReadyAPI(t)
	_, _, pie := bakery(t, api)

	many := func(n int) []map[string]any {
		var ls []map[string]any
		for i := 0; i < n; i++ {
			ls = append(ls, ql(pie, 1, ""))
		}
		return ls
	}
	if status, raw := api.do("POST", "/quotations", quote("Ana", 0)); status != http.StatusBadRequest {
		t.Errorf("0 lines: status = %d, want 400: %s", status, raw)
	}
	if status, raw := api.do("POST", "/quotations", quote("Ana", 0, many(31)...)); status != http.StatusBadRequest {
		t.Errorf("31 lines: status = %d, want 400: %s", status, raw)
	}
	if n := quotationsCount(t, admin); n != 0 {
		t.Fatalf("a refused quotation was saved (%d)", n)
	}
	q := createQuote(t, api, quote("Ana", 0, many(30)...))
	if got := number(t, q, "total"); got != 30000 {
		t.Errorf("30 pies: total = %v, want 30000", got)
	}
}

func TestQ1_BadQuotationsAreRefusedWholeAndSaveNothing(t *testing.T) {
	api, admin := newReadyAPI(t)
	_, cake, pie := bakery(t, api)
	good := ql(pie, 1, "")
	withQty := func(q any) map[string]any { l := ql(cake, 1, ""); l["quantity"] = q; return l }
	withField := func(k string, v any) map[string]any { l := ql(cake, 1, ""); l[k] = v; return l }
	cases := map[string]map[string]any{
		"no client":         quote("", 0, good),
		"blank client":      quote("   ", 0, good),
		"client too long":   quote(strings.Repeat("a", 101), 0, good),
		"zero quantity":     quote("Ana", 0, good, withQty(0)),
		"negative quantity": quote("Ana", 0, good, withQty(-2)),
		"decimal quantity":  quote("Ana", 0, good, withQty(1.5)),
		"huge quantity":     quote("Ana", 0, good, withQty(100001)),
		"unknown recipe":    quote("Ana", 0, good, ql("00000000-0000-0000-0000-000000000000", 1, "")),
		"bad recipe id":     quote("Ana", 0, good, ql("not-an-id", 1, "")),
		"bad sold_as":       quote("Ana", 0, good, withField("sold_as", "slice")),
		"negative delivery": quote("Ana", -500, good),
		"markup too big":    quote("Ana", 0, good, withField("margin_pct", 50)),
		"negative extra":    quote("Ana", 0, good, withField("extra_charge", -1)),
	}
	for label, body := range cases {
		if status, raw := api.do("POST", "/quotations", body); status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400: %s", label, status, raw)
		}
	}
	if n := quotationsCount(t, admin); n != 0 {
		t.Errorf("%d quotation(s) saved from refused requests, want 0 (all or nothing)", n)
	}
}

// Q2: the price of each line.
func TestQ2_PriceOfALine(t *testing.T) {
	api, _ := newReadyAPI(t)
	_, cake, _ := bakery(t, api)

	check := func(label string, l map[string]any, wantUnit, wantTotal float64) {
		t.Helper()
		q := createQuote(t, api, quote("Ana", 0, l))
		got := quoteLines(t, q)[0]
		if number(t, got, "unit_price") != wantUnit || number(t, got, "line_total") != wantTotal {
			t.Errorf("%s: unit %v, total %v; want %v and %v", label, got["unit_price"], got["line_total"], wantUnit, wantTotal)
		}
		for _, k := range []string{"unit_price", "line_total"} {
			if !whole(number(t, got, k)) {
				t.Errorf("%s: %s is not whole pesos", label, k)
			}
		}
	}
	check("a whole recipe", ql(cake, 1, ""), 1500, 1500)
	check("two recipes", ql(cake, 2, "recipe"), 1500, 3000)
	// 4 portions of 9: ingredients 444.44, labor 20% -> 533.33, markup 25% -> 666.67 -> 1000
	check("four portions", ql(cake, 4, "portion"), 250, 1000)

	over := ql(cake, 1, "")
	over["margin_pct"] = 0.5 // 1200 x 1.5 = 1800 -> 2000
	check("another markup", over, 2000, 2000)

	zero := ql(cake, 1, "")
	zero["margin_pct"], zero["labor_cost_pct"] = 0, 0 // cost 1000, markup 0%
	check("an explicit 0%", zero, 1000, 1000)

	extra := ql(cake, 1, "")
	extra["extra_charge"] = 300 // 1500 + 300 = 1800 -> 2000, the extra is not shown apart
	check("the hidden extra charge", extra, 2000, 2000)
}

func TestQ2_WithoutPercentagesTheRecipeOnesAreUsedAndTheRecipeIsNotChanged(t *testing.T) {
	api, _ := newReadyAPI(t)
	_, cake, _ := bakery(t, api)
	setCosts(t, api, cake, 0, 0.2, 0.25)

	q := createQuote(t, api, quote("Ana", 0, map[string]any{"recipe_id": cake, "quantity": 1}))
	l := quoteLines(t, q)[0]
	if number(t, l, "unit_price") != 1500 {
		t.Errorf("unit_price = %v, want 1500 from the recipe's own percentages", l["unit_price"])
	}
	if number(t, l, "margin_pct") != 0.25 || number(t, l, "labor_cost_pct") != 0.2 || number(t, l, "indirect_cost_pct") != 0 {
		t.Errorf("the line does not show the percentages it used: %v", l)
	}

	over := ql(cake, 1, "")
	over["margin_pct"] = 0.5
	createQuote(t, api, quote("Ana", 0, over))
	if got := number(t, getJSON(t, api, "/recipes/"+cake+"/cost"), "margin_pct"); got != 0.25 {
		t.Errorf("the recipe's markup became %v after adjusting a line, want 0.25", got)
	}
}

// Q3: one delivery per quotation.
func TestQ3_DeliveryIsAddedOnce(t *testing.T) {
	api, _ := newReadyAPI(t)
	_, cake, pie := bakery(t, api)

	q := createQuote(t, api, quote("Ana", 1000, ql(cake, 2, ""), ql(pie, 3, "")))
	if number(t, q, "delivery_cost") != 1000 {
		t.Errorf("delivery_cost = %v, want 1000", q["delivery_cost"])
	}
	var sum float64
	for _, l := range quoteLines(t, q) {
		sum += number(t, l, "line_total")
	}
	if sum != 6000 {
		t.Errorf("the lines add up to %v, want 6000 (3000 + 3000): the delivery is not inside them", sum)
	}
	if got := number(t, q, "total"); got != 7000 || !whole(got) {
		t.Errorf("total = %v, want 7000", got)
	}

	noDelivery := createQuote(t, api, quote("Ana", 0, ql(cake, 1, "")))
	if number(t, noDelivery, "total") != 1500 || number(t, noDelivery, "delivery_cost") != 0 {
		t.Errorf("without delivery: %v", noDelivery)
	}
}

// Q4: the quotation is saved by creating it; the PDF is a separate step that needs it saved.
func TestQ4_CreatingSavesAndAnswersJSONNotAPDF(t *testing.T) {
	api, admin := newQuotesAPI(t, config.Business{}) // no business data: creating still works
	_, cake, _ := bakery(t, api)

	status, raw := api.do("POST", "/quotations", quote("Ana", 0, ql(cake, 1, "")))
	if status != http.StatusCreated || bytes.HasPrefix(raw, []byte("%PDF")) {
		t.Fatalf("status = %d, body starts %.10q: want 201 and JSON", status, raw)
	}
	if n := quotationsCount(t, admin); n != 1 {
		t.Errorf("quotations saved = %d, want 1", n)
	}
	status, raw = api.do("GET", "/quotations/00000000-0000-0000-0000-000000000000/pdf", nil)
	if status != http.StatusNotFound || bytes.HasPrefix(raw, []byte("%PDF")) {
		t.Errorf("the PDF of a quotation that was not saved: status = %d, want 404", status)
	}
}

// Q5: the saved quotation keeps its prices and names.
func TestQ5_TheSavedQuotationDoesNotChangeWhenCostsDo(t *testing.T) {
	api, admin := newReadyAPI(t)
	_, cake, _ := bakery(t, api)
	q := createQuote(t, api, quote("Ana", 500, ql(cake, 2, "")))
	id := q["id"].(string)

	if _, err := admin.Exec(context.Background(), `update ingredients set package_price = 10000, price_per_unit = 10`); err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(context.Background(), `update recipes set name = 'Otra cosa', photo_url = 'https://example.test/otra.jpg'`); err != nil {
		t.Fatal(err)
	}

	got := getJSON(t, api, "/quotations/"+id)
	l := quoteLines(t, got)[0]
	if number(t, got, "total") != 3500 || number(t, l, "unit_price") != 1500 || number(t, l, "line_total") != 3000 {
		t.Errorf("the prices changed: total %v, unit %v, line %v", got["total"], l["unit_price"], l["line_total"])
	}
	if l["recipe_name"] != "Torta" {
		t.Errorf("recipe_name = %v, want the name it had when it was quoted (Torta)", l["recipe_name"])
	}
}

func TestQ5_ThePDFKeepsTheSnapshotToo(t *testing.T) {
	api, admin := newReadyAPI(t)
	_, cake, _ := bakery(t, api)
	id := createQuote(t, api, quote("Ana", 0, ql(cake, 1, "")))["id"].(string)
	if _, err := admin.Exec(context.Background(), `update recipes set name = 'Otra cosa'`); err != nil {
		t.Fatal(err)
	}
	status, pdf := api.do("GET", "/quotations/"+id+"/pdf", nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	text := pdfTextOf(t, pdf)
	if !strings.Contains(text, "Torta") || strings.Contains(text, "Otra cosa") {
		t.Errorf("the PDF does not carry the quoted name:\n%s", text)
	}
}

// Q6: the PDF can be downloaded again, always the same.
func TestQ6_DownloadThePDFOfAnyQuotationAgain(t *testing.T) {
	api, _ := newReadyAPI(t)
	_, cake, pie := bakery(t, api)
	q := createQuote(t, api, quote("Ana Pérez", 500, ql(cake, 2, ""), ql(pie, 1, "")))
	id := q["id"].(string)

	download := func() []byte {
		t.Helper()
		status, raw := api.do("GET", "/quotations/"+id+"/pdf", nil)
		if status != http.StatusOK {
			t.Fatalf("status = %d: %.200s", status, raw)
		}
		return raw
	}
	first := download()
	if !bytes.HasPrefix(first, []byte("%PDF")) {
		t.Fatal("not a PDF")
	}
	second := download()
	if pdfTextOf(t, first) != pdfTextOf(t, second) {
		t.Error("downloading it twice gives different content")
	}
	text := pdfTextOf(t, first)
	for _, want := range []string{"Ana Pérez", "Torta", "Pie", "$3.000", "$4.000"} {
		if !strings.Contains(text, want) {
			t.Errorf("the PDF does not contain %q", want)
		}
	}
}

func TestQ6_FileNameAndHeaders(t *testing.T) {
	api, _ := newReadyAPI(t)
	_, cake, _ := bakery(t, api)
	santiago, _ := time.LoadLocation("America/Santiago")

	plain := createQuote(t, api, quote("Ana Perez", 0, ql(cake, 1, "")))
	rec := newRecorder(api, "GET", "/quotations/"+plain["id"].(string)+"/pdf")
	if got := rec.Header().Get("Content-Type"); got != "application/pdf" {
		t.Errorf("Content-Type = %q", got)
	}
	created, err := time.Parse(time.RFC3339Nano, plain["created_at"].(string))
	if err != nil {
		t.Fatal(err)
	}
	want := `attachment; filename="cotizacion_Ana_Perez_` + created.In(santiago).Format("2006-01-02") + `.pdf"`
	if got := rec.Header().Get("Content-Disposition"); got != want {
		t.Errorf("Content-Disposition = %q, want %q (the date of the quotation, in Santiago)", got, want)
	}

	// Whatever the client is called, the header stays plain ASCII and safe.
	odd := createQuote(t, api, quote(`Ñandú "El Rey" / <b>`, 0, ql(cake, 1, "")))
	rec = newRecorder(api, "GET", "/quotations/"+odd["id"].(string)+"/pdf")
	if got := rec.Header().Get("Content-Disposition"); !regexp.MustCompile(`^attachment; filename="cotizacion_[A-Za-z0-9_.-]+_\d{4}-\d{2}-\d{2}\.pdf"$`).MatchString(got) {
		t.Errorf("Content-Disposition = %q, want a plain ASCII file name", got)
	}
}

func TestQ6_UnknownOrMalformedIdsAre404(t *testing.T) {
	api, _ := newReadyAPI(t)
	for _, path := range []string{"/quotations/not-an-id", "/quotations/not-an-id/pdf", "/quotations/00000000-0000-0000-0000-000000000000"} {
		if status, raw := api.do("GET", path, nil); status != http.StatusNotFound {
			t.Errorf("GET %s: status = %d, want 404: %s", path, status, raw)
		}
	}
}

// Q7 (API side): the list carries the lines.
func TestQ7_TheListCarriesTheLinesAndFilters(t *testing.T) {
	api, _ := newReadyAPI(t)
	_, cake, pie := bakery(t, api)
	a := createQuote(t, api, quote("Ana", 0, ql(cake, 1, ""), ql(pie, 1, "")))
	createQuote(t, api, quote("Bea", 500, ql(pie, 2, "")))
	if status, raw := api.do("PUT", "/quotations/"+a["id"].(string)+"/confirm", nil); status != http.StatusNoContent {
		t.Fatalf("confirm: status = %d: %s", status, raw)
	}

	status, raw := api.do("GET", "/quotations", nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	list := decodeList(t, raw)
	if len(list) != 2 {
		t.Fatalf("quotations = %d, want 2", len(list))
	}
	byClient := map[string]map[string]any{}
	for _, q := range list {
		byClient[q["client_name"].(string)] = q
	}
	if n := len(quoteLines(t, byClient["Ana"])); n != 2 {
		t.Errorf("Ana's quotation lists %d lines, want 2", n)
	}
	if got := number(t, byClient["Bea"], "total"); got != 2500 {
		t.Errorf("Bea's total = %v, want 2500 (2 pies and the delivery)", got)
	}
	status, raw = api.do("GET", "/quotations?status=confirmed", nil)
	if l := decodeList(t, raw); status != http.StatusOK || len(l) != 1 || l[0]["client_name"] != "Ana" {
		t.Errorf("confirmed = %v (%d)", l, status)
	}
}

// Q11f: no placeholder data in the PDF.
func TestQ11_TheOldPlaceholdersAreGone(t *testing.T) {
	// The PDF used to print made-up bank data when nothing was configured.
	api, _ := newReadyAPI(t)
	_, cake, _ := bakery(t, api)
	id := createQuote(t, api, quote("Ana", 0, ql(cake, 1, "")))["id"].(string)
	_, pdf := api.do("GET", "/quotations/"+id+"/pdf", nil)
	text := pdfTextOf(t, pdf)
	for _, fake := range []string{"123-456789-0", "12.345.678-9", "1234 5678", "Ángeles Martínez", "angelesscoffee"} {
		if strings.Contains(text, fake) {
			t.Errorf("the PDF still prints the placeholder %q", fake)
		}
	}
	for _, real := range []string{fullBusiness.Account, fullBusiness.RUT, profile["holder"].(string)} {
		if !strings.Contains(text, real) {
			t.Errorf("the PDF does not print %q", real)
		}
	}
}

// Q12: both accounts do everything.
func TestQ12_BothAccountsCreateConfirmCancelAndDownload(t *testing.T) {
	a, admin := newReadyAPI(t)
	_, cake, _ := bakery(t, a)
	hash, _ := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if _, err := admin.Exec(context.Background(),
		`insert into users (name, email, password, role) values ('Mamá', 'mama@example.test', $1, 'owner')`, string(hash)); err != nil {
		t.Fatal(err)
	}
	b := &client{t: t, h: a.h}
	status, raw := b.do("POST", "/auth/login", map[string]string{"email": "mama@example.test", "password": testPassword})
	if status != http.StatusOK {
		t.Fatalf("login: %d", status)
	}
	b.token, _ = decode(t, raw)["token"].(string)

	fromA := createQuote(t, a, quote("De A", 0, ql(cake, 1, "")))
	fromB := createQuote(t, b, quote("De B", 0, ql(cake, 1, "")))
	if fromA["created_by"] == nil || fromB["created_by"] == nil || fromA["created_by"] == fromB["created_by"] {
		t.Errorf("created_by: %v and %v, want two different creators", fromA["created_by"], fromB["created_by"])
	}
	if _, raw := a.do("GET", "/quotations", nil); len(decodeList(t, raw)) != 2 {
		t.Error("A does not see both quotations")
	}
	if status, _ := b.do("GET", "/quotations/"+fromA["id"].(string)+"/pdf", nil); status != http.StatusOK {
		t.Errorf("B downloading A's PDF: status = %d", status)
	}
	if status, _ := b.do("PUT", "/quotations/"+fromA["id"].(string)+"/confirm", nil); status != http.StatusNoContent {
		t.Errorf("B confirming A's quotation: status = %d", status)
	}
	if status, _ := a.do("PUT", "/quotations/"+fromB["id"].(string)+"/cancel", nil); status != http.StatusNoContent {
		t.Errorf("A cancelling B's quotation: status = %d", status)
	}
	if got := getJSON(t, b, "/quotations/"+fromA["id"].(string))["status"]; got != "confirmed" {
		t.Errorf("status = %v, want confirmed", got)
	}
}
