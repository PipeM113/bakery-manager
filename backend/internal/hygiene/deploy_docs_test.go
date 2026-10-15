package hygiene

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func readRepoFile(t *testing.T, parts ...string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(append([]string{repoRoot(t)}, parts...)...))
	if err != nil {
		t.Fatalf("%v", err)
	}
	return string(data)
}

// connectionURL captures the password and the host of every postgres:// URL in a text.
var connectionURL = regexp.MustCompile(`postgres(?:ql)?://[^\s:@/]+:([^\s@]+)@([^\s:/?]+)`)

// assertOnlyPlaceholderPasswords accepts a placeholder (<...>, __...__, [...] or the
// literal word "password") or the throwaway credentials of the local test database,
// which only ever lives on this machine. Anything else could be a real secret.
func assertOnlyPlaceholderPasswords(t *testing.T, name, text string) {
	t.Helper()
	for _, m := range connectionURL.FindAllStringSubmatch(text, -1) {
		pw, host := m[1], m[2]
		placeholder := strings.HasPrefix(pw, "<") || strings.HasPrefix(pw, "__") ||
			strings.HasPrefix(pw, "[") || pw == "password"
		localTestDB := host == "127.0.0.1" || host == "localhost"
		if !placeholder && !localTestDB {
			// The value is not printed: it may be a real secret.
			t.Errorf("%s holds a connection string whose password is not a placeholder", name)
		}
	}
}

// B7: the frontend documents the variable it needs, with a placeholder only.
func TestB7_FrontendEnvExampleDocumentsTheAPIURL(t *testing.T) {
	text := readRepoFile(t, "frontend", ".env.example")

	if !regexp.MustCompile(`(?m)^VITE_API_URL=https://\S+$`).MatchString(text) {
		t.Error("frontend/.env.example must hold a line VITE_API_URL=https://<placeholder>")
	}
	if strings.Contains(text, "localhost") {
		t.Error("the example must not point to localhost: that is the silent default this rule removes")
	}
}

// B8: the CI runs the frontend checks and every action is pinned to a full commit SHA.
func TestB8_CIHasAFrontendJobAndPinsEveryAction(t *testing.T) {
	text := readRepoFile(t, ".github", "workflows", "ci.yml")

	for _, want := range []string{"backend-tests:", "frontend-tests:", "setup-node", "npm ci", "VITE_API_URL", "npm test", "npm run build"} {
		if !strings.Contains(text, want) {
			t.Errorf("ci.yml should contain %q", want)
		}
	}
	pinned := regexp.MustCompile(`@[0-9a-f]{40}\b`)
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, "uses:") && !pinned.MatchString(line) {
			t.Errorf("an action is not pinned to a full SHA: %s", strings.TrimSpace(line))
		}
	}
}

// B9: the runbook exists, follows the order of the procedure, and holds no real secret.
func TestB9_RunbookCoversTheStepsInOrderAndHoldsNoSecrets(t *testing.T) {
	text := readRepoFile(t, "docs", "despliegue.md")

	ordered := []string{
		"01_manager_schema.sql", // 1. role and schema
		"migrate",               // 2. migrations
		"hashpassword",          // 3. first user
		"Root Directory",        // 4. the second Vercel project
		"CORS_ORIGINS",          // 5. variables
		"/health",               // 6. verification
		"pg_dump",               // 7. backup
	}
	last := -1
	for _, token := range ordered {
		i := strings.Index(text, token)
		if i < 0 {
			t.Errorf("the runbook never mentions %q", token)
			continue
		}
		if i < last {
			t.Errorf("%q appears before the previous step: the order of the procedure is broken", token)
		}
		last = i
	}

	for _, token := range []string{
		"JWT_SECRET", "DATABASE_URL", "DB_MAX_CONNS", "VITE_API_URL", "openssl rand",
		"Exposed schemas", "go version", "pooler de sesión", "pooler de transacciones",
	} {
		if !strings.Contains(text, token) {
			t.Errorf("the runbook should mention %q", token)
		}
	}
	assertOnlyPlaceholderPasswords(t, "docs/despliegue.md", text)
}

// B9: no document or script that ships with the repo may hold a real connection password.
func TestB9_NoRealConnectionPasswordsInDocsOrSetupScripts(t *testing.T) {
	root := repoRoot(t)
	for _, rel := range []string{
		"README.md", "docs/despliegue.md", "docs/backlog.md", "docs/fuentes.md",
		"backend/.env.example", "backend/db/setup/01_manager_schema.sql", "frontend/.env.example",
	} {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			if rel == "docs/despliegue.md" || rel == "backend/db/setup/01_manager_schema.sql" || rel == "frontend/.env.example" {
				t.Errorf("%s is missing", rel)
			}
			continue
		}
		assertOnlyPlaceholderPasswords(t, rel, string(data))
	}
}
