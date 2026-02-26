package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewRoleUsesRunnerImageOverridesFromConfig(t *testing.T) {
	dir := t.TempDir()
	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	writeCheckFile(t, filepath.Join(wfDir, "ci.yml"), `
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: echo ok
`)
	if err := os.MkdirAll(filepath.Join(dir, ".streep"), 0o755); err != nil {
		t.Fatalf("mkdir .streep: %v", err)
	}
	writeCheckFile(t, filepath.Join(dir, ".streep", "config.yaml"), `
runner_images:
  ubuntu-latest: ghcr.io/custom/ubuntu:act-latest
`)

	var out bytes.Buffer
	if err := executeNewRole([]string{dir}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executeNewRole() error: %v", err)
	}

	actrc, err := os.ReadFile(filepath.Join(dir, ".actrc"))
	if err != nil {
		t.Fatalf("read .actrc: %v", err)
	}
	if !strings.Contains(string(actrc), "-P ubuntu-latest=ghcr.io/custom/ubuntu:act-latest") {
		t.Fatalf("expected custom runner mapping in .actrc, got:\n%s", actrc)
	}
}
