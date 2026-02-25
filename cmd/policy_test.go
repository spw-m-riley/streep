package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPolicyShowsHelp(t *testing.T) {
	var out bytes.Buffer
	if err := executePolicy([]string{"--help"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executePolicy() error: %v", err)
	}
	if !strings.Contains(out.String(), "streep policy check") {
		t.Fatalf("expected help output, got: %q", out.String())
	}
}

func TestPolicyCheckFindings(t *testing.T) {
	dir := t.TempDir()
	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	writeCheckFile(t, filepath.Join(wfDir, "ci.yml"), `
on: [pull_request_target]
permissions: write-all
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
`)

	var out bytes.Buffer
	if err := executePolicy([]string{"check", dir}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("executePolicy(check) error: %v", err)
	}
	got := out.String()
	for _, want := range []string{"write-all-permissions", "pull-request-target", "unpinned-action"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in output, got:\n%s", want, got)
		}
	}
}
