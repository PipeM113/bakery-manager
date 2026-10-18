package app_test

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"sync"
	"testing"

	"github.com/PipeM113/bakery-manager/internal/app"
	"github.com/PipeM113/bakery-manager/internal/config"
	"github.com/PipeM113/bakery-manager/internal/shared/kernel"
	"github.com/PipeM113/bakery-manager/internal/testsupport/pgtest"
	"golang.org/x/crypto/bcrypt"
)

// Minimal files that net/http's content sniffing recognises as images.
var (
	jpegBytes = append([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0x01}, bytes.Repeat([]byte{0x01}, 64)...)
	pngBytes  = append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0x02}, 64)...)
	webpBytes = append([]byte("RIFF\x24\x00\x00\x00WEBPVP8 "), bytes.Repeat([]byte{0x03}, 64)...)
	gifBytes  = append([]byte("GIF89a"), bytes.Repeat([]byte{0x04}, 64)...)
	htmlBytes = []byte("<html><script>alert(1)</script></html>")
	exeBytes  = append([]byte("MZ\x90\x00"), bytes.Repeat([]byte{0x05}, 64)...)
)

const maxPhotoBytes = 4 << 20

// fakeUploader is the photo store the router talks to in these tests.
type fakeUploader struct {
	mu          sync.Mutex
	calls       int
	recipeID    string
	contentType string
	data        []byte
	url         string
	err         error
}

func (f *fakeUploader) UploadRecipePhoto(_ context.Context, recipeID, contentType string, data []byte) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.recipeID, f.contentType, f.data = recipeID, contentType, append([]byte(nil), data...)
	if f.err != nil {
		return "", f.err
	}
	return f.url, nil
}

func (f *fakeUploader) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// newPhotoAPI is newAPI with a photo store, logged in, plus a recipe to attach photos to.
func newPhotoAPI(t *testing.T, photos app.PhotoUploader) (*client, string) {
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
	api := &client{t: t, h: app.New(config.Config{Port: "8080", DatabaseURL: "unused", JWTSecret: goodSecret, DBMaxConns: 4}, pool, photos)}

	status, raw := api.do("POST", "/auth/login", map[string]string{"email": testEmail, "password": testPassword})
	if status != http.StatusOK {
		t.Fatalf("login: %d %s", status, raw)
	}
	api.token, _ = decode(t, raw)["token"].(string)

	status, raw = api.do("POST", "/ingredients", map[string]any{
		"name": "Harina", "brand": "Marca", "default_unit": "gr", "package_size": 1000, "package_price": 2000,
	})
	if status != http.StatusCreated {
		t.Fatalf("ingredient: %d %s", status, raw)
	}
	ingredientID, _ := decode(t, raw)["id"].(string)
	status, raw = api.do("POST", "/recipes", map[string]any{
		"name": "Bizcocho", "description": "x", "yield": 10, "yield_unit": "porciones",
		"ingredients": []map[string]any{{"ingredient_id": ingredientID, "quantity": 500, "unit": "gr"}},
	})
	if status != http.StatusCreated {
		t.Fatalf("recipe: %d %s", status, raw)
	}
	recipeID, _ := decode(t, raw)["id"].(string)
	return api, recipeID
}

// upload sends a multipart POST with the given file under the given form field. The declared
// content type is what a lying or sloppy client would put in the part header.
func (c *client) upload(recipeID, field, filename, declaredType string, content []byte, token string) (int, []byte) {
	c.t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="`+field+`"; filename="`+filename+`"`)
	h.Set("Content-Type", declaredType)
	part, err := mw.CreatePart(h)
	if err != nil {
		c.t.Fatal(err)
	}
	_, _ = part.Write(content)
	_ = mw.Close()

	req := httptest.NewRequest("POST", "/recipes/"+recipeID+"/photo", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, req)
	return rec.Code, rec.Body.Bytes()
}

func (c *client) storedPhotoURL(recipeID string) *string {
	c.t.Helper()
	status, raw := c.do("GET", "/recipes/"+recipeID, nil)
	if status != http.StatusOK {
		c.t.Fatalf("get recipe: %d %s", status, raw)
	}
	if v, ok := decode(c.t, raw)["photo_url"].(string); ok {
		return &v
	}
	return nil
}

// C3: the file type is decided by its content, never by the name or the declared type.
func TestC3_AcceptsJPEGPNGAndWebPByContent(t *testing.T) {
	cases := []struct {
		name         string
		content      []byte
		declared     string
		wantType     string
		fileNameHint string
	}{
		{"jpeg", jpegBytes, "image/jpeg", "image/jpeg", "foto.jpg"},
		{"png", pngBytes, "image/png", "image/png", "foto.png"},
		{"webp", webpBytes, "image/webp", "image/webp", "foto.webp"},
		{"jpeg declared as octet-stream", jpegBytes, "application/octet-stream", "image/jpeg", "foto"},
		{"png with a misleading name", pngBytes, "image/png", "image/png", "foto.exe"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			up := &fakeUploader{url: "https://files.example.test/recipes/x/photo.img"}
			api, recipeID := newPhotoAPI(t, up)

			status, raw := api.upload(recipeID, "photo", tc.fileNameHint, tc.declared, tc.content, api.token)

			if status != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", status, raw)
			}
			if got := decode(t, raw)["photo_url"]; got != up.url {
				t.Errorf("photo_url = %v, want %s", got, up.url)
			}
			if up.callCount() != 1 || up.contentType != tc.wantType || up.recipeID != recipeID || !bytes.Equal(up.data, tc.content) {
				t.Errorf("the uploader got type=%q recipe=%q (%d calls); want type=%q recipe=%q with the exact bytes",
					up.contentType, up.recipeID, up.callCount(), tc.wantType, recipeID)
			}
			if stored := api.storedPhotoURL(recipeID); stored == nil || *stored != up.url {
				t.Errorf("the recipe should keep %s, has %v", up.url, stored)
			}
		})
	}
}

// C3: whatever is not a JPEG, PNG or WebP is refused, whatever the client claims.
func TestC3_RefusesOtherContentEvenIfItClaimsToBeAnImage(t *testing.T) {
	cases := []struct {
		name    string
		content []byte
	}{
		{"gif", gifBytes}, {"html", htmlBytes}, {"executable", exeBytes}, {"empty", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			up := &fakeUploader{url: "https://files.example.test/x.png"}
			api, recipeID := newPhotoAPI(t, up)

			status, _ := api.upload(recipeID, "photo", "foto.png", "image/png", tc.content, api.token)

			if status != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", status)
			}
			if up.callCount() != 0 {
				t.Error("the uploader must not be called")
			}
			if stored := api.storedPhotoURL(recipeID); stored != nil {
				t.Errorf("the recipe must not change, has %q", *stored)
			}
		})
	}
}

// C3: 4 MB is the limit; the function of Vercel accepts bodies of at most 4.5 MB.
func TestC3_SizeLimitIsFourMegabytes(t *testing.T) {
	big := func(n int) []byte {
		return append(append([]byte(nil), jpegBytes...), bytes.Repeat([]byte{0x07}, n-len(jpegBytes))...)
	}

	t.Run("exactly 4 MB passes", func(t *testing.T) {
		up := &fakeUploader{url: "https://files.example.test/ok.jpg"}
		api, recipeID := newPhotoAPI(t, up)

		status, raw := api.upload(recipeID, "photo", "a.jpg", "image/jpeg", big(maxPhotoBytes), api.token)

		if status != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", status, raw)
		}
	})
	t.Run("one byte more is refused with 413", func(t *testing.T) {
		up := &fakeUploader{url: "https://files.example.test/no.jpg"}
		api, recipeID := newPhotoAPI(t, up)

		status, _ := api.upload(recipeID, "photo", "a.jpg", "image/jpeg", big(maxPhotoBytes+1), api.token)

		if status != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d, want 413", status)
		}
		if up.callCount() != 0 {
			t.Error("the uploader must not be called")
		}
	})
}

// C3: the recipe id goes into the object path, so it must be a UUID.
func TestC3_RecipeIDMustBeAUUID(t *testing.T) {
	up := &fakeUploader{url: "https://files.example.test/x.jpg"}
	api, _ := newPhotoAPI(t, up)

	for _, bad := range []string{"not-a-uuid", "..%2Fother", "1234"} {
		status, _ := api.upload(bad, "photo", "a.jpg", "image/jpeg", jpegBytes, api.token)
		if status != http.StatusBadRequest {
			t.Errorf("id %q: status = %d, want 400", bad, status)
		}
	}
	if up.callCount() != 0 {
		t.Error("the uploader must not be called for an invalid id")
	}
}

// C3: the field must be called "photo"; the route needs a login.
func TestC3_RequiresTheFieldAndALogin(t *testing.T) {
	up := &fakeUploader{url: "https://files.example.test/x.jpg"}
	api, recipeID := newPhotoAPI(t, up)

	if status, _ := api.upload(recipeID, "other", "a.jpg", "image/jpeg", jpegBytes, api.token); status != http.StatusBadRequest {
		t.Errorf("wrong field: status = %d, want 400", status)
	}
	if status, _ := api.upload(recipeID, "photo", "a.jpg", "image/jpeg", jpegBytes, ""); status != http.StatusUnauthorized {
		t.Errorf("no token: status = %d, want 401", status)
	}
	if up.callCount() != 0 {
		t.Error("the uploader must not be called")
	}
}

// C1/C3: without a configured store the route answers 503, as before.
func TestC3_WithoutAPhotoStoreTheRouteIs503(t *testing.T) {
	api, recipeID := newPhotoAPI(t, nil)

	status, _ := api.upload(recipeID, "photo", "a.jpg", "image/jpeg", jpegBytes, api.token)

	if status != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", status)
	}
}

// C2/C3: when the store fails the client gets a generic error, the details stay out of the
// response, and the recipe keeps its previous photo.
func TestC3_StoreFailureIsGenericAndKeepsThePreviousPhoto(t *testing.T) {
	up := &fakeUploader{url: "https://files.example.test/first.jpg"}
	api, recipeID := newPhotoAPI(t, up)
	if status, raw := api.upload(recipeID, "photo", "a.jpg", "image/jpeg", jpegBytes, api.token); status != http.StatusOK {
		t.Fatalf("first upload: %d %s", status, raw)
	}

	up.err = errors.New("storage said no: sb_secret_LEAK_ME")
	status, raw := api.upload(recipeID, "photo", "b.jpg", "image/jpeg", jpegBytes, api.token)

	if status != http.StatusInternalServerError && status != http.StatusBadGateway {
		t.Fatalf("status = %d, want 500 or 502", status)
	}
	if strings.Contains(string(raw), "sb_secret") || strings.Contains(string(raw), "storage said no") {
		t.Errorf("the response leaks the store's error: %s", raw)
	}
	if stored := api.storedPhotoURL(recipeID); stored == nil || *stored != "https://files.example.test/first.jpg" {
		t.Errorf("the recipe should keep its first photo, has %v", stored)
	}
}
