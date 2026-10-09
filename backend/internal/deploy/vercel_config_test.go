package deploy_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// B6: backend/vercel.json sends every path to a function that exists, and keeps secrets out.
func TestB6_VercelConfigRoutesEverythingToTheFunction(t *testing.T) {
	raw, err := os.ReadFile(backendPath("vercel.json"))
	if err != nil {
		t.Fatalf("backend/vercel.json is missing: %v", err)
	}
	var cfg struct {
		Rewrites []struct {
			Source      string `json:"source"`
			Destination string `json:"destination"`
		} `json:"rewrites"`
		Functions map[string]struct {
			MaxDuration int `json:"maxDuration"`
		} `json:"functions"`
		Env map[string]any `json:"env"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("backend/vercel.json is not valid JSON: %v", err)
	}

	if len(cfg.Rewrites) == 0 {
		t.Fatal("there must be a rewrite that sends every path to the function")
	}
	catchAll := false
	for _, r := range cfg.Rewrites {
		if r.Source == "/(.*)" {
			catchAll = true
		}
		dest := strings.TrimPrefix(r.Destination, "/")
		_, errFile := os.Stat(backendPath(dest + ".go"))
		_, errIndex := os.Stat(backendPath(dest, "index.go"))
		if errFile != nil && errIndex != nil {
			t.Errorf("the rewrite destination %q does not match any function file", r.Destination)
		}
	}
	if !catchAll {
		t.Error("no rewrite catches every path with the source /(.*)")
	}

	fn, ok := cfg.Functions["api/index.go"]
	if !ok {
		t.Fatal("functions must configure api/index.go")
	}
	// 300 s is the verified maximum for Hobby (docs/fuentes.md).
	if fn.MaxDuration < 1 || fn.MaxDuration > 300 {
		t.Errorf("maxDuration = %d, want between 1 and 300", fn.MaxDuration)
	}

	if len(cfg.Env) != 0 {
		t.Error("vercel.json must not hold environment values: they go in the Vercel dashboard")
	}
}
