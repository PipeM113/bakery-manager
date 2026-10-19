package storage_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PipeM113/bakery-manager/internal/storage"
)

const (
	testKey    = "sb_secret_TESTONLY_not_a_real_key_0123456789"
	testBucket = "recipe-photos"
	recipeID   = "0b6a2a52-1d3c-4a53-8d3f-5c0f6e3a9a11"
)

// fakeStorage stands in for Supabase Storage and records what the client sent.
type fakeStorage struct {
	mu       sync.Mutex
	requests []recorded
	status   int
	body     string
	delay    time.Duration
}

type recorded struct {
	method, path string
	header       http.Header
	body         []byte
}

func (f *fakeStorage) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.requests = append(f.requests, recorded{r.Method, r.URL.Path, r.Header.Clone(), raw})
		status, body, delay := f.status, f.body, f.delay
		f.mu.Unlock()
		time.Sleep(delay)
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	})
}

func (f *fakeStorage) last(t *testing.T) recorded {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.requests) == 0 {
		t.Fatal("the fake storage received no request")
	}
	return f.requests[len(f.requests)-1]
}

func (f *fakeStorage) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
}

func newClient(t *testing.T, f *fakeStorage) (*storage.Client, string) {
	t.Helper()
	srv := httptest.NewServer(f.handler())
	t.Cleanup(srv.Close)
	return storage.New(storage.Config{URL: srv.URL, SecretKey: testKey, Bucket: testBucket}, srv.Client()), srv.URL
}

// C2: the client talks to the Storage REST API with the secret key and uploads the bytes as they are.
// The endpoint, the headers and the response shape are the ones to confirm with the real project
// (rule C6): if the real API disagrees, these expectations change, with evidence and approval.
func TestC2_UploadSendsTheFileToTheBucketWithTheSecretKey(t *testing.T) {
	f := &fakeStorage{body: `{"Key":"recipe-photos/x"}`}
	c, _ := newClient(t, f)
	data := []byte("\x89PNG\r\n\x1a\nfake-png-body")

	if _, err := c.UploadRecipePhoto(context.Background(), recipeID, "image/png", data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := f.last(t)
	if got.method != http.MethodPost {
		t.Errorf("method = %s, want POST", got.method)
	}
	prefix := "/storage/v1/object/" + testBucket + "/recipes/" + recipeID + "/"
	if !strings.HasPrefix(got.path, prefix) {
		t.Errorf("path = %q, want it to start with %q", got.path, prefix)
	}
	if got.header.Get("apikey") != testKey {
		t.Error("the secret key must travel in the apikey header")
	}
	if got.header.Get("Authorization") != "Bearer "+testKey {
		t.Error("the secret key must also travel as a bearer token")
	}
	if got.header.Get("Content-Type") != "image/png" {
		t.Errorf("Content-Type = %q, want image/png", got.header.Get("Content-Type"))
	}
	if !bytes.Equal(got.body, data) {
		t.Error("the body must be the file, byte for byte")
	}
	if got.header.Get("x-upsert") != "" {
		t.Error("no upsert: every upload gets its own name")
	}
}

// C2: the returned URL is the public one, and its name keeps the extension that
// the PDF code reads to decide the image type.
func TestC2_ReturnsThePublicURLWithTheRightExtension(t *testing.T) {
	cases := map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp"}
	for contentType, ext := range cases {
		t.Run(contentType, func(t *testing.T) {
			f := &fakeStorage{}
			c, base := newClient(t, f)

			got, err := c.UploadRecipePhoto(context.Background(), recipeID, contentType, []byte("data"))
			if err != nil {
				t.Fatal(err)
			}

			want := regexp.MustCompile("^" + regexp.QuoteMeta(base+"/storage/v1/object/public/"+testBucket+"/recipes/"+recipeID+"/") + `[A-Za-z0-9_-]+` + regexp.QuoteMeta(ext) + "$")
			if !want.MatchString(got) {
				t.Errorf("URL = %q, want <url>/storage/v1/object/public/%s/recipes/%s/<name>%s", got, testBucket, recipeID, ext)
			}
			if !strings.HasSuffix(f.last(t).path, ext) {
				t.Errorf("the uploaded path %q does not end in %s", f.last(t).path, ext)
			}
		})
	}
}

// C2: a new name per upload, so a replaced photo is never served from the old cache.
func TestC2_EveryUploadGetsAFreshName(t *testing.T) {
	f := &fakeStorage{}
	c, _ := newClient(t, f)

	first, err1 := c.UploadRecipePhoto(context.Background(), recipeID, "image/jpeg", []byte("a"))
	second, err2 := c.UploadRecipePhoto(context.Background(), recipeID, "image/jpeg", []byte("a"))

	if err1 != nil || err2 != nil {
		t.Fatalf("errors: %v %v", err1, err2)
	}
	if first == second {
		t.Fatal("two uploads of the same photo produced the same URL")
	}
}

// C2: a failure from Storage is an error that neither carries the key nor repeats the response.
func TestC2_StorageErrorsDoNotLeakTheKeyOrTheResponse(t *testing.T) {
	f := &fakeStorage{status: http.StatusForbidden, body: `{"message":"bad key ` + testKey + `"}`}
	c, _ := newClient(t, f)

	_, err := c.UploadRecipePhoto(context.Background(), recipeID, "image/png", []byte("data"))

	if err == nil {
		t.Fatal("expected an error for a 403")
	}
	if strings.Contains(err.Error(), testKey) || strings.Contains(err.Error(), "bad key") {
		t.Errorf("the error leaks the key or the response: %v", err)
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("the error should say the status code to help diagnose: %v", err)
	}
}

// C2: an unreachable Storage and a slow Storage both end in an error, not a hang.
func TestC2_TimeoutsAndNetworkFailuresAreErrors(t *testing.T) {
	t.Run("deadline", func(t *testing.T) {
		f := &fakeStorage{delay: 2 * time.Second}
		c, _ := newClient(t, f)
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		start := time.Now()
		_, err := c.UploadRecipePhoto(ctx, recipeID, "image/png", []byte("data"))

		if err == nil {
			t.Fatal("expected an error")
		}
		if time.Since(start) > time.Second {
			t.Errorf("took %v: the context deadline was ignored", time.Since(start))
		}
	})
	t.Run("connection refused", func(t *testing.T) {
		c := storage.New(storage.Config{URL: "http://127.0.0.1:1", SecretKey: testKey, Bucket: testBucket}, nil)

		_, err := c.UploadRecipePhoto(context.Background(), recipeID, "image/png", []byte("data"))

		if err == nil {
			t.Fatal("expected an error")
		}
		if strings.Contains(err.Error(), testKey) {
			t.Error("the error leaks the key")
		}
	})
}

// C2: the recipe id and the content type end up in the object path; anything that could change
// the path or that is not an accepted image is refused before any request leaves.
func TestC2_RefusesUnsafeInputsWithoutCallingStorage(t *testing.T) {
	cases := []struct {
		name, id, contentType string
	}{
		{"path traversal", "../other", "image/png"},
		{"slash in the id", "a/b", "image/png"},
		{"empty id", "", "image/png"},
		{"query characters", "abc?x=1", "image/png"},
		{"gif", recipeID, "image/gif"},
		{"html", recipeID, "text/html"},
		{"empty type", recipeID, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeStorage{}
			c, _ := newClient(t, f)

			_, err := c.UploadRecipePhoto(context.Background(), tc.id, tc.contentType, []byte("data"))

			if err == nil {
				t.Fatal("expected an error")
			}
			if f.count() != 0 {
				t.Errorf("Storage received %d requests, want none", f.count())
			}
		})
	}
}

func TestC2_EmptyFileIsRefused(t *testing.T) {
	f := &fakeStorage{}
	c, _ := newClient(t, f)

	_, err := c.UploadRecipePhoto(context.Background(), recipeID, "image/png", nil)

	if err == nil || f.count() != 0 {
		t.Fatalf("err = %v, requests = %d: an empty file must be refused before any request", err, f.count())
	}
}

// Sprint 13, C6: removing a photo. The contract is the one of the official storage-js client
// (docs/fuentes.md): DELETE /storage/v1/object/<bucket> with {"prefixes": [<paths>]}.
func TestC6_DeleteSendsTheObjectPathToTheBucket(t *testing.T) {
	f := &fakeStorage{body: `[{"name":"x"}]`}
	c, base := newClient(t, f)
	url := base + "/storage/v1/object/public/" + testBucket + "/recipes/" + recipeID + "/abc123.jpg"

	if err := c.DeleteRecipePhoto(context.Background(), url); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := f.last(t)
	if got.method != http.MethodDelete || got.path != "/storage/v1/object/"+testBucket {
		t.Errorf("request = %s %s, want DELETE /storage/v1/object/%s", got.method, got.path, testBucket)
	}
	if got.header.Get("apikey") != testKey || got.header.Get("Authorization") != "Bearer "+testKey {
		t.Error("the secret key must travel in the apikey and Authorization headers")
	}
	if got.header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got.header.Get("Content-Type"))
	}
	if want := `{"prefixes":["recipes/` + recipeID + `/abc123.jpg"]}`; strings.TrimSpace(string(got.body)) != want {
		t.Errorf("body = %s, want %s", got.body, want)
	}
}

func TestC6_DeleteIgnoresWhatIsNotOurs(t *testing.T) {
	f := &fakeStorage{}
	c, base := newClient(t, f)
	for _, url := range []string{
		"https://other.example.test/storage/v1/object/public/" + testBucket + "/recipes/" + recipeID + "/a.jpg", // another host
		base + "/storage/v1/object/public/other-bucket/recipes/" + recipeID + "/a.jpg",                          // another bucket
		base + "/storage/v1/object/public/" + testBucket + "/recipes/../secrets/a.jpg",                          // climbs out
		base + "/storage/v1/object/public/" + testBucket + "/other/a.jpg",                                       // not a recipe photo
		"", "not a url",
	} {
		if err := c.DeleteRecipePhoto(context.Background(), url); err != nil {
			t.Errorf("%q: unexpected error %v (a foreign address is ignored, not an error)", url, err)
		}
	}
	if f.count() != 0 {
		t.Errorf("%d request(s) were sent for addresses that are not ours", f.count())
	}
}

func TestC6_DeleteErrorsNeverCarryTheKey(t *testing.T) {
	f := &fakeStorage{status: http.StatusInternalServerError, body: "boom " + testKey}
	c, base := newClient(t, f)
	err := c.DeleteRecipePhoto(context.Background(), base+"/storage/v1/object/public/"+testBucket+"/recipes/"+recipeID+"/a.jpg")
	if err == nil {
		t.Fatal("a 500 must be an error")
	}
	if strings.Contains(err.Error(), testKey) {
		t.Error("the error carries the secret key")
	}
}
