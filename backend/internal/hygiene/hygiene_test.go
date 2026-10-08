// Package hygiene holds checks about the repository itself, not about the program.
package hygiene

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Skipf("not inside a git work tree: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func git(t *testing.T, root string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err == nil {
		return string(out), 0
	}
	if exit, ok := err.(*exec.ExitError); ok {
		return string(out), exit.ExitCode()
	}
	t.Fatalf("git %v: %v", args, err)
	return "", -1
}

// AC5: secrets and build output must be ignored by git.
func TestAC5_SensitiveAndGeneratedPathsAreIgnored(t *testing.T) {
	root := repoRoot(t)
	for _, path := range []string{
		"backend/.env",
		".env",
		"backend/.env.local",
		"tmp/build-errors.log",
		"backend/tmp/main",
	} {
		// --no-index: judge the patterns alone, even for a file that is already tracked.
		if _, code := git(t, root, "check-ignore", "-q", "--no-index", path); code != 0 {
			t.Errorf("%s is not covered by .gitignore", path)
		}
	}
}

// AC5: the example file must stay tracked, or nobody learns which variables exist.
func TestAC5_EnvExampleIsTracked(t *testing.T) {
	root := repoRoot(t)
	out, _ := git(t, root, "ls-files", "backend/.env.example")
	if strings.TrimSpace(out) == "" {
		t.Fatal("backend/.env.example is not tracked")
	}
	if _, code := git(t, root, "check-ignore", "-q", "backend/.env.example"); code == 0 {
		t.Fatal("backend/.env.example is ignored")
	}
}

// AC5: generated output must not be committed (tmp/build-errors.log is today).
func TestAC5_NoGeneratedFilesAreTracked(t *testing.T) {
	root := repoRoot(t)
	out, _ := git(t, root, "ls-files", "tmp", "backend/tmp")
	if strings.TrimSpace(out) != "" {
		t.Fatalf("generated files are tracked:\n%s", out)
	}
}

// AC5: the example file only holds placeholders, never real values.
func TestAC5_EnvExampleHasOnlyPlaceholders(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "backend", ".env.example"))
	if err != nil {
		t.Fatal(err)
	}
	placeholder := regexp.MustCompile(`(?i)(your_|<[^>]+>|user:password@host|^$)`)
	secretKeys := map[string]bool{"DATABASE_URL": true, "JWT_SECRET": true, "CLOUDINARY_URL": true}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			t.Errorf("line without =: %q", key)
			continue
		}
		if secretKeys[key] && !placeholder.MatchString(value) {
			// The value is not printed: it may be a real secret.
			t.Errorf("%s does not look like a placeholder", key)
		}
	}
}
