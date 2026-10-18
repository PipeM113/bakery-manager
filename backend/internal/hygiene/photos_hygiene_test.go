package hygiene

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// C4: Cloudinary is gone from the code, the dependencies and the docs.
func TestC4_CloudinaryIsGone(t *testing.T) {
	root := repoRoot(t)

	if _, err := os.Stat(filepath.Join(root, "backend", "internal", "cloudinary")); err == nil {
		t.Error("backend/internal/cloudinary must be deleted")
	}
	for _, rel := range []string{"backend/go.mod", "backend/go.sum", "backend/.env.example", "docs/despliegue.md", "README.md"} {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.ToLower(string(data)), "cloudinary") {
			t.Errorf("%s still mentions cloudinary", rel)
		}
	}
	out, _ := git(t, root, "ls-files", "backend", "frontend/src")
	for _, rel := range strings.Fields(out) {
		if !strings.HasSuffix(rel, ".go") && !strings.HasSuffix(rel, ".ts") && !strings.HasSuffix(rel, ".tsx") {
			continue
		}
		// These two tests name the word on purpose (the second checks that the old variable is ignored).
		if strings.HasSuffix(rel, "photos_hygiene_test.go") || strings.HasSuffix(rel, "internal/config/storage_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(string(data)), "cloudinary") {
			t.Errorf("%s still mentions cloudinary", rel)
		}
	}
}

// C5: the example file documents the three variables with placeholders only.
func TestC5_EnvExampleDocumentsTheStorageVariablesWithPlaceholders(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "backend", ".env.example"))
	if err != nil {
		t.Fatal(err)
	}
	placeholder := regexp.MustCompile(`(?i)(your_|<[^>]+>)`)
	found := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok || strings.HasPrefix(key, "#") {
			continue
		}
		switch key {
		case "SUPABASE_URL", "SUPABASE_SECRET_KEY", "SUPABASE_STORAGE_BUCKET":
			found[key] = true
			if !placeholder.MatchString(value) {
				// The value is not printed: it may be a real secret.
				t.Errorf("%s does not look like a placeholder", key)
			}
		}
	}
	for _, key := range []string{"SUPABASE_URL", "SUPABASE_SECRET_KEY", "SUPABASE_STORAGE_BUCKET"} {
		if !found[key] {
			t.Errorf("backend/.env.example must document %s", key)
		}
	}
}

// C5: no tracked file holds something shaped like a real Supabase secret key.
func TestC5_NoSupabaseSecretKeyInTrackedFiles(t *testing.T) {
	root := repoRoot(t)
	// A real key is sb_secret_ followed by a long run of key characters. Placeholders such as
	// sb_secret_<...> and the obviously fake test value are fine.
	real := regexp.MustCompile(`sb_secret_[A-Za-z0-9_-]{20,}`)
	fake := regexp.MustCompile(`TESTONLY|not_a_real|LEAK_ME|example|placeholder`)

	out, _ := git(t, root, "ls-files")
	for _, rel := range strings.Fields(out) {
		if strings.HasPrefix(rel, "frontend/node_modules") || strings.HasSuffix(rel, ".lock") || strings.HasSuffix(rel, "package-lock.json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil || len(data) > 2<<20 {
			continue
		}
		for _, m := range real.FindAllString(string(data), -1) {
			if !fake.MatchString(m) {
				t.Errorf("%s holds something shaped like a Supabase secret key", rel)
				break
			}
		}
	}
}

// C5/C6: the runbook explains the bucket, the dedicated key and the variables, and keeps the key out of the repo.
func TestC5_RunbookCoversTheStorageSetup(t *testing.T) {
	text := readRepoFile(t, "docs", "despliegue.md")

	for _, want := range []string{
		"SUPABASE_URL", "SUPABASE_SECRET_KEY", "SUPABASE_STORAGE_BUCKET",
		"recipe-photos", "Allowed MIME", "4 MB", "manager-backend",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the runbook should mention %q", want)
		}
	}
	assertOnlyPlaceholderPasswords(t, "docs/despliegue.md", text)
}
