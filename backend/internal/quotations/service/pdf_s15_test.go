package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// Sprint 15, P2: the PDF only downloads photos that live in the public bucket of the project (the address
// the app itself stored), and it never follows a redirect to somewhere else.

func countingServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	inner := photoServer(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/redirect.jpg", func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Redirect(w, r, inner.URL+"/jpeg-named.png", http.StatusFound)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		resp, err := http.Get(inner.URL + strings.TrimPrefix(r.URL.Path, "/bucket"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		buf := make([]byte, 8<<20)
		n, _ := resp.Body.Read(buf)
		w.Write(buf[:n]) //nolint:errcheck
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &hits
}

func TestP2_APhotoOfTheBucketIsDrawn(t *testing.T) {
	srv, _ := countingServer(t)
	base := images(raw(t, sample(line("Torta", 1, "recipe", 1500, 1500))))
	data := sample(withPhoto(line("Torta", 1, "recipe", 1500, 1500), srv.URL+"/bucket/jpeg-named.png"))
	data.PhotoBase = srv.URL + "/bucket/"
	if got := images(raw(t, data)); got != base+1 {
		t.Errorf("%d images, want %d (the logo and the photo)", got, base+1)
	}
}

func TestP2_APhotoOfAnotherAddressIsNeverRequested(t *testing.T) {
	srv, hits := countingServer(t)
	base := images(raw(t, sample(line("Torta", 1, "recipe", 1500, 1500))))
	for label, photoBase := range map[string]string{
		"another host":                "https://abc.supabase.co/storage/v1/object/public/recipe-photos/",
		"another bucket":              srv.URL + "/other-bucket/",
		"no bucket (none configured)": "",
	} {
		data := sample(withPhoto(line("Torta", 1, "recipe", 1500, 1500), srv.URL+"/bucket/jpeg-named.png"), line("Pie", 1, "recipe", 1000, 1000))
		data.PhotoBase = photoBase
		out := raw(t, data)
		if got := images(out); got != base {
			t.Errorf("%s: %d images, want %d (the line goes without photo)", label, got, base)
		}
	}
	if got := hits.Load(); got != 0 {
		t.Errorf("the server received %d requests, want none", got)
	}
}

func TestP2_ARedirectIsNotFollowed(t *testing.T) {
	srv, _ := countingServer(t)
	base := images(raw(t, sample(line("Torta", 1, "recipe", 1500, 1500))))
	data := sample(withPhoto(line("Torta", 1, "recipe", 1500, 1500), srv.URL+"/redirect.jpg"))
	data.PhotoBase = srv.URL + "/"
	if got := images(raw(t, data)); got != base {
		t.Errorf("%d images, want %d: a redirect to another address must not be followed", got, base)
	}
}

func TestP2_ADotSegmentThatClimbsOutOfTheBucketIsNotRequested(t *testing.T) {
	srv, hits := countingServer(t)
	base := images(raw(t, sample(line("Torta", 1, "recipe", 1500, 1500))))
	for _, path := range []string{"/bucket/../jpeg-named.png", "/bucket/%2e%2e/jpeg-named.png", "/bucket/%2E%2E/jpeg-named.png"} {
		data := sample(withPhoto(line("Torta", 1, "recipe", 1500, 1500), srv.URL+path))
		data.PhotoBase = srv.URL + "/bucket/"
		if got := images(raw(t, data)); got != base {
			t.Errorf("%s: %d images, want %d", path, got, base)
		}
	}
	if got := hits.Load(); got != 0 {
		t.Errorf("the server received %d requests, want none", got)
	}
}
