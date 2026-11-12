package config_test

import (
	"testing"

	"github.com/PipeM113/bakery-manager/internal/config"
)

// Sprint 15, P2: the address where the photos of the project are served.
func TestP2_PhotoBaseIsThePublicAddressOfTheBucket(t *testing.T) {
	vars := validEnv()
	vars["SUPABASE_URL"] = "https://abc.supabase.co"
	vars["SUPABASE_SECRET_KEY"] = "sb_secret_x"
	vars["SUPABASE_STORAGE_BUCKET"] = "recipe-photos"
	cfg, err := config.Load(env(vars))
	if err != nil {
		t.Fatal(err)
	}
	if want := "https://abc.supabase.co/storage/v1/object/public/recipe-photos/"; cfg.PhotoBase() != want {
		t.Errorf("PhotoBase() = %q, want %q", cfg.PhotoBase(), want)
	}
	plain, err := config.Load(env(validEnv()))
	if err != nil {
		t.Fatal(err)
	}
	if plain.PhotoBase() != "" {
		t.Errorf("without storage PhotoBase() = %q, want empty (no photo is downloaded)", plain.PhotoBase())
	}
}
