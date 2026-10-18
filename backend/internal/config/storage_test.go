package config_test

import (
	"strings"
	"testing"

	"github.com/PipeM113/bakery-manager/internal/config"
)

const (
	storageKey = "sb_secret_TESTONLY_not_a_real_key_0123456789"
	storageURL = "https://abcdefghijklmnop.supabase.co"
)

func storageEnv() map[string]string {
	e := validEnv()
	e["SUPABASE_URL"] = storageURL
	e["SUPABASE_SECRET_KEY"] = storageKey
	e["SUPABASE_STORAGE_BUCKET"] = "recipe-photos"
	return e
}

// C1: without any of the three variables the service starts and photos stay disabled.
func TestC1_NoStorageVariablesMeansPhotosDisabled(t *testing.T) {
	cfg, err := config.Load(env(validEnv()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SupabaseURL != "" || cfg.SupabaseSecretKey != "" || cfg.SupabaseBucket != "" {
		t.Errorf("storage settings should be empty, got url=%q bucket=%q", cfg.SupabaseURL, cfg.SupabaseBucket)
	}
}

// C1: with the three variables, they are loaded; a trailing slash of the URL is dropped.
func TestC1_AllThreeVariablesAreLoaded(t *testing.T) {
	e := storageEnv()
	e["SUPABASE_URL"] = storageURL + "/"

	cfg, err := config.Load(env(e))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SupabaseURL != storageURL || cfg.SupabaseSecretKey != storageKey || cfg.SupabaseBucket != "recipe-photos" {
		t.Errorf("settings not loaded as expected: url=%q bucket=%q", cfg.SupabaseURL, cfg.SupabaseBucket)
	}
}

// C1: one or two of the three is a configuration mistake; the error names what is missing.
func TestC1_PartialConfigurationIsAnErrorNamingTheMissingVariables(t *testing.T) {
	all := []string{"SUPABASE_URL", "SUPABASE_SECRET_KEY", "SUPABASE_STORAGE_BUCKET"}
	for _, missing := range all {
		t.Run("without "+missing, func(t *testing.T) {
			e := storageEnv()
			delete(e, missing)

			_, err := config.Load(env(e))

			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), missing) {
				t.Errorf("the error should name %s: %v", missing, err)
			}
			if strings.Contains(err.Error(), storageKey) {
				t.Error("the error repeats the secret key")
			}
		})
	}
	t.Run("only the key", func(t *testing.T) {
		e := validEnv()
		e["SUPABASE_SECRET_KEY"] = storageKey

		_, err := config.Load(env(e))

		if err == nil || !strings.Contains(err.Error(), "SUPABASE_URL") || !strings.Contains(err.Error(), "SUPABASE_STORAGE_BUCKET") {
			t.Fatalf("the error should name both missing variables: %v", err)
		}
		if strings.Contains(err.Error(), storageKey) {
			t.Error("the error repeats the secret key")
		}
	})
}

// C1: SUPABASE_URL is the project origin: https, a host, nothing else.
func TestC1_InvalidSupabaseURLsAreRejected(t *testing.T) {
	for _, bad := range []string{
		"http://abcdefghijklmnop.supabase.co", // not https
		storageURL + "/storage/v1",            // a path
		storageURL + "?a=b",                   // a query
		"abcdefghijklmnop.supabase.co",        // no scheme
		"not a url",
		"https://" + storageKey + "@x.supabase.co", // credentials inside
	} {
		t.Run(bad, func(t *testing.T) {
			e := storageEnv()
			e["SUPABASE_URL"] = bad

			_, err := config.Load(env(e))

			if err == nil || !strings.Contains(err.Error(), "SUPABASE_URL") {
				t.Fatalf("expected an error naming SUPABASE_URL, got %v", err)
			}
			if strings.Contains(err.Error(), storageKey) {
				t.Error("the error repeats the secret key")
			}
		})
	}
}

// C1: the bucket name goes into URLs; keep it to the characters buckets use.
func TestC1_InvalidBucketNamesAreRejected(t *testing.T) {
	for _, bad := range []string{"a/b", "has space", "UPPER?", "../x", "bu%cket"} {
		t.Run(bad, func(t *testing.T) {
			e := storageEnv()
			e["SUPABASE_STORAGE_BUCKET"] = bad

			_, err := config.Load(env(e))

			if err == nil || !strings.Contains(err.Error(), "SUPABASE_STORAGE_BUCKET") {
				t.Fatalf("expected an error naming SUPABASE_STORAGE_BUCKET, got %v", err)
			}
		})
	}
}

// C1: Cloudinary is gone; its variable has no effect and does not break the start.
func TestC1_TheOldCloudinaryVariableIsIgnored(t *testing.T) {
	e := validEnv()
	e["CLOUDINARY_URL"] = "cloudinary://key:secret@cloud"

	cfg, err := config.Load(env(e))

	if err != nil {
		t.Fatalf("a leftover CLOUDINARY_URL must not break the start: %v", err)
	}
	if cfg.SupabaseURL != "" {
		t.Error("photos must stay disabled")
	}
}
