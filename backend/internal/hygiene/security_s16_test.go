package hygiene

import (
	"strings"
	"testing"
)

// Security audit of 2026-11 (docs/auditoria-seguridad-2026-11.md), points 5, 17 and 20.

// S17: the queries name their columns; a table that later gets a private column does not leak it.
func TestS17_NoQueryReturnsEveryColumn(t *testing.T) {
	root := repoRoot(t)
	for path, text := range goSources(t, root, "backend/internal") {
		upper := strings.ToUpper(text)
		if strings.Contains(upper, "RETURNING *") {
			t.Errorf("%s uses RETURNING *: name the columns", path)
		}
	}
}

// S5: the runbook tells to ask for TLS in the database connection.
func TestS5_RunbookAsksForTLSInTheDatabaseURL(t *testing.T) {
	text := readRepoFile(t, "docs", "despliegue.md")
	if !strings.Contains(text, "sslmode=require") {
		t.Error("docs/despliegue.md should say that DATABASE_URL ends in ?sslmode=require")
	}
	if !strings.Contains(readRepoFile(t, "backend", ".env.example"), "sslmode=require") {
		t.Error("backend/.env.example should show sslmode=require in DATABASE_URL")
	}
}
