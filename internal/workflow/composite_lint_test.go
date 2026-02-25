package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLintCompositeActionDir(t *testing.T) {
	dir := t.TempDir()
	actionsDir := filepath.Join(dir, ".github", "actions")
	if err := os.MkdirAll(filepath.Join(actionsDir, "bad"), 0o755); err != nil {
		t.Fatalf("mkdir bad: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(actionsDir, "good"), 0o755); err != nil {
		t.Fatalf("mkdir good: %v", err)
	}

	writeFile(t, filepath.Join(actionsDir, "bad", "action.yml"), `
runs:
  using: composite
  steps:
    - run: |
        if [[ -f file ]]; then
          echo ok
        fi
`)

	writeFile(t, filepath.Join(actionsDir, "good", "action.yml"), `
runs:
  using: composite
  steps:
    - shell: bash
      run: |
        if [[ -f file ]]; then
          echo ok
        fi
`)

	issues, err := LintCompositeActionDir(actionsDir)
	if err != nil {
		t.Fatalf("LintCompositeActionDir() error: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d (%+v)", len(issues), issues)
	}
	if issues[0].Rule != "composite-shell-safety" {
		t.Fatalf("unexpected rule: %+v", issues[0])
	}
	if !strings.Contains(issues[0].File, "bad/action.yml") {
		t.Fatalf("expected bad action file path, got: %s", issues[0].File)
	}
}
