package policy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckDirFindings(t *testing.T) {
	dir := t.TempDir()
	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wfDir, "ci.yml"), []byte(`
on:
  pull_request_target:
permissions: write-all
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
`), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	findings, err := CheckDir(dir)
	if err != nil {
		t.Fatalf("CheckDir() error: %v", err)
	}
	if len(findings) < 3 {
		t.Fatalf("expected at least 3 findings, got %d (%+v)", len(findings), findings)
	}
}

func TestCheckDirWriteAllInMapping(t *testing.T) {
	dir := t.TempDir()
	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wfDir, "ci.yml"), []byte(`
on: [push]
permissions:
  contents: write-all
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@abc1234abc1234abc1234abc1234abc1234abc12
`), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	findings, err := CheckDir(dir)
	if err != nil {
		t.Fatalf("CheckDir() error: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.Rule == "write-all-permissions" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected write-all-permissions finding for mapping node, got %+v", findings)
	}
}

func TestCheckDirPullRequestTargetInSequence(t *testing.T) {
	dir := t.TempDir()
	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wfDir, "ci.yml"), []byte(`
on: [pull_request_target]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@abc1234abc1234abc1234abc1234abc1234abc12
`), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	findings, err := CheckDir(dir)
	if err != nil {
		t.Fatalf("CheckDir() error: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.Rule == "pull-request-target" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected pull-request-target finding for sequence item, got %+v", findings)
	}
}

func TestCheckDirInvalidPolicyConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".github", "workflows"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".streep"), 0o755); err != nil {
		t.Fatalf("mkdir .streep: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".streep", "policy.yaml"), []byte(":\tinvalid:\n\tyaml"), 0o644); err != nil {
		t.Fatalf("write bad config: %v", err)
	}

	_, err := CheckDir(dir)
	if err == nil {
		t.Fatal("expected error for invalid policy config, got nil")
	}
}

func TestCheckDirRespectsConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".github", "workflows"), 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".github", "workflows", "ci.yml"), []byte(`
on: [pull_request_target]
permissions: write-all
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
`), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".streep"), 0o755); err != nil {
		t.Fatalf("mkdir .streep: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".streep", "policy.yaml"), []byte(`
rules:
  write_all_permissions: false
  pull_request_target: false
`), 0o644); err != nil {
		t.Fatalf("write policy config: %v", err)
	}

	findings, err := CheckDir(dir)
	if err != nil {
		t.Fatalf("CheckDir() error: %v", err)
	}
	for _, f := range findings {
		if f.Rule == "write-all-permissions" || f.Rule == "pull-request-target" {
			t.Fatalf("expected configured rules disabled, got finding: %+v", f)
		}
	}
}
