package app_test

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/PipeM113/bakery-manager/internal/config"
	"golang.org/x/crypto/bcrypt"
)

// Sprint 12, rule Q11 (revised): name, phone, Instagram, bank and holder are edited in the app;
// the RUT and the bank account come from the environment and are never sent to the screen.

func missingOf(t *testing.T, body map[string]any) []string {
	t.Helper()
	raw, _ := body["missing"].([]any)
	var out []string
	for _, r := range raw {
		out = append(out, r.(string))
	}
	slices.Sort(out)
	return out
}

func TestQ11_TheProfileStartsEmptyAndSaysWhatIsMissing(t *testing.T) {
	api, _ := newQuotesAPI(t, fullBusiness)
	body := getJSON(t, api, "/business")
	for _, f := range []string{"name", "phone", "instagram", "bank", "account_type", "holder", "email"} {
		if v, ok := body[f].(string); !ok || v != "" {
			t.Errorf("%s = %v, want an empty text", f, body[f])
		}
	}
	want := []string{"account_type", "bank", "email", "holder", "instagram", "name", "phone"}
	if got := missingOf(t, body); !slices.Equal(got, want) {
		t.Errorf("missing = %v, want %v", got, want)
	}

	bare, _ := newQuotesAPI(t, config.Business{})
	want = []string{"BIZ_ACCOUNT", "BIZ_RUT", "account_type", "bank", "email", "holder", "instagram", "name", "phone"}
	if got := missingOf(t, getJSON(t, bare, "/business")); !slices.Equal(got, want) {
		t.Errorf("missing without the environment = %v, want %v", got, want)
	}
}

func TestQ11_SaveAndReadTheProfile(t *testing.T) {
	api, _ := newQuotesAPI(t, fullBusiness)
	status, raw := api.do("PUT", "/business", map[string]any{
		"name": "  Panadería de Prueba ", "phone": "+56 9 1111 2222", "instagram": "prueba",
		"bank": "Banco Prueba", "account_type": "Cuenta Corriente", "holder": " Titular Prueba", "email": " Pagos@Prueba.test ",
	})
	if status != http.StatusOK {
		t.Fatalf("PUT: status = %d: %s", status, raw)
	}
	saved := decode(t, raw)
	want := map[string]string{"name": "Panadería de Prueba", "phone": "+56 9 1111 2222", "instagram": "@prueba", "bank": "Banco Prueba", "account_type": "Cuenta Corriente", "holder": "Titular Prueba", "email": "pagos@prueba.test"}
	for k, v := range want {
		if saved[k] != v {
			t.Errorf("saved %s = %q, want %q (trimmed, with the @ of Instagram, the e-mail in lower case)", k, saved[k], v)
		}
	}
	read := getJSON(t, api, "/business")
	for k, v := range want {
		if read[k] != v {
			t.Errorf("read %s = %q, want %q", k, read[k], v)
		}
	}
	if m := missingOf(t, read); len(m) != 0 {
		t.Errorf("missing = %v, want none", m)
	}
}

func TestQ11_BadProfilesAreRefusedAndNothingChanges(t *testing.T) {
	api, _ := newQuotesAPI(t, fullBusiness)
	fillProfile(t, api)
	with := func(k string, v any) map[string]any {
		m := map[string]any{}
		for key, val := range profile {
			m[key] = val
		}
		m[k] = v
		return m
	}
	cases := map[string]map[string]any{
		"empty name":       with("name", ""),
		"blank holder":     with("holder", "   "),
		"empty bank":       with("bank", ""),
		"name too long":    with("name", strings.Repeat("a", 101)),
		"letters as phone": with("phone", "llámame"),
		"short phone":      with("phone", "123"),
		"instagram space":  with("instagram", "@mi tienda"),
		"empty instagram":  with("instagram", ""),
	}
	for label, body := range cases {
		if status, raw := api.do("PUT", "/business", body); status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400: %s", label, status, raw)
		}
	}
	read := getJSON(t, api, "/business")
	if read["name"] != profile["name"] || read["holder"] != profile["holder"] {
		t.Errorf("a refused request changed the profile: %v", read)
	}
}

func TestQ11_TheEnvironmentSecretsNeverReachTheScreen(t *testing.T) {
	api, _ := newReadyAPI(t)
	_, raw := api.do("GET", "/business", nil)
	_, put := api.do("PUT", "/business", profile)
	for _, body := range [][]byte{raw, put} {
		for _, secret := range []string{fullBusiness.Account, fullBusiness.RUT} {
			if strings.Contains(string(body), secret) {
				t.Errorf("the answer carries %q: %s", secret, body)
			}
		}
	}
}

func TestQ11_RequiresLogin(t *testing.T) {
	api, _ := newQuotesAPI(t, fullBusiness)
	anon := &client{t: t, h: api.h}
	if status, _ := anon.do("GET", "/business", nil); status != http.StatusUnauthorized {
		t.Errorf("GET without a token: status = %d, want 401", status)
	}
	if status, _ := anon.do("PUT", "/business", profile); status != http.StatusUnauthorized {
		t.Errorf("PUT without a token: status = %d, want 401", status)
	}
}

func TestQ11_BothAccountsCanEditIt(t *testing.T) {
	a, admin := newQuotesAPI(t, fullBusiness)
	fillProfile(t, a)
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

	changed := map[string]any{}
	for k, v := range profile {
		changed[k] = v
	}
	changed["holder"] = "Otra Titular"
	if status, raw := b.do("PUT", "/business", changed); status != http.StatusOK {
		t.Fatalf("B editing: status = %d: %s", status, raw)
	}
	if got := getJSON(t, a, "/business")["holder"]; got != "Otra Titular" {
		t.Errorf("A reads holder = %v, want what B saved", got)
	}
}

// Q11e: the PDF is refused until everything is there, and says where to complete it.
func TestQ11_ThePDFNeedsTheProfileAndTheEnvironment(t *testing.T) {
	pdfStatus := func(api *client) (int, string) {
		_, cake, _ := bakery(t, api)
		id := createQuote(t, api, quote("Ana", 0, ql(cake, 1, "")))["id"].(string)
		status, raw := api.do("GET", "/quotations/"+id+"/pdf", nil)
		return status, string(raw)
	}

	noProfile, _ := newQuotesAPI(t, fullBusiness)
	status, body := pdfStatus(noProfile)
	if status != http.StatusServiceUnavailable || !strings.Contains(body, "Datos del negocio") {
		t.Errorf("without the profile: status = %d, body %.200s; want 503 pointing to 'Datos del negocio'", status, body)
	}

	noEnv, _ := newQuotesAPI(t, config.Business{})
	fillProfile(t, noEnv)
	status, body = pdfStatus(noEnv)
	if status != http.StatusServiceUnavailable || !strings.Contains(body, "BIZ_ACCOUNT") || !strings.Contains(body, "BIZ_RUT") {
		t.Errorf("without the environment: status = %d, body %.200s; want 503 naming BIZ_ACCOUNT and BIZ_RUT", status, body)
	}

	ready, _ := newReadyAPI(t)
	if status, body = pdfStatus(ready); status != http.StatusOK {
		t.Errorf("with everything: status = %d: %.200s", status, body)
	}
}

// Q11d: the PDF prints the data as they are when it is downloaded.
func TestQ11_ThePDFUsesTheCurrentProfile(t *testing.T) {
	api, _ := newReadyAPI(t)
	_, cake, _ := bakery(t, api)
	id := createQuote(t, api, quote("Ana", 0, ql(cake, 1, "")))["id"].(string)

	_, before := api.do("GET", "/quotations/"+id+"/pdf", nil)
	if !strings.Contains(pdfTextOf(t, before), "Titular Prueba") {
		t.Fatal("the PDF does not carry the holder")
	}
	changed := map[string]any{}
	for k, v := range profile {
		changed[k] = v
	}
	changed["holder"] = "Titular Nuevo"
	changed["bank"] = "Banco Nuevo"
	changed["email"] = "nuevo@prueba.test"
	if status, raw := api.do("PUT", "/business", changed); status != http.StatusOK {
		t.Fatalf("PUT: %d %s", status, raw)
	}
	_, after := api.do("GET", "/quotations/"+id+"/pdf", nil)
	text := pdfTextOf(t, after)
	if !strings.Contains(text, "Titular Nuevo") || !strings.Contains(text, "Banco Nuevo") || !strings.Contains(text, "nuevo@prueba.test") || strings.Contains(text, "Titular Prueba") {
		t.Errorf("the old PDF did not pick up the new data:\n%s", text)
	}
}
