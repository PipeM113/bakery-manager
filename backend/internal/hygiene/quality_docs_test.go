package hygiene

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Sprint 13 (C10, C11, C12, C13): dependencies, CI and documentation say what is true.

func TestC10_GoModIsTidy(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go is not on the PATH")
	}
	cmd := exec.Command("go", "mod", "tidy", "-diff")
	cmd.Dir = filepath.Join(repoRoot(t), "backend")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("go mod tidy would change go.mod/go.sum (run it in a commit of its own):\n%.600s", out)
	}
}

func TestC11_CIInstallsPopplerAndRunsTheLint(t *testing.T) {
	ci := readRepoFile(t, ".github", "workflows", "ci.yml")
	if !strings.Contains(ci, "poppler-utils") {
		t.Error("the backend job does not install poppler-utils: the text tests of the quotation PDF would be skipped")
	}
	if !strings.Contains(ci, "npm run lint") {
		t.Error("the frontend job does not run npm run lint")
	}
}

func TestC12_TheXlsxDecisionIsWrittenDownWithItsSource(t *testing.T) {
	fuentes := readRepoFile(t, "docs", "fuentes.md")
	if !strings.Contains(fuentes, "xlsx") {
		t.Error("docs/fuentes.md says nothing about the xlsx package")
	}
	if !strings.Contains(fuentes, "storage-js") {
		t.Error("docs/fuentes.md does not cite the storage-js source of the delete contract")
	}
}

func TestC13_BacklogAndDocsAreUpToDate(t *testing.T) {
	backlog := readRepoFile(t, "docs", "backlog.md")
	for _, stale := range []string{"Última actualización: 2026-10-08", "- [ ] `GetAll` de recetas hace N+1", "- [ ] Límites de tamaño del cuerpo en los endpoints JSON"} {
		if strings.Contains(backlog, stale) {
			t.Errorf("docs/backlog.md still says %q", stale)
		}
	}
	for _, want := range []string{"Sprint 12", "Sprint 13"} {
		if !strings.Contains(backlog, want) {
			t.Errorf("docs/backlog.md does not mention %s", want)
		}
	}
	acceso := readRepoFile(t, "docs", "acceso.md")
	for _, want := range []string{"15 minutos", "Mi cuenta", "contraseña"} {
		if !strings.Contains(acceso, want) {
			t.Errorf("docs/acceso.md does not mention %q", want)
		}
	}
	deploy := readRepoFile(t, "docs", "despliegue.md")
	if !strings.Contains(deploy, "`26`") || !strings.Contains(deploy, "login_attempts") {
		t.Error("docs/despliegue.md does not describe migration 26 (login_attempts)")
	}
}
