package app_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/PipeM113/bakery-manager/internal/app"
	"github.com/PipeM113/bakery-manager/internal/config"
	"github.com/PipeM113/bakery-manager/internal/shared/kernel"
	"github.com/PipeM113/bakery-manager/internal/testsupport/pgtest"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// Sprint 13: C3 (size of JSON bodies), C4 (no free text in the logs), C5 (queries that do not grow
// with the data) and C6 (photos that nobody uses any more are removed from the bucket).

// --- C3 ---

func rawJSON(t *testing.T, h http.Handler, token, method, path string, body []byte) int {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

func TestC3_JSONBodiesOverOneMiBAreRefusedWith413(t *testing.T) {
	api, _ := newQuotesAPI(t, fullBusiness)
	big := []byte(`{"description":"` + strings.Repeat("a", 2<<20) + `"}`)
	routes := []struct{ method, path string }{
		{"POST", "/expenses"}, {"POST", "/ingredients"}, {"POST", "/recipes"}, {"POST", "/sales"}, {"POST", "/orders"},
		{"POST", "/quotations"}, {"PUT", "/business"}, {"POST", "/fixed-costs"}, {"PUT", "/auth/password"},
	}
	for _, r := range routes {
		if got := rawJSON(t, api.h, api.token, r.method, r.path, big); got != http.StatusRequestEntityTooLarge {
			t.Errorf("%s %s with a 2 MiB body: status = %d, want 413", r.method, r.path, got)
		}
	}
	if got := rawJSON(t, api.h, "", "POST", "/auth/login", big); got != http.StatusRequestEntityTooLarge {
		t.Errorf("POST /auth/login with a 2 MiB body: status = %d, want 413 (it is open to anyone)", got)
	}
}

func TestC3_ABodyUnderTheLimitIsNotRefusedForItsSize(t *testing.T) {
	api, _ := newQuotesAPI(t, fullBusiness)
	body := []byte(`{"description":"` + strings.Repeat("a", 900<<10) + `"}`)
	if got := rawJSON(t, api.h, api.token, "POST", "/expenses", body); got == http.StatusRequestEntityTooLarge {
		t.Error("900 KiB was refused as too large")
	}
}

func TestC3_PhotosKeepTheirOwnLimit(t *testing.T) {
	up := &fakeUploader{url: "https://example.test/p.jpg"}
	api, recipe := newPhotoAPI(t, up)
	jpg := append([]byte("\xff\xd8\xff\xe0"), bytes.Repeat([]byte{1}, 2<<20)...) // 2 MiB: over the JSON limit, under the photo limit
	if status, raw := api.upload(recipe, "photo", "a.jpg", "image/jpeg", jpg, api.token); status != http.StatusOK {
		t.Errorf("a 2 MiB photo: status = %d, want 200: %s", status, raw)
	}
}

// --- C4 ---

type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}
func (s *syncBuffer) String() string { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

func TestC4_LogsNeverCarryWhatPeopleType(t *testing.T) {
	logs := &syncBuffer{}
	log.SetOutput(logs)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	api := newAPIAt(t, noonMarch31)
	recipe, _ := stockedFlourRecipe(t, api.client)
	gas := categoryID(t, api.client, "gas")
	markers := map[string]string{
		"expense description": "MARCA-DESCRIPCION-7731",
		"sale notes":          "MARCA-NOTA-7732",
		"order customer":      "MARCA-CLIENTE-7733",
		"quotation client":    "MARCA-COTIZACION-7734",
		"login e-mail":        "marca-correo-7735@example.test",
	}

	// Expenses: the valid path, the update, the refused path and the delete.
	status, raw := api.do("POST", "/expenses", map[string]any{"description": markers["expense description"], "amount": 5000, "category_id": gas, "expense_date": "2026-03-31"})
	if status != http.StatusCreated {
		t.Fatalf("expense: %d %s", status, raw)
	}
	id := decode(t, raw)["id"].(string)
	api.do("PUT", "/expenses/"+id, map[string]any{"description": markers["expense description"] + "-2", "amount": 6000, "category_id": gas, "expense_date": "2026-03-31"})
	api.do("POST", "/expenses", map[string]any{"description": markers["expense description"] + "-3", "amount": -1, "category_id": gas, "expense_date": "2026-03-31"})
	api.do("DELETE", "/expenses/"+id, nil)

	api.do("POST", "/sales", map[string]any{"recipe_id": recipe, "quantity": 1, "unit_price": 1000, "notes": markers["sale notes"]})
	api.do("POST", "/sales", map[string]any{"recipe_id": "not-an-id", "quantity": 1, "unit_price": 1000, "notes": markers["sale notes"] + "-2"})
	api.do("POST", "/orders", map[string]any{"customer_name": markers["order customer"], "notes": markers["order customer"] + "-n",
		"lines": []line{ln(recipe, 1, 1000, "")}})
	api.do("POST", "/orders", map[string]any{"customer_name": markers["order customer"] + "-2", "lines": []line{}})
	api.do("POST", "/quotations", quote(markers["quotation client"], 0, map[string]any{"recipe_id": recipe, "quantity": 1}))
	api.do("POST", "/quotations", quote(markers["quotation client"]+"-2", 0, map[string]any{"recipe_id": "not-an-id", "quantity": 1}))
	loginAs(api.h, markers["login e-mail"], "wrong-password")

	out := logs.String()
	for what, marker := range markers {
		if strings.Contains(out, marker) {
			t.Errorf("the logs carry the %s (%q)", what, marker)
		}
	}
}

// --- C5 ---

func queriesOf(api *timeAPI, f func()) int64 {
	api.count.n.Store(0)
	f()
	return api.count.n.Load()
}

func TestC5_ListingRecipesTakesTheSameNumberOfQueriesWhateverTheirNumber(t *testing.T) {
	api := newAPIAt(t, noonMarch31)
	flour, _, _ := bakery(t, api.client)
	one := queriesOf(api, func() { getList(t, api.client, "/recipes") })
	for i := 0; i < 11; i++ {
		createRecipe(t, api.client, map[string]any{
			"name": fmt.Sprintf("Receta %d", i), "description": "x", "yield": 4, "yield_unit": "porciones",
			"ingredients": []map[string]any{{"ingredient_id": flour, "quantity": 100, "unit": "gr"}},
		})
	}
	many := queriesOf(api, func() {
		if got := getList(t, api.client, "/recipes"); len(got) != 13 {
			t.Fatalf("recipes = %d, want 13", len(got))
		}
	})
	if many != one || many > 3 {
		t.Errorf("queries: %d for 2 recipes and %d for 13; want the same number and at most 3", one, many)
	}
}

func getList(t *testing.T, api *client, path string) []map[string]any {
	t.Helper()
	status, raw := api.do("GET", path, nil)
	if status != http.StatusOK {
		t.Fatalf("GET %s: %d %s", path, status, raw)
	}
	return decodeList(t, raw)
}

func TestC5_AQuotationQueriesOncePerRecipeNotPerLine(t *testing.T) {
	api := newAPIAt(t, noonMarch31)
	_, cake, pie := bakery(t, api.client)
	lines := func(n int, recipe string) []map[string]any {
		var ls []map[string]any
		for i := 0; i < n; i++ {
			ls = append(ls, ql(recipe, 1, ""))
		}
		return ls
	}
	one := queriesOf(api, func() { createQuote(t, api.client, quote("Ana", 0, lines(1, cake)...)) })
	twenty := queriesOf(api, func() { createQuote(t, api.client, quote("Ana", 0, lines(20, cake)...)) })
	if twenty > one+2 {
		t.Errorf("queries: %d for 1 line and %d for 20 lines of the same recipe; want almost the same", one, twenty)
	}
	two := queriesOf(api, func() { createQuote(t, api.client, quote("Ana", 0, append(lines(10, cake), lines(10, pie)...)...)) })
	if two > one+8 {
		t.Errorf("queries: %d for 20 lines of 2 recipes against %d for 1 line; they should follow the recipes, not the lines", two, one)
	}
}

// --- C6 ---

// removingStore is a photo store that also removes: it hands out one URL per upload and remembers
// what was asked to remove.
type removingStore struct {
	mu        sync.Mutex
	next      int
	removed   []string
	removeErr error
}

func (s *removingStore) UploadRecipePhoto(_ context.Context, recipeID, _ string, _ []byte) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	return fmt.Sprintf("https://photos.example.test/recipes/%s/photo-%d.jpg", recipeID, s.next), nil
}

func (s *removingStore) DeleteRecipePhoto(_ context.Context, url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removed = append(s.removed, url)
	return s.removeErr
}

func (s *removingStore) removedList() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.removed...)
}

func newRemovingAPI(t *testing.T, store app.PhotoUploader) (*client, *pgxpool.Pool, string) {
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
	api := &client{t: t, h: app.New(config.Config{Port: "8080", DatabaseURL: "unused", JWTSecret: goodSecret, DBMaxConns: 4}, pool, store)}
	status, raw := api.do("POST", "/auth/login", map[string]string{"email": testEmail, "password": testPassword})
	if status != http.StatusOK {
		t.Fatalf("login: %d", status)
	}
	api.token, _ = decode(t, raw)["token"].(string)
	_, cake, _ := bakery(t, api)
	return api, admin, cake
}

func upload(t *testing.T, api *client, recipe string) {
	t.Helper()
	jpg := append([]byte("\xff\xd8\xff\xe0"), bytes.Repeat([]byte{1}, 100)...)
	if status, raw := api.upload(recipe, "photo", "a.jpg", "image/jpeg", jpg, api.token); status != http.StatusOK {
		t.Fatalf("upload: %d %s", status, raw)
	}
}

func photoOf(api *client, recipe string) string {
	if p := api.storedPhotoURL(recipe); p != nil {
		return *p
	}
	return ""
}

func TestC6_ReplacingAPhotoRemovesTheOldOne(t *testing.T) {
	store := &removingStore{}
	api, _, recipe := newRemovingAPI(t, store)
	upload(t, api, recipe)
	if got := store.removedList(); len(got) != 0 {
		t.Fatalf("the first photo removed %v, want nothing (there was no previous one)", got)
	}
	first := photoOf(api, recipe)
	upload(t, api, recipe)
	if got := store.removedList(); len(got) != 1 || got[0] != first {
		t.Errorf("removed = %v, want [%s]", got, first)
	}
	if photoOf(api, recipe) == first {
		t.Error("the recipe still points to the old photo")
	}
}

func TestC6_DeletingARecipeRemovesItsPhoto(t *testing.T) {
	store := &removingStore{}
	api, admin, _ := newRemovingAPI(t, store)
	var spare string
	if err := admin.QueryRow(context.Background(), `select id from recipes where name = 'Pie'`).Scan(&spare); err != nil {
		t.Fatal(err)
	}
	upload(t, api, spare)
	url := photoOf(api, spare)
	if status, raw := api.do("DELETE", "/recipes/"+spare, nil); status != http.StatusNoContent {
		t.Fatalf("delete: %d %s", status, raw)
	}
	if got := store.removedList(); len(got) != 1 || got[0] != url {
		t.Errorf("removed = %v, want [%s]", got, url)
	}
}

func TestC6_APhotoStillUsedElsewhereIsKept(t *testing.T) {
	store := &removingStore{}
	api, admin, cake := newRemovingAPI(t, store)
	upload(t, api, cake)
	url := photoOf(api, cake)

	// Another recipe points to the same photo (a scaled copy, for instance).
	if _, err := admin.Exec(context.Background(), `update recipes set photo_url = $1 where name = 'Pie'`, url); err != nil {
		t.Fatal(err)
	}
	upload(t, api, cake) // replaces the photo of the cake
	if got := store.removedList(); len(got) != 0 {
		t.Errorf("removed %v although another recipe uses it", got)
	}
}

func TestC6_APhotoOfASavedQuotationIsKept(t *testing.T) {
	store := &removingStore{}
	api, _, cake := newRemovingAPI(t, store)
	upload(t, api, cake)
	createQuote(t, api, quote("Ana", 0, ql(cake, 1, ""))) // the line copies the photo
	upload(t, api, cake)                                  // replaced afterwards
	if got := store.removedList(); len(got) != 0 {
		t.Errorf("removed %v although a saved quotation shows it", got)
	}
}

func TestC6_AFailedRemovalNeverFailsTheRequest(t *testing.T) {
	store := &removingStore{removeErr: errors.New("storage is down")}
	api, _, cake := newRemovingAPI(t, store)
	upload(t, api, cake)
	upload(t, api, cake) // would remove the first one, and fails
	if got := store.removedList(); len(got) != 1 {
		t.Errorf("removal attempts = %d, want 1", len(got))
	}
}
