package hygiene

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Sprint 5, rules T1 and T8.

func goSources(t *testing.T, root, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.Walk(filepath.Join(root, dir), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		out[filepath.ToSlash(rel)] = string(raw)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// T1: the program reads the time only through the business clock. The login token has its own
// expiry in real time, which is the one exception.
func TestT1_OnlyTheClockPackageReadsTheTime(t *testing.T) {
	root := repoRoot(t)
	for _, dir := range []string{"backend/internal", "backend/pkg", "backend/api", "backend/cmd"} {
		for path, text := range goSources(t, root, dir) {
			if strings.HasPrefix(path, "backend/internal/shared/clock/") || strings.HasPrefix(path, "backend/internal/auth/") {
				continue
			}
			if strings.Contains(text, "time.Now(") {
				t.Errorf("%s reads time.Now() directly; use the clock", path)
			}
			if strings.Contains(text, "Truncate(24") {
				t.Errorf("%s truncates to 24 hours, which cuts at midnight UTC, not at midnight in Santiago", path)
			}
		}
	}
}

// T1: the time zone database travels inside the program, so it does not depend on the server.
func TestT1_TimeZoneDatabaseIsEmbedded(t *testing.T) {
	root := repoRoot(t)
	var found bool
	for path, text := range goSources(t, root, "backend/internal/shared/clock") {
		if strings.Contains(text, `_ "time/tzdata"`) {
			found = true
			_ = path
		}
	}
	if !found {
		t.Error(`the clock package should import _ "time/tzdata"`)
	}
}

func TestT8_TimeDocExists(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "docs", "tiempo.md"))
	if err != nil {
		t.Fatalf("docs/tiempo.md is missing: %v", err)
	}
	for _, want := range []string{"America/Santiago", "UTC", "reloj", "primer día del mes", "TIMESTAMPTZ"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("docs/tiempo.md should mention %q", want)
		}
	}
}
